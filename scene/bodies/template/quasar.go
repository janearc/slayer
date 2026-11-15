package template

import (
	"fmt"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
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
// is what the bending makes. With a NewHorizon's hexes a ball of hex
// lines sits just outside the horizon.
//
// The disc has no streaks: a shot that wants them turns them on. A
// blazar is this body seen down its jet: a shot of a Quasar, not a
// template, since the body cannot know where it is pointed.
func Quasar() (bodies.Body, error) {
	mass := 6.6e10 * bodies.Sun
	rs := bodies.Schwarzschild(mass)
	disc, err := bodies.NewDisc(km(3*rs), km(30*rs), km(0.3*rs),
		bodies.PerKilometre(10/rs), measure.Kelvin(12000), 60)
	if err != nil {
		return bodies.Body{}, fmt.Errorf("quasar: %w", err)
	}
	disc, err = disc.Beamed(maths.Share(0.41))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("quasar: %w", err)
	}
	disc = disc.Dressed(0, 1.5)
	jets, err := bodies.NewJets(km(300*rs), km(1*rs), 0.03, km(2*rs),
		bodies.PerKilometre(2/rs), deg(0), maths.Share(0.6),
		swatches.Light(swatches.JetBlue))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("quasar: %w", err)
	}
	jets = jets.Radiating(thing.Spectrum{
		Gives: map[radiation.Range]float64{radiation.XRay(): 1,
			radiation.Gamma(): 0.5},
		Passes: map[radiation.Range]float64{radiation.XRay(): 0.97,
			radiation.Gamma(): 0.98}})
	quasar, err := bodies.NewBody("quasar", bodies.Hole, km(rs),
		kg(mass), disc, jets)
	if err != nil {
		return bodies.Body{}, err
	}
	return quasar.Seeded(7), nil
}
