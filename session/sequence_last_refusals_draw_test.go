//go:build draw

package session

import (
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"testing"
)

func TestAnActiveCPUProfileRefusesAnotherBeforeDrawing(t *testing.T) {
	active, err := os.Create(filepath.Join(t.TempDir(), "active.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	defer active.Close()
	if err := pprof.StartCPUProfile(active); err != nil {
		t.Fatal(err)
	}
	defer pprof.StopCPUProfile()
	called := false
	err = profiled(filepath.Join(t.TempDir(), "second.pprof"), func() error {
		called = true
		return nil
	})
	if called || err == nil || !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("profile conflict reached drawing: %v, called %v", err, called)
	}
}
