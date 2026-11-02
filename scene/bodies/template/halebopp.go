package template

import (
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
func HaleBopp() bodies.Body {
	return bodies.Body{Name: "hale-bopp", Kind: bodies.Rock, Radius: 30, Mass: 1e16,
		Seed: 3,
		Coma: &bodies.Coma{Radius: 1e6, Dense: 6e-6,
			Colour: swatches.Light(swatches.ComaDust),
			Glow:   swatches.Light(swatches.ComaGreen)},
		Tails: []bodies.Tail{
			{Length: 4e7, Base: 6e5, Spread: 0.08, Curve: 0.35, Dense: 2e-7,
				Colour: swatches.Light(swatches.DustTail),
				Wisp:   0.5, Seed: 97},
			{Length: 6e7, Base: 3e5, Spread: 0.03, Curve: 0.02, Dense: 2e-7,
				Colour: swatches.Light(swatches.IonTail),
				Glow:   swatches.Light(swatches.IonGlow),
				Wisp:   0.7,
				Seed:   98},
		},
	}
}
