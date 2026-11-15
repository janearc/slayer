//go:build draw

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
