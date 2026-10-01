package main

import (
	"errors"
	"testing"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/checkpoint"
	"github.com/kadragon/dhlottery-worker/internal/dhlottery"
	"github.com/kadragon/dhlottery-worker/internal/notify"
)

func restoreVars(t *testing.T) {
	t.Helper()
	origValidate, origChecks, origClient, origNow, origLoad := validateEnv, runChecks, newClient, nowFn, loadCheckpoint
	loadCheckpoint = func() *checkpoint.Checkpoint { return nil }
	t.Cleanup(func() {
		validateEnv, runChecks, newClient, nowFn, loadCheckpoint = origValidate, origChecks, origClient, origNow, origLoad
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
	aggCalls           int
	aggPrev            *checkpoint.Checkpoint
	aggPrevs           []*checkpoint.Checkpoint
	incSummary         dhlottery.LedgerSummary // returned when prev != nil
	incFailed          bool
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

func (f *fakeSmokeClient) AggregateLedgerIncremental(_ string, _ time.Time, prev *checkpoint.Checkpoint) (dhlottery.LedgerSummary, *checkpoint.Checkpoint, bool) {
	f.aggCalls++
	f.aggPrev = prev
	f.aggPrevs = append(f.aggPrevs, prev)
	if prev != nil {
		return f.incSummary, nil, !f.incFailed
	}
	return dhlottery.LedgerSummary{}, &checkpoint.Checkpoint{Through: "2026-08-23"}, true
}

func (f *fakeSmokeClient) Collector() *notify.Collector { return &f.collector }

// realtest must drive the production ledger path (settled/tail split) with no
// checkpoint, so the full scan is still verified against the real account.
func TestDefaultRunChecksUsesIncrementalLedgerWithoutCheckpoint(t *testing.T) {
	restoreVars(t)
	client := &fakeSmokeClient{}
	newClient = func() smokeClient { return client }

	if code := defaultRunChecks(); code != 0 {
		t.Fatalf("defaultRunChecks() = %d, want 0", code)
	}
	if client.aggCalls != 1 {
		t.Errorf("AggregateLedgerIncremental calls = %d, want 1", client.aggCalls)
	}
	if client.aggPrev != nil {
		t.Errorf("prev = %+v, want nil (full scan)", client.aggPrev)
	}
}

// With a gist checkpoint, realtest replays the resume path from it and the
// incremental totals must equal the full scan's (read-only: never saved).
func TestDefaultRunChecksComparesCheckpointResume(t *testing.T) {
	stored := &checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-08-23"}
	cases := map[string]struct {
		client *fakeSmokeClient
		want   int
	}{
		"totals match":     {&fakeSmokeClient{}, 0},
		"totals differ":    {&fakeSmokeClient{incSummary: dhlottery.LedgerSummary{CumulativeWinning: 5000}}, 1},
		"incremental fail": {&fakeSmokeClient{incFailed: true}, 1},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			restoreVars(t)
			loadCheckpoint = func() *checkpoint.Checkpoint { return stored }
			newClient = func() smokeClient { return tc.client }

			if code := defaultRunChecks(); code != tc.want {
				t.Errorf("defaultRunChecks() = %d, want %d", code, tc.want)
			}
			if len(tc.client.aggPrevs) != 2 || tc.client.aggPrevs[0] != nil || tc.client.aggPrevs[1] != stored {
				t.Errorf("aggregate prevs = %v, want [nil (full scan), stored checkpoint]", tc.client.aggPrevs)
			}
		})
	}
}

func TestDefaultRunChecksSkipsResumeWithoutCheckpoint(t *testing.T) {
	restoreVars(t)
	client := &fakeSmokeClient{}
	newClient = func() smokeClient { return client }

	if code := defaultRunChecks(); code != 0 {
		t.Fatalf("defaultRunChecks() = %d, want 0", code)
	}
	if client.aggCalls != 1 {
		t.Errorf("AggregateLedgerIncremental calls = %d, want 1 (no resume without a checkpoint)", client.aggCalls)
	}
}
