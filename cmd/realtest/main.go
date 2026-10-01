// Command realtest is a money-free real-server smoke test, invoked by the
// manual `.github/workflows/realtest.yml` workflow. It exercises only the
// read-only paths — login (RSA + cookie session), account info (balance +
// round JSON parse), the previous-week winning check, and the lifetime ledger
// aggregate (full scan, then a resume from the gist checkpoint when one is
// configured) — and prints the collected notifications instead of sending
// them. It NEVER purchases lotto, reserves pension, sends Telegram, or writes
// the gist checkpoint, so it is safe to run any time.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/checkpoint"
	"github.com/kadragon/dhlottery-worker/internal/constants"
	"github.com/kadragon/dhlottery-worker/internal/dhlottery"
	"github.com/kadragon/dhlottery-worker/internal/env"
	"github.com/kadragon/dhlottery-worker/internal/format"
	"github.com/kadragon/dhlottery-worker/internal/notify"
)

type smokeClient interface {
	Login() error
	GetAccountInfo() (dhlottery.AccountInfo, error)
	CheckWinning(time.Time) []dhlottery.WinningResult
	AggregateLedgerIncremental(startDate string, now time.Time, prev *checkpoint.Checkpoint) (dhlottery.LedgerSummary, *checkpoint.Checkpoint, bool)
	Collector() *notify.Collector
}

// Indirected for tests.
var (
	validateEnv = env.Validate
	runChecks   = defaultRunChecks
	newClient   = func() smokeClient { return dhlottery.NewClient() }
	nowFn       = time.Now
	// Read-only toward the gist: never saves (enforced by save_ban_test.go).
	loadCheckpoint      = checkpoint.Load
	checkpointResumable = dhlottery.CheckpointResumable
)

func main() {
	os.Exit(run())
}

// run validates the environment then runs the read-only checks. Returns 1 if
// the environment is invalid, otherwise the result of the checks.
func run() int {
	if err := validateEnv(); err != nil {
		fmt.Println("❌ env:", err)
		return 1
	}
	return runChecks()
}

func defaultRunChecks() int {
	c := newClient()

	fmt.Println("== 1) Login (RSA + cookie session) ==")
	if err := c.Login(); err != nil {
		fmt.Println("❌ login failed:", err)
		return 1
	}
	fmt.Println("✅ login ok")

	fmt.Println("\n== 2) GetAccountInfo (balance + round JSON parse) ==")
	if info, err := c.GetAccountInfo(); err != nil {
		fmt.Println("❌ account info failed:", err)
		return 1
	} else {
		fmt.Printf("✅ balance=%d KRW  currentRound=%d\n", info.Balance, info.CurrentRound)
	}

	fmt.Println("\n== 3) CheckWinning (previous week, lotto + pension) ==")
	wins := c.CheckWinning(nowFn())
	if len(wins) == 0 {
		fmt.Println("✅ checked — no wins in the previous week")
	}
	for _, w := range wins {
		fmt.Printf("  🎉 %s round=%d rank=%d prize=%d\n", w.Product, w.RoundNumber, w.Rank, w.PrizeAmount)
	}

	fmt.Println("\n== 4) AggregateLedgerIncremental (production path, no checkpoint → full scan; lifetime cumulative — verify vs real account) ==")
	startDate := constants.DefaultLedgerStartDate
	if v, err := env.Get("LEDGER_START_DATE"); err == nil {
		startDate = v
	}
	now := nowFn()
	s, next, ok := c.AggregateLedgerIncremental(startDate, now, nil)
	if !ok {
		fmt.Printf("  ❌ ledger aggregate lookup failed (start=%s)\n", startDate)
	} else {
		net := s.CumulativeWinning - s.CumulativePurchase
		fmt.Printf("  start=%s  누적 구매=%s  누적 당첨=%s  결산=%s\n",
			startDate, format.Currency(s.CumulativePurchase), format.Currency(s.CumulativeWinning), format.Currency(net))
		if next != nil {
			fmt.Printf("  settled through=%s (checkpoint NOT saved)\n", next.Through)
		}
	}

	fmt.Println("\n== 5) Checkpoint resume (gist read-only; incremental must equal the full scan) ==")
	if code := compareResume(c, startDate, now, s, ok); code != 0 {
		return code
	}

	fmt.Println("\n== collected payloads (NOT sent) ==")
	payloads := c.Collector().Payloads()
	if len(payloads) == 0 {
		fmt.Println("(none)")
	}
	for _, p := range payloads {
		fmt.Printf("  [%s] %s — %s\n", p.Type, p.Title, p.Message)
	}
	return 0
}

// compareResume replays the production resume path from the stored gist
// checkpoint and fails (1) when its lifetime totals differ from the full
// scan's, which would mean the checkpoint froze in a bad delta or the
// Through+1/settled/tail seams drop or double-count rows. It also fails when
// the gist is configured but the checkpoint cannot be loaded or would not be
// resumed (the comparison would then be a second full scan). Skips (0) when
// the gist is unconfigured or the full scan itself failed.
func compareResume(c smokeClient, startDate string, now time.Time, full dhlottery.LedgerSummary, fullOK bool) int {
	if !gistConfigured() {
		fmt.Println("  ⏭️  skipped — GIST_TOKEN/GIST_ID unset")
		return 0
	}
	if !fullOK {
		fmt.Println("  ⏭️  skipped — full scan failed, nothing to compare against")
		return 0
	}
	prev := loadCheckpoint()
	if prev == nil {
		fmt.Println("  ❌ checkpoint load failed (see checkpoint_load_failed log)")
		return 1
	}
	fmt.Printf("  checkpoint start=%s through=%s\n", prev.Start, prev.Through)
	if !checkpointResumable(prev, startDate, now) {
		fmt.Printf("  ❌ checkpoint not resumable for start=%s (start mismatch or through past the settled cutoff)\n", startDate)
		return 1
	}
	inc, _, ok := c.AggregateLedgerIncremental(startDate, now, prev)
	if !ok {
		fmt.Println("  ❌ incremental aggregate lookup failed")
		return 1
	}
	if inc != full {
		fmt.Printf("  ❌ mismatch: incremental 구매=%s 당첨=%s vs full 구매=%s 당첨=%s\n",
			format.Currency(inc.CumulativePurchase), format.Currency(inc.CumulativeWinning),
			format.Currency(full.CumulativePurchase), format.Currency(full.CumulativeWinning))
		return 1
	}
	fmt.Println("  ✅ incremental totals match the full scan")
	return 0
}

// gistConfigured reports whether both gist credentials are set.
func gistConfigured() bool {
	_, tokenErr := env.Get("GIST_TOKEN")
	_, idErr := env.Get("GIST_ID")
	return tokenErr == nil && idErr == nil
}
