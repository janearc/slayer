package trace

import (
	"math"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

var gammaBand = maths.Band{Nanometres: 1e-4}

// source is a ball that gives off gamma, far along minus z.
func source() thing.Thing {
	return thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "magnetar",
			Radiant: thing.Spectrum{
				Gives: map[radiation.Range]float64{
					radiation.Gamma(): 5}}},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -20)})}
}

// Excited gas glows in its line only where a lamp reaches it: a ball of
// it lit from one side glows on that side and not in its own shadow.
func TestExcitedGasGlowsWhereTheLampReaches(t *testing.T) {
	halpha := maths.Band{Nanometres: 656}
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 2, Thick: 3},
		Material: &thing.Material{Name: "gas", Colour: maths.Grey(0.5),
			Radiant: thing.Excited{
				Lines: map[float64]float64{656: 1}, Width: 2,
				Passes: 0.5}}})
	light := maths.Grey(1)
	world.Lamp(thing.Sun(maths.XAxis, light))
	eye := tenBack()
	lens := film.Pinhole{Wide: 30 * maths.Degree}
	picture := drawIn(world, eye, lens, 0, 41, 41, halpha)
	// the sun is from plus x, the picture's right seen from plus z
	lit, shadow := picture.At(32, 20).R, picture.At(8, 20).R
	// measured 0.415 and 0.0079: the shadowed side is a fiftieth of
	// the lit one, and a tenth of it is let
	if !(lit > 0.4 && shadow < 0.1*lit) {
		t.Errorf("the lit side glows %v, the shadowed %v", lit, shadow)
	}
	world.Lamps = nil
	dark := drawIn(world, eye, lens, 0, 41, 41, halpha).At(8, 20).R
	if dark != 0 {
		t.Errorf("with no lamp the gas glows %v", dark)
	}
}

// lineGlow is how bright a ball of excited gas is on its lit side, in
// its line at 656 nanometres, under a sun of a strength and with the
// line wanting the lamplight to an excitation.
func lineGlow(sun, excitation float64) float64 {
	glows := thing.Excited{Lines: map[float64]float64{656: 1},
		Excitations: map[float64]float64{656: excitation},
		Width:       2, Passes: 0.5}
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 2, Thick: 3},
		Material: &thing.Material{Name: "gas", Colour: maths.Grey(0.5),
			Radiant: glows}})
	light := maths.Grey(sun)
	world.Lamp(thing.Sun(maths.XAxis, light))
	eye := tenBack()
	lens := film.Pinhole{Wide: 30 * maths.Degree}
	line := maths.Band{Nanometres: 656}
	return float64(drawIn(world, eye, lens, 0, 41, 41, line).At(32, 20).R)
}

// An excited gas glows as the lamplight reaching it, to the power of
// its line's excitation: under half the sun a line of excitation one
// glows half as much, and one of excitation two a quarter.
func TestExcitedGasGlowsByItsExcitation(t *testing.T) {
	for _, excitation := range []float64{1, 2} {
		full, half := lineGlow(1, excitation), lineGlow(0.5, excitation)
		want := math.Pow(0.5, excitation)
		if full > 0 && within(half/full, want, 1e-6) {
			continue
		}
		t.Errorf("at excitation %v half the sun gives %v of the "+
			"glow, %v of %v; want %v", excitation, half/full, half,
			full, want)
	}
}

// A volume that gives off a band and passes it glows along its whole
// depth; one that gives it off and does not pass it shows only its
// skin, and is far fainter.
func TestAVolumeMustPassTheBandItGivesOff(t *testing.T) {
	xray := maths.Band{Nanometres: 1}
	shot := func(passing float64) float32 {
		world := thing.NewWorld()
		world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 2,
			Thick: 0.5},
			Material: &thing.Material{Name: "gas",
				Radiant: thing.Spectrum{
					Gives: map[radiation.Range]float64{
						radiation.XRay(): 1},
					Passes: map[radiation.Range]float64{
						radiation.XRay(): passing}}}})
		eye := tenBack()
		lens := film.Pinhole{Wide: 20 * maths.Degree}
		return drawIn(world, eye, lens, 0, 21, 21, xray).At(10, 10).R
	}
	if deep, skin := shot(0.9), shot(0); !(deep > 10*skin) {
		t.Errorf("passing its band the gas glows %v, not passing it %v",
			deep, skin)
	}
}
