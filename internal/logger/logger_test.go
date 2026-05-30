package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestDebugGate(t *testing.T) {
	var info, errb bytes.Buffer
	SetWriters(&info, &errb)
	defer SetWriters(os.Stdout, os.Stderr)

	SetDebug(false)
	Debug("should be suppressed", nil)
	if info.Len() != 0 {
		t.Errorf("debug should be suppressed when DEBUG is off, got %q", info.String())
	}

	SetDebug(true)
	defer SetDebug(false)
	Debug("now visible", Fields{"k": "v"})
	if !strings.Contains(info.String(), "now visible") {
		t.Errorf("debug should be emitted when DEBUG is on, got %q", info.String())
	}
}

func TestLevelsRouteToWriters(t *testing.T) {
	var info, errb bytes.Buffer
	SetWriters(&info, &errb)
	defer SetWriters(os.Stdout, os.Stderr)

	Info("hello", nil)
	Warn("careful", nil)
	Error("boom", nil)

	if !strings.Contains(info.String(), "hello") {
		t.Errorf("info missing: %q", info.String())
	}
	out := errb.String()
	if !strings.Contains(out, "careful") || !strings.Contains(out, "boom") {
		t.Errorf("warn/error missing: %q", out)
	}
	if !strings.Contains(out, `"level":"warn"`) || !strings.Contains(out, `"level":"error"`) {
		t.Errorf("level field missing: %q", out)
	}
}
