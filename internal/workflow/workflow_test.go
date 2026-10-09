package workflow

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/checkpoint"
	"github.com/kadragon/dhlottery-worker/internal/constants"
	"github.com/kadragon/dhlottery-worker/internal/dhlottery"
	"github.com/kadragon/dhlottery-worker/internal/notify"
)

type fakeClient struct {
	loginErr     error
	depositOK    bool
	depositErr   error
	collector    *notify.Collector
	depositArg   int
	buyOutcome   dhlottery.PurchaseOutcome
	reserveOut   dhlottery.PensionReserveOutcome
	wins         []dhlottery.WinningResult
	summary      dhlottery.LedgerSummary
	summaryOK    bool
	aggStartDate string
	aggPrev      *checkpoint.Checkpoint
	aggPrevs     []*checkpoint.Checkpoint
	aggNext      *checkpoint.Checkpoint
	failFullScan bool // fail only calls with prev == nil

	login, checkDeposit, reserve, buy, checkWinning, aggregate int
}

// TestMain disables real gist access so no test depends on GIST_* in the
// developer's environment.
func TestMain(m *testing.M) {
	loadCheckpoint = func() *checkpoint.Checkpoint { return nil }
	saveCheckpoint = func(checkpoint.Checkpoint) bool { return true }
	saveUndelivered = func(string) bool { return true }
	os.Exit(m.Run())
}

func newFake() *fakeClient {
	return &fakeClient{depositOK: true, summaryOK: true, collector: &notify.Collector{}}
}

func (f *fakeClient) Login() error {
	f.login++
	return f.loginErr
}
func (f *fakeClient) CheckDeposit(required int) (bool, error) {
	f.checkDeposit++
	f.depositArg = required
	return f.depositOK, f.depositErr
}
func (f *fakeClient) ReservePensionNextWeek() dhlottery.PensionReserveOutcome {
	f.reserve++
	return f.reserveOut
}
func (f *fakeClient) Buy() dhlottery.PurchaseOutcome {
	f.buy++
	return f.buyOutcome
}
func (f *fakeClient) CheckWinning(time.Time) []dhlottery.WinningResult {
	f.checkWinning++
	return f.wins
}
func (f *fakeClient) AggregateLedgerIncremental(startDate string, _ time.Time, prev *checkpoint.Checkpoint) (dhlottery.LedgerSummary, *checkpoint.Checkpoint, bool) {
	f.aggregate++
	f.aggStartDate = startDate
	f.aggPrev = prev
	f.aggPrevs = append(f.aggPrevs, prev)
	if prev == nil && f.failFullScan {
		return dhlottery.LedgerSummary{}, nil, false
	}
	return f.summary, f.aggNext, f.summaryOK
}
func (f *fakeClient) Collector() *notify.Collector { return f.collector }

type sendCapture struct {
	calls    int
	payloads []notify.Payload
}

func installSend(t *testing.T, result bool) *sendCapture {
	t.Helper()
	cap := &sendCapture{}
	orig := SendCombined
	SendCombined = func(payloads []notify.Payload) bool {
		cap.calls++
		cap.payloads = payloads
		return result
	}
	t.Cleanup(func() { SendCombined = orig })
	return cap
}

func TestRunWorkflowComplete(t *testing.T) {
	cap := installSend(t, true)
	f := newFake()

	if !RunWorkflow(time.Now(), f) {
		t.Error("expected true")
	}
	if f.login != 1 || f.checkDeposit != 1 || f.reserve != 1 || f.buy != 1 || f.checkWinning != 1 {
		t.Errorf("calls = %+v", f)
	}
	if f.depositArg != constants.WeeklyCombinedRequiredBalance {
		t.Errorf("checkDeposit required = %d, want %d", f.depositArg, constants.WeeklyCombinedRequiredBalance)
	}
	if f.aggregate != 1 {
		t.Errorf("AggregateLedgerIncremental calls = %d, want 1", f.aggregate)
	}
	if f.aggStartDate != constants.DefaultLedgerStartDate {
		t.Errorf("AggregateLedgerIncremental startDate = %q, want %q", f.aggStartDate, constants.DefaultLedgerStartDate)
	}
	// Settlement is always added now, so a combined send always fires.
	if cap.calls != 1 {
		t.Errorf("SendCombined calls = %d, want 1", cap.calls)
	}
	last := cap.payloads[len(cap.payloads)-1]
	if last.Title != "주간 결산" {
		t.Errorf("last payload title = %q, want 주간 결산", last.Title)
	}
}

