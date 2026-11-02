package template

import (
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
func Magnetar() bodies.Body {
	return bodies.Body{Name: "magnetar", Kind: bodies.Star, Radius: 12, Mass: 1.4 * bodies.Sun,
		Day:  7540 * time.Millisecond,
		Glow: swatches.Light(swatches.MagnetarBlue),
		Radiant: thing.Spectrum{Gives: map[radiation.Range]float64{
			radiation.XRay(): 1, radiation.Gamma(): 0.2}},
		Flare: &bodies.Flare{At: time.Second, Speed: 30000, Opacity: 0.8,
			Glow: swatches.Light(swatches.FlareWhite),
			Radiant: thing.Spectrum{
				Gives:  map[radiation.Range]float64{radiation.Gamma(): 2, radiation.XRay(): 1},
				Passes: map[radiation.Range]float64{radiation.Gamma(): 0.9}}},
	}
}
