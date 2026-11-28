package generate

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/scene/bodies"
)

// The relations below are cited from memory: none of the papers is on
// the slayer shelf yet, and the design asks that each be shelved before
// code relies on it. Each comment names the source to shelve.

// makeupSlack is how far a makeup's shares may add from one: room for
// the last digit of a decimal said in a set, and no more.
const makeupSlack = 1e-6

// keplerSlack is the share by which a year given beside a distance may
// differ from the year Kepler's third law gives for that distance.
const keplerSlack = 0.01

// speedOfLight is in metres per second, exact by the SI's definition. It
// turns measure's natural lengths back into metres per second.
const speedOfLight = 299792458

// julianYear is a Julian year in seconds: 365.25 days.
const julianYear = 365.25 * 86400

// metresPerKilometre turns metres into the kilometres bodies speak.
const metresPerKilometre = 1e3

// earthRadius is the Earth's mean radius in metres (NASA's Earth fact
// sheet, Williams, NSSDCA): the unit Chen & Kipping's radii are said in.
const earthRadius = 6.371e6

// distance is a planet's distance from its sun in metres: the one given,
// or the one Kepler's third law gives for its year. Given both, it
// refuses them unless the law makes them agree to within keplerSlack.
func distance(planet Planet) (float64, error) {
	given := planet.Orbit.Distance.InMetres()
	if planet.Orbit.Flux != 0 {
		var err error
		if given, err = lit(planet); err != nil {
			return 0, err
		}
	}
	if planet.Orbit.Year == 0 {
		return given, nil
	}
	// Kepler's third law with Newton's constant: the period squared is
	// four pi squared times the distance cubed, over G times both masses
	heavy := measure.Kilograms(planet.Sun.Mass.InKilograms() +
		planet.Mass.InKilograms())
	pull := measure.GM(heavy).InMetres() * speedOfLight * speedOfLight
	year := float64(planet.Orbit.Year) * julianYear
	if given == 0 {
		return math.Cbrt(pull * math.Pow(year/(2*math.Pi), 2)), nil
	}
	kepler := 2 * math.Pi * math.Sqrt(given*given*given/pull)
	if math.Abs(kepler-year) > keplerSlack*year {
		return 0, fmt.Errorf("generate: %s's distance makes a year of %.3f "+
			"years, not the %.3f given", planet.Name, kepler/julianYear,
			float64(planet.Orbit.Year))
	}
	return given, nil
}

// Distance is where the planet stands from its sun: the one it is given,
// or the one its Flux, or its Year, gives. It is the same distance Derive
// works the planet's warmth from.
func (planet Planet) Distance() (measure.Length, error) {
	metres, err := distance(planet)
	return measure.Metres(metres), err
}

// lit is the distance in metres at which a sun gives a planet the light of
// its Flux, in the Earth's. The light falls as the square of the distance
// (the inverse square law), and the Earth's is the Sun's at one
// astronomical unit, so a sun of a Luminosity in the Sun's gives that
// light at the square root of the luminosity over the flux, in
// astronomical units. It refuses a planet given a distance beside its
// flux, a flux that is not above nought, and a sun that has no light.
func lit(planet Planet) (float64, error) {
	switch {
	case planet.Orbit.Distance.InMetres() != 0:
		return 0, fmt.Errorf("generate: %s is given a distance and the "+
			"light it gets, which would be said twice", planet.Name)
	case !(planet.Orbit.Flux > 0):
		return 0, fmt.Errorf("generate: %s is given %v times the Earth's "+
			"sunlight, which is no light to be put at", planet.Name,
			planet.Orbit.Flux)
	case !(planet.Sun.Luminosity > 0):
		return 0, fmt.Errorf("generate: %s is given the light it gets, "+
			"and its sun gives none", planet.Name)
	}
	return AstronomicalUnit * math.Sqrt(planet.Sun.Luminosity/
		planet.Orbit.Flux), nil
}

// giantGasShare is the share of hydrogen and helium past which a planet
// is a giant. Lopez & Fortney 2014 (ApJ 792, 1) find an envelope of one
// per cent of the mass roughly doubles a rocky core's radius, which takes
// it off the terran branch onto the Neptunian.
const giantGasShare = 0.01

// kindOf is the body kind a makeup is drawn as: a giant when it holds
// enough gas to be on the Neptunian or Jovian branch, else a rock.
func kindOf(makeup Makeup) bodies.Kind {
	if makeup.Gas >= giantGasShare {
		return bodies.Giant
	}
	return bodies.Rock
}

