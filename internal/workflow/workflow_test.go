package workflow

import (
	"errors"
	"strings"
	"testing"
	"time"

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
	aggStartDate string

	login, checkDeposit, reserve, buy, checkWinning, aggregate int
}

func newFake() *fakeClient {
	return &fakeClient{depositOK: true, collector: &notify.Collector{}}
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
func (f *fakeClient) AggregateLedger(startDate string, _ time.Time) dhlottery.LedgerSummary {
	f.aggregate++
	f.aggStartDate = startDate
	return f.summary
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
		t.Errorf("AggregateLedger calls = %d, want 1", f.aggregate)
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
	f.buyOutcome = dhlottery.PurchaseOutcome{TotalAmount: 5000}
	f.reserveOut = dhlottery.PensionReserveOutcome{TotalAmount: 5000}
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
