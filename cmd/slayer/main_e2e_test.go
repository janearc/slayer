package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ball is the ball's set file, which door's tests judge.
const ball = "../../door/testdata/ball.yaml"

// ballAt is the ball's set file, absolute, for a command run elsewhere.
func ballAt(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(ball)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

// the smokes are judged through the door: a group there is not is
// refused after the program's and the verb's names, with the groups
// there are, and exit one.
func TestSlayerAssessesTheSmokes(t *testing.T) {
	refused := exec.Command(coverageCommand, "assess", "--smokes", "fire")
	told, _ := refused.CombinedOutput()
	if refused.ProcessState.ExitCode() != exitFailed ||
		!strings.HasPrefix(string(told), `slayer assess: smoke: no `+
			`group "fire"; the groups are light,`) {
		t.Errorf("refused, slayer said %q, exit %d", told,
			refused.ProcessState.ExitCode())
	}
}

// slayer draw through the door: a dry run of a set says where its
// picture would go and exits nought; --age is pzm's, from the stamps.
func TestSlayerDrawsThroughTheDoor(t *testing.T) {
	directory := t.TempDir()
	run := exec.Command(coverageCommand, "draw", ballAt(t), "--dry-run")
	run.Dir = directory
	told, err := run.CombinedOutput()
	want := "would write picture " + filepath.Join(directory, "out",
		"ball.png")
	if err != nil || !strings.Contains(string(told), want) {
		t.Errorf("slayer draw --dry-run said %q, %v", told, err)
	}
	if _, err := os.Stat(filepath.Join(directory, "out")); err == nil {
		t.Error("a dry run made out/")
	}
	aged, err := exec.Command(coverageCommand, "--age").CombinedOutput()
	if err != nil || len(aged) == nothing {
		t.Errorf("slayer --age said %q, %v", aged, err)
	}
}

// slayer list through the door: the headings, then one of them.
func TestSlayerListsThroughTheDoor(t *testing.T) {
	told, err := exec.Command(coverageCommand, "list").CombinedOutput()
	if err != nil || !strings.HasPrefix(string(told), "scenes ") {
		t.Errorf("slayer list said %q, %v", told, err)
	}
	told, err = exec.Command(coverageCommand, "list", "smokes").Output()
	if err != nil || !strings.Contains(string(told), "\n1  ") {
		t.Errorf("slayer list smokes said %q, %v", told, err)
	}
}

// the anti-test: a word that is neither an older verb nor the door's is
// refused by the door with exit two, and draw with no file fails with
// one.
func TestSlayerRefusesWhatItHasNot(t *testing.T) {
	stray := exec.Command(coverageCommand, "paint")
	told, _ := stray.CombinedOutput()
	if stray.ProcessState.ExitCode() != exitUsage {
		t.Errorf("slayer paint said %q, exit %d", told,
			stray.ProcessState.ExitCode())
	}
	bare := exec.Command(coverageCommand, "draw")
	told, _ = bare.CombinedOutput()
	if bare.ProcessState.ExitCode() != exitFailed ||
		!strings.Contains(string(told), "one file") {
		t.Errorf("slayer draw said %q, exit %d", told,
			bare.ProcessState.ExitCode())
	}
}

// set and smokes went into assess: each says what to type now, and exits
// as a verb slayer does not have.
func TestSlayerSaysWhereSetAndSmokesWent(t *testing.T) {
	for verb, said := range map[string]string{
		"set": "slayer: set is now: slayer assess FILE --judge\n",
		"smokes": "slayer: smokes is now: slayer assess --smokes " +
			"[GROUP|N...]\n",
	} {
		run := exec.Command(coverageCommand, verb, "anything")
		told, _ := run.CombinedOutput()
		if run.ProcessState.ExitCode() != exitUsage || string(told) != said {
			t.Errorf("slayer %s said %q, exit %d", verb, told,
				run.ProcessState.ExitCode())
		}
	}
}