func TestRunWorkflowSettlementAmounts(t *testing.T) {
	cap := installSend(t, true)
	f := newFake()
	f.buyOutcome = dhlottery.PurchaseOutcome{Success: true, TotalAmount: 5000}
	f.reserveOut = dhlottery.PensionReserveOutcome{Success: true, TotalAmount: 5000}
	f.wins = []dhlottery.WinningResult{{PrizeAmount: 3000}, {PrizeAmount: 2000}}
	f.summary = dhlottery.LedgerSummary{CumulativePurchase: 1250000, CumulativeWinning: 2003005000}

	RunWorkflow(time.Now(), f)

	last := cap.payloads[len(cap.payloads)-1]
	want := map[string]string{
		"이번 주 구매": "10,000원",
		"이번 주 당첨": "5,000원",
		"누적 구매":   "1,250,000원",
		"누적 당첨":   "2,003,005,000원",
		"결산":      "+2,001,755,000원",
	}
	for _, kv := range last.Details {
		if w, ok := want[kv.Key]; ok {
			if kv.Value != w {
				t.Errorf("%s = %q, want %q", kv.Key, kv.Value, w)
			}
			delete(want, kv.Key)
		}
	}
	if len(want) != 0 {
		t.Errorf("missing settlement details: %v", want)
	}
}

func settlementDetail(p notify.Payload, key string) string {
	for _, kv := range p.Details {
		if kv.Key == key {
			return kv.Value
		}
	}
	return ""
}

// A skipped (duplicate) pension reserve still reports TotalAmount=5000, but no
// money was spent this run, so it must not be counted in 이번 주 구매.
func TestRunWorkflowSkippedPensionNotCounted(t *testing.T) {
	cap := installSend(t, true)
	f := newFake()
	f.reserveOut = dhlottery.PensionReserveOutcome{Skipped: true, TotalAmount: 5000}
	f.buyOutcome = dhlottery.PurchaseOutcome{Success: true, TotalAmount: 5000}

	RunWorkflow(time.Now(), f)

	last := cap.payloads[len(cap.payloads)-1]
	if got := settlementDetail(last, "이번 주 구매"); got != "5,000원" {
		t.Errorf("이번 주 구매 = %q, want 5,000원 (skipped reserve excluded)", got)
	}
}

func TestRunWorkflowSettlementLookupFailed(t *testing.T) {
	cap := installSend(t, true)
	f := newFake()
	f.summaryOK = false
	f.summary = dhlottery.LedgerSummary{CumulativePurchase: 999, CumulativeWinning: 999}

	RunWorkflow(time.Now(), f)

	last := cap.payloads[len(cap.payloads)-1]
	for _, key := range []string{"누적 구매", "누적 당첨", "결산"} {
		if got := settlementDetail(last, key); got != "조회 실패" {
			t.Errorf("%s = %q, want 조회 실패", key, got)
		}
	}
}

func TestRunWorkflowSettlementNegativeNet(t *testing.T) {
	cap := installSend(t, true)
	f := newFake()
	f.summary = dhlottery.LedgerSummary{CumulativePurchase: 240000, CumulativeWinning: 25000}

	RunWorkflow(time.Now(), f)

	last := cap.payloads[len(cap.payloads)-1]
	for _, kv := range last.Details {
		if kv.Key == "결산" && kv.Value != "-215,000원" {
			t.Errorf("결산 = %q, want -215,000원", kv.Value)
		}
	}
}

