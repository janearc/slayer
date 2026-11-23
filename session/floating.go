package session

import (
	"fmt"
	"math/rand/v2"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// A stand of a kind that floats, as a boat does, stands on the sea and
// never on the ground: where the terrain is under the sea by at least
// the stand's least water, at the sea's height less the kind's waterline.
//
// The sea is the terrain's own `sea`, or, where the terrain says none, the
// height of the middle of the set's `world.ocean`. A place on land, or in
// water too shallow, is not offered. The stand's habit lays out its sites
// as it would anywhere; this only says where the ground is.

// afloatKind is the waterline of a kind that floats, in metres, and
// whether it floats.
func afloatKind(kind string) (waterline float64, floats bool) {
	lowered, floats := manufacture.Buoyant(kind)
	return lowered.Waterline.InMetres(), floats
}

// seaLevel is the height of the sea in the map's metres, and whether
// the set has one: the terrain's `sea`, or the middle of the ocean.
func (stage *staging) seaLevel() (float64, bool) {
	if ground, has := stage.recipes.Ground(); has {
		if level, said := ground.Sea(); said {
			return level, true
		}
	}
	world, has := stage.recipes.World()
	if !has {
		return 0, false
	}
	surface, has := world.OceanSurface()
	if !has {
		return 0, false
	}
	return surface.MapMiddle()[2], true
}

// onSea is the water of a scene on terrain, for a stand that floats: a
// place has water where the ground is under the sea by the least water,
// and the place is at the sea's height less the kind's waterline.
func (stage *staging) onSea(waterline, least float64) groundAt {
	level, has := stage.seaLevel()
	return func(east, north float64) (maths.Vec, bool) {
		ground, err := stage.at(east, north)
		if err != nil || !has || ground.Y+stage.lowest > level-least {
			return maths.Vec{}, false
		}
		return maths.V(ground.X, level-stage.lowest-waterline, ground.Z), true
	}
}

// floated stands a stand of a kind that floats: by its habit if it has
// one, and otherwise scattered over the water of the ground.
func (stage *staging) floated(each set.Stand, waterline float64) error {
	if each.Habit() != "" {
		return stage.habited(each, stage.onSea(waterline,
			each.LeastWater()), false)
	}
	return stage.scattered(each, stage.seaPlace(waterline,
		each.LeastWater()))
}

// seaPlace is a chooser of places for a stand that is not laid out by a
// habit: one the generator offers over the ground that has water, clear
// of what the stand excludes.
func (stage *staging) seaPlace(waterline, least float64) func(set.Stand,
	*rand.Rand) (maths.Vec, error) {
	on := stage.onSea(waterline, least)
	return func(each set.Stand, places *rand.Rand) (maths.Vec, error) {
		offer, err := stage.offering(each, places)
		if err != nil {
			return maths.Vec{}, err
		}
		for range tries {
			place, water := on(offer())
			if water && stage.allowed(each, place) {
				return place, nil
			}
		}
		return maths.Vec{}, fmt.Errorf("no water %v m deep or more in %d "+
			"tries", least, tries)
	}
}

// standPick is the pick that stands a stand's thing at an index: one of
// a few varieties, so that a forest is made once in each; but a stand of
// a kind that floats, or of houses laid by the grid or along the ground's
// contours, draws each afresh, so that no two are alike.
func standPick(each set.Stand, seed uint64, index int) uint64 {
	_, floats := afloatKind(each.Kind())
	habit := each.Habit()
	if floats || habit == cities.HabitGrid || habit == cities.HabitContour ||
		habit == cities.HabitFrontage {
		return seed<<gridPickShift + uint64(index)
	}
	return seed*standVariety + uint64(index%standVariety)
}
