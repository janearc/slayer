//go:build draw

package main

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestRealCommandHelpOrUsagePreservesItsPublicOptions(t *testing.T) {
	out, err := runCoverageCommand(t, "-help")
	code := 0
	if err != nil {
		var stopped *exec.ExitError
		if !errors.As(err, &stopped) {
			t.Fatalf("command did not exit: %v", err)
		}
		code = stopped.ExitCode()
	}
	if code != 0 || !strings.Contains(string(out), "-out") {
		t.Fatalf("exit%d: %q", code, out)
	}
}
