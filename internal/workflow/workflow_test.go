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
	loginErr   error
	depositOK  bool
	depositErr error
	collector  *notify.Collector
	depositArg int

	login, checkDeposit, reserve, buy, checkWinning int
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
	return dhlottery.PensionReserveOutcome{}
}
func (f *fakeClient) Buy() dhlottery.PurchaseOutcome {
	f.buy++
	return dhlottery.PurchaseOutcome{}
}
func (f *fakeClient) CheckWinning(time.Time) []dhlottery.WinningResult {
	f.checkWinning++
	return nil
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
	if cap.calls != 0 {
		t.Errorf("SendCombined should not be called for an empty collector, got %d", cap.calls)
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
}

func TestRunWorkflowSendFails(t *testing.T) {
	installSend(t, false)
	f := newFake()
	f.loginErr = errors.New("login error")

	if RunWorkflow(time.Now(), f) {
		t.Error("expected false when SendCombined fails")
	}
}
