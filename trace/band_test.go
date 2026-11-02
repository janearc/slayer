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

// In gamma, a source shows at its strength; a wall with no radiant in
// front of it is black; a ball that passes gamma dims it by its through
// to the power of the path, and the sky is dark.
func TestABandPassesThroughWhatPassesIt(t *testing.T) {
	eye := tenBack()
	lens := film.Pinhole{Wide: 20 * maths.Degree}
	at := func(world *thing.World) float64 {
		picture := drawIn(world, eye, lens, 0, 21, 21, gammaBand)
		return float64(picture.At(10, 10).R)
	}
	bare := thing.NewWorld()
	bare.Add(source())
	if got := at(bare); !within(got, 5, 1e-6) {
		t.Fatalf("the source alone is %v", got)
	}
	wall := thing.NewWorld()
	wall.Add(source())
	wall.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "wall"}})
	if got := at(wall); got != 0 {
		t.Fatalf("behind a wall, %v", got)
	}
	glass := thing.NewWorld()
	glass.Add(source())
	glass.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "gas",
			Radiant: thing.Spectrum{
				Passes: map[radiation.Range]float64{
					radiation.Gamma(): 0.5}}}})
	// through two units of a ball that passes a half a unit: a quarter
	if got := at(glass); !within(got, 5*0.25, 1e-3) {
		t.Fatalf("through the ball, %v, want %v", got, 5*0.25)
	}
	empty := thing.NewWorld()
	if got := at(empty); got != 0 {
		t.Fatalf("the sky gives %v", got)
	}
	// visible is the ordinary render: the source is dark without a lamp
	vis := drawIn(bare, eye, lens, 0, 21, 21, maths.Band{}).At(10, 10)
	if vis.R != 0 || vis.Cover != 1 {
		t.Fatalf("in visible light the source is %+v", vis)
	}
}

// boostAlong is how much brighter a beamed jet's glow is than the same
// jet's at rest, seen down its axis from above: the beam's boost at
// each point of the axis, weighed by the light that point sends up
// through the jet above it. It is added up here in fine steps, apart
// from the tracer's own march.
func boostAlong(jet medium.Jet, beam thing.Beam) float64 {
	const cuts = 4000
	step := 2 * jet.Length / cuts
	keep, rest, moving := 1.0, 0.0, 0.0
	for i := range cuts {
		point := maths.V(0, jet.Length-(float64(i)+0.5)*step, 0)
		took := 1 - math.Exp(-jet.Density(point)*step)
		rest += keep * took
		moving += keep * took * beam.Boost(maths.Up, point)
		keep *= 1 - took
	}
	return moving / rest
}

// A beamed jet is brighter seen coming than at rest, and dimmer seen
// side-on, in visible light and in a band alike.
func TestABeamedJetIsBrightComingAndFaintGoing(t *testing.T) {
	xray := maths.Band{Nanometres: 1}
	shot := func(speed float64, from maths.Vec) (float32, float32) {
		world := thing.NewWorld()
		jet := thing.Volume{Medium: medium.Jet{Length: 4, Base: 1,
			Spread: 0.1, Thick: 1},
			Material: &thing.Material{Name: "jet",
				Glow: maths.Grey(0.2),
				Radiant: thing.Spectrum{
					Gives: map[radiation.Range]float64{
						radiation.XRay(): 0.2}}}}
		if speed > 0 {
			jet.Beam = &thing.Beam{Speed: speed}
		}
		world.AddVolume(jet)
		eye := lookingAt(from, maths.Vec{})
		lens := film.Pinhole{Wide: 20 * maths.Degree}
		vis := draw(world, eye, lens, 0, 21, 21).At(10, 10)
		band := drawIn(world, eye, lens, 0, 21, 21, xray).At(10, 10)
		return vis.R, band.R
	}
	restVis, restBand := shot(0, maths.V(0, 12, 0))
	comingVis, comingBand := shot(0.9, maths.V(0, 12, 0))
	sideVis, sideBand := shot(0.9, maths.V(12, 0, 0))
	// how much brighter: the beam's own boost, added up along the
	// jet's axis as the light is, within five percent. The band shows
	// only the jet's near skin, where the boost is the near jet's.
	beam := thing.Beam{Speed: 0.9}
	jet := medium.Jet{Length: 4, Base: 1, Spread: 0.1, Thick: 1}
	wantVis := boostAlong(jet, beam)
	wantBand := beam.Boost(maths.Up, maths.V(0, 2, 0))
	gotVis := float64(comingVis / restVis)
	gotBand := float64(comingBand / restBand)
	if !within(gotVis, wantVis, 0.05*wantVis) ||
		!within(gotBand, wantBand, 0.05*wantBand) {
		t.Errorf("coming, the jet is %.2f times itself at rest and "+
			"%.2f in the band; want %.2f and %.2f", gotVis, gotBand,
			wantVis, wantBand)
	}
	if !(sideVis < restVis && sideBand < restBand) {
		t.Errorf("side-on %v/%v, at rest %v/%v", sideVis, sideBand,
			restVis, restBand)
	}
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
	world.Lamp(thing.Sun(maths.XAxis, maths.Grey(1)))
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
	world.Lamp(thing.Sun(maths.XAxis, maths.Grey(sun)))
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
