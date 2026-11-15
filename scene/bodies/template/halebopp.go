package template

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
)

// HaleBopp is the great comet of 1997: a nucleus sixty kilometres
// across, a coma of a million kilometres' radius, two million across,
// one of the largest seen, glowing the green of carbon.
//
// Its dust tail is broad, white, forty million kilometres long and
// curved back along the orbit; its ion tail straight, blue and longer
// still.
//
// The shot aims it.
func HaleBopp() (bodies.Body, error) {
	return comet("hale-bopp", km(30), kg(1e16),
		bodies.PerKilometre(6e-6), km(1e6),
		[2]maths.Light{swatches.Light(swatches.ComaDust),
			swatches.Light(swatches.ComaGreen)},
		tailOf{length: km(4e7), base: km(6e5), spread: 0.08,
			curve: 0.35, loss: bodies.PerKilometre(2e-7),
			colour: swatches.Light(swatches.DustTail),
			wisp:   maths.Percent(50), seed: 97},
		tailOf{length: km(6e7), base: km(3e5), spread: 0.03,
			curve: 0.02, loss: bodies.PerKilometre(2e-7),
			colour: swatches.Light(swatches.IonTail),
			glow:   swatches.Light(swatches.IonGlow),
			wisp:   maths.Percent(70), seed: 98},
		3)
}
