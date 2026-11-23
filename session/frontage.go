package session

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// A stand of the habit frontage stands along the line of a channel of the
// set's world: terraces end to end with their fronts to the water, trees
// along the quay, boats along the wall. The set says the channel, the side
// of its line, how far from the wall the fronts stand and the room between
// runs; the channel says where its walls are.
//
// A scene on no terrain stands such a stand on flat ground at the scene's
// zero, which is where a thing with an up of nought stands, and a boat
// stand on the level of the world's ocean, in a channel whose water is as
// deep as the boat draws.

// channelOf is the channel of the world that a frontage stand stands
// along, or why it has none.
func (stage *staging) channelOf(each set.Stand) (set.Channel, error) {
	of, _, _, _, _ := each.Frontage()
	if world, has := stage.recipes.World(); has {
		for _, channel := range world.Channels() {
			if channel.Name() == of {
				return channel, nil
			}
		}
	}
	return set.Channel{}, fmt.Errorf("the frontage of stand %s is along "+
		"%q, which is not a channel of the world", each.Name(), of)
}

// frontageOf is the line of fronts of a frontage stand: the channel's line
// moved to the side by half its width, the thickness of its wall and the
// stand's setback.
func (stage *staging) frontageOf(each set.Stand) (cities.Frontage, error) {
	channel, err := stage.channelOf(each)
	if err != nil {
		return cities.Frontage{}, err
	}
	_, right, setback, gap, _ := each.Frontage()
	offset := channel.Width()/2 + channel.WallThickness() + setback
	if !(offset > 0) {
		return cities.Frontage{}, fmt.Errorf("the setback of stand %s, %v m, "+
			"puts its line of fronts at or past the middle of channel %s, "+
			"which is %v m from its wall", each.Name(), setback,
			channel.Name(), channel.Width()/2+channel.WallThickness())
	}
	return cities.FrontageAlong(channel.Line(), offset, right, gap), nil
}

// fronted is the sites a frontage stand's habit lays along its channel.
// A terrace is laid as the runs its seeds draw, and a tree or a boat at
// the spacing its data gives.
func (stage *staging) fronted(each set.Stand) (cities.Laid, error) {
	along, err := stage.frontageOf(each)
	if err != nil {
		return cities.Laid{}, err
	}
	item, err := cities.ItemOf(each.Kind())
	if err != nil {
		return cities.Laid{}, err
	}
	if !item.Spaced {
		item.Run = frontRuns{stage: stage, each: each}.room
	}
	habit, err := cities.HabitNamed(each.Habit())
	if err != nil {
		return cities.Laid{}, err
	}
	count, seed := each.Count()
	return habit.Sites(cities.OnFrontage(along), item, cities.Seed(seed),
		count)
}

// frontRuns is the room each run of a frontage stand takes, by its number
// among the sites its habit lays. Each run is drawn from a number of its
// own, and a street lies between one run and the next, so no run is drawn
// with a thought for its neighbour's colour.
type frontRuns struct {
	stage *staging
	each  set.Stand
}

// room is the room of the terrace that will stand at a run's number, from
// the number the stage will make it from.
func (runs frontRuns) room(number int) (cities.Run, error) {
	_, seed := runs.each.Count()
	return runs.stage.runFrom(runs.each, standPick(runs.each, seed, number))
}

// groundlessStand stands a stand of a scene on no terrain. A boat stand
// floats on the water of a channel; any other stands on flat ground at the
// scene's zero, by its habit or over its within region. A stand that
// needs terrain heights or a mist floor is refused by name.
func (stage *staging) groundlessStand(each set.Stand) error {
	if wants := terrainWants(each); wants != "" {
		return fmt.Errorf("a scene with no terrain has no ground for a "+
			"stand that says %s", wants)
	}
	if _, floats := afloatKind(each.Kind()); floats {
		return stage.moored(each)
	}
	return stage.afloatStand(each)
}

// moored stands a boat stand along the water of a channel, in the habit
// frontage, each boat at the level of the world's ocean.
func (stage *staging) moored(each set.Stand) error {
	if each.Habit() != cities.HabitFrontage {
		return fmt.Errorf("a boat stand on a scene with no terrain floats "+
			"in a channel's water by the habit frontage, not %q",
			each.Habit())
	}
	water, err := stage.channelWater(each)
	if err != nil {
		return err
	}
	return stage.habited(each, water.at, false)
}

// waterAt is the water of a scene on no terrain: its height in the scene's
// metres, and the square of the world's ocean that holds it, as the east
// and north of its middle and half its side. The channel holds no water of
// its own, so a place has water only where the ocean's square reaches it.
type waterAt struct {
	level, east, north, half float64
}

// at is the place on the water at an easting and a northing, and whether
// the ocean's square covers it.
func (water waterAt) at(east, north float64) (maths.Vec, bool) {
	covered := math.Abs(east-water.east) <= water.half &&
		math.Abs(north-water.north) <= water.half
	return maths.V(east, water.level, -north), covered
}

// oceanWater is the water the world's ocean lays, or why it lays none.
func (stage *staging) oceanWater(each set.Stand) (waterAt, error) {
	world, _ := stage.recipes.World()
	surface, has := world.OceanSurface()
	if !has {
		return waterAt{}, fmt.Errorf("the world has no ocean to lay water "+
			"for stand %s", each.Name())
	}
	middle := surface.MapMiddle()
	return waterAt{level: middle[2], east: middle[0], north: middle[1],
		half: surface.Extent() / 2}, nil
}

// channelWater is the water a boat stand floats on: the level of the
// world's ocean, in a channel whose water is as deep as the most that a
// boat of the stand's kind draws, and whose line the ocean's square
// covers. A channel with less water, a kind that is not a boat, or a line
// the ocean does not reach is refused.
func (stage *staging) channelWater(each set.Stand) (waterAt, error) {
	channel, err := stage.channelOf(each)
	if err != nil {
		return waterAt{}, err
	}
	water, err := stage.oceanWater(each)
	if err != nil {
		return waterAt{}, err
	}
	draught, boat := manufacture.BoatDraught(each.Kind())
	if !boat {
		return waterAt{}, fmt.Errorf("a %s is not a boat with a draught to "+
			"float in channel %s", each.Kind(), channel.Name())
	}
	deep := water.level - (channel.Top() - channel.Depth())
	if !(deep >= draught) {
		return waterAt{}, fmt.Errorf("the water of channel %s is %.2f m "+
			"deep, and a %s draws up to %.2f m", channel.Name(), deep,
			each.Kind(), draught)
	}
	for _, point := range channel.Line() {
		if _, covered := water.at(point[0], point[1]); !covered {
			return waterAt{}, fmt.Errorf("the ocean's square does not reach "+
				"the point %v of channel %s, so it has no water there",
				point, channel.Name())
		}
	}
	return water, nil
}
