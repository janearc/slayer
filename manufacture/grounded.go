package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/manufacture/cities"
)

// SlopeAware is whether a kind of building can be stood on a slope: it is
// drawn by package cities with a plinth under each of its buildings, from
// the ground it is told. The terraces, the condominium, the village and
// its house are.
func SlopeAware(name string) bool {
	row, err := kindNamed(name)
	if err != nil {
		return false
	}
	switch row.how.(type) {
	case terraceRow, condominiumRow, villageHouseRow, villageRow:
		return true
	}
	return false
}

// Grounded is a kind of building stood on a ground, in the building's own
// frame, so that each of its buildings stands on the ground under it, on a
// plinth to the lowest of it. It refuses a kind that is not slope aware,
// and a ground there is not.
func Grounded(row Kind, ground cities.Terrain) (Kind, error) {
	if ground == nil {
		return Kind{}, fmt.Errorf("manufacture: %s was given no ground to "+
			"stand on", row.name)
	}
	switch how := row.how.(type) {
	case terraceRow:
		how.ground = ground
		return Kind{name: row.name, how: how}, nil
	case condominiumRow:
		how.ground = ground
		return Kind{name: row.name, how: how}, nil
	case villageHouseRow:
		how.ground = ground
		return Kind{name: row.name, how: how}, nil
	case villageRow:
		how.ground = ground
		return Kind{name: row.name, how: how}, nil
	}
	return Kind{}, fmt.Errorf("manufacture: %s is not a kind that can be "+
		"stood on a slope", row.name)
}

// Founded is a terrace stood on a ground as Grounded stands it, with the
// floor of each bay founded as said: at the highest ground under it, or at
// the ground outside its door. It refuses a kind that is not a terrace,
// since only a terrace has a front door in a wall that is all of its
// front, and a ground there is not.
func Founded(row Kind, ground cities.Terrain,
	founding cities.Founding) (Kind, error) {
	how, terrace := row.how.(terraceRow)
	if !terrace {
		return Kind{}, fmt.Errorf("manufacture: %s is not a terrace, and has "+
			"no front to found on a door", row.name)
	}
	if ground == nil {
		return Kind{}, fmt.Errorf("manufacture: %s was given no ground to "+
			"stand on", row.name)
	}
	how.ground, how.founding = ground, founding
	return Kind{name: row.name, how: how}, nil
}

// RunOf is the room the terrace of a kind takes when it is made from a
// seed number, the number the manufactory sows: how long its row is, and
// how deep. It draws the row's plan and none of its bays.
func RunOf(name string, number uint64) (cities.Run, error) {
	row, err := kindNamed(name)
	if err != nil {
		return cities.Run{}, err
	}
	how, terrace := row.how.(terraceRow)
	if !terrace {
		return cities.Run{}, fmt.Errorf("manufacture: %s is not a terrace, "+
			"and takes no run of a row", name)
	}
	return cities.TerraceRun(how.culture, cities.Seed(number))
}