// The mean relations of Chen & Kipping 2017 (ApJ 834, 17, table 1): a
// radius in the Earth's is a power of the mass in the Earth's, the power
// changing at each branch's top. terranLogRadius is the log of the radius
// at one Earth mass.
const terranLogRadius = 0.00346

// branch is one of Chen & Kipping's branches: the mass in Earth masses
// where it ends, and the power of the mass the radius goes as below it.
type branch struct {
	top, slope float64
}

// branches are the terran, Neptunian and Jovian branches. The terran
// ends at 2.04 Earth masses, the Neptunian at 0.414 Jupiter masses (131.6
// Earth masses), and the Jovian at 0.08 Suns; above that is the stellar
// branch, which a planet is not on.
var branches = []branch{
	{top: 2.04, slope: 0.2790},
	{top: 131.6, slope: 0.589},
	{top: math.Inf(1), slope: -0.044},
}

// forecast is Chen & Kipping's mean radius in Earth radii for a mass in
// Earth masses, branch by branch, each joined to the one below.
func forecast(earthMasses float64) float64 {
	logMass := math.Log10(earthMasses)
	floor := math.Log10(branches[0].top)
	logRadius := terranLogRadius + branches[0].slope*min(logMass, floor)
	for _, each := range branches[1:] {
		if logMass <= floor {
			break
		}
		top := math.Log10(each.top)
		logRadius += each.slope * (min(logMass, top) - floor)
		floor = top
	}
	return math.Pow(10, logRadius)
}

// Zeng, Sasselov & Jacobsen 2016 (ApJ 819, 127) give a rocky planet's
// radius as (1.07 - 0.21 times its core's share of the mass) times a
// power of the mass. Their factor, taken against the Earth's, moves a
// rocky world within the terran branch by at most about a tenth.
const (
	zengBase  = 1.07
	zengSlope = 0.21
	// earthMetal is the Earth's core share, near Zeng's 0.33
	earthMetal = 0.32
)

// giantRadius is a giant's radius in metres at a distance in metres from
// its sun, from Fortney, Marley & Barnes 2007's tables: by its mass, its
// age and the sunlight it gets, with its core the share of its mass that
// is not gas. Their models are for the Sun, so a brighter sun's distance
// is taken as the Sun's distance with the same light, the distance over
// the root of the luminosity; a planet with no sun has the faintest light
// they give. Their radius is the least a planet gets: the hottest giants
// are larger, and inflation adds what Sestovic, Demory & Queloz 2018 fit.
func giantRadius(planet Planet, metres float64) float64 {
	earthMasses := planet.Mass.InKilograms() / EarthMass
	core := (1 - planet.Makeup.Gas) * earthMasses
	light := math.Inf(1)
	if planet.Sun.Luminosity > 0 {
		light = metres / AstronomicalUnit / math.Sqrt(planet.Sun.Luminosity)
	}
	gigayears := float64(planet.Age) / gigayear
	// the flux is the sun's light over a sphere of the distance's radius
	flux := planet.Sun.Luminosity * solarLuminosity /
		(4 * math.Pi * metres * metres)
	swell := inflation(planet.Mass.InKilograms()/jupiterMass, flux)
	return (coldGiantRadius(earthMasses, core, light, gigayears) + swell) *
		jupiterEquatorial
}

// gigayear is a billion years, the unit of the tables' ages.
const gigayear = 1e9

// radiusOf is a planet's radius in metres at a distance in metres from
// its sun: a rock on the terran branch, however heavy, moved by its share
// of metal; a giant from Fortney, Marley & Barnes's tables.
func radiusOf(planet Planet, metres float64, kind bodies.Kind) float64 {
	if kind == bodies.Giant {
		return giantRadius(planet, metres)
	}
	earthMasses := planet.Mass.InKilograms() / EarthMass
	terran := math.Pow(10, terranLogRadius+
		branches[0].slope*math.Log10(earthMasses))
	metal := (zengBase - zengSlope*planet.Makeup.Metal) /
		(zengBase - zengSlope*earthMetal)
	return terran * metal * earthRadius
}

// gravity is the pull at the surface, G M over R squared, in metres per
// second squared, for a radius in metres.
func gravity(mass measure.Mass, radius float64) float64 {
	pull := measure.GM(mass).InMetres() * speedOfLight * speedOfLight
	return pull / (radius * radius)
}

