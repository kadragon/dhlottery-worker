// Command realtest is a money-free real-server smoke test, invoked by the
// manual `.github/workflows/realtest.yml` workflow. It exercises only the
// read-only paths — login (RSA + cookie session), account info (balance +
// round JSON parse), and the previous-week winning check — and prints the
// collected notifications instead of sending them. It NEVER purchases lotto,
// reserves pension, or sends Telegram, so it is safe to run any time.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/dhlottery"
	"github.com/kadragon/dhlottery-worker/internal/env"
	"github.com/kadragon/dhlottery-worker/internal/notify"
)

type smokeClient interface {
	Login() error
	GetAccountInfo() (dhlottery.AccountInfo, error)
	CheckWinning(time.Time) []dhlottery.WinningResult
	Collector() *notify.Collector
}

// Indirected for tests.
var (
	validateEnv = env.Validate
	runChecks   = defaultRunChecks
	newClient   = func() smokeClient { return dhlottery.NewClient() }
	nowFn       = time.Now
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
