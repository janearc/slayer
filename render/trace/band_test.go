package trace

import (
	"math"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

var gamma = maths.Band{Nanometres: 1e-4}

// source is a ball that gives off gamma, far along minus z.
func source() thing.Thing {
	return thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "magnetar",
			Radiant: thing.Spectrum{Gives: map[radiation.Range]float64{radiation.Gamma(): 5}}},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -20)})}
}

// In gamma, a source shows at its strength; a wall with no radiant in
// front of it is black; a ball that passes gamma dims it by its through
// to the power of the path, and the sky is dark.
func TestABandPassesThroughWhatPassesIt(t *testing.T) {
	eye := maths.Subspace{Origin: maths.V(0, 0, 10),
		Orientation: maths.LookAt(maths.V(0, 0, 10), maths.Vec{})}
	lens := film.Pinhole{Wide: 20 * maths.Degree}
	at := func(w *thing.World) float64 {
		return float64(RenderIn(w, eye, lens, 0, 21, 21, gamma).At(10, 10).R)
	}
	bare := thing.NewWorld()
	bare.Add(source())
	if got := at(bare); math.Abs(got-5) > 1e-6 {
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
			Radiant: thing.Spectrum{Passes: map[radiation.Range]float64{radiation.Gamma(): 0.5}}}})
	// through two units of a ball that passes a half a unit: a quarter
	if got := at(glass); math.Abs(got-5*0.25) > 1e-3 {
		t.Fatalf("through the ball, %v, want %v", got, 5*0.25)
	}
	empty := thing.NewWorld()
	if got := at(empty); got != 0 {
		t.Fatalf("the sky gives %v", got)
	}
	// visible is the ordinary render: the source is dark without a lamp
	vis := RenderIn(bare, eye, lens, 0, 21, 21, maths.Band{}).At(10, 10)
	if vis.R != 0 || vis.Cover != 1 {
		t.Fatalf("in visible light the source is %+v", vis)
	}
}

// A radar lights nothing in visible light; in its own band a thing that
// returns it shows by how squarely the beam lands, how rough it is
// there, and what passes between; a thing that returns nothing is dark.
// Twelve and a half centimetres, magellan's band, is radio as libtheme
// draws the ranges: its microwave stops at fifteen millimetres.
func TestARadarSeesWhatScattersBack(t *testing.T) {
	micro := maths.Band{Nanometres: 1.26e8}
	eye := maths.Subspace{Origin: maths.V(0, 0, 10),
		Orientation: maths.LookAt(maths.V(0, 0, 10), maths.Vec{})}
	lens := film.Pinhole{Wide: 30 * maths.Degree}
	rough := thing.NewWorld()
	rough.Add(thing.Thing{Shape: shape.Sphere{Radius: 2},
		Material: &thing.Material{Name: "ground",
			Radiant: thing.Spectrum{Returns: map[radiation.Range]float64{radiation.Radio(): 0.8}}}})
	rough.Lamp(film.Radar{At: maths.V(0, 0, 10), Range: radiation.Radio(), Strength: 1})
	f := RenderIn(rough, eye, lens, 0, 41, 41, micro)
	mid, edge := f.At(20, 20).R, f.At(20, 7).R
	if math.Abs(float64(mid)-0.8) > 1e-3 || !(edge > 0 && edge < mid) {
		t.Fatalf("the middle returns %v, the edge %v", mid, edge)
	}
	if vis := Render(rough, eye, lens, 0, 41, 41).At(20, 20); vis.R != 0 {
		t.Fatalf("in visible light the radar lit the ground: %+v", vis)
	}
	smooth := thing.NewWorld()
	smooth.Add(thing.Thing{Shape: shape.Sphere{Radius: 2},
		Material: &thing.Material{Name: "mirror",
			Radiant: thing.Spectrum{Returns: map[radiation.Range]float64{radiation.Radio(): 0.8}},
			Rough:   texture.Checker(100, maths.Grey(0.25), maths.Grey(0.25))}})
	smooth.Lamp(film.Radar{At: maths.V(0, 0, 10), Range: radiation.Radio(), Strength: 1})
	if got := RenderIn(smooth, eye, lens, 0, 41, 41, micro).At(20, 20).R; math.Abs(float64(got)-0.2) > 1e-3 {
		t.Fatalf("a quarter as rough returns %v", got)
	}
	dark := thing.NewWorld()
	dark.Add(thing.Thing{Shape: shape.Sphere{Radius: 2},
		Material: &thing.Material{Name: "wall"}})
	dark.Lamp(film.Radar{At: maths.V(0, 0, 10), Range: radiation.Radio(), Strength: 1})
	if got := RenderIn(dark, eye, lens, 0, 41, 41, micro).At(20, 20).R; got != 0 {
		t.Fatalf("a wall returns %v", got)
	}
}