// The Bond albedos, the share of sunlight a planet sends back, by kind:
// the Earth's and Jupiter's (NASA's fact sheets, Williams, NSSDCA).
const (
	rockAlbedo  = 0.306
	giantAlbedo = 0.343
)

// albedo is the share of light a kind of planet sends back.
func albedo(kind bodies.Kind) float64 {
	if kind == bodies.Giant {
		return giantAlbedo
	}
	return rockAlbedo
}

// darkest is the temperature of a planet no star lights: the cosmic
// microwave background, 2.7255 kelvin (Fixsen 2009, ApJ 707, 916). A
// rogue's own heat from inside is a later step; until then this is its
// floor, and it is the floor of a planet very far out too.
const darkest = 2.7255

// equilibrium is the temperature, in kelvin, at which a planet gives off
// as much heat as it takes from a sun of a luminosity, in the Sun's, at a
// distance in metres: the sunlight it keeps, L (1 - A) over 4 pi d
// squared, on its disc, given off from its whole sphere, four times the
// disc. This is the radiative balance of Seager 2010 (Exoplanet
// Atmospheres, Princeton University Press). No sun gives darkest.
func equilibrium(luminosity, metres, sendsBack float64) float64 {
	if luminosity == 0 {
		return darkest
	}
	kept := measure.SolarLuminosities(luminosity).Times((1 - sendsBack) / 4)
	heat := measure.SurfaceTemperature(kept, measure.Sphere(
		measure.Metres(metres))).InKelvin()
	return max(heat, darkest)
}

// The main gas of each kind's air, in daltons: a giant's is hydrogen; a
// rock's is taken to be nitrogen, as the Earth's is. A rock with an air
// of carbon dioxide, at 44, would keep it more easily.
const (
	hydrogen = 2.016
	nitrogen = 28.014
)

// gasOf is the mass of one molecule of a kind's main gas.
func gasOf(kind bodies.Kind) measure.Mass {
	if kind == bodies.Giant {
		return measure.Daltons(hydrogen)
	}
	return measure.Daltons(nitrogen)
}

// jeansRatio is how many times its gas's thermal speed a planet's
// escape speed must be for it to keep that gas for billions of years:
// the rule of thumb from Jeans escape (Jeans 1925, The Dynamical Theory
// of Gases, Cambridge University Press).
const jeansRatio = 6

// keepsAir is whether a planet of a mass and a radius in metres keeps
// its main gas at a temperature in kelvin: its escape speed, the root of
// 2 G M over R, is at least jeansRatio times the gas's most likely
// speed, the root of 2 k T over m. Both are said as shares of the speed
// of light, as measure holds them.
func keepsAir(mass measure.Mass, radius, kelvin float64,
	kind bodies.Kind) bool {
	escape := math.Sqrt(2 * measure.GM(mass).InMetres() / radius)
	thermal := math.Sqrt(2 * measure.Kelvin(kelvin).InMeV() /
		gasOf(kind).InMeV())
	return escape >= jeansRatio*thermal
}

// The edges of the habitable zone for a star like the Sun, in sunlight
// at the planet as a share of the Earth's: inside the moist greenhouse
// edge an ocean is lost to space as steam, and outside the maximum
// greenhouse edge no air keeps it from freezing (Kopparapu et al. 2013,
// ApJ 765, 131, table 3). Both are for the Sun's temperature.
const (
	moistGreenhouse   = 1.0146
	maximumGreenhouse = 0.3507
)

// freezing is water's freezing point in kelvin, where an airless world's
// water is ice below it and lost above it.
const freezing = 273.15

// leastWater is the share of a planet's mass in ice below which it is
// dry: a twentieth of the Earth's oceans, which are two in ten thousand.
const leastWater = 1e-5

// waterOf is the state of a planet's water from its share of ice, its
// temperature in kelvin and whether it keeps an air. With air, the
// habitable zone's edges, said as the temperatures a rock has in them,
// part ice, ocean and steam; with none, water is ice when frozen and
// gone when not.
func waterOf(ice, kelvin float64, air bool) Water {
	inner := equilibrium(moistGreenhouse, AstronomicalUnit, rockAlbedo)
	outer := equilibrium(maximumGreenhouse, AstronomicalUnit, rockAlbedo)
	switch {
	case ice < leastWater:
		return Dry
	case !air && kelvin < freezing, air && kelvin < outer:
		return Ice
	case !air:
		return Dry
	case kelvin <= inner:
		return Ocean
	}
	return Steam
}
