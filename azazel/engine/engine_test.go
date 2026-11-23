package engine_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/azazel/engine"
	"github.com/janearc/slayer/session"
)

// TestMain gives the tests a cache home of their own, so nothing a draw
// keeps is kept in a real cache.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "engine-cache")
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	os.Setenv("SLAYER_CACHE", home)
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}

// widow is a set file that says a world and no ground, so it draws with
// no shelf of sources.
const widow = "../../sets/widow.yaml"

// the numbers hillside gives back, and a count of none.
const (
	nothing    = 0 // defined: no amount at all, and a picture written
	exitFailed = 1 // defined: a set that could not be staged or drawn
	exitUsage  = 2 // defined: a setting hillside does not take
)

// readable is the mode a test's file is written with.
const readable = 0o644 // defined: the owner writes, everyone reads

// stamp is a build that says nothing of itself.
func stamp() session.Stamp { return session.Stamped("", "", "", "") }

// engineDraws is what the adapter gives back for a set and settings: the
// picture's bytes, what it said, what it complained of, and its number.
func engineDraws(t *testing.T, path string, given ...string) ([]byte, string,
	string, int) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "engine.png")
	var said, wrong bytes.Buffer
	code := engine.Render(path, append(given, "-out", out), &said, &wrong,
		stamp())
	picture, _ := os.ReadFile(out)
	return picture, said.String(), wrong.String(), code
}

// hillsideDraws is what hillside gives back for the same set and settings.
func hillsideDraws(t *testing.T, path string, given ...string) ([]byte,
	string, int) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "hillside.png")
	var wrong bytes.Buffer
	settings := append([]string{"-set", path}, given...)
	settings = append(settings, "-out", out)
	code := session.Run(settings, io.Discard, &wrong, stamp())
	picture, _ := os.ReadFile(out)
	return picture, wrong.String(), code
}

// a set that cannot be staged gives hillside's number and says what
// hillside says: one for a file that is not there and for a set with no
// film, two for a setting hillside does not take, nothing written.
func TestRenderGivesHillsidesNumberForASetItCannotStage(t *testing.T) {
	dir := t.TempDir()
	filmless := filepath.Join(dir, "filmless.yaml")
	if err := os.WriteFile(filmless, []byte("scene:\n  name: filmless\n"),
		readable); err != nil {
		t.Fatal(err)
	}
	for _, x := range []struct {
		path  string
		given []string
		code  int
	}{
		{filepath.Join(dir, "nowhere.yaml"), nil, exitFailed},
		{filmless, nil, exitFailed},
		{widow, []string{"-nonsense", "3"}, exitUsage},
	} {
		picture, said, wrong, code := engineDraws(t, x.path, x.given...)
		_, wanted, expected := hillsideDraws(t, x.path, x.given...)
		if code != x.code || expected != x.code || len(picture) != nothing ||
			wrong != wanted || strings.Contains(said, "timing") {
			t.Errorf("%s %v: gave %d and %q, hillside %d and %q, not %d "+
				"and no picture", x.path, x.given, code, wrong, expected,
				wanted, x.code)
		}
	}
}
