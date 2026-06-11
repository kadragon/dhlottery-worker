package main

import (
	"errors"
	"testing"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/dhlottery"
	"github.com/kadragon/dhlottery-worker/internal/notify"
)

func restoreVars(t *testing.T) {
	t.Helper()
	origValidate, origChecks, origClient, origNow := validateEnv, runChecks, newClient, nowFn
	t.Cleanup(func() {
		validateEnv, runChecks, newClient, nowFn = origValidate, origChecks, origClient, origNow
	})
}

func TestRunEnvValidationFails(t *testing.T) {
	restoreVars(t)
	validateEnv = func() error { return errors.New("missing env") }
	called := false
	runChecks = func() int { called = true; return 0 }

	if code := run(); code != 1 {
		t.Errorf("run() = %d, want 1", code)
	}
	if called {
		t.Error("checks must not run when env validation fails")
	}
}

func TestRunSuccess(t *testing.T) {
	restoreVars(t)
	validateEnv = func() error { return nil }
	runChecks = func() int { return 0 }

	if code := run(); code != 0 {
		t.Errorf("run() = %d, want 0", code)
	}
}

func TestRunChecksFailure(t *testing.T) {
	restoreVars(t)
	validateEnv = func() error { return nil }
	runChecks = func() int { return 1 }

	if code := run(); code != 1 {
		t.Errorf("run() = %d, want 1", code)
	}
}

func TestDefaultRunChecksFailsWhenAccountInfoFails(t *testing.T) {
	restoreVars(t)
	client := &fakeSmokeClient{accountErr: errors.New("account down")}
	newClient = func() smokeClient { return client }

	if code := defaultRunChecks(); code != 1 {
		t.Errorf("defaultRunChecks() = %d, want 1", code)
	}
	if client.checkWinningCalled {
		t.Error("CheckWinning must not run when account info fails")
	}
}

type fakeSmokeClient struct {
	accountErr         error
	checkWinningCalled bool
	collector          notify.Collector
}

func (f *fakeSmokeClient) Login() error { return nil }

func (f *fakeSmokeClient) GetAccountInfo() (dhlottery.AccountInfo, error) {
	if f.accountErr != nil {
		return dhlottery.AccountInfo{}, f.accountErr
	}
	return dhlottery.AccountInfo{Balance: 10000, CurrentRound: 1206}, nil
}

func (f *fakeSmokeClient) CheckWinning(time.Time) []dhlottery.WinningResult {
	f.checkWinningCalled = true
	return nil
}

func (f *fakeSmokeClient) AggregateLedger(string, time.Time) dhlottery.LedgerSummary {
	return dhlottery.LedgerSummary{}
}

func (f *fakeSmokeClient) Collector() *notify.Collector { return &f.collector }
