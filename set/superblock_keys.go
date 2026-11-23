package set

import (
	"github.com/janearc/slayer/manufacture"
	yaml "gopkg.in/yaml.v3"
)

// What a thing of the kind superblock asks of its parcel, which is the
// set's to say: how wide and deep the parcel is, how many storeys the
// tallest building has, and how the parcel is built over. Its parts, if it
// has any, are the kinds that stand on its roofs, read as a cluster's are.

// mostCeiling is the most storeys a ceiling may be.
const mostCeiling = 200

// deepestSea and highestSea bound the elevation of a terrain's still water,
// in metres: below the deepest ocean trench's floor and above the highest
// mountain's top there is no water to ask for.
const (
	deepestSea = 11000
	highestSea = 9000
)

// leastParcel and mostParcel are the smallest and the largest side of a
// parcel, in metres.
const (
	leastParcel = 1
	mostParcel  = 1e5
)

// mostOccupants is the most people a set may plan a superblock for: ten
// million, past any one block.
const mostOccupants = 1e7

// leastCrowding and mostCrowding bound the crowding a set may say a
// superblock's people tolerate: from a hundredth of the design occupancy
// to a hundred times it. The culture's crush limit holds the rest.
const (
	leastCrowding = 0.01
	mostCrowding  = 100
)

// superblockAsk is what a superblock thing asks, as read: the parcel's
// width and depth, the ceiling, the habit, the people and their crowding,
// nought or empty where the set did not say; and its accretion, drawn
// from the seed where the set did not say or said drawn.
type superblockAsk struct {
	width, depth float64
	ceiling      float64
	habit        string
	occupants    float64
	crowding     float64
	accretion    float64
	drawn        bool
}

// Occupants are the people the superblock is planned for, or nought if
// the set did not say.
func (made Thing) Occupants() int { return int(made.superblock.occupants) }

// Crowding is the crowding the superblock's people tolerate, or nought if
// the set did not say.
func (made Thing) Crowding() float64 { return made.superblock.crowding }

// Accretion is how far the superblock is grown over, from nought to one,
// and whether it draws that from its seed instead.
func (made Thing) Accretion() (float64, bool) {
	return made.superblock.accretion, made.superblock.drawn
}

// accretion is a step that reads a superblock's accretion: a number from
// nought to one, or the word drawn.
func (reading reader) accretion(target *superblockAsk) step {
	return func(key, value *yaml.Node) error {
		if value.Value == "drawn" {
			target.drawn = true
			return nil
		}
		return reading.number(&target.accretion, 0, 1)(key, value)
	}
}

// ParcelWidth is the width of the parcel along east, in metres, or nought
// if the set did not say.
func (made Thing) ParcelWidth() float64 { return made.superblock.width }

// ParcelDepth is the depth of the parcel along north, in metres, or
// nought if the set did not say.
func (made Thing) ParcelDepth() float64 { return made.superblock.depth }

// Ceiling is the most storeys a building of the superblock has, or nought
// if the set did not say.
func (made Thing) Ceiling() int { return int(made.superblock.ceiling) }

// Habit is the way the parcel is built over, or nothing if the set did not
// say.
func (made Thing) Habit() string { return made.superblock.habit }

// parcel is a step that reads a parcel: its width and its depth, each a
// length in metres.
func (reading reader) parcel(target *superblockAsk) step {
	return reading.block(map[string]step{
		"width": reading.quantity(&target.width, "m", leastParcel,
			mostParcel),
		"depth": reading.quantity(&target.depth, "m", leastParcel,
			mostParcel)})
}

// superblockKeys refuses a parcel, a ceiling or a habit on a thing that
// is not a superblock, which has no use for them.
func (reading reader) superblockKeys(block *yaml.Node, made Thing) error {
	if made.kind == manufacture.SuperblockKind {
		return nil
	}
	for _, key := range []string{"parcel", "ceiling", "habit", "accretion",
		"occupants", "crowding"} {
		if says(block, key) {
			return reading.refuse(valueNode(block, key), "thing %s says %s, "+
				"which only a superblock has", made.name, key)
		}
	}
	return nil
}