func TestRunWorkflowInsufficientDeposit(t *testing.T) {
	installSend(t, true)
	f := newFake()
	f.depositOK = false

	RunWorkflow(time.Now(), f)
	if f.reserve != 0 || f.buy != 0 {
		t.Error("reserve/buy must be skipped when deposit insufficient")
	}
	if f.checkWinning != 1 {
		t.Error("checkWinning must still run")
	}
}

func TestRunWorkflowLoginError(t *testing.T) {
	cap := installSend(t, true)
	f := newFake()
	f.loginErr = errors.New("boom")

	RunWorkflow(time.Now(), f)
	if f.checkDeposit != 0 || f.buy != 0 || f.checkWinning != 0 {
		t.Error("login error must short-circuit the workflow")
	}
	if cap.calls != 1 {
		t.Fatalf("SendCombined calls = %d, want 1", cap.calls)
	}
	if cap.payloads[0].Type != notify.Error || cap.payloads[0].Title != "워크플로 오류" {
		t.Errorf("payload = %+v", cap.payloads[0])
	}
}

func TestRunWorkflowDepositError(t *testing.T) {
	cap := installSend(t, true)
	f := newFake()
	f.depositErr = errors.New("deposit failed")

	if !RunWorkflow(time.Now(), f) {
		t.Error("expected true when send succeeds")
	}
	if f.buy != 0 {
		t.Error("buy must be skipped after deposit error")
	}
	if f.checkWinning != 1 {
		t.Error("checkWinning must still run after deposit error")
	}
	if cap.calls != 1 {
		t.Fatalf("SendCombined calls = %d, want 1", cap.calls)
	}
	if !strings.Contains(cap.payloads[0].Message, "deposit failed") {
		t.Errorf("payload message = %q", cap.payloads[0].Message)
	}
	// Settlement is added last; the deposit error stays at index 0.
	if last := cap.payloads[len(cap.payloads)-1]; last.Title != "주간 결산" {
		t.Errorf("last payload title = %q, want 주간 결산", last.Title)
	}
}

func TestResolveLedgerStartDate(t *testing.T) {
	if got := resolveLedgerStartDate(); got != constants.DefaultLedgerStartDate {
		t.Errorf("default = %q, want %q", got, constants.DefaultLedgerStartDate)
	}
	t.Setenv("LEDGER_START_DATE", "20210303")
	if got := resolveLedgerStartDate(); got != "20210303" {
		t.Errorf("env = %q, want 20210303", got)
	}
}

func TestRunWorkflowSendFails(t *testing.T) {
	installSend(t, false)
	f := newFake()
	f.loginErr = errors.New("login error")

	if RunWorkflow(time.Now(), f) {
		t.Error("expected false when SendCombined fails")
	}
}

func installUndelivered(t *testing.T, ok bool) *[]string {
	t.Helper()
	var saved []string
	orig := saveUndelivered
	saveUndelivered = func(content string) bool {
		saved = append(saved, content)
		return ok
	}
	t.Cleanup(func() { saveUndelivered = orig })
	return &saved
}

func TestRunWorkflowSavesUndeliveredOnSendFailure(t *testing.T) {
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	t.Setenv("GITHUB_RUN_ID", "123")
	cap := installSend(t, false)
	saved := installUndelivered(t, true)

	now := time.Date(2026, 10, 12, 1, 0, 0, 0, time.UTC)
	if RunWorkflow(now, newFake()) {
		t.Error("RunWorkflow = true, want false even when the gist write succeeds")
	}
	if len(*saved) != 1 {
		t.Fatalf("saveUndelivered calls = %d, want 1", len(*saved))
	}
	want := "미전송 알림 — 2026-10-12 10:00 KST\n" +
		"Run: https://github.com/owner/repo/actions/runs/123\n\n" +
		notify.FormatCombined(cap.payloads)
	if got := (*saved)[0]; got != want {
		t.Errorf("content =\n%s\nwant\n%s", got, want)
	}
}

