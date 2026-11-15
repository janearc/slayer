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

// the commands the tests run, built once: azazel as it is tested, and
// slayer, whose render azazel's must draw as.
var azazel, slayer string

// root is the module's root, where slayer's command is built from, and
// widow the set file the pictures are drawn from.
const (
	root  = "../../.."
	widow = "../../../sets/widow.yaml"
)

// the numbers the commands exit with, and a count of none.
const (
	nothing    = 0 // defined: no amount at all, and a picture written
	exitFailed = 1 // defined: a set that could not be staged or drawn
)

// TestMain builds azazel's command, and slayer's, once.
func TestMain(suite *testing.M) {
	directory, err := os.MkdirTemp("", "azazel-command-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFailed)
	}
	azazel = filepath.Join(directory, "azazel")
	slayer = filepath.Join(directory, "slayer")
	err = commandtest.Build(".", ".", azazel, testing.CoverMode())
	if err == nil {
		err = commandtest.Build(root, "./cmd/slayer", slayer, "")
	}
	if err != nil {
		os.RemoveAll(directory)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFailed)
	}
	code := suite.Run()
	os.RemoveAll(directory)
	os.Exit(code)
}

// run is a command run for a minute at most, with what it said and the
// number it exited with.
func run(t *testing.T, command string, args ...string) (string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	told, err := exec.CommandContext(ctx, command, args...).CombinedOutput()
	var stopped *exec.ExitError
	if err != nil && !errors.As(err, &stopped) {
		t.Fatalf("%s did not run: %v", command, err)
	}
	if stopped != nil {
		return string(told), stopped.ExitCode()
	}
	return string(told), nothing
}

// azazel has one verb: with none, or with one it does not have, it says
// how it is used and exits with two; render with no file does the same;
// and a set it cannot draw gives hillside's number, one.
func TestAzazelHasOneVerbAndRefusesTheRest(t *testing.T) {
	for _, x := range []struct {
		args []string
		code int
	}{
		{nil, exitUsage}, {[]string{"queue"}, exitUsage},
		{[]string{"queue", widow}, exitUsage}, {[]string{"render"}, exitUsage},
		{[]string{"render", filepath.Join(t.TempDir(), "nowhere.yaml")},
			exitFailed},
	} {
		told, code := run(t, azazel, x.args...)
		if code != x.code {
			t.Errorf("%v: exit %d and %q, not %d", x.args, code, told, x.code)
		}
		if x.code == exitUsage && !strings.Contains(told, "usage: azazel render FILE") {
			t.Errorf("%v: said %q", x.args, told)
		}
	}
}

// the command says which build it is.
func TestAzazelSaysWhichBuildItIs(t *testing.T) {
	told, code := run(t, azazel, "--age")
	if code != nothing || !strings.HasPrefix(told, "azazel") {
		t.Errorf("--age said %q, exit %d", told, code)
	}
}
