package session

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/generate"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

// A star is a body that glows and lights every other body from where it
// stands (doc/system-generators-design.md): a ball of its radius glowing
// with its temperature's colour, limb-darkened, and a lamp at its centre
// whose light falls with the square of the distance. A set's pairs of
// stars are drawn this way, so a body near two stars is lit by both, in
// two colours, with a terminator and a shadow for each.
//
// The film's exposure is what a watt a steradian is in the picture, as it
// is for a sun's lamp, so the ball and the lamp are both scaled by it.
// The sun's direction, which a set says with sun:, is unchanged.

// wholeSphere is four pi, the solid angle round a point in steradians,
// which turns a star's watts into watts a steradian.
const wholeSphere = 4 * math.Pi

// limbMeanShare is what the linear limb law's disc averages to, as a
// share of the centre's light, for a coefficient u: one minus u over
// three.
func limbMeanShare(coefficient float64) float64 { return 1 - coefficient/3 }

// starOf is a star as a thing and a lamp: a ball of its radius at a
// place in the world's unit, glowing at the centre of its disc so that
// the disc's mean light is the star's exitance, the light it gives a
// square metre of its surface, times the exposure; and a point lamp at
// its centre, of its luminosity a steradian over the unit squared, times
// the exposure, that starts at the ball's surface, so the ball does not
// shade what it lights. The ball casts no shadow.
//
// A star too far off to cover a pixel is a point source of its true
// brightness (the design's Drawing): seen is the least radius, in the
// world's unit, the ball is drawn at, and a ball drawn wider than the
// star is dimmed by the share of the area it gains, so its light in the
// picture is the star's. Nought draws the star at its own size.
func starOf(shine generate.Shine, at maths.Vec, unit, exposure,
	seen float64) (thing.Thing, emit.Lit, error) {
	radius, watts := shine.Radius.InMetres(), shine.Luminosity.InWatts()
	heat := shine.Temperature.InKelvin()
	limb := generate.LimbCoefficient(heat)
	exitance := watts / (wholeSphere * radius * radius)
	drawn := max(radius, seen*unit)
	dimmed := (radius / drawn) * (radius / drawn)
	_, glow, err := stage.Blackbody(heat,
		exitance*exposure*dimmed/limbMeanShare(limb))
	if err != nil {
		return thing.Thing{}, emit.Lit{}, fmt.Errorf("%s: %w", shine.Name,
			err)
	}
	spectrum, _, err := stage.Blackbody(heat,
		watts/wholeSphere/(unit*unit)*exposure)
	if err != nil {
		return thing.Thing{}, emit.Lit{}, fmt.Errorf("%s: %w", shine.Name,
			err)
	}
	ball := thing.Thing{
		Shape:    shape.Sphere{Radius: drawn / unit},
		Material: surfaceOf(shine, glow, limb, drawn/unit),
		Motion:   thing.Still(maths.Subspace{Origin: at}),
		Attrs:    thing.Attrs{Clear: true},
	}
	lamp := emit.Point{At: at, Light: spectrum,
		Beyond: radius / unit}.When(still)
	return ball, lamp, nil
}

// lavaSeed is the seed of a lava drop's plates. Picked: any number
// makes plates, and the same drop is the same drop.
const lavaSeed = 7

// surfaceOf is the material a star's ball is drawn with: its glow and limb
// darkening, or, for a molten body, a crust drifting on magma whose mean
// light over the ball is that glow (texture.Lava), with the same limb
// darkening. The radius is the ball's, in the world's unit.
func surfaceOf(shine generate.Shine, glow maths.Light, limb,
	radius float64) *thing.Material {
	if !shine.Molten {
		return &thing.Material{Name: shine.Name, Glow: glow, Limb: limb}
	}
	return &thing.Material{Name: shine.Name, Limb: limb,
		Shine: texture.NewLava(lavaSeed, radius, glow)}
}

// still is the moment a star's lamp is lit at: no star here pulses.
const still = 0