func TestRunWorkflowUndeliveredLocalRun(t *testing.T) {
	t.Setenv("GITHUB_SERVER_URL", "")
	t.Setenv("GITHUB_REPOSITORY", "")
	t.Setenv("GITHUB_RUN_ID", "")
	installSend(t, false)
	saved := installUndelivered(t, false)

	RunWorkflow(time.Date(2026, 10, 12, 1, 0, 0, 0, time.UTC), newFake())
	if len(*saved) != 1 || !strings.Contains((*saved)[0], "Run: (local run)\n") {
		t.Errorf("saved = %q, want a local-run marker", *saved)
	}
}

func TestRunWorkflowNoUndeliveredOnSuccess(t *testing.T) {
	installSend(t, true)
	saved := installUndelivered(t, true)
	if !RunWorkflow(time.Now(), newFake()) {
		t.Error("RunWorkflow = false, want true")
	}
	if len(*saved) != 0 {
		t.Errorf("saveUndelivered calls = %d, want 0", len(*saved))
	}
}

type checkpointCapture struct {
	saved []checkpoint.Checkpoint
}

func installCheckpoint(t *testing.T, prev *checkpoint.Checkpoint, saveOK bool) *checkpointCapture {
	t.Helper()
	cap := &checkpointCapture{}
	origLoad, origSave := loadCheckpoint, saveCheckpoint
	loadCheckpoint = func() *checkpoint.Checkpoint { return prev }
	saveCheckpoint = func(cp checkpoint.Checkpoint) bool {
		cap.saved = append(cap.saved, cp)
		return saveOK
	}
	t.Cleanup(func() { loadCheckpoint, saveCheckpoint = origLoad, origSave })
	return cap
}

// midMonth is a run outside the monthly forced-rescan window (KST day > 7),
// so the stored checkpoint is loaded.
var midMonth = time.Date(2026, 10, 12, 1, 0, 0, 0, time.UTC)

func TestRunWorkflowCheckpointSavedOnSuccess(t *testing.T) {
	installSend(t, true)
	prev := &checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-04-27", Purchase: 1000}
	next := &checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-05-04", Purchase: 2000}
	cap := installCheckpoint(t, prev, true)
	f := newFake()
	f.aggNext = next

	RunWorkflow(midMonth, f)

	if f.aggPrev != prev {
		t.Errorf("AggregateLedgerIncremental prev = %+v, want loaded checkpoint", f.aggPrev)
	}
	if len(cap.saved) != 1 || cap.saved[0] != *next {
		t.Errorf("saved = %+v, want [%+v]", cap.saved, *next)
	}
}

func TestRunWorkflowCheckpointNotSaved(t *testing.T) {
	prev := &checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-05-04", Purchase: 2000}
	cases := map[string]func(f *fakeClient){
		"aggregation failed": func(f *fakeClient) {
			f.summaryOK = false
			f.aggNext = &checkpoint.Checkpoint{Through: "2026-05-11"}
		},
		"nothing settled": func(f *fakeClient) { f.aggNext = nil },
		"unchanged":       func(f *fakeClient) { cp := *prev; f.aggNext = &cp },
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			installSend(t, true)
			cap := installCheckpoint(t, prev, true)
			f := newFake()
			setup(f)

			RunWorkflow(midMonth, f)

			if len(cap.saved) != 0 {
				t.Errorf("saved = %+v, want none", cap.saved)
			}
		})
	}
}

func TestRunWorkflowCheckpointSaveFailureNonBlocking(t *testing.T) {
	send := installSend(t, true)
	installCheckpoint(t, nil, false)
	f := newFake()
	f.aggNext = &checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-05-04"}
	f.summary = dhlottery.LedgerSummary{CumulativePurchase: 1000}

	if !RunWorkflow(time.Now(), f) {
		t.Error("expected true despite checkpoint save failure")
	}
	last := send.payloads[len(send.payloads)-1]
	if got := settlementDetail(last, "누적 구매"); got != "1,000원" {
		t.Errorf("누적 구매 = %q, want 1,000원", got)
	}
}

