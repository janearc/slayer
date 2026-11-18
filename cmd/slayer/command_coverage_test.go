package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/internal/commandtest"
)

var coverageCommand string

// TestMain builds the real command once, with the test's coverage mode.
func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "slayer-slayer-command-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// the cache home the command and the tests keep what they work out
	// in, removed with the rest, so nothing is kept in a real cache.
	os.Setenv("SLAYER_CACHE", filepath.Join(directory, "cache"))
	coverageCommand = filepath.Join(directory, "slayer")
	if err = commandtest.Build(".", ".", coverageCommand, testing.CoverMode()); err != nil {
		os.RemoveAll(directory)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(directory)
	os.Exit(code)
}

// runCoverageCommand bounds a real invocation and retains its diagnostics.
func runCoverageCommand(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	return exec.CommandContext(ctx, coverageCommand, args...).CombinedOutput()
}

func TestRealCommandHelpOrUsagePreservesItsPublicOptions(t *testing.T) {
	out, err := runCoverageCommand(t, "draw", "--help")
	code := 0
	if err != nil {
		var stopped *exec.ExitError
		if !errors.As(err, &stopped) {
			t.Fatalf("command did not exit: %v", err)
		}
		code = stopped.ExitCode()
	}
	if code != nothing || !strings.Contains(string(out), "--dry-run") ||
		!strings.Contains(string(out), "--renderer") {
		t.Fatalf("exit%d: %q", code, out)
	}
}
