package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	compositor "github.com/janearc/slayer/improv/moonlake/omnipositor"
)

func TestTracePublishesWithoutClobbering(t *testing.T) {
	profile, err := compositor.NewProfile(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := profile.Record(compositor.ProfileSpan{Frame: 1, Stage: "matte", Start: time.Unix(10, 0), Duration: time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "trace.json")
	if err := writeTrace(path, profile); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(first), "matte") {
		t.Fatalf("trace: %q %v", first, err)
	}
	if err := writeTrace(path, profile); err == nil {
		t.Fatal("existing trace was replaced")
	}
	second, err := os.ReadFile(path)
	if err != nil || string(first) != string(second) {
		t.Fatal("existing trace changed")
	}
}

func TestTraceRequiresExperimentalGPU(t *testing.T) {
	code, _ := run(stretchArgs([]string{"-settings", "night.yaml", "-out", "film", "-png", "-trace", "trace.json", "--dry-run"})...)
	if code == 0 {
		t.Fatal("CPU trace accepted despite unavailable profiling")
	}
}
