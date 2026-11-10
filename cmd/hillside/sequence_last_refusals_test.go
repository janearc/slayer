package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"testing"
)

func TestSequenceCannotCreateAnOutputUnderAReadOnlyParent(t *testing.T) {
	parent := t.TempDir()
	if err := os.Chmod(parent, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(parent, 0700); err != nil {
			t.Error(err)
		}
	})
	output := filepath.Join(parent, "frames")
	_, err := sequenceFolder(output, sequenceState{Complete: map[int]sequenceEntry{}}, false)
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("uncreatable output: %v", err)
	}
	if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("output appeared after refusal: %v", err)
	}
}

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
