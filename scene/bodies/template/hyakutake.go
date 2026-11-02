package template

import (
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
)

// Hyakutake is the great comet of 1996: a small nucleus, a bright
// coma, a short dust tail and the longest ion tail ever seen, five
// hundred million kilometres, straight and blue.
//
// The shot aims it.
func Hyakutake() bodies.Body {
	return bodies.Body{Name: "hyakutake", Kind: bodies.Rock, Radius: 2, Mass: 1e13,
		Seed: 5,
		Coma: &bodies.Coma{Radius: 5e5, Dense: 1e-5,
			Colour: swatches.Light(swatches.ComaDustCool),
			Glow:   swatches.Light(swatches.ComaGreenBright)},
		Tails: []bodies.Tail{
			{Length: 8e6, Base: 4e5, Spread: 0.06, Curve: 0.3, Dense: 1.5e-7,
				Colour: swatches.Light(swatches.DustTailPale),
				Wisp:   0.5, Seed: 96},
			{Length: 5e8, Base: 6e5, Spread: 0.01, Curve: 0.01, Dense: 3e-7,
				Colour: swatches.Light(swatches.IonTail),
				Glow:   swatches.Light(swatches.IonGlow),
				Wisp:   0.7,
				Seed:   95},
		},
	}
}
