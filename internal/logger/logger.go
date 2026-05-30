// Package logger provides minimal structured (JSON line) logging.
//
// Behaviour mirrors the original implementation:
//   - debug entries are suppressed unless DEBUG=true,
//   - info/debug go to stdout, warn/error go to stderr,
//   - each entry is a single JSON object with level, message and fields.
package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Fields is an arbitrary set of structured key/value pairs attached to a log
// entry. Nil is allowed.
type Fields map[string]any

var (
	mu           sync.Mutex
	debugEnabled           = os.Getenv("DEBUG") == "true"
	infoWriter   io.Writer = os.Stdout
	errWriter    io.Writer = os.Stderr
	// now is overridable in tests; production uses the wall clock.
	now = time.Now
)

// SetDebug toggles debug logging. Intended for tests and explicit overrides.
func SetDebug(enabled bool) {
	mu.Lock()
	defer mu.Unlock()
	debugEnabled = enabled
}

// SetWriters overrides the output destinations. Intended for tests.
func SetWriters(info, err io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	infoWriter, errWriter = info, err
}

func write(w io.Writer, level, message string, fields Fields) {
	entry := map[string]any{
		"level":     level,
		"message":   message,
		"timestamp": now().UTC().Format(time.RFC3339),
	}
	for k, v := range fields {
		entry[k] = v
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	fmt.Fprintln(w, string(b))
}

// Debug logs at debug level (suppressed unless DEBUG=true).
func Debug(message string, fields Fields) {
	mu.Lock()
	enabled := debugEnabled
	w := infoWriter
	mu.Unlock()
	if !enabled {
		return
	}
	write(w, "debug", message, fields)
}

// Info logs at info level.
func Info(message string, fields Fields) {
	mu.Lock()
	w := infoWriter
	mu.Unlock()
	write(w, "info", message, fields)
}

// Warn logs at warn level.
func Warn(message string, fields Fields) {
	mu.Lock()
	w := errWriter
	mu.Unlock()
	write(w, "warn", message, fields)
}

// Error logs at error level.
func Error(message string, fields Fields) {
	mu.Lock()
	w := errWriter
	mu.Unlock()
	write(w, "error", message, fields)
}
