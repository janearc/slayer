package template

import (
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/world"
)

// Saturn is the ringed giant: paler bands than jupiter's, the rings from
// the inner edge of the c ring to the outer edge of the a ring with the
// cassini division between, and titan, rhea and enceladus.
func Saturn() bodies.Body {
	stripes := bands(0.04, 0.03, 0.35, 8,
		swatches.Light(swatches.SaturnCream),
		swatches.Light(swatches.SaturnBelt),
		swatches.Light(swatches.SaturnCream),
		swatches.Light(swatches.SaturnOchre),
		swatches.Light(swatches.SaturnPale),
		swatches.Light(swatches.SaturnBelt),
		swatches.Light(swatches.SaturnCream),
		swatches.Light(swatches.SaturnOchre),
		swatches.Light(swatches.SaturnCream))
	stripes.Seed = 15
	return bodies.Body{Name: "saturn", Kind: bodies.Giant, Radius: 58232, Mass: 5.683e26,
		Day: hours(10.7), Tilt: deg(26.73),
		Gas: &bodies.Gas{Depth: 0.08, Opacity: 40, Bands: stripes},
		// the oval fifteen degrees from the pole, on the spin, in blues
		Aurora: &bodies.Aurora{Low: 800, High: 2000, Lat: deg(75), Width: deg(2),
			Strength: 0.02, Curtains: 0.5, Seed: 15,
			Fringe: swatches.Light(swatches.AuroraViolet),
			Bottom: swatches.Light(swatches.AuroraBlue),
			Top:    swatches.Light(swatches.AuroraMauve)},
		// the rings from the c ring's inner edge to the a ring's outer,
		// as shares of the radius, in twelve samples about 5,450 km
		// apart, blended between.
		//
		// the c ring faint (1.24 to 1.53), the b ring dense (1.53 to
		// 1.95), the cassini division the dip at the ninth sample (1.95
		// to 2.03), the a ring beyond it (2.03 to 2.27).
		//
		// what twelve samples cannot say: the encke gap (325 km at
		// 2.21) and keeler are too narrow; the ringlets are noise where
		// saturn's are made by its moons; the d, f, g and e rings are
		// not here; the colour is one swatch for ice and dust alike.
		//
		// one profile cannot name a ring or give two rings different
		// stuff, as an earlier review found. the shares are of the
		// equatorial 60,268 km but meet the mean, 58,232 km, so the
		// edges sit three per cent in. figures from memory.
		Rings: &bodies.Rings{Inner: 1.24, Outer: 2.27, Thick: 1, Dense: 1.2,
			Profile: []float64{0.2, 0.25, 0.3, 0.5, 0.95, 1, 1,
				0.95,
				0.08, 0.7, 0.65, 0.5},
			Grain:    0.6,
			Ringlets: 60,
			Colour:   swatches.Light(swatches.Ice)},
		Moons: []bodies.Moon{
			{Body: bodies.Body{Name: "titan", Kind: bodies.Rock, Radius: 2574.7, Mass: 1.345e23,
				Ground: &bodies.Ground{Terrain: world.Terrain{Seed: 11, Ocean: 0, Plates: 5},
					Look: look(0, twoTone(swatches.Titan))},
				Air: &bodies.Air{Height: 40, Dense: 0.02,
					Colour: swatches.Light(swatches.TitanAir)}},
				Orbit: 1221870},
			{Body: bodies.Body{Name: "rhea", Kind: bodies.Rock, Radius: 763.8, Mass: 2.31e21}, Orbit: 527108, Phase: 2},
			{Body: bodies.Body{Name: "enceladus", Kind: bodies.Rock, Radius: 252.1, Mass: 1.08e20}, Orbit: 237948, Phase: 4},
		},
	}
}
