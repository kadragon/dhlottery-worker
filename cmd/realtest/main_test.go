package main

import (
	"errors"
	"testing"
)

func restoreVars(t *testing.T) {
	t.Helper()
	origValidate, origChecks := validateEnv, runChecks
	t.Cleanup(func() { validateEnv, runChecks = origValidate, origChecks })
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
