package template

import (
	"fmt"
	"time"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
)

// Pulsar is the crab's: a neutron star twenty kilometres across turning
// thirty times a second, bright in x-ray, its beams fifteen hundred
// kilometres long along a magnetic axis forty-five degrees off the
// spin, so they sweep.
func Pulsar() (bodies.Body, error) {
	beams, err := bodies.NewJets(km(1500), km(30), 0.12, km(0),
		bodies.PerKilometre(0.004), deg(45), maths.Share(0),
		swatches.Light(swatches.PulsarBeam))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("pulsar: %w", err)
	}
	beams = beams.Radiating(thing.Spectrum{
		Gives: map[radiation.Range]float64{radiation.XRay(): 0.8,
			radiation.Radio(): 1},
		Passes: map[radiation.Range]float64{radiation.XRay(): 0.97,
			radiation.Radio(): 0.97}})
	pulsar, err := bodies.NewBody("pulsar", bodies.Star, km(10),
		kg(1.4*bodies.Sun), beams)
	if err != nil {
		return bodies.Body{}, err
	}
	pulsar = pulsar.Seeded(8).
		Turning(33*time.Millisecond, deg(0)).
		Glowing(swatches.Light(swatches.PulsarWhite),
			thing.Spectrum{Gives: map[radiation.Range]float64{
				radiation.XRay(): 1}})
	return pulsar, nil
}
