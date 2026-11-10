package main

import (
	"os/exec"
	"strings"
	"testing"
)

// slayer run as a person runs it: with no verb it says how it is used
// and exits with two; a set it draws honestly exits with nought and says
// the verdict; one it cannot draw it says why after its name and the
// verb's, and exits with one.
func TestSlayerRunsAsAPersonRunsIt(t *testing.T) {
	built := coverageCommand
	usage := exec.Command(built)
	told, _ := usage.CombinedOutput()
	if usage.ProcessState.ExitCode() != 2 ||
		!strings.HasPrefix(string(told), "usage: slayer set FILE") {
		t.Errorf("with no verb slayer said %q, exit %d", told,
			usage.ProcessState.ExitCode())
	}
	said, err := exec.Command(built, "set", ball(t), "-k", "1", "-out",
		t.TempDir()).Output()
	if err != nil || !strings.HasPrefix(string(said), "set ball at 0s: honest.") {
		t.Errorf("slayer said %q, %v", said, err)
	}
	refused := exec.Command(built, "set", ball(t), "-camera", "back")
	told, _ = refused.CombinedOutput()
	if refused.ProcessState.ExitCode() != 1 ||
		string(told) != "slayer set: the set has no camera back\n" {
		t.Errorf("refused, slayer said %q, exit %d", told,
			refused.ProcessState.ExitCode())
	}
}
