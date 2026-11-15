//go:build draw

package door

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/cheeptune"
)

// a tune shorter than the film loops to its end when its score loops,
// and is cut there; one longer is cut at the film's end.
func TestTunedLoopsAndCuts(t *testing.T) {
	piece, err := cheeptune.Open("@crypt", "@crypt-dmg")
	if err != nil {
		t.Fatal(err)
	}
	if piece.LoopPeriod() == nothing {
		t.Fatal("crypt no longer loops; choose a looping tune")
	}
	longer := piece.Duration() * stereo
	looped, err := tuned("@crypt", "@crypt-dmg", longer)
	if err != nil {
		t.Fatal(err)
	}
	if looped.sound.Length() != sampleCount(piece.Rate, longer)/stereo ||
		!strings.Contains(looped.said, "loops to the film's end") {
		t.Errorf("a looping tune for %s gave %s: %q", longer,
			looped.sound.Duration(), looped.said)
	}
	cut, err := tuned("@crypt", "@crypt-dmg", time.Second)
	if err != nil || !strings.Contains(cut.said, "cut at the film's end") {
		t.Errorf("a tune longer than the film said %q, %v", cut.said, err)
	}
}

// a tune that does not loop is followed by silence, never
// played twice, and a voicing given as a score is refused.
// anti-test of TestTunedLoopsAndCuts
func TestTunedLeavesAOneShotAlone(t *testing.T) {
	score := filepath.Join(t.TempDir(), "once.json")
	raw, err := os.ReadFile("../cheeptune/examples/sunrun.json")
	if err != nil {
		t.Fatal(err)
	}
	once := bytes.Replace(raw, []byte(`"loop": true`),
		[]byte(`"loop": false`), one)
	if bytes.Equal(once, raw) {
		t.Fatal("sunrun no longer says it loops")
	}
	if err := os.WriteFile(score, once, private); err != nil {
		t.Fatal(err)
	}
	piece, err := cheeptune.Open(score, "@leap-dmg")
	if err != nil {
		t.Fatal(err)
	}
	made, err := tuned(score, "@leap-dmg", piece.Duration()*stereo)
	if err != nil {
		t.Fatal(err)
	}
	if made.sound.Length() != len(piece.Left) ||
		!strings.Contains(made.said, "silence follows") {
		t.Errorf("a one-shot tune was stretched: %q", made.said)
	}
	if _, err := tuned("@crypt-dmg", "@crypt", time.Second); err == nil {
		t.Error("a voicing as a score was taken")
	}
}
