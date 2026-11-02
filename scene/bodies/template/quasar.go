package template

import (
	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
)

// Quasar is a hole with a disc round it that outshines its galaxy: the
// mass of ton 618, a disc from three times the hole's radius to thirty,
// twelve thousand kelvin at its hottest and orange at the rim, its gas
// at the inner edge going round at two fifths of the speed of light.
//
// Its jets run along the spin from two radii out to three hundred,
// narrow, their bulk at six tenths of the speed of light, x-ray and
// gamma bright. A real disc is hotter inside, into the ultraviolet;
// this one is tuned to be looked at.
//
// The hole bends light within twenty of its schwarzschild radii; its
// black is the rays that came from the horizon, and its shadow, wider,
// is what the bending makes. With Edge.Hexes a ball of hex lines sits
// just outside the horizon.
//
// The disc has no streaks: a shot that wants them turns them on. A
// blazar is this body seen down its jet: a shot of a Quasar, not a
// template, since the body cannot know where it is pointed.
func Quasar() bodies.Body {
	mass := 6.6e10 * bodies.Sun
	rs := bodies.Schwarzschild(mass)
	return bodies.Body{Name: "quasar", Kind: bodies.Hole, Mass: mass, Radius: rs,
		Seed: 7,
		Disc: &bodies.Disc{Inner: 3 * rs, Outer: 30 * rs, Thick: 0.3 * rs, Dense: 10 / rs,
			Hot: 12000, Bright: 60, Speed: 0.41, Shear: 1.5},
		Jets: &bodies.Jets{Length: 300 * rs, Base: 1 * rs, Spread: 0.03, From: 2 * rs,
			Dense: 2 / rs, Speed: 0.6,
			Glow: swatches.Light(swatches.JetBlue),
			Radiant: thing.Spectrum{
				Gives:  map[radiation.Range]float64{radiation.XRay(): 1, radiation.Gamma(): 0.5},
				Passes: map[radiation.Range]float64{radiation.XRay(): 0.97, radiation.Gamma(): 0.98}}},
	}
}