// starFloor is the least radius, in pixels, a star is drawn at. Picked by
// eye, 2026-11-26: two pixels across a radius reads as a disc at a
// glance and not as a speck, and the light is kept either way.
const starFloor = 2.0

// unexposed is the exposure of stars staged for a picture the print
// exposes: one, which leaves their light as it is.
const unexposed = 1.0

// starred puts each pair of the set's stars into the world: both balls
// and both lamps, at their places on their orbits about the pair's
// middle at the pair's time. A star is drawn at least a pixel's width
// across, as seen from the eye. It refuses a pair the generator refuses.
//
// exposure is what the stars' light is scaled by: the film's, where
// staging scales the lamps by it, and unexposed where the print does, as
// it does for a set drawn from its world half alone.
//
// It gives back a line for each number of a lava drop that is made up,
// "made up: ...", which the draw says before it draws.
func (making *staging) starred(read set.Set, given asked, eye maths.Vec,
	exposure float64) ([]string, error) {
	said, has := read.World()
	if !has {
		return nil, nil
	}
	shift := making.rebasing(read)
	pixel := rayWide(read, given) * float64(given.samples) * starFloor
	var lines []string
	for _, body := range said.Bodies() {
		planet, generated := body.Generated()
		if generated && planet.AirLook != nil {
			look := planet.AirLook
			lines = append(lines, fmt.Sprintf("made up: %s air: scale height "+
				"%g km, loss %g/km, linear RGB tint %v", planet.Name,
				look.Height, look.Loss, look.Colour))
		}
		if generated && planet.Clouds != nil {
			look := planet.Clouds
			lines = append(lines, fmt.Sprintf("made up: %s cloud shell: "+
				"base %g km, top %g km, cover %g, loss %g/km, weather %q",
				planet.Name, look.Base, look.Top, look.Cover, look.Loss,
				look.Weather))
			if look.StormTurn != nil {
				lines = append(lines, fmt.Sprintf("made up: %s cloud storm "+
					"turn %g radians", planet.Name, *look.StormTurn))
			}
		}
		if generated && planet.GroundLook != nil {
			look := planet.GroundLook
			lines = append(lines, fmt.Sprintf("made up: %s ground look: ocean %g, variation %g, ridge contrast %g, seed %d", planet.Name, look.Ocean, look.Variation, look.RidgeContrast, planet.Seed))
		}
		if generated && planet.GiantLook != nil {
			look := planet.GiantLook
			lines = append(lines, fmt.Sprintf("made up: %s giant appearance: paints %v, roil %g, shear %g, grain %g, fine %g, mottle %g, jets %d, spots %v, streets %v, seed %d", planet.Name, look.Paints, look.Roil, look.Shear, look.Grain, look.Fine, look.Mottle, look.Jets, look.Spots, look.Streets, planet.Seed))
		}
		if generated && planet.GiantGas != nil {
			gas := planet.GiantGas
			lines = append(lines, fmt.Sprintf("made up: %s giant gas: depth %g of radius, opacity %g", planet.Name, gas.Depth, gas.Opacity))
		}
	}
	for _, each := range said.Binaries() {
		pair, middle, err := each.Placed(said.Unit())
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", making.given.set,
				each.Name(), err)
		}
		stars := []generate.Placed{pair.Primary, pair.Companion}
		if pair.Drop != nil {
			stars = append(stars, *pair.Drop)
			lines = append(lines, each.Generated().Drop.MadeUp()...)
		}
		for _, placed := range stars {
			at := middle.Add(shift).Add(placed.At.Scale(1 / said.Unit()))
			seen := at.Sub(eye).Len() * pixel
			ball, lamp, err := starOf(placed.Shine, at, said.Unit(),
				exposure, seen)
			if err != nil {
				return nil, fmt.Errorf("%s: %s: %w", making.given.set,
					each.Name(), err)
			}
			making.world.Add(ball)
			making.world.Lamp(lamp)
			making.made.things++
			making.made.lamps++
		}
	}
	return lines, nil
}
