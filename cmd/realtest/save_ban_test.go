package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// realtest runs with a write-capable GIST_TOKEN but must never overwrite the
// production checkpoint, so no non-test source here may reference
// checkpoint.Save.
func TestRealtestNeverSavesCheckpoint(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(src), "checkpoint.Save") {
			t.Errorf("%s references checkpoint.Save; realtest must stay read-only toward the gist", f)
		}
	}
}
