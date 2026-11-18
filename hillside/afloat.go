package hillside

import (
	"fmt"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/hillside/placement"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// The stage for a set that says setting: space: the manufactory's craft
// afloat with no ground under them, as smoke 50 builds Webb and the
// Larson by hand. It draws what that smoke draws and no more, and refuses
// by name whatever else such a set says.
//
// Each thing is a manufactured kind, made at its complexity from its seed
// as a thing on the ground is, at (east, height, minus north) in the
// scene's metres, with no ground asked for a height. It is turned by its
// heading where it says one, and not at all where it says none.
//
// The sky is black unless the set's world half hangs one. The light is
// the sun as one distant lamp at any altitude, as bright as sunlight at
// the earth at the film's exposure; or none. What stands on ground, a
// planet, a burning thing and a recipe are refused.

// sunlightAtEarth is the sun's light at the earth's distance, in watts a
// square metre. Measured: the total solar irradiance, 1361 W/m^2 (Kopp
// and Lean 2011), the figure moonlight.go and smoke 50 work from.
const sunlightAtEarth = 1361.0

// groundPart is a part of a set that stands on ground, by the name the
// set file gives it, and whether a set says one.
type groundPart struct {
	name string
	says func(read set.Set) bool
}

// groundParts are the parts a set may say that a scene in space has no
// ground for, in the order they are looked for.
var groundParts = []groundPart{
	{"stands", func(read set.Set) bool { return len(read.Stands()) > nothing }},
	{"texts", func(read set.Set) bool { return len(read.Texts()) > nothing }},
	{"ecosystems", func(read set.Set) bool {
		return len(read.Ecosystems()) > nothing
	}},
	{"fire-fronts", func(read set.Set) bool {
		return len(read.FireFronts()) > nothing
	}},
	{"fuel-beds", func(read set.Set) bool {
		return len(read.FuelBeds()) > nothing
	}},
	{"atmosphere layers", func(read set.Set) bool {
		return len(read.AtmosphericLayers()) > nothing
	}},
	{"a planet", func(read set.Set) bool {
		_, has := read.Planet()
		return has
	}},
}

// craftPart is a thing's key that a craft in space does not draw, by its
// name, and whether a thing says it.
type craftPart struct {
	name string
	says func(each set.Thing) bool
}

// craftParts are the keys of a thing a scene in space refuses.
var craftParts = []craftPart{
	{"recipe", func(each set.Thing) bool { return each.Recipe() != "" }},
	{"alight", func(each set.Thing) bool { return each.Alight() }},
}

// spaceLights are the lights a scene in space is lit by, each by the
// word light: says, and how it lights the stage.
var spaceLights = map[string]func(stage *staging, read set.Set) error{
	"sun":  (*staging).sunned,
	"moon": (*staging).mooned,
	"none": func(*staging, set.Set) error { return nil },
}

// afloat stages a set in space with no planet: its sky, its light and
// its craft; or refuses what it says that the stage does not draw.
func (stage *staging) afloat(read set.Set) error {
	if err := stage.groundless(read); err != nil {
		return err
	}
	lit, known := spaceLights[read.Light()]
	if !known {
		return fmt.Errorf("%s: a scene in space is lit by light: sun, moon or "+
			"light: none, not %q", stage.given.set, read.Light())
	}
	if !hangsASky(read) {
		stage.world.Sky = darkness{}
	}
	if err := lit(stage, read); err != nil {
		return err
	}
	for _, each := range read.Things() {
		if err := stage.craft(each); err != nil {
			return fmt.Errorf("%s: %s: %w", stage.given.set, each.Name(), err)
		}
	}
	return nil
}

// mooned stages the set's moonlight, or refuses when the set has no moon.
func (stage *staging) mooned(read set.Set) error {
	moon, has := read.Moonlit()
	if !has {
		return fmt.Errorf("%s: light: moon says no moon: where it stands",
			stage.given.set)
	}
	way, err := stage.moonlit(moon)
	if err != nil {
		return err
	}
	return stage.skies(read, way)
}

// groundless refuses a set in space that says a part that stands on
// ground, by the part's name, or asks for the lake, which floods ground.
func (stage *staging) groundless(read set.Set) error {
	for _, part := range groundParts {
		if part.says(read) {
			return fmt.Errorf("%s: a scene in space has no ground, and "+
				"this one says %s", stage.given.set, part.name)
		}
	}
	if stage.given.lake > nothing {
		return fmt.Errorf("%s: -lake floods the ground, and a scene in "+
			"space has none", stage.given.set)
	}
	return nil
}

// hangsASky is whether the set's world half says a sky, which the world
// half hangs once the stage is built.
func hangsASky(read set.Set) bool {
	said, has := read.World()
	if !has {
		return false
	}
	_, hung := said.Sky()
	return hung
}

// sunned lights a scene in space by its sun: one distant lamp from the
// sun's way, wherever it stands, as bright as sunlight at the earth at
// the film's exposure, tinted as the sun says.
func (stage *staging) sunned(read set.Set) error {
	sun, has := read.Sunlit()
	if !has {
		return fmt.Errorf("%s: light: sun says no sun: where it stands",
			stage.given.set)
	}
	lamp, err := emit.NewDistant(skyward(sun.Where()),
		tintedSun(sun, sunlightAtEarth*stage.given.exposure))
	if err != nil {
		return err
	}
	stage.world.Lamp(lamp)
	stage.made.lamps++
	return nil
}

// craft stands one thing afloat: made as its kind, at its complexity and
// seed, at its east, north and height, turned by its heading if it says
// one; or refuses a key of it the stage does not draw.
func (stage *staging) craft(each set.Thing) error {
	for _, part := range craftParts {
		if part.says(each) {
			return fmt.Errorf("a craft in space says %s, which is not "+
				"drawn there", part.name)
		}
	}
	one, err := stage.stockedCraft(each)
	if err != nil {
		return err
	}
	at := afloatAt(each)
	stage.places[each.Name()] = at.Origin
	stage.note(each.Name(), each.Kind(), at.Origin)
	start := len(stage.world.Things)
	if err := stage.add(one, each.Kind(), at, each.Seed(), false); err != nil {
		return err
	}
	return stage.partSpin(start, each)
}

// stockedCraft is a thing's kind as the manufactory makes it at its
// complexity and seed: the stock's, or made now and kept.
func (stage *staging) stockedCraft(each set.Thing) (kept, error) {
	key := stocked{name: each.Kind(), pick: int(each.Seed()),
		detail: each.Complexity()}
	if one, known := stage.have[key]; known {
		return one, nil
	}
	fresh, err := stage.constructed(each.Kind(), "", each.Complexity(),
		each.Seed())
	if err != nil {
		return kept{}, err
	}
	stage.have[key] = *fresh
	stage.made.kept += fresh.stored
	return *fresh, nil
}

// afloatAt is where a thing stands in space and how it is turned: at
// (east, height, minus north) in the scene's metres, turned clockwise
// from north by its heading, or not turned where it says none.
func afloatAt(each set.Thing) maths.Subspace {
	return placement.SpaceAt(each)
}
