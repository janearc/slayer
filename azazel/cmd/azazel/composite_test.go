package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompositeCommandRefusesUnsafeDirectStart(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "frames")
	for _, args := range [][]string{
		{"composite", filepath.Join(directory, "missing.yaml")},
		{"composite", filepath.Join(directory, "missing.yaml"), "--from", "0", "--to", "1", "--out", output, "--memory-bytes", "1024"},
		{"composite", "--unsafe"},
	} {
		told, code := run(t, azazel, args...)
		if code != exitFailed || !strings.Contains(told, "shared admission") || !strings.Contains(told, "unavailable") {
			t.Fatalf("%v: exit %d, %q", args, code, told)
		}
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("output touched: %v", err)
	}
}

func TestCompositeCommandRequiresAnArgument(t *testing.T) {
	told, code := run(t, azazel, "composite")
	if code != exitUsage || !strings.Contains(told, "azazel composite") {
		t.Fatalf("exit %d: %q", code, told)
	}
}
