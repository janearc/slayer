//go:build draw

package fixture_test

import (
	"math"
	"testing"

	binned "github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/burning-trousers/fake"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// An rgb tracer lights a colour with a colour by multiplying three
// numbers; light multiplies spectra, wavelength by wavelength. Blue paint
// lit by a warm lamp: how far slayer's picture is from the paint as light
// makes it.
//
// Render's pixel, the paint's rgb taken under daylight as a painter would,
// is logged. slayer drawn band by band, the paint scattering each band and
// the lamp shining its own spectrum, is held to one just-noticeable
// difference.
func TestColourUnderColouredLight(t *testing.T) {
	seen := readObserver(t)
	lamp := swatch.Blackbody(2700)
	truth := seen.integrate(times(lamp, blue))
	t.Logf("blue paint under a 2700 K lamp: the rgb tracer is %.2f JND off",
		apart(rgbUnder(t, seen, lamp), truth))
	drawn := seen.integrate(drawnInBands(t, blueAt, lamp))
	gap := apart(drawn, truth)
	t.Logf("drawn band by band it is %.2f JND off", gap)
	if gap > 1 {
		t.Errorf("blue paint under a 2700 K lamp, drawn band by band, is "+
			"%.2f JND from the light's own colour", gap)
	}
}

// rgbUnder is Render's colour of blue paint under a lamp of a spectrum:
// the paint's rgb as daylight shows it, times the lamp's rgb.
func rgbUnder(t *testing.T, seen observer,
	lamp map[int]float64) swatch.Swatch {
	t.Helper()
	daylight := swatch.D65()
	white := channels(seen.integrate(daylight))
	paint := channels(seen.integrate(times(daylight, blue)))
	for i := range paint {
		paint[i] /= white[i]
	}
	glow := channels(seen.integrate(lamp))
	pixel := drawnUnder(t, maths.RGB(paint[0], paint[1], paint[2]),
		maths.RGB(glow[0], glow[1], glow[2]))
	return srgb.FromLight(pixel[0], pixel[1], pixel[2])
}

// scattering is a paint as a band sees it: it passes and gives off
// nothing, and sends back of each band the share it says.
type scattering func(nm float64) float64

// shortest and longest are the observer table's ends, in nanometres.
const shortest, longest = 360, 830

// drawnInBands is slayer's picture of a plane of a paint straight under a
// lamp of a spectrum, drawn in each band the observer table spans: each
// band's light at the nanometre nearest its middle.
func drawnInBands(t *testing.T, paint scattering,
	spectrum map[int]float64) map[int]float64 {
	t.Helper()
	ground := &thing.Thing{Shape: fake.NewPlane(100, nil),
		Material: &thing.Material{Radiant: paint}}
	lamp := emit.Point{At: maths.V(0, 2, 0),
		Light: binned.Split(spectrum)}.When(0)
	world := &thing.World{Things: []*thing.Thing{ground},
		Lamps: []thing.Lamp{lamp}}
	look := maths.Ray{At: maths.V(0, 1, 1), Dir: maths.V(0, -1, -1).Unit()}
	drawn := map[int]float64{}
	for band := binned.Of(shortest); band <= binned.Of(longest); band++ {
		short, long := band.Edges()
		middle := math.Sqrt(short * long)
		picture, err := trace.RenderIn(world, maths.Subspace{},
			fake.NewAimed(look), 0, 1, 1, maths.Band{Nanometres: middle})
		if err != nil {
			t.Fatal(err)
		}
		drawn[int(math.Round(middle))] += float64(picture.At(0, 0).R)
	}
	return drawn
}

// channels is a colour's linear srgb.
func channels(colour swatch.Swatch) [3]float64 {
	red, green, blue := srgb.Light(colour)
	return [3]float64{red, green, blue}
}

// drawnUnder is the tracer's pixel of a plane of a colour straight under
// a lamp of another.
func drawnUnder(t *testing.T, colour, glow maths.Light) [3]float64 {
	t.Helper()
	ground := &thing.Thing{Shape: fake.NewPlane(100, nil),
		Material: &thing.Material{Colour: colour}}
	world := &thing.World{Things: []*thing.Thing{ground},
		Lamps: []thing.Lamp{fake.NewLamp(maths.V(0, 2, 0), glow, nil)}}
	look := maths.Ray{At: maths.V(0, 1, 1), Dir: maths.V(0, -1, -1).Unit()}
	picture, err := trace.Render(world, maths.Subspace{}, fake.NewAimed(look),
		0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	seen := picture.At(0, 0)
	return [3]float64{float64(seen.R), float64(seen.G), float64(seen.B)}
}

// sodium is a low-pressure sodium lamp: nearly all its light in the two
// lines at 589.0 and 589.6 nanometres.
func sodium() map[int]float64 { return map[int]float64{589: 1, 590: 1} }

// held is how far from the truth slayer may draw blue paint under sodium,
// in just-noticeable differences, where the bar is one. it is loosened, with
// sign-off on 2026-11-01, to what was measured that day, 4.29.
//
// celestial carries light in 24 bands an octave. both sodium lines fall in
// the band from 574.7 to 591.6 nanometres, one number weighed at its
// middle, 583, and not at 589. no change in slayer can place them; the fix
// is celestial's bands carrying where in a band their light sits.
const held = 4.3

// The same blue paint under sodium light: the paint gives back what it
// gives back at 589 nanometres, so it shows as the lamp's own orange,
// darker. three numbers cannot know that. drawn band by band, slayer is
// held to held, until it comes within one just-noticeable difference and
// the hold goes.
func TestColourUnderLinesOfLight(t *testing.T) {
	seen := readObserver(t)
	truth := seen.integrate(times(sodium(), blue))
	t.Logf("blue paint under sodium: the rgb tracer is %.2f JND off",
		apart(rgbUnder(t, seen, sodium()), truth))
	drawn := seen.integrate(drawnInBands(t, blueAt, sodium()))
	gap := apart(drawn, truth)
	t.Logf("drawn band by band it is %.2f JND off", gap)
	switch {
	case gap > held:
		t.Errorf("blue paint under sodium light, drawn band by band, is "+
			"%.2f JND from the light's own colour, past the %.2f held",
			gap, held)
	case gap <= 1:
		t.Errorf("blue paint under sodium light is %.2f JND off, within "+
			"one: the hold has nothing behind it, so take it out", gap)
	}
}

func (scattering) Through(maths.Band) float64 { return 0 }

func (scattering) Emits(maths.Band) float64 { return 0 }

func (paint scattering) Scatters(band maths.Band) float64 {
	return paint(band.Nanometres)
}
