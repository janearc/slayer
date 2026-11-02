package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// built is house, built where the test keeps its files.
func built(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "house")
	made, err := exec.Command("go", "build", "-o", path, ".").
		CombinedOutput()
	if err != nil {
		t.Fatalf("house would not build: %v\n%s", err, made)
	}
	return path
}

// house run as a person runs it draws what its flags ask and says so on
// its output; refused, it says why on its errors, after its name, and
// exits with one.
func TestHouseRunsAsAPersonRunsIt(t *testing.T) {
	house := built(t)
	out := filepath.Join(t.TempDir(), "front.png")
	said, err := exec.Command(house, "-set", porch(t), "-camera", "front",
		"-out", out).Output()
	if err != nil || string(said) != out+": 16x9, 1 rays a pixel\n" {
		t.Errorf("house said %q, %v", said, err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Error(err)
	}
	pair := filepath.Join(t.TempDir(), "pair.png")
	said, err = exec.Command(house, "-set", porch(t), "-camera",
		"front,side", "-at", "the box=0,1,0", "-out", pair).Output()
	if err != nil || !strings.HasSuffix(string(said),
		": side, \"the side\"\n") {
		t.Errorf("house said %q, %v", said, err)
	}
	refused := exec.Command(house, "-set", porch(t), "-camera", "back",
		"-out", out)
	told, err := refused.CombinedOutput()
	if refused.ProcessState.ExitCode() != 1 ||
		string(told) != "house: the set has no camera back\n" {
		t.Errorf("house refused with %q, exit %v", told, err)
	}
}
