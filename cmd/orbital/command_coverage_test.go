package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/janearc/slayer/internal/commandtest"
)

var coverageCommand string

// TestMain builds the real command once, with the test's coverage mode.
func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "slayer-orbital-command-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	coverageCommand = filepath.Join(directory, "orbital")
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