// A beamed jet is brighter seen coming than at rest, and dimmer seen
// side-on, in visible light and in a band alike.
func TestABeamedJetIsBrightComingAndFaintGoing(t *testing.T) {
	xray := maths.Band{Nanometres: 1}
	render := func(speed float64, from maths.Vec) (float32, float32) {
		w := thing.NewWorld()
		v := thing.Volume{Medium: medium.Jet{Length: 4, Base: 1, Spread: 0.1, Thick: 1},
			Material: &thing.Material{Name: "jet", Glow: maths.Grey(0.2),
				Radiant: thing.Spectrum{Gives: map[radiation.Range]float64{radiation.XRay(): 0.2}}}}
		if speed > 0 {
			v.Beam = &thing.Beam{Speed: speed}
		}
		w.AddVolume(v)
		eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
		lens := film.Pinhole{Wide: 20 * maths.Degree}
		vis := Render(w, eye, lens, 0, 21, 21).At(10, 10)
		band := RenderIn(w, eye, lens, 0, 21, 21, xray).At(10, 10)
		return vis.R, band.R
	}
	restVis, restBand := render(0, maths.V(0, 12, 0))
	comingVis, comingBand := render(0.9, maths.V(0, 12, 0))
	sideVis, sideBand := render(0.9, maths.V(12, 0, 0))
	if !(comingVis > 5*restVis && comingBand > 5*restBand) {
		t.Errorf("coming %v/%v, at rest %v/%v", comingVis, comingBand, restVis, restBand)
	}
	if !(sideVis < restVis && sideBand < restBand) {
		t.Errorf("side-on %v/%v, at rest %v/%v", sideVis, sideBand, restVis, restBand)
	}
}

// Excited gas glows in its line only where a lamp reaches it: a ball of
// it lit from one side glows on that side and not in its own shadow.
func TestExcitedGasGlowsWhereTheLampReaches(t *testing.T) {
	halpha := maths.Band{Nanometres: 656}
	w := thing.NewWorld()
	w.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 2, Thick: 3},
		Material: &thing.Material{Name: "gas", Colour: maths.Grey(0.5),
			Radiant: thing.Excited{Lines: map[float64]float64{656: 1}, Width: 2,
				Passes: 0.5}}})
	w.Lamp(thing.Sun(maths.XAxis, maths.Grey(1)))
	eye := maths.Subspace{Origin: maths.V(0, 0, 10),
		Orientation: maths.LookAt(maths.V(0, 0, 10), maths.Vec{})}
	lens := film.Pinhole{Wide: 30 * maths.Degree}
	f := RenderIn(w, eye, lens, 0, 41, 41, halpha)
	// the sun is from plus x, the picture's right seen from plus z
	lit, shadow := f.At(32, 20).R, f.At(8, 20).R
	if !(lit > 0.05 && shadow < 0.2*lit) {
		t.Errorf("the lit side glows %v, the shadowed %v", lit, shadow)
	}
	w.Lamps = nil
	if dark := RenderIn(w, eye, lens, 0, 41, 41, halpha).At(8, 20).R; dark != 0 {
		t.Errorf("with no lamp the gas glows %v", dark)
	}
}

// A volume that gives off a band and passes it glows along its whole
// depth; one that gives it off and does not pass it shows only its
// skin, and is far fainter.
func TestAVolumeMustPassTheBandItGivesOff(t *testing.T) {
	xray := maths.Band{Nanometres: 1}
	render := func(passes float64) float32 {
		w := thing.NewWorld()
		w.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 2, Thick: 0.5},
			Material: &thing.Material{Name: "gas",
				Radiant: thing.Spectrum{Gives: map[radiation.Range]float64{radiation.XRay(): 1},
					Passes: map[radiation.Range]float64{radiation.XRay(): passes}}}})
		eye := maths.Subspace{Origin: maths.V(0, 0, 10),
			Orientation: maths.LookAt(maths.V(0, 0, 10), maths.Vec{})}
		return RenderIn(w, eye, film.Pinhole{Wide: 20 * maths.Degree}, 0, 21, 21, xray).At(10, 10).R
	}
	if deep, skin := render(0.9), render(0); !(deep > 10*skin) {
		t.Errorf("passing its band the gas glows %v, not passing it %v", deep, skin)
	}
}
