package main

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/session"
)

// the widow set, which says a world and no ground, and the small size the
// tests draw it at.
const (
	widow     = "../../sets/widow.yaml"
	smallWide = "48"
	smallTall = "27"
	small     = "-w " + smallWide + " -h " + smallTall + " -n 1"
)

// defined: one place

// exitOf is the number a command exited with, from what running it said.
func exitOf(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return nothing
	}
	var stopped *exec.ExitError
	if !errors.As(err, &stopped) {
		t.Fatalf("the command did not run: %v", err)
	}
	return stopped.ExitCode()
}

// renders is the picture slayer's render writes for a set and settings:
// its bytes, and the number the command exited with.
func renders(t *testing.T, path, settings string) ([]byte, int) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "render.png")
	arguments := append([]string{"render", path}, strings.Fields(settings)...)
	_, err := runCoverageCommand(t, append(arguments, "-out", out)...)
	picture, _ := os.ReadFile(out)
	return picture, exitOf(t, err)
}

// hillsides is the picture hillside writes for the same set and settings,
// and the number it gives back.
func hillsides(t *testing.T, path, settings string) ([]byte, int) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "hillside.png")
	arguments := append([]string{"-set", path}, strings.Fields(settings)...)
	code := session.Run(append(arguments, "-out", out), io.Discard,
		io.Discard, session.Stamped("", "", "", ""))
	picture, _ := os.ReadFile(out)
	return picture, code
}

// render says the numbers hillside says: a file that is not there is one,
// a setting hillside does not know is two, and a render with no file
// says how it is used and is two.
func TestRenderExitsAsHillsideDoes(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nowhere.yaml")
	for _, x := range []struct {
		path, settings string
		code           int
	}{
		{missing, "", exitFailed},
		{widow, "-nonsense 3", exitUsage},
	} {
		_, got := renders(t, x.path, x.settings)
		_, wanted := hillsides(t, x.path, x.settings)
		if got != x.code || wanted != x.code {
			t.Errorf("%s %q: render exited %d, hillside %d, not %d",
				x.path, x.settings, got, wanted, x.code)
		}
	}
	told, err := runCoverageCommand(t, "render")
	if exitOf(t, err) != exitUsage || !strings.HasPrefix(string(told),
		"usage: slayer render FILE") {
		t.Errorf("render with no file said %q", told)
	}
}

// the command says which build it is, as hillside does.
func TestSlayerSaysWhichBuildItIs(t *testing.T) {
	told, err := runCoverageCommand(t, "--age")
	if exitOf(t, err) != nothing || !strings.HasPrefix(string(told), "slayer") {
		t.Errorf("--age said %q", told)
	}
}

// render refuses the short names and any other long name, with a message
// that names the long flags it knows, and draws nothing.
func TestRenderRefusesWhatItDoesNotKnowByName(t *testing.T) {
	for _, settings := range []string{"-w 48", "-h 27", "-n 1", "--bogus 3",
		"--width"} {
		told, code := rendered(t, widow, settings)
		if code != exitUsage {
			t.Errorf("%q: exit %d, not %d", settings, code, exitUsage)
		}
		for _, name := range []string{"--width", "--height", "--rays"} {
			if !strings.Contains(told, name) {
				t.Errorf("%q: said %q, which does not name %s", settings,
					told, name)
			}
		}
	}
}

// rendered is what render says for a set and settings, and the number it
// exits with.
func rendered(t *testing.T, path, settings string) (string, int) {
	t.Helper()
	arguments := append([]string{"render", path}, strings.Fields(settings)...)
	told, err := runCoverageCommand(t, arguments...)
	return string(told), exitOf(t, err)
}
