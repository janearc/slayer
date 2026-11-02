package template

import (
	"time"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
)

// Pulsar is the crab's: a neutron star twenty kilometres across turning
// thirty times a second, bright in x-ray, its beams fifteen hundred
// kilometres long along a magnetic axis forty-five degrees off the
// spin, so they sweep.
func Pulsar() bodies.Body {
	return bodies.Body{Name: "pulsar", Kind: bodies.Star, Radius: 10, Mass: 1.4 * bodies.Sun,
		Seed:    8,
		Day:     33 * time.Millisecond,
		Glow:    swatches.Light(swatches.PulsarWhite),
		Radiant: thing.Spectrum{Gives: map[radiation.Range]float64{radiation.XRay(): 1}},
		Jets: &bodies.Jets{Length: 1500, Base: 30, Spread: 0.12, Dense: 0.004,
			Tilt: deg(45),
			Glow: swatches.Light(swatches.PulsarBeam),
			Radiant: thing.Spectrum{
				Gives:  map[radiation.Range]float64{radiation.XRay(): 0.8, radiation.Radio(): 1},
				Passes: map[radiation.Range]float64{radiation.XRay(): 0.97, radiation.Radio(): 0.97}}},
	}
}
