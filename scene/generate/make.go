package generate

import (
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/world"
)

// The parts below take their look from the Earth and Jupiter templates,
// in scene/bodies/template: what those set by eye, these set the same,
// and what follows from a planet's numbers is worked from them.

// partsOf is the parts a planet's numbers call for: a giant's gas, or a
// rock's ground with the air and clouds it keeps. For now there are no
// rings, and no moons until the system generator makes them.
func partsOf(planet Planet, derived Derived) ([]bodies.Part, error) {
	if derived.Kind == bodies.Giant {
		bands := stripes(planet.Seed)
		if planet.GiantLook != nil {
			bands = *planet.GiantLook
			bands.Seed = planet.Seed
		}
		depth, opacity := giantDepth/100.0, float64(giantOpacity)
		if planet.GiantGas != nil {
			depth, opacity = planet.GiantGas.Depth, planet.GiantGas.Opacity
		}
		gas, err := bodies.NewGas(maths.Share(depth), opacity, bands)
		return []bodies.Part{gas}, err
	}
	water := derived.Water
	if planet.GroundLook != nil && planet.GroundLook.Ocean > 0 {
		water = Ocean
	}
	ground, err := bodies.NewGround(terrain(planet, derived), worn(water))
	if err != nil {
		return nil, err
	}
	vary := fineVary
	if planet.GroundLook != nil {
		vary = planet.GroundLook.Variation
	}
	fine, err := bodies.NewFineness(fineRungs, maths.Share(vary),
		maths.Share(fineMix))
	if err != nil {
		return nil, err
	}
	if planet.GroundLook != nil {
		fine, err = fine.Ridges(maths.Share(planet.GroundLook.RidgeContrast))
		if err != nil {
			return nil, err
		}
	}
	parts := []bodies.Part{ground.Fined(fine)}
	if !derived.Air && planet.Clouds == nil && planet.AirLook == nil {
		return parts, nil
	}
	return withAir(parts, planet, derived)
}

// The giant's gas, as the templates' giantGas has it: opaque eight per
// cent of the radius down, forty optical depths from the tops to there.
const (
	giantDepth   = 8
	giantOpacity = 40
)

// The Earth template's air: how much light a kilometre of it takes, and
// its scale height in kilometres, which a generated air is scaled from.
const (
	earthAirLoss   = 0.012
	earthAirHeight = 8.5
)

// The clouds: the bottom and top of their shell in kilometres, as the
// Earth template has them; how much of the sky they cover in per cent,
// and how much light a kilometre of them takes, set by eye (2026-11-26).
// The template's 55 and 1 drew a world from orbit nearly all solid white.
const (
	cloudBottom = 2
	cloudTop    = 9
	cloudCover  = 24
	cloudLoss   = 0.5
)

// lockedCloudRise is how many times higher the top of a tidally locked
// world's clouds is than the Earth template's: the deep convection at its
// substellar point reaches the top of the troposphere, taken as twice the
// Earth's mean cloud top. Picked by eye 2026-11-27, so that the towers' relief
// is a few kilometres; the factor itself is from memory, to shelve.
const lockedCloudRise = 2.5

// withAir adds an air to a rock's parts, its scale height worked from its
// gas, temperature and gravity, and clouds where it has oceans, their
// shell scaled with the air.
func withAir(parts []bodies.Part, planet Planet,
	derived Derived) ([]bodies.Part, error) {
	height := scaleHeight(derived)
	if derived.Air || planet.AirLook != nil {
		loss, colour := earthAirLoss, swatches.Light(swatches.EarthAir)
		if planet.AirLook != nil {
			height, loss, colour = planet.AirLook.Height, planet.AirLook.Loss,
				planet.AirLook.Colour
		}
		air, err := bodies.NewAir(bodies.Kilometres(height),
			bodies.PerKilometre(loss), colour)
		if err != nil {
			return nil, err
		}
		parts = append(parts, air)
	}
	if planet.Clouds != nil {
		look := planet.Clouds
		clouds, err := bodies.NewClouds(bodies.NewShell(
			bodies.Kilometres(look.Base), bodies.Kilometres(look.Top)),
			maths.Share(look.Cover), bodies.PerKilometre(look.Loss), planet.Seed)
		if err != nil {
			return nil, err
		}
		if look.Cover == 0 {
			return parts, nil
		}
		if look.Weather == "bands" {
			clouds = clouds.TurningWeather()
		} else if look.Weather == "soft-bands" {
			clouds = clouds.SoftWeather()
		}
		if look.StormTurn != nil {
			clouds = clouds.StormTurn(*look.StormTurn)
		}
		return append(parts, clouds), nil
	}
	if derived.Water != Ocean || !derived.Air {
		return parts, nil
	}
	stretch := height / earthAirHeight
	top := cloudTop * stretch
	if planet.Locked {
		top *= lockedCloudRise
	}
	clouds, err := bodies.NewClouds(bodies.NewShell(
		bodies.Kilometres(cloudBottom*stretch),
		bodies.Kilometres(top)), maths.Percent(cloudCover),
		bodies.PerKilometre(cloudLoss), planet.Seed)
	if err != nil {
		return nil, err
	}
	return append(parts, clouds), nil
}

