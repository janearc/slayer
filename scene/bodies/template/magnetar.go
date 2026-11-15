package template

import (
	"fmt"
	"time"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
)

// Magnetar is sgr 1806-20's: a neutron star of twelve kilometres
// turning once in seven and a half seconds, dim blue and hot in x-ray,
// and a giant flare a second in: a fireball from the surface at a tenth
// of the speed of light, gamma bright and nearly clear to gamma.
func Magnetar() (bodies.Body, error) {
	flare, err := bodies.NewFlare(time.Second, 30000, 0.8,
		swatches.Light(swatches.FlareWhite))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("magnetar: %w", err)
	}
	flare = flare.Radiating(thing.Spectrum{
		Gives: map[radiation.Range]float64{radiation.Gamma(): 2,
			radiation.XRay(): 1},
		Passes: map[radiation.Range]float64{radiation.Gamma(): 0.9}})
	magnetar, err := bodies.NewBody("magnetar", bodies.Star, km(12),
		kg(1.4*bodies.Sun), flare)
	if err != nil {
		return bodies.Body{}, err
	}
	magnetar = magnetar.Turning(7540*time.Millisecond, deg(0)).
		Glowing(swatches.Light(swatches.MagnetarBlue),
			thing.Spectrum{Gives: map[radiation.Range]float64{
				radiation.XRay(): 1, radiation.Gamma(): 0.2}})
	return magnetar, nil
}
