package colour

import (
	"image/color"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	ninjacolour "github.com/janearc/slayer/ninjafilm/colour"
	"github.com/janearc/slayer/ninjafilm/data"
)

// This package's tests reach the print that ninjafilm/colour holds by
// the names they knew it by when this package held a copy, so each one
// now checks the one definition through this package's door.

// bloomed is ninjafilm's bloom, which a tone's pictures are bloomed by.
func bloomed(lights []maths.Light, w, h int,
	exposure, radius float64) []maths.Light {
	return ninjacolour.Bloom(lights, w, h, exposure, radius)
}

// widest is the widest bloom, in pixels' sigma: ninjafilm's picked one.
var widest = data.Picked("bloom-widest")

// scaleOf is the shoulder's scale for a knee and a top, as a tone's
// print solves it.
func scaleOf(knee, top float64) float64 {
	return Tone{Knee: knee, Top: top}.Print().Scale()
}

// printed is one light as the screen's bytes through a tone.
func printed(tone Tone, light maths.Light) color.RGBA {
	return tone.Print().Picture([]maths.Light{light}, onePixel,
		onePixel).RGBAAt(0, 0)
}

// onePixel is the side of a picture of one light.
const onePixel = 1 // defined: one pixel across and one down

// shown is every range a band may stand in, for the test that walks
// them: gamma to microwave, fermi left out, an instrument's sensitivity
// and not a range of the spectrum.
var shown = []radiation.Range{radiation.Gamma(), radiation.XRay(),
	radiation.ExtremeUltraviolet(), radiation.Ultraviolet(),
	radiation.Infrared(), radiation.Radio(), radiation.Microwave()}

// the anti-test of the door: a tone whose print loses its knee draws
// bright light as the clamp does, and this package's pictures do not,
// so they are drawn through the print with the tone's own numbers.
func TestATonesPrintKeepsItsKnee(t *testing.T) {
	bright := maths.Grey(brightGrey)
	clamped := printed(Tone{}, bright)
	rolled := printed(Tone{Knee: doorKnee}, bright)
	if rolled == clamped {
		t.Fatalf("a knee of %v printed %v, as the clamp does", doorKnee,
			rolled)
	}
	want := ninjacolour.NewTone(0, doorKnee, 0, 0) // defined: no other part
	if got := (Tone{Knee: doorKnee}).Print(); got != want {
		t.Errorf("the print of a knee of %v is %+v", doorKnee, got)
	}
}

// the door's numbers
const (
	// brightGrey is a light well past white, past any knee.
	brightGrey = 3 // picked: past white, under the default top
	// doorKnee is a knee the door's test prints through.
	doorKnee = 0.6 // picked: the knee colour_test develops with
)
