//go:build draw

package hillside

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/interfaces/file"
)

// playedJob is a film of zappy, one frame at the tests' size, played by
// a sheet of the cues given, with sunrun as its sound and its clock.
func playedJob(t *testing.T, cues string) string {
	t.Helper()
	sheet := filepath.Join(t.TempDir(), "cues.yaml")
	if err := os.WriteFile(sheet, []byte("cues: "+cues+"\n"),
		0o600); err != nil {
		t.Fatal(err)
	}
	tune, err := filepath.Abs("../cheeptune/examples/sunrun.json")
	if err != nil {
		t.Fatal(err)
	}
	return namedJob("named: zappy, play: "+sheet,
		"fps: 2\nduration: {value: 0.5, unit: s}\n"+
			"sound: {tune: "+tune+", voicing: '@leap-dmg'}\n")
}

// filmFrame is the developed first frame of a film job, as png bytes.
func filmFrame(t *testing.T, document string) []byte {
	t.Helper()
	session := opening(t, jobFrom(t, document))
	drawn, err := session.Frame(context.Background(), nothing, Ask{})
	if err != nil {
		t.Fatal(err)
	}
	picture := session.Tone().Develop(drawn.Light())
	out := filepath.Join(t.TempDir(), "frame.png")
	if err := file.PNG(out, picture, sessionSquare); err != nil {
		t.Fatal(err)
	}
	read, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return read
}

// A job's play sheet reaches the picture: zappy with its star put out
// from the first bar is not zappy at rest, and a sheet that leaves every
// channel at rest is.
func TestAPlaySheetPlaysTheScene(t *testing.T) {
	rest := filmFrame(t, playedJob(t, "[]"))
	dark := filmFrame(t, playedJob(t,
		`[{at: "1.1", star-brightness: 0, beam-brightness: 0}]`))
	if bytes.Equal(rest, dark) {
		t.Error("the sheet's dark star did not reach the picture")
	}
	plain := filmFrame(t, namedJob("named: zappy",
		"fps: 2\nduration: {value: 0.5, unit: s}\n"))
	if !bytes.Equal(rest, plain) {
		t.Error("a sheet at rest is not zappy unplayed")
	}
}

// anti-tests: a sheet naming a channel zappy has not is refused at Open,
// by name, with the job file.
func TestOpenRefusesASheetZappyCannotPlay(t *testing.T) {
	wanted := jobFrom(t, playedJob(t, `[{at: "1.1", star-shine: 2}]`))
	_, err := Open(context.Background(), wanted, Stamp{})
	if err == nil || !strings.Contains(err.Error(), "star-shine") ||
		!strings.Contains(err.Error(), wanted.File()) {
		t.Errorf("a sheet with star-shine gave %v", err)
	}
}
