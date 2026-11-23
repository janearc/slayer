// Package stage builds a set's world for the tracer: each piece the set
// file says, made as the tracer holds it.
//
// a set says what is there and stage makes it: a box as triangles, a
// black body's light from celestial, and its colour through observer.
package stage

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// World is the set's world, built: its boxes and spheres in the
// materials they wear, its lamps, spots, floods and suns lit, its sky
// hung, its bodies placed and its wisps of gas and its fizzes let in.
//
// each list goes in where the set file writes it, and each piece in its
// list's order, so the world drawn from a file is the same each time.
//
// it refuses a black body whose light the kit cannot carry, too cold to
// see or too hot to hold, a box too thin to have faces, a sun whose way
// is too short to point anywhere, and a body its template cannot place.
func World(said set.World) (*thing.World, error) {
	world := thing.NewWorld()
	if err := Add(world, said, unmoved); err != nil {
		return nil, err
	}
	return world, nil
}

// Add builds a set's world as World does, into a world that is there
// already, each place the set says moved by shift. A world that has
// things, lamps and a sky keeps them, in their order, and the set's come
// after; a sky the world already has stays, and one it has none of is the
// set's.
//
// shift is in the world's space, as every place a set says is. A flood, a
// sun and a spot's aim name no place, so they do not move. On a refusal
// the world may hold some of what was added, so a caller drops it.
func Add(world *thing.World, said set.World, shift maths.Vec) error {
	worn := map[string]*thing.Material{}
	for _, each := range said.Materials() {
		made, err := material(each, said.Unit())
		if err != nil {
			return err
		}
		worn[each.Name()] = made
	}
	for _, key := range said.Order() {
		if build, builds := builders[key]; builds {
			if err := build(world, said, worn, shift); err != nil {
				return err
			}
		}
	}
	return nil
}

// Lens is what the film's camera sees through: a pinhole as wide as the
// film says, or of no width, which a pinhole reads as its own, where the
// film does not say.
//
// every angle a set says is turned to radians by maths.Degree, as the
// css sheets turned theirs, so a scene drawn from either is the same.
func Lens(said set.Film) film.Pinhole {
	return film.Pinhole{Wide: said.Wide() * maths.Degree}
}

// Print is how the film's raw becomes a picture: its exposure, and its
// print's knee and top where it says a print.
func Print(said set.Film) colour.Tone {
	knee, top, _ := said.Print()
	return colour.Tone{Exposure: said.Exposure(), Knee: knee, Top: top}
}

// material is a material: its colour, glow and clear interface, built for
// the renderer.
func material(said set.Material, metresPerUnit float64) (*thing.Material, error) {
	made := &thing.Material{Name: said.Name(), Source: said.Source()}
	if clear, has := said.Clear(); has {
		made.WaterComposition, _ = clear.WaterComposition()
		made.Through = clear.Through
		made.Index = clear.Index
		made.Gloss = clear.Gloss
		made.Rank = clear.Rank
		if name, extinction, optical := clear.OpticalMedium(); optical {
			absorption, scattering, phaseG, steps, maxDistance, participating :=
				clear.ParticipatingMedium()
			var medium *thing.ClearMedium
			var err error
			if participating {
				medium, err = thing.NewClearMediumWithScattering(name,
					maths.RGB(absorption[0], absorption[1], absorption[2]),
					maths.RGB(scattering[0], scattering[1], scattering[2]),
					phaseG, metresPerUnit, steps, maxDistance)
			} else {
				medium, err = thing.NewClearMedium(name, maths.RGB(
					extinction[0], extinction[1], extinction[2]), metresPerUnit)
			}
			if err != nil {
				return nil, fmt.Errorf("material %s: %w", said.Name(), err)
			}
			if water, hasWater := clear.WaterComposition(); hasWater {
				medium, err = medium.WithWaterComposition(water, clear.WaterOpticalSource())
				if err != nil {
					return nil, fmt.Errorf("material %s: %w", said.Name(), err)
				}
			}
			made.ClearMedium = medium
		}
	}
	if painted, has := said.Colour(); has {
		made.Colour = light(painted)
	}
	if glows, glowing := said.Glowing(); glowing {
		made.Glow = light(glows)
	}
	if heat, luminance, glows := said.Glow(); glows {
		_, glow, err := blackbody(heat, luminance)
		if err != nil {
			return nil, fmt.Errorf("material %s: %w", said.Name(), err)
		}
		made.Glow = glow
	}
	return made, nil
}

// Colour is a colour a set says, as the tracer carries light.
func Colour(said set.Colour) maths.Light { return light(said) }

// light is a colour as the tracer carries light: a swatch's, by its
// name, or the linear light the set says.
func light(said set.Colour) maths.Light {
	if named, isSwatch := said.Swatch(); isSwatch {
		found, _ := swatches.Named(named)
		return swatches.Light(found)
	}
	return maths.RGB(said.Linear())
}

// blackbody is a black body's spectrum at a temperature in kelvin, and
// its colour, both scaled so the colour's luminance is the one wanted.
// It refuses a body so cold its light is all nought, or so hot it
// overflows, rather than let either through as a NaN.
func blackbody(kelvin, luminance float64) (bands.Light, maths.Light,
	error) {
	heat := measure.Kelvin(kelvin)
	spectrum := bands.Split(swatch.Blackbody(heat.InKelvin()))
	red, green, blue := srgb.Light(observer.Swatch(spectrum))
	shade := maths.RGB(red, green, blue)
	scale := luminance / shade.Luminance()
	if math.IsNaN(scale) || math.IsInf(scale, 0) {
		return bands.Light{}, maths.Light{}, fmt.Errorf("a black body at "+
			"%v K, whose light the kit can't carry", kelvin)
	}
	return spectrum.Scale(scale), shade.Scale(scale), nil
}
