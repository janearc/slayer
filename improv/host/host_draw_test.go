//go:build draw

package host

import (
	"math"
	"testing"

	"github.com/janearc/slayer/cheeptune"
	"github.com/janearc/slayer/interfaces/movie"
)

func libraryTune(t *testing.T) (*cheeptune.Piece, *cheeptune.CutSheet,
	movie.Sound) {
	t.Helper()
	names := cheeptune.Library()
	if len(names) == none {
		t.Fatal("the tune library is empty")
	}
	piece, sheet, err := cheeptune.OpenTune(names[none])
	if err != nil {
		t.Fatal(err)
	}
	samples := make([]float32, stereo*len(piece.Left))
	for i := range piece.Left {
		samples[stereo*i], samples[stereo*i+one] = piece.Left[i],
			piece.Right[i]
	}
	sound, err := movie.NewSound(piece.Rate, samples)
	if err != nil {
		t.Fatal(err)
	}
	return piece, sheet, sound
}

func TestATuneSaysItsPartsKicksAndLoudness(t *testing.T) {
	piece, sheet, sound := libraryTune(t)
	last := sheet.Parts[len(sheet.Parts)-one]
	tune, err := NewTune(piece.Score, sheet, sound, []string{last.Name})
	if err != nil {
		t.Fatal(err)
	}
	if tune.part(none) != sheet.Parts[none].Name ||
		tune.part(last.Second) != last.Name {
		t.Fatalf("the parts are %s and %s", tune.part(none),
			tune.part(last.Second))
	}
	onBeat := piece.Score.SecondsAt(math.Floor(
		piece.Score.BeatsAt(last.Second)) + one)
	if kick := tune.at(onBeat, false).Kick; kick < strong {
		t.Errorf("on a beat in a kicking part the kick is %v", kick)
	}
	if kick := tune.at(seconds/two*frameGap, false).Kick; kick != none &&
		sheet.Parts[none].Name != last.Name {
		t.Errorf("a part that does not kick kicks %v", kick)
	}
	for i := range frames {
		at := tune.at(float64(i)*frameGap, true)
		if at.Loudness < none || at.Loudness > one || !at.Speaking {
			t.Fatalf("frame %d is %+v", i, at)
		}
	}
	if tune.at(-one, false).Loudness != none ||
		tune.sound.Length() == none {
		t.Error("a moment before the tune is not silent")
	}
}

func TestNewTuneRefusesWhatIsNotThere(t *testing.T) {
	piece, sheet, sound := libraryTune(t)
	if _, err := NewTune(piece.Score, nil, sound, nil); err == nil {
		t.Error("a tune with no sheet was taken")
	}
	if _, err := NewTune(piece.Score, sheet, sound,
		[]string{"no such part"}); err == nil {
		t.Error("a part the sheet lacks was taken to kick")
	}
	if _, err := NewTune(piece.Score, sheet, movie.Sound{},
		nil); err == nil {
		t.Error("a tune with no sound was taken")
	}
}
