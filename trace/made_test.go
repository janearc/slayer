package trace

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// The made worlds the estimator tests share: media, skies and lights
// simple enough that what a ray gathers from them is worked by hand.

// steady is a medium of one density at every point, in its box and out
// of it. A march toward a lamp starts part way into its first step, by a
// noise that can fall a hair outside the step; a medium as thick there
// reads the same, so the density added up along a way through the box is
// the density times the way, however the steps fall.
type steady struct {
	density float64
	box     maths.Box
}

// Density is the one density, wherever it is asked.
func (block steady) Density(maths.Vec) float64 { return block.density }

// Bounds is the box the medium is crossed by.
func (block steady) Bounds() maths.Box { return block.box }

// sheet is a steady medium wide and tall, and deep along z from near to
// far.
func sheet(density, near, far float64) steady {
	return steady{density: density, box: maths.Box{
		Min: maths.V(-10, -10, near), Max: maths.V(10, 10, far)}}
}

// smokeColour is what the smoke of these tests scatters: a different
// share of each channel, so a channel taken for another shows.
var smokeColour = maths.RGB(0.8, 0.4, 0.2)

// smokeBit is alongZ's crossing of a medium worn by smokeColour, staged
// alone. A test lights one bit of it in a scene that holds no volume, so
// that nothing stands between the bit and a lamp.
func smokeBit(made medium.Medium) crossing {
	smoke := thing.Volume{Medium: made,
		Material: &thing.Material{Colour: smokeColour}}
	return crossings(staging(smoke).volumes, alongZ, math.Inf(1))[0]
}

// midway is the stride whose middle is fifty along alongZ, which is the
// world's origin.
var midway = stride{along: 50}

// sunlit is a scene of one sun and nothing else.
func sunlit(toward maths.Vec, strength maths.Light) *staged {
	world := thing.NewWorld()
	world.Lamp(thing.Sun(toward, strength))
	return stage(world, 0)
}

// sunWays are three ways a sun can stand to a look down alongZ: behind
// the eye, ahead of it and overhead.
var sunWays = map[string]maths.Vec{"behind the eye": alongZ.Dir.Neg(),
	"ahead of the eye": alongZ.Dir, "overhead": maths.Up}

// rising is a sky brighter the higher it is looked at: a floor plus a
// slope times the elevation in radians, every channel alike. The sky's
// table keeps a sky that is a straight line in elevation as it is,
// between its rows, anywhere more than one row, 5.625 degrees, from
// straight up or straight down.
type rising struct{ floor, slope float64 }

// Light is the floor plus the slope times how high the way looks.
func (sky rising) Light(way maths.Vec) maths.Light {
	capped := math.Min(1, way.Unit().Y)
	high := math.Asin(math.Max(-1, capped))
	return maths.Grey(sky.floor + sky.slope*high)
}

// wayLit is a sky whose light says which way it was looked at: a half
// plus half the way's x, y and z, in red, green and blue.
type wayLit struct{}

// Light is the way looked along, as a light.
func (wayLit) Light(way maths.Vec) maths.Light {
	return maths.RGB(0.5+0.5*way.X, 0.5+0.5*way.Y, 0.5+0.5*way.Z)
}

// skyBlue is the one radiance of these tests' even sky, a different
// amount in each channel.
var skyBlue = maths.RGB(0.2, 0.4, 0.8)

// skyOver is a scene of a sky and nothing else.
func skyOver(sky thing.Sky) *staged {
	world := thing.NewWorld()
	world.Sky = sky
	return stage(world, 0)
}

// waterIndex is the index of the water of these tests.
const waterIndex = 1.333

// mirrorShare is the share of unpolarised light a clear surface of an
// index mirrors at a cosine of incidence from the air, by Fresnel's
// equations: the mean of the two polarisations' squared amplitude
// ratios, (c - n c') / (c + n c') and (n c - c') / (n c + c'), where c'
// is the cosine of the refracted angle by Snell's law.
func mirrorShare(cosine, index float64) float64 {
	inside := math.Sqrt(1 - (1-cosine*cosine)/(index*index))
	across := (cosine - index*inside) / (cosine + index*inside)
	along := (index*cosine - inside) / (index*cosine + inside)
	return (across*across + along*along) / 2
}
