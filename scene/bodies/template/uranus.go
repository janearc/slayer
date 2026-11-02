package template

import (
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
)

// Uranus is the ice giant on its side: a pale cyan, almost featureless,
// with thin dark rings and three of its five large moons: titania, oberon
// and miranda; ariel and umbriel are not here.
func Uranus() bodies.Body {
	stripes := bands(0.02, 0.01, 0.15, 6,
		swatches.Light(swatches.UranusCyan),
		swatches.Light(swatches.UranusDeep),
		swatches.Light(swatches.UranusPale),
		swatches.Light(swatches.UranusDeep),
		swatches.Light(swatches.UranusCyan))
	stripes.Seed = 16
	// uranus turns backwards, on its side: the tilt past ninety says so,
	// as pluto's does, and the day is positive
	return bodies.Body{Name: "uranus", Kind: bodies.Giant, Radius: 25362, Mass: 8.681e25,
		Day: hours(17.24), Tilt: deg(97.77),
		Gas: &bodies.Gas{Depth: 0.08, Opacity: 40, Bands: stripes},
		Rings: &bodies.Rings{Inner: 1.65, Outer: 2.02, Thick: 0.5, Dense: 0.5,
			Profile: []float64{0.2, 0.05, 0.4, 0.05, 0.3, 0.05, 1, 0.05, 0.5},
			Grain:   0.4, Ringlets: 40,
			Colour: swatches.Light(swatches.UranusRing)},
		Moons: []bodies.Moon{
			{Body: bodies.Body{Name: "titania", Kind: bodies.Rock, Radius: 788.4, Mass: 3.4e21}, Orbit: 435910},
			{Body: bodies.Body{Name: "oberon", Kind: bodies.Rock, Radius: 761.4, Mass: 3.08e21}, Orbit: 583520, Phase: 2},
			{Body: bodies.Body{Name: "miranda", Kind: bodies.Rock, Radius: 235.8, Mass: 6.6e19}, Orbit: 129390, Phase: 1},
		},
	}
}
