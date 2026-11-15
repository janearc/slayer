package named

import (
	"math"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/lines"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/observer"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// the cities' lights' temperatures in kelvin, from the edges of a city in
// to its heart, and on the gap walls and hulls; and the one luminance
// they are all scaled to
const (
	sodium, warmWhite, warmer, blueWhite = 1900, 2600, 3400, 9000
	wallSodium, wallAmber                = 1900, 2300
	wallWarm, wallWhite                  = 2700, 3200
	cityLuminance                        = 0.9
)

// urbanLights are a city's from its edges in to its heart: sodium amber
// at the ends of the arms, warm white, and a dense blue-white in the
// middle, as black bodies through libtheme at one brightness
var urbanLights = warmLights(sodium, warmWhite, warmer, blueWhite)

// cityLights are the cities' and the hull's lights: sodium amber to
// warm white, black bodies from 1900 to 3200 kelvin through libtheme, at
// one brightness, so they read as cities and not as glitter
var cityLights = warmLights(wallSodium, wallAmber, wallWarm, wallWhite)

// warmLights is a black body at each temperature as three channels, all
// scaled to one luminance
func warmLights(kelvins ...float64) []maths.Light {
	lit := make([]maths.Light, 0, len(kelvins))
	for _, kelvin := range kelvins {
		spectrum := bands.Split(swatch.Blackbody(kelvin))
		r, g, b := srgb.Light(observer.Swatch(spectrum))
		colour := maths.RGB(r, g, b)
		lit = append(lit, colour.Scale(cityLuminance/colour.Luminance()))
	}
	return lit
}

// the nebula the pair is parked in: its radius, and the seed, scale and
// octaves of the noise that makes it ragged, picked for the picture; and
// the colour it scatters
const (
	nebulaReach                        = 5200.0
	nebulaSeed                         = 1806
	nebulaScale                        = 260.0
	nebulaOctaves                      = 5
	nebulaRed, nebulaGreen, nebulaBlue = 0.22, 0.14, 0.17
)

// nebula is the gas's settings: how thick, and what share of it is neon
// and argon, the rest hydrogen
type nebula struct {
	thick, neon, argon float64
	// glow is how bright its glow is, whatever it is made of, as a
	// luminance
	glow float64
}

// share is one element and how much of the gas it is
type share struct {
	element string
	part    float64
}

// material is the gas's look: its scatter, and its glow mixed from its
// elements' lines, in a fixed order so the sum is the same every time
func (gas nebula) material() (*thing.Material, error) {
	shares := []share{{"hydrogen", max(0, 1-gas.neon-gas.argon)},
		{"neon", gas.neon}, {"argon", gas.argon}}
	var glow maths.Light
	for _, one := range shares {
		element, err := lines.Of(one.element)
		if err != nil {
			return nil, err
		}
		glow = glow.Add(element.Glow(gas.glow).Scale(one.part))
	}
	return &thing.Material{Name: "nebula",
		Colour: maths.RGB(nebulaRed, nebulaGreen, nebulaBlue),
		Glow:   glow}, nil
}

// addNebula puts the nebula round the pair into the world: glowing
// hydrogen red and neon orange, scattering the sun, with the channel the
// pair has swept through it, twice a band's width about each band's path.
func addNebula(world *thing.World, pair []hoop, gas nebula) error {
	gasMaterial, err := gas.material()
	if err != nil {
		return err
	}
	paths, err := pathsOf(pair)
	if err != nil {
		return err
	}
	grain, err := medium.NewGrain(nebulaSeed, nebulaScale, nebulaOctaves)
	if err != nil {
		return err
	}
	nebulaWake, err := medium.NewWake(nebulaReach, 2*bandWidth,
		gas.thick, grain, paths)
	if err != nil {
		return err
	}
	world.AddVolume(thing.Volume{Medium: nebulaWake,
		Material: gasMaterial})
	return nil
}

// the gap walls' marble: the size of a crowd of cities and of one city,
// in thousands of km, and the noise's octaves; where a crowd starts and
// is full; how many cells of a crowd are lit, how many of those in neon,
// and how bright
const (
	crowdSize, citySize    = 30, 0.1
	crowdOctaves           = 4
	crowdStarts, crowdFull = 0.05, 0.45
	litShare, neonShare    = 0.6, 0.12
	neonDim                = 0.8
	cityRed, cityGreen     = 1.0, 0.68
	cityBlue               = 0.32
	cityLeast, cityByCrowd = 0.6, 0.6
)

// marble is a black marble for a wall: cities gathered where a slow noise
// says people are, each a speck lit warm, a few in neon, and dark between.
type marble struct{ seed uint64 }

// At is the light of the marble at point: a speck where a city stands, else
// dark.
func (stone marble) At(point maths.Vec) maths.Light {
	// where the people are: clusters about thirty thousand km across
	crowd := maths.Smooth(crowdStarts, crowdFull,
		noise.Fractal(stone.seed, point.X/crowdSize, point.Y/crowdSize,
			point.Z/crowdSize, crowdOctaves))
	if crowd <= 0 {
		return maths.Light{}
	}
	// a city to every cell of about a hundred km, lit if the crowd is
	// enough
	i := int64(math.Floor(point.X / citySize))
	j := int64(math.Floor(point.Y / citySize))
	k := int64(math.Floor(point.Z / citySize))
	if noise.Hash3(stone.seed, i, j, k) > litShare*crowd {
		return maths.Light{}
	}
	if noise.Hash3(stone.seed+1, i, j, k) < neonShare {
		pick := noise.Hash3(stone.seed+2, i, j, k) *
			float64(len(cityLights))
		return cityLights[int(pick)%len(cityLights)].Scale(neonDim)
	}
	return maths.RGB(cityRed, cityGreen, cityBlue).Scale(cityLeast +
		cityByCrowd*crowd)
}
