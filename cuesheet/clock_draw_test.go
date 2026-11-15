//go:build draw

package cuesheet_test

import (
	"errors"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/cheeptune"
	"github.com/janearc/slayer/cuesheet"
)

// the tune the piece clock is held to, the one she loves, and the moments
// it is asked about: picked by the author of this package, 2026-11-12
const (
	tune    = "../cheeptune/examples/sunrun.json"
	voicing = "@leap-dmg"
)

var offsets = []time.Duration{0, 1300 * time.Millisecond,
	7 * time.Second, 15250 * time.Millisecond}

// opened is sunrun, or the test stops.
func opened(t *testing.T) (*cheeptune.Piece, cuesheet.Clock) {
	t.Helper()
	piece, err := cheeptune.Open(tune, voicing)
	if err != nil {
		t.Fatal(err)
	}
	made, err := cuesheet.Piece(piece)
	if err != nil {
		t.Fatal(err)
	}
	return piece, made
}

// The piece clock agrees with the piece's own playhead on how many beats
// a moment is, and has its meter and bars.
func TestThePieceClockAgreesWithThePiece(t *testing.T) {
	piece, made := opened(t)
	for _, at := range offsets {
		_, _, beats, _, ok := piece.At(at)
		if !ok || math.Abs(made.Beats(at)-beats) > near {
			t.Errorf("at %v the clock says %v beats, the piece %v", at,
				made.Beats(at), beats)
		}
		if back := made.Moment(made.Beats(at)); (back - at).Abs() >
			time.Microsecond {
			t.Errorf("%v beats is %v, not %v", made.Beats(at), back, at)
		}
	}
	if made.Meter() != piece.Score.Meter ||
		made.Bars() != piece.Score.Bars() {
		t.Errorf("the clock has %d beats a bar and %d bars", made.Meter(),
			made.Bars())
	}
}

// Each of the piece's roles is a voice with the notes the piece plays.
func TestThePiecesRolesAreVoices(t *testing.T) {
	piece, made := opened(t)
	for _, part := range piece.Score.Parts {
		notes, err := made.Notes(part.Role)
		if err != nil {
			t.Errorf("%s: %v", part.Role, err)
		}
		if len(notes) != len(piece.Notes(part.Role)) {
			t.Errorf("%s has %d notes, the piece %d", part.Role,
				len(notes), len(piece.Notes(part.Role)))
		}
	}
}

// anti-test of TestThePiecesRolesAreVoices
// A voice the piece has not is refused by name.
func TestAVoiceThePieceHasNotIsRefused(t *testing.T) {
	_, made := opened(t)
	if _, err := made.Notes("kazoo"); !errors.Is(err, cuesheet.ErrVoice) {
		t.Errorf("kazoo: %v; want ErrVoice", err)
	}
}

// the tempo change the stepped tune is given: from bar five, half again
// as fast, picked by the author of this package, 2026-11-12
const steppedTempo = `"tempos": [{"bar": 5, "tempo": 210}],`

// steppedPiece is sunrun with its tempo stepped from bar five, played once.
func steppedPiece(t *testing.T) *cheeptune.Piece {
	t.Helper()
	text, err := os.ReadFile(tune)
	if err != nil {
		t.Fatal(err)
	}
	stepped := strings.Replace(string(text), `"meter": 4,`,
		`"meter": 4, `+steppedTempo, 1)
	stepped = strings.Replace(stepped, `"loop": true`, `"loop": false`, 1)
	score, err := cheeptune.ParseScore([]byte(stepped))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := cheeptune.ReadDoc(voicing)
	if err != nil {
		t.Fatal(err)
	}
	voiced, err := cheeptune.ParseVoicing(raw)
	if err != nil {
		t.Fatal(err)
	}
	piece, err := cheeptune.Render(score, voiced)
	if err != nil {
		t.Fatal(err)
	}
	return piece
}

// A piece whose tempo steps keeps its beats where the steps put them:
// the clock agrees with the piece's own playhead past the step, and a
// count of beats comes back to its moment.
func TestThePieceClockFollowsTempoSteps(t *testing.T) {
	piece := steppedPiece(t)
	made, err := cuesheet.Piece(piece)
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range offsets {
		_, _, beats, _, ok := piece.At(at)
		if !ok || math.Abs(made.Beats(at)-beats) > near ||
			(made.Moment(beats)-at).Abs() > time.Microsecond {
			t.Errorf("at %v the clock says %v beats, the piece %v", at,
				made.Beats(at), beats)
		}
	}
	// anti-test of TestThePieceClockFollowsTempoSteps
	plain := float64(15250*time.Millisecond) / float64(time.Minute) * 140
	if made.Beats(15250*time.Millisecond) <= plain {
		t.Error("past the step the clock counts beats at the old tempo")
	}
}
