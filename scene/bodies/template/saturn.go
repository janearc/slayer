package template

import (
	"fmt"

	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/world"
)

// Saturn is the ringed giant: paler bands than jupiter's, the rings from
// the inner edge of the c ring to the outer edge of the a ring with the
// cassini division between, and titan, rhea and enceladus.
func Saturn() (bodies.Body, error) {
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
	gas, err := giantGas(stripes)
	if err != nil {
		return bodies.Body{}, fmt.Errorf("saturn: %w", err)
	}
	// the oval fifteen degrees from the pole, on the spin, in blues
	aurora, err := giantAurora(km(800), km(2000), deg(75), deg(0), 15)
	if err != nil {
		return bodies.Body{}, fmt.Errorf("saturn: %w", err)
	}
	// the rings from the c ring's inner edge to the a ring's outer,
	// as times the radius, in twelve samples about 5,450 km apart,
	// blended between.
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
	rings, err := bodies.NewRings(1.24, 2.27, km(1),
		bodies.PerKilometre(1.2), swatches.Light(swatches.Ice))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("saturn: %w", err)
	}
	rings = rings.Profiled([]float64{0.2, 0.25, 0.3, 0.5, 0.95, 1, 1,
		0.95,
		0.08, 0.7, 0.65, 0.5}).Grained(0.6, 60)
	titan, err := titanMoon()
	if err != nil {
		return bodies.Body{}, err
	}
	rhea, err := smallMoon("rhea", km(763.8), kg(2.31e21), km(527108), rad(2))
	if err != nil {
		return bodies.Body{}, err
	}
	enceladus, err := smallMoon("enceladus", km(252.1), kg(1.08e20), km(237948), rad(4))
	if err != nil {
		return bodies.Body{}, err
	}
	saturn, err := bodies.NewBody("saturn", bodies.Giant, km(58232),
		kg(5.683e26), gas, aurora, rings, titan, rhea, enceladus)
	if err != nil {
		return bodies.Body{}, err
	}
	return saturn.Turning(hours(10.7), deg(26.73)), nil
}

// titanMoon is titan: a two-tone ground under a tall orange haze.
func titanMoon() (bodies.Moon, error) {
	ground, err := bodies.NewGround(
		world.Terrain{Seed: 11, Ocean: 0, Plates: 5},
		look(0, twoTone(swatches.Titan)))
	if err != nil {
		return bodies.Moon{}, fmt.Errorf("titan: %w", err)
	}
	air, err := bodies.NewAir(km(40), bodies.PerKilometre(0.02),
		swatches.Light(swatches.TitanAir))
	if err != nil {
		return bodies.Moon{}, fmt.Errorf("titan: %w", err)
	}
	body, err := bodies.NewBody("titan", bodies.Rock, km(2574.7),
		kg(1.345e23), ground, air)
	if err != nil {
		return bodies.Moon{}, err
	}
	return bodies.NewMoon(body, km(1221870), rad(0))
}
