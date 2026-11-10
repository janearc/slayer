package main

import (
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

// localLamps retains descriptions and conversion inputs, not the specimen's
// procedural parts, construction caches or the whole asked scene.
type localLamps struct {
	lights   []manufacture.AttachedLight
	exposure float64
	moment   time.Duration
}

// at converts placed descriptions with the retained exposure and moment.
func (lamps localLamps) at(place maths.Subspace) ([]emit.Lit, error) {
	var lit []emit.Lit
	given := asked{exposure: lamps.exposure, at: lamps.moment}
	lights, err := manufacture.PlaceLights(lamps.lights, place)
	if err != nil {
		return nil, err
	}
	for _, placed := range lights {
		source := placed.Light()
		shone, err := lampOf(source, placed.At(), placed.Toward(),
			placed.Beyond().InMetres(), given)
		if err != nil {
			return nil, err
		}
		lit = append(lit, shone)
	}
	return lit, nil
}
