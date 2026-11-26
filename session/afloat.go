package session

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
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
// planet, a burning thing and a recipe other than a spacecraft are refused.
//
// A stand in space grows on a flat world ground at height 0, the level a
// thing with height 0 stands at, and offers its places by its within
// wedge. A stand that needs terrain for anything else is refused by name.

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
		return len(read.AtmosphericLayers()) > len(read.FlatAir())
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

// recipeKinds are the kinds a craft in space may be made from a recipe
// of: the observatory, the tower, the bridge, the street and the lashed
// street, the village, a composition, and the four signs that stand on their own, which
// a recipe can give a script.
var recipeKinds = []string{"space-observatory", "tower", "bridge", "street",
	"lashed-street", manufacture.VillageKind, "composition", "sign-roof", "sign-screen",
	"sign-blade", "sign-awning"}

// craftParts are the keys of a thing a scene in space refuses.
var craftParts = []craftPart{
	{"recipe", func(each set.Thing) bool {
		return each.Recipe() != "" && !slices.Contains(recipeKinds,
			each.Kind())
	}},
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
	for _, each := range read.Stands() {
		start := len(stage.world.Things)
		if err := stage.afloatStand(each); err != nil {
			return fmt.Errorf("%s: %s: %w", stage.given.set, each.Name(), err)
		}
		stage.recordAuthor(start, "scene.set.stands", each.Name())
	}
	return nil
}

// flatGround is the height, in the scene's metres, of the ground a stand
// in space stands on: the same level a thing with height 0 stands at.
const flatGround = 0.0

// onFlat is a place a stand's seed offers on the flat world ground of a
// scene in space: east and north from the stand's wedge, at ground level,
// clear of what the stand excludes. The wedge is required, so a stand in
// space always has a thing to be seen from.
func (stage *staging) onFlat(each set.Stand,
	places *rand.Rand) (maths.Vec, error) {
	offer, err := stage.offering(each, places)
	if err != nil {
		return maths.Vec{}, err
	}
	for range tries {
		east, north := offer()
		place := maths.V(east, flatGround, -north)
		if stage.allowed(each, place) {
			return place, nil
		}
	}
	return maths.Vec{}, fmt.Errorf("no room clear of what it excludes in "+
		"%d tries", tries)
}

// terrainWants is what a stand says that only terrain can answer, by the
// name the set file gives it, or nothing if it says none. A stand in
// space stands on flat ground, which has no heights, no mist floor and
// no fire bed, and offers places only by a wedge.
func terrainWants(each set.Stand) string {
	above, below := each.Band()
	_, _, _, _, _, within := each.Within()
	weather, _ := each.Atmosphere()
	switch {
	case !math.IsInf(above, -1):
		return "above, which needs terrain heights"
	case !math.IsInf(below, 1):
		return "below, which needs terrain heights"
	case !math.IsInf(each.Steepest(), 1):
		return "steepest, which needs terrain slopes"
	case weather != "":
		return "atmosphere, which needs terrain under its mist"
	case each.Alight() > nothing:
		return "alight, which needs terrain for its fire"
	case !within && each.Habit() != cities.HabitFrontage:
		return "no within, so it grows over all the terrain: say within"
	}
	return ""
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
	for _, each := range read.Stands() {
		if wants := terrainWants(each); wants != "" {
			return fmt.Errorf("%s: a scene in space has no terrain, and "+
				"stand %s says %s", stage.given.set, each.Name(), wants)
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
		if part.name == "recipe" && each.Recipe() != "" {
			if recipe, found := stage.recipes.Recipe(each.Recipe()); found && (recipe.IsPlant() || recipe.IsAnimal() || recipe.IsAlga() || recipe.HasTreeProfiles()) {
				continue
			}
		}
		if part.says(each) {
			return fmt.Errorf("a craft in space says %s, which is not "+
				"drawn there", part.name)
		}
	}
	one, err := stage.stockedCraft(each)
	if err != nil {
		return err
	}
	if scale, has := each.Scale(); has {
		if one.deform != nil {
			return fmt.Errorf("scale is not yet supported for deforming geometry")
		}
		if stage.radii == nil {
			stage.radii = map[string]float64{}
		}
		stage.radii[each.Name()] = horizontalRadius(one.bounds) * scale
	}
	at := afloatAt(each)
	stage.places[each.Name()] = at.Origin
	stage.note(each.Name(), stage.reportedKind(each.Kind(), each.Recipe()), at.Origin)
	if _, spinning := each.PartSpin(); spinning && one.deform != nil {
		return fmt.Errorf("deforming geometry cannot use part-spin")
	}
	start := len(stage.world.Things)
	if err := stage.add(one, each.Kind(), at, each.Seed(), false); err != nil {
		return err
	}
	if err := stage.partSpin(start, each); err != nil {
		return err
	}
	stage.recordThingAuthor(start, each.Name())
	return nil
}

// errLevelledCluster is the refusal of a cluster or a superblock that is
// levelled, which is drawn from its parts and has no level of its own.
var errLevelledCluster = errors.New("a cluster or a superblock is not " +
	"levelled; give it a complexity")

// stockedCraft is a thing's kind as the manufactory makes it at its
// complexity and seed: the stock's, or made now and kept. Levelled, it is
// made at the level the camera gives it, or for its place.
func (stage *staging) stockedCraft(each set.Thing) (kept, error) {
	if each.Complexity() == manufacture.Levelled {
		if clusterKey(each) != "" {
			return kept{}, errLevelledCluster
		}
		return stage.levelledStock(each.Kind(), each.Recipe(),
			each.Complexity(), each.Seed(), each.Name(), afloatAt(each))
	}
	key := stocked{name: each.Kind() + "/" + each.Recipe() + clusterKey(each),
		pick: int(each.Seed()), detail: each.Complexity()}
	if one, known := stage.have[key]; known {
		return one, nil
	}
	fresh, err := stage.madeCraft(each)
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
