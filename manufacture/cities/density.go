package cities

import (
	"fmt"
	"math"
)

// Density is people per square metre of ground, a unit fixed for good so
// that nothing added later rescales what is written: a one-person bivouac
// of two square metres is a half, and nothing one storey high is over one
// at its design occupancy. Above one is stacking.
//
// Nobody writes a density. DensityOf derives it from a place's Facts, the
// same a winnebago and a bivouac give: how many it sleeps, the floor they
// have, and the ground it takes.
//
// Crowding is the other number: how many times its design occupancy a
// floor holds. When the demand on a footprint is more than one floor
// holds at the crowding its people tolerate, it rises.

// leastPerPerson is the least floor a design may give a person, in square
// metres: one, below which one storey over the whole of its ground would
// hold more than one person to the square metre.
const leastPerPerson = 1

// wholeSlack is how far over a whole number of storeys a demand may come,
// by the arithmetic alone, and still be that number and not the next.
const wholeSlack = 1e-9

// DensityOf is a place's design occupancy as people per square metre of
// its ground: the people it sleeps over the ground it takes. It refuses
// facts that are not a place: no ground, fewer than no one, or people
// given less floor each than the least a design gives.
func DensityOf(facts Facts) (float64, error) {
	if !(facts.PitchArea > 0) || facts.Sleeps < 0 ||
		!(facts.FloorArea >= float64(facts.Sleeps)*leastPerPerson) {
		return 0, fmt.Errorf("cities: a density needs ground and at least "+
			"%v square metre of floor a person, not %+v", leastPerPerson,
			facts)
	}
	return float64(facts.Sleeps) / facts.PitchArea, nil
}

// demand is the people a footprint must hold and what holding them asks:
// the footprint in square metres, the floor a person is given, the
// crowding the people tolerate, and the crush limit, the most people a
// square metre of floor holds standing.
type demand struct {
	Occupants                             int
	Footprint, PerPerson, Crowding, Crush float64
}

// storeys is how many storeys a footprint rises to so that it holds its
// occupants at the crowding tolerated: the storeys the demand needs at
// the design occupancy, over the crowding, rounded up. Nobody picks a
// height.
//
// It refuses a crowding at which a floor would hold more people to a
// square metre than the crush limit, so that demand past it adds storeys,
// never people on a floor; and a demand of no one, or on no footprint.
func (demand demand) storeys() (int, error) {
	if demand.Occupants < 1 || !(demand.Footprint > 0) ||
		!(demand.PerPerson >= leastPerPerson) || !(demand.Crowding > 0) {
		return 0, fmt.Errorf("cities: a demand needs people, a footprint, "+
			"at least %v square metre a person and a crowding, not %+v",
			leastPerPerson, demand)
	}
	if demand.Crowding/demand.PerPerson > demand.Crush {
		return 0, fmt.Errorf("cities: a crowding of %v at %v square metres "+
			"a person is %v people to the square metre, over the crush "+
			"limit of %v", demand.Crowding, demand.PerPerson,
			demand.Crowding/demand.PerPerson, demand.Crush)
	}
	floors := float64(demand.Occupants) * demand.PerPerson /
		(demand.Footprint * demand.Crowding)
	return int(math.Ceil(floors - wholeSlack)), nil
}

// held is how many people a number of storeys on the footprint holds at
// the crowding tolerated, which is how many fit under a ceiling.
func (demand demand) held(storeys int) int {
	return int(math.Floor(float64(storeys) * demand.Footprint *
		demand.Crowding / demand.PerPerson))
}
