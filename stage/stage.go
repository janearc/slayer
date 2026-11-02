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
// hung, its bodies placed and its wisps of gas let in.
//
// each list goes in where the set file writes it, and each piece in its
// list's order, so the world drawn from a file is the same each time.
//
// it refuses a black body whose light the kit cannot carry, too cold to
// see or too hot to hold, a box too thin to have faces, a sun whose way
// is too short to point anywhere, and a body its template cannot place.
func World(said set.World) (*thing.World, error) {
	worn := map[string]*thing.Material{}
	for _, each := range said.Materials() {
		made, err := material(each)
		if err != nil {
			return nil, err
		}
		worn[each.Name()] = made
	}
	world := thing.NewWorld()
	for _, key := range said.Order() {
		if build, builds := builders[key]; builds {
			if err := build(world, said, worn); err != nil {
				return nil, err
			}
		}
	}
	return world, nil
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

// material is a material: its colour, its glow, or both; a glow is a
// black body's light, or a colour.
func material(said set.Material) (*thing.Material, error) {
	made := &thing.Material{Name: said.Name()}
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
