package preview

import (
	"context"
	"io"
	"path/filepath"
	"testing"
)

// TestProcessRunnerRefusesAnUnavailableControlBinary proves there is no shell fallback.
func TestProcessRunnerRefusesAnUnavailableControlBinary(t *testing.T) {
	raw, err := (processRunner{}).Run(context.Background(), invocation{Executable: filepath.Join(t.TempDir(), "missing-azazel"), Output: io.Discard})
	if err == nil || len(raw) != 0 {
		t.Fatal("unavailable control executable succeeded", err)
	}
}
