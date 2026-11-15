//go:build draw

package main

import (
	"os/exec"
	"strings"
	"testing"
)

// slayer run as a person runs it: with no verb it is the door, which
// says its help and names its verbs; a set it judges honest exits with
// nought and says the verdict; one it cannot judge it says why after its
// name and the verb's, and exits with one.
func TestSlayerRunsAsAPersonRunsIt(t *testing.T) {
	built := coverageCommand
	usage := exec.Command(built)
	told, _ := usage.CombinedOutput()
	if usage.ProcessState.ExitCode() != nothing ||
		!strings.Contains(string(told), "draw") ||
		!strings.Contains(string(told), "assess") ||
		!strings.Contains(string(told), "list") {
		t.Errorf("with no verb slayer said %q, exit %d", told,
			usage.ProcessState.ExitCode())
	}
	said, err := exec.Command(built, "assess", ball, "--judge", "--k", "1",
		"--out", t.TempDir()).Output()
	if err != nil || !strings.HasPrefix(string(said), "set ball at 0s: honest.") {
		t.Errorf("slayer said %q, %v", said, err)
	}
	refused := exec.Command(built, "assess", ball, "--judge", "--camera",
		"back")
	told, _ = refused.CombinedOutput()
	if refused.ProcessState.ExitCode() != exitFailed ||
		string(told) != "slayer assess: the set has no camera back\n" {
		t.Errorf("refused, slayer said %q, exit %d", told,
			refused.ProcessState.ExitCode())
	}
}
