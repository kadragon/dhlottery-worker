package main

import (
	"errors"
	"testing"
	"time"
)

func restoreVars(t *testing.T) {
	t.Helper()
	origValidate, origRun := validateEnv, runWorkflow
	t.Cleanup(func() { validateEnv, runWorkflow = origValidate, origRun })
}

func TestRunSuccess(t *testing.T) {
	restoreVars(t)
	validateEnv = func() error { return nil }
	runWorkflow = func(time.Time) bool { return true }

	if code := run(); code != 0 {
		t.Errorf("run() = %d, want 0", code)
	}
}

func TestRunNotificationFailed(t *testing.T) {
	restoreVars(t)
	validateEnv = func() error { return nil }
	runWorkflow = func(time.Time) bool { return false }

	if code := run(); code != 2 {
		t.Errorf("run() = %d, want 2", code)
	}
}

func TestRunEnvValidationFails(t *testing.T) {
	restoreVars(t)
	validateEnv = func() error { return errors.New("missing env") }
	called := false
	runWorkflow = func(time.Time) bool { called = true; return true }

	if code := run(); code != 1 {
		t.Errorf("run() = %d, want 1", code)
	}
	if called {
		t.Error("workflow must not run when env validation fails")
	}
}
