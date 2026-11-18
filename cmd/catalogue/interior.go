package main

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// interiorPlan holds a stationary floor-level study in model coordinates:
// X east, Y up, Z south. The shell stays present unless cutaway says otherwise.
// Stars and air are authored display choices, not a measured night sky.
type interiorPlan struct {
	At         maths.Vec      `json:"at" yaml:"at"`
	Look       maths.Vec      `json:"look-at" yaml:"look-at"`
	Stars      sky.Stars      `json:"stars" yaml:"stars"`
	AirRadius  float64        `json:"air-radius-metres" yaml:"air-radius-metres"`
	Extinction float64        `json:"extinction-per-metre" yaml:"extinction-per-metre"`
	Beams      []interiorBeam `json:"beams" yaml:"beams"`
}

// interiorBeam describes an authored launcher pose; it does not assert that
// the real instrument has this laser installation.
type interiorBeam struct {
	At         maths.Vec   `json:"at" yaml:"at"`
	Toward     maths.Vec   `json:"toward" yaml:"toward"`
	Strength   maths.Light `json:"strength" yaml:"strength"`
	Radius     float64     `json:"radius-metres" yaml:"radius-metres"`
	Reach      float64     `json:"reach-metres" yaml:"reach-metres"`
	Divergence float64     `json:"divergence-deg" yaml:"divergence-deg"`
}

// interiorSky uses the same sky and participating air for beams on and off,
// so the comparison changes only the sources. Geometry remains opaque.
func (plan plan) interiorSky(world *thing.World) error {
	if plan.Interior == nil {
		return nil
	}
	view := plan.Interior
	world.Sky = view.Stars
	if view.Extinction > 0 {
		world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: view.AirRadius, Thick: view.Extinction},
			Material: &thing.Material{Name: "authored interior study air", Colour: maths.Grey(0.9)}})
	}
	for _, source := range view.Beams {
		beam, err := emit.NewBeam(emit.Lit{At: source.At, Toward: source.Toward, Strength: source.Strength},
			measure.Metres(source.Radius), measure.Metres(source.Reach), measure.Degrees(source.Divergence))
		if err != nil {
			return err
		}
		world.Lamp(beam)
	}
	return nil
}
