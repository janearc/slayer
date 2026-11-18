package main

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/improv/walk"
)

func TestRunRefusesAShortWithNoScript(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-short", "x", "-out", "y"}, &out, &problems) !=
		failed {
		t.Fatal("a short with no script was taken")
	}
}

func TestRunRefusesAFlagItDoesNotKnow(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-colour", "red"}, &out, &problems) != failed {
		t.Fatal("an unknown flag was taken")
	}
}

func TestRunRefusesAScriptThatIsNotThere(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-script", t.TempDir() + "/none.yaml", "-short",
		"x", "-out", "y"}, &out, &problems) != failed {
		t.Fatal("a missing script was taken")
	}
}

// markerScript is a script over walk's marker fixture with one short of
// five seconds in two shots.
const markerScript = `set: set.yaml
places: places.csv
shorts:
  - name: where
    title: Where the markers are
    shots:
      - camera: down
        picture: down.png
        seconds: 3
        lit: show
        name: The markers
        sentence: Six boulders glow alone.
      - set: set.yaml
        camera: above
        picture: above.png
        seconds: 2
        lit: inspection
        name: From above
        sentence: The same, from above.
`

// fixture copies walk's marker fixture and the script into a temporary
// folder and returns the script's path.
func fixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	source := filepath.Join("..", "..", "walk", "testdata", "marker")
	for _, name := range []string{"set.yaml", "places.csv", "down.png",
		"above.png"} {
		data, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, "script.yaml")
	if err := os.WriteFile(path, []byte(markerScript), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestReadTakesEveryFlag checks that each flag lands in its field and
// that the defaults are the ones the usage names.
func TestReadTakesEveryFlag(t *testing.T) {
	var problems bytes.Buffer
	got, err := read([]string{"-script", "s.yaml", "-pictures", "p", "-short",
		"x", "-out", "o.mp4", "-still", "7", "-score", "@crypt", "-voicing",
		"@crypt-dmg", "-bar", "3"}, &problems)
	if err != nil {
		t.Fatal(err)
	}
	want := asked{script: "s.yaml", pictures: "p", short: "x", out: "o.mp4",
		score: "@crypt", voicing: "@crypt-dmg", still: 7, bar: 3}
	if got != want {
		t.Errorf("read %+v, want %+v", got, want)
	}
	got, err = read([]string{"-script", "s", "-short", "x", "-out", "o"}, &problems)
	if err != nil || got.still != noStill || got.bar != firstBar {
		t.Errorf("defaults %+v (%v): want still %d, bar %d", got, err, noStill, firstBar)
	}
}

// TestRunWritesOneStill writes a frame as a PNG through run and reads it
// back.
func TestRunWritesOneStill(t *testing.T) {
	script := fixture(t)
	out := filepath.Join(t.TempDir(), "frame.png")
	var stdout, problems bytes.Buffer
	status := run([]string{"-script", script, "-short", "where", "-out", out,
		"-still", "0"}, &stdout, &problems)
	if status != ok {
		t.Fatalf("status %d: %s", status, problems.String())
	}
	file, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	config, err := png.DecodeConfig(file)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := walk.ReadScript(script, "")
	if err != nil {
		t.Fatal(err)
	}
	short, _ := loaded.Short("where")
	frame, err := short.Frame(0, walk.NewStills())
	if err != nil {
		t.Fatal(err)
	}
	if config.Width != frame.Bounds().Dx() || config.Height != frame.Bounds().Dy() {
		t.Errorf("png is %dx%d, frame is %v", config.Width, config.Height, frame.Bounds())
	}
}

// TestWriteRefusals covers what write turns away once the script reads.
func TestWriteRefusals(t *testing.T) {
	script := fixture(t)
	dir := t.TempDir()
	cases := []struct {
		name  string
		given asked
		want  string
	}{
		{"unknown short", asked{script: script, short: "nope", out: filepath.Join(dir, "x.mp4"), still: noStill},
			`no short called "nope"`},
		{"frame past the end", asked{script: script, short: "where", out: filepath.Join(dir, "x.png"), still: 100000},
			""},
		{"still into a folder that is not there", asked{script: script, short: "where", out: filepath.Join(dir, "none", "x.png"), still: 0},
			""},
		{"bar before the first", asked{script: script, short: "where", out: filepath.Join(dir, "x.mp4"), still: noStill, score: "@crypt", voicing: "@crypt-dmg", bar: 0},
			"bar 0 is before the first"},
		{"film into a folder that is not there", asked{script: script, short: "where", out: filepath.Join(dir, "none", "x.mp4"), still: noStill},
			""},
	}
	for _, c := range cases {
		var out bytes.Buffer
		err := write(c.given, &out)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want it to contain %q", c.name, err, c.want)
		}
		if out.Len() != 0 {
			t.Errorf("%s: wrote %q", c.name, out.String())
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "x.png")); err == nil {
		t.Error("a frame past the end left a file")
	}
}
