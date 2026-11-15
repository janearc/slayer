package template

import (
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
)

// Sol is the sun: a star of its radius and mass, glowing white a
// little warm. It has no air to speak of in a picture.
func Sol() (bodies.Body, error) {
	sol, err := bodies.NewBody("sol", bodies.Star, km(696000),
		kg(bodies.Sun))
	if err != nil {
		return bodies.Body{}, err
	}
	sol = sol.Turning(hours(609), deg(0)).
		Glowing(swatches.Light(swatches.SunWhite), thing.Radiant(nil))
	return sol, nil
}