// scaleHeight is the height in kilometres over which a rock's air thins
// by a factor of e: k T over m g, its gas's heat over its weight.
func scaleHeight(derived Derived) float64 {
	heat := measure.Kelvin(derived.Temperature).InMeV() /
		gasOf(derived.Kind).InMeV()
	return heat * speedOfLight * speedOfLight / derived.Gravity /
		metresPerKilometre
}

// The Earth's ground as the Earth template has it: seven tenths sea and
// twelve plates.
const (
	earthOcean  = 0.71
	earthPlates = 12
)

// earthIce is the Earth's share of its mass in water, two in ten
// thousand: the share that pours an earthOcean of sea.
const earthIce = 2e-4

// terrain is a rock's ground: its seed, a sea poured from its ice where
// the water is liquid or frozen, as many plates for its size as the Earth
// has for its, and cold from how far it is below the Earth's warmth.
func terrain(planet Planet, derived Derived) world.Terrain {
	sea := 0.0
	if derived.Water == Ocean || derived.Water == Ice {
		// each Earth's worth of water covers the share of what is still
		// dry that the Earth's covers
		sea = 1 - math.Pow(1-earthOcean, planet.Makeup.Ice/earthIce)
	}
	if planet.GroundLook != nil {
		sea = planet.GroundLook.Ocean
	}
	size := derived.Radius.InKilometres() * metresPerKilometre / earthRadius
	return world.Terrain{Seed: planet.Seed, Ocean: sea,
		Plates: int(math.Round(earthPlates * size * size)),
		Cold:   cold(derived.Temperature)}
}

// cold is a ground's cold for a temperature in kelvin: nought at the
// Earth's, rising to one at the outer edge of the habitable zone, where
// the sea freezes over, and on past it.
func cold(kelvin float64) float64 {
	earth := equilibrium(1, AstronomicalUnit, rockAlbedo)
	frozen := equilibrium(maximumGreenhouse, AstronomicalUnit, rockAlbedo)
	return max(0, (earth-kelvin)/(earth-frozen))
}

// groundLines is how dark the lines between tiles are: none. The Earth
// template draws them at 0.4, which shows a planet's hex tiling as a
// grid; a planet like the Earth shows no tiling.
const groundLines = 0

// The ground is painted finer than its tiles, with smoke's Io's numbers:
// the picture's own level of detail (no rungs fixed), the light varied by
// a quarter, and a neighbouring class let in about a third of the time.
// The gpu hands each painted place back with its pixel's width, so the
// ground is worked as finely as that pixel can see.
const (
	fineRungs = 0
	fineVary  = 0.25
	fineMix   = 0.35
)

// worn is a rock's look by its water: the Earth's paints where it has a
// sea, liquid or frozen, and Mercury's bare rock where it has none.
func worn(water Water) world.Look {
	paints := swatches.Mercury
	if water == Ocean || water == Ice {
		paints = swatches.Earth
	}
	light := swatches.Light
	return world.Look{Lines: groundLines, Paints: [world.Biomes]maths.Light{
		world.Deep: light(paints.Deep), world.Sea: light(paints.Sea),
		world.Ice:   light(paints.Ice),
		world.Beach: light(paints.Beach), world.Desert: light(paints.Desert),
		world.Grass: light(paints.Grass), world.Forest: light(paints.Forest),
		world.Jungle: light(paints.Jungle), world.Tundra: light(paints.Tundra),
		world.Rock: light(paints.Rock), world.Snow: light(paints.Snow),
		world.Lava: light(paints.Lava), world.Ash: light(paints.Ash),
	}}
}

// The Jupiter template's banding: eleven bands, ten jets, and the roil,
// fine detail, mottle, shear and grain of the giants' shared bands.
const (
	bandCount    = 11
	fewestJets   = 8
	jetsByChance = 5
	bandRoil     = 0.06
	bandFine     = 0.05
	bandMottle   = 0.5
	bandShear    = 7
	bandGrain    = 0.08
)

// stripes is a giant's bands from its seed: cream between darker bands,
// each rust or ochre as the stirred seed's bits say, and eight to twelve
// jets.
func stripes(seed uint64) texture.Banded {
	stirred := noise.Mix(seed)
	cream := swatches.Light(swatches.Cream)
	darks := [2]maths.Light{swatches.Light(swatches.Rust),
		swatches.Light(swatches.Ochre)}
	paints := make([]maths.Light, bandCount)
	for index := range paints {
		paints[index] = cream
		if index%2 == 1 {
			paints[index] = darks[stirred>>index&1]
		}
	}
	return texture.Banded{Paints: paints, Roil: bandRoil, Shear: bandShear,
		Grain: bandGrain, Fine: bandFine, Mottle: bandMottle, Seed: seed,
		Jets: fewestJets + int(stirred%jetsByChance)}
}
