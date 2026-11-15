package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ringArgs(out string) []string {
	return []string{"-around", "8", "-w", "8", "-h", "6", "-n", "1", "-out", out}
}

// anti-tests: fewer steps round than close the ring are refused by name,
// not drawn as more.
func TestRingCommandRefusesTooFewStepsRound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ring.png")
	args := append(ringArgs(path), "-around", "7")
	out, err := runCoverageCommand(t, args...)
	if err == nil || !strings.Contains(string(out), "around") {
		t.Fatalf("-around 7: %v %q", err, out)
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("-around 7 drew a picture")
	}
}
