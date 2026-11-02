package template

import (
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
)

// Sol is the sun: a star of its radius and mass, glowing white a
// little warm. It has no air to speak of in a picture.
func Sol() bodies.Body {
	return bodies.Body{Name: "sol", Kind: bodies.Star, Radius: 696000, Mass: bodies.Sun,
		Day: hours(609), Glow: swatches.Light(swatches.SunWhite)}
}
