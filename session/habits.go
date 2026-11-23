package session

import (
	"fmt"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// An area of a stand is in metres east of the terrain's west edge and
// north of its south edge, as a thing's place is; the stage keeps the
// terrain's north edge, so the south edge is the depth of the ground
// below it.

// A stand with a habit is not scattered by the stage: its habit, from
// package cities, lays out the sites, each a place and a facing, over the
// stand's within region, and the stage stands one thing on each. A stand
// with no habit is scattered as it always was.

// groundAt is the place a thing stands for an easting and a northing, and
// whether the ground there will have one: the flat world ground of a
// scene in space always will, and terrain will where there is ground and
// it is between the stand's heights.
type groundAt func(east, north float64) (maths.Vec, bool)

// afloatStand stands one stand in space: by its habit if it has one, and
// otherwise scattered over the flat world ground.
func (stage *staging) afloatStand(each set.Stand) error {
	if each.Habit() == cities.HabitContour {
		return fmt.Errorf("the habit contour lays rows along the contours "+
			"of terrain, and the scene of stand %s has none", each.Name())
	}
	if each.Habit() != "" {
		return stage.habited(each, flatAt, false)
	}
	return stage.scattered(each, stage.onFlat)
}

// flatAt is the flat world ground of a scene in space: every place has
// ground, at ground level.
func flatAt(east, north float64) (maths.Vec, bool) {
	return maths.V(east, flatGround, -north), true
}

// onTerrain is the ground of a scene on terrain, for a stand: a place has
// ground where the terrain is, between the stand's heights.
func (stage *staging) onTerrain(each set.Stand) groundAt {
	above, below := each.Band()
	return func(east, north float64) (maths.Vec, bool) {
		place, err := stage.at(east, north)
		if err != nil {
			return maths.Vec{}, false
		}
		height := place.Y + stage.lowest
		return place, height > above && height < below
	}
}

// laid is the sites a stand's habit lays out over its within region, and
// the line the habit says of what it did.
func (stage *staging) laid(each set.Stand) (cities.Laid, error) {
	if each.Habit() == cities.HabitFrontage {
		return stage.fronted(each)
	}
	if west, south, east, north, boxed := each.Area(); boxed {
		_, down, step := stage.ground.Size()
		base := stage.north - float64(down-1)*step
		return stage.gridded(each, cities.Quarter{West: stage.west + west,
			South: base + south, East: stage.west + east,
			North: base + north})
	}
	of, from, to, near, far, within := each.Within()
	seen, known := stage.places[of]
	if !within || !known {
		return cities.Laid{}, fmt.Errorf("the habit %s lays sites over a "+
			"within region about a thing that stands, and %q is not one",
			each.Habit(), of)
	}
	habit, err := cities.HabitNamed(each.Habit())
	if err != nil {
		return cities.Laid{}, err
	}
	item, err := cities.ItemOf(each.Kind())
	if err != nil {
		return cities.Laid{}, err
	}
	count, seed := each.Count()
	ground := cities.Ground{East: stage.west + seen.X,
		North: stage.north - seen.Z, From: from, To: to, Near: near,
		Far: far}
	if climb, climbs := each.Climb(); climbs {
		ground.Climb = &climb
		ground.Height = stage.heightAt
		item.Run = stage.runOf(each)
	}
	return habit.Sites(ground, item, cities.Seed(seed), count)
}

// heightAt is the height of the ground at an easting and a northing in the
// model's metres, in the scene's frame. Beyond the ground it is the height
// of the nearest ground there is.
func (stage *staging) heightAt(east, north float64) float64 {
	return stage.groundAt(east-stage.west, stage.north-north)
}

// runOf is the room each terrace of a stand takes, by its number among the
// sites its habit lays: the room of the terrace that will stand there, from
// the same seed the stage will make it from.
func (stage *staging) runOf(each set.Stand) func(int) (cities.Run, error) {
	return contourRuns{stage: stage, each: each}.room
}

// gridded is the sites a stand's habit lays out over a rectangle of the
// map, which the habit fills in place of a wedge about a thing.
func (stage *staging) gridded(each set.Stand, box cities.Quarter) (cities.Laid,
	error) {
	habit, err := cities.HabitNamed(each.Habit())
	if err != nil {
		return cities.Laid{}, err
	}
	item, err := cities.ItemOf(each.Kind())
	if err != nil {
		return cities.Laid{}, err
	}
	count, seed := each.Count()
	return habit.Sites(cities.Ground{Box: &box}, item, cities.Seed(seed),
		count)
}

// facingOf is the turn about the vertical, in radians, that makes a
// thing's front, which is its positive z, look along a site's bearing; or
// nothing for a site that leaves the facing to the stage. A bearing is
// clockwise from north and positive z looks south, so the turn is the
// bearing taken from half a turn.
func facingOf(site cities.Site) *float64 {
	if !site.Faced {
		return nil
	}
	turn := (halfTurnDegrees - site.Heading) * maths.Degree
	return &turn
}

// halfTurnDegrees is half a turn, in degrees, which is how far a bearing
// of south is from one of north.
const halfTurnDegrees = 180

// gridPickShift keeps a grid stand's seed above the site's number in its
// pick, so that every row of houses of a grid is drawn afresh and no two
// are alike: a street is not four rows over and over.
const gridPickShift = 20

// habited stands a stand's things on the sites its habit lays out, one
// each, skipping a site the ground has no place for or the stand excludes.
// The stand's first alight things burn, as in a scatter.
func (stage *staging) habited(each set.Stand, ground groundAt,
	sloped bool) error {
	laid, err := stage.laid(each)
	if err != nil {
		return err
	}
	if laid.Said != "" {
		stage.given.departures = append(stage.given.departures,
			each.Name()+": "+laid.Said)
	}
	_, seed := each.Count()
	var grove []groveTree
	for number, site := range laid.Sites {
		place, stands := ground(site.East, site.North)
		if !stands || !stage.allowed(each, place) {
			continue
		}
		index := len(grove)
		stage.note(each.Name(), each.Kind(), place)
		pick, err := stage.pickOf(each, seed, index, number)
		if err != nil {
			return err
		}
		bounds, err := stage.standSite(each, sloped, pick, uint64(index),
			place, facingOf(site))
		if err != nil {
			return err
		}
		grove = append(grove, groveTree{at: place, bounds: bounds})
	}
	return stage.kept(each, grove)
}
