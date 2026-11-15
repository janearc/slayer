package template

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
)

// Hyakutake is the great comet of 1996: a small nucleus, a bright
// coma, a short dust tail and the longest ion tail ever seen, five
// hundred million kilometres, straight and blue.
//
// The shot aims it.
func Hyakutake() (bodies.Body, error) {
	return comet("hyakutake", km(2), kg(1e13),
		bodies.PerKilometre(1e-5), km(5e5),
		[2]maths.Light{swatches.Light(swatches.ComaDustCool),
			swatches.Light(swatches.ComaGreenBright)},
		tailOf{length: km(8e6), base: km(4e5), spread: 0.06,
			curve: 0.3, loss: bodies.PerKilometre(1.5e-7),
			colour: swatches.Light(swatches.DustTailPale),
			wisp:   maths.Percent(50), seed: 96},
		tailOf{length: km(5e8), base: km(6e5), spread: 0.01,
			curve: 0.01, loss: bodies.PerKilometre(3e-7),
			colour: swatches.Light(swatches.IonTail),
			glow:   swatches.Light(swatches.IonGlow),
			wisp:   maths.Percent(70), seed: 95},
		5)
}