var firstMonday = time.Date(2026, 10, 5, 1, 0, 0, 0, time.UTC)

func TestRunWorkflowForcedRescan(t *testing.T) {
	cases := map[string]struct {
		now    time.Time
		forced bool
	}{
		"first Monday":           {firstMonday, true},
		"KST day 7 late evening": {time.Date(2026, 10, 7, 14, 0, 0, 0, time.UTC), true},
		"KST day 8 (UTC day 7)":  {time.Date(2026, 10, 7, 15, 30, 0, 0, time.UTC), false},
		"mid month":              {midMonth, false},
	}
	prev := &checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-08-31", Purchase: 2000}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			installSend(t, true)
			cap := installCheckpoint(t, prev, true)
			f := newFake()
			unchanged := *prev
			f.aggNext = &unchanged
			grown := checkpoint.Checkpoint{Start: prev.Start, Through: "2026-09-07", Purchase: 3000}
			if tc.forced {
				f.aggNext = &grown
			}

			RunWorkflow(tc.now, f)

			if tc.forced {
				if len(f.aggPrevs) != 1 || f.aggPrevs[0] != nil {
					t.Errorf("aggregate prevs = %v, want one full scan (prev=nil)", f.aggPrevs)
				}
				if len(cap.saved) != 1 || cap.saved[0] != grown {
					t.Errorf("saved = %+v, want the full-scan checkpoint written", cap.saved)
				}
				return
			}
			if len(f.aggPrevs) != 1 || f.aggPrevs[0] != prev {
				t.Errorf("aggregate prevs = %v, want one resume from the stored checkpoint", f.aggPrevs)
			}
			if len(cap.saved) != 0 {
				t.Errorf("saved = %+v, want none for an unchanged checkpoint", cap.saved)
			}
		})
	}
}

func TestRunWorkflowForcedRescanFallsBackToResume(t *testing.T) {
	installSend(t, true)
	prev := &checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-08-31", Purchase: 2000}
	installCheckpoint(t, prev, true)
	f := newFake()
	f.failFullScan = true
	f.summary = dhlottery.LedgerSummary{CumulativePurchase: 3000}

	summary, ok := aggregateLedger(f, firstMonday)

	if !ok || summary.CumulativePurchase != 3000 {
		t.Errorf("aggregateLedger = %+v, %v; want the resumed totals", summary, ok)
	}
	if len(f.aggPrevs) != 2 || f.aggPrevs[0] != nil || f.aggPrevs[1] != prev {
		t.Errorf("aggregate prevs = %v, want [nil (full scan), stored checkpoint]", f.aggPrevs)
	}
}

func TestRunWorkflowCheckpointNotSavedWhenTotalsShrink(t *testing.T) {
	prev := &checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-08-31", Purchase: 2000, Winning: 500}
	cases := map[string]struct {
		next *checkpoint.Checkpoint
		save bool
	}{
		"purchase shrank":    {&checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-09-07", Purchase: 1000, Winning: 500}, false},
		"winning shrank":     {&checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-09-07", Purchase: 3000, Winning: 0}, false},
		"grew":               {&checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-09-07", Purchase: 3000, Winning: 500}, true},
		"start date changed": {&checkpoint.Checkpoint{Start: "2026-01-01", Through: "2026-09-07", Purchase: 100}, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			installSend(t, true)
			cap := installCheckpoint(t, prev, true)
			f := newFake()
			f.aggNext = tc.next

			RunWorkflow(firstMonday, f)

			if got := len(cap.saved) == 1; got != tc.save {
				t.Errorf("saved = %+v, want save=%v", cap.saved, tc.save)
			}
		})
	}
}
