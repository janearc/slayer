package cities

import "testing"

// The facts of a box the tests derive densities from: a hundred square
// metres of ground and two square metres a person, a bivouac's.
const (
	boxGround    = 100.0
	bivouacFloor = 2.0
)

// bivouacDensity is a one-person bivouac's density, the plan's half.
const bivouacDensity = 0.5

// crushStanding is the crush limit the demand tests hold a floor to, the
// plan's six standing.
const crushStanding = 6.0

// sixtyFour is the demand of the plan's example: a box of density one
// asked to hold sixty-four storeys' worth of people.
const sixtyFour = 64

// The floors a person is given that the tests try, in square metres: the
// least, a bivouac's, a flat's and a house's.
var floorsAPerson = []float64{leastPerPerson, bivouacFloor, 7, 40}

// storeysOfBox is how many storeys of floor the box the doubling test
// derives from has.
const storeysOfBox = 4

// factsOf are the facts of a place of some ground, with a floor, whose
// people are given a floor each: as many as the floor sleeps.
func factsOf(ground, floor, per float64) Facts {
	return Facts{Sleeps: int(floor / per), FloorArea: floor,
		PitchArea: ground}
}

func TestOneStoreyIsNeverOverOne(t *testing.T) {
	for _, per := range floorsAPerson {
		density, err := DensityOf(factsOf(boxGround, boxGround, per))
		if err != nil {
			t.Fatal(err)
		}
		if density > 1 {
			t.Errorf("one storey at %v square metres a person is %v",
				per, density)
		}
	}
}

func TestABivouacIsAHalf(t *testing.T) {
	density, err := DensityOf(factsOf(bivouacFloor, bivouacFloor,
		bivouacFloor))
	if err != nil || density != bivouacDensity {
		t.Errorf("a bivouac of two square metres for one is %v, %v",
			density, err)
	}
}

func TestDoublingFloorAPersonHalvesDensityAndDoublingStoreysDoublesIt(
	t *testing.T) {
	floor := storeysOfBox * boxGround
	one, _ := DensityOf(factsOf(boxGround, floor, bivouacFloor))
	half, _ := DensityOf(factsOf(boxGround, floor, 2*bivouacFloor))
	twice, _ := DensityOf(factsOf(boxGround, 2*floor, bivouacFloor))
	if half != one/2 || twice != one*2 {
		t.Errorf("density %v; at twice the floor a person %v, at twice "+
			"the storeys %v", one, half, twice)
	}
}

func TestFactsThatAreNoPlaceAreRefused(t *testing.T) {
	for name, facts := range map[string]Facts{
		"no ground":       {Sleeps: 1, FloorArea: boxGround},
		"negative ground": {Sleeps: 1, FloorArea: boxGround, PitchArea: -1},
		"fewer than none": {Sleeps: -1, FloorArea: 1, PitchArea: boxGround},
		"crushed people": {Sleeps: sixtyFour, FloorArea: 1,
			PitchArea: boxGround},
	} {
		if _, err := DensityOf(facts); err == nil {
			t.Errorf("%s: %+v gave a density", name, facts)
		}
	}
}

func TestCrowdingForcesHeightAsThePlanSays(t *testing.T) {
	demand := demand{Occupants: sixtyFour, Footprint: 1, PerPerson: 1,
		Crush: crushStanding}
	for crowding, want := range map[float64]int{1: 64, 4: 16, 3: 22} {
		demand.Crowding = crowding
		storeys, err := demand.storeys()
		if err != nil || storeys != want {
			t.Errorf("crowding %v: %d storeys, %v, want %d", crowding,
				storeys, err, want)
		}
	}
}

func TestStoreysNeverFallAsDemandRises(t *testing.T) {
	demand := demand{Footprint: 340, PerPerson: 7, Crowding: 1.5,
		Crush: crushStanding}
	last := 0
	for people := 1; people < 5000; people += 37 {
		demand.Occupants = people
		storeys, err := demand.storeys()
		if err != nil {
			t.Fatal(err)
		}
		if storeys < last {
			t.Fatalf("%d people need %d storeys, fewer than %d", people,
				storeys, last)
		}
		if demand.held(storeys) < people {
			t.Fatalf("%d storeys hold %d, not the %d asked", storeys,
				demand.held(storeys), people)
		}
		last = storeys
	}
}

func TestNoFloorIsCrowdedPastTheCrush(t *testing.T) {
	demand := demand{Occupants: sixtyFour, Footprint: 1, PerPerson: 1,
		Crowding: crushStanding + 1, Crush: crushStanding}
	if storeys, err := demand.storeys(); err == nil {
		t.Errorf("a crowding over the crush limit gave %d storeys", storeys)
	}
	demand.Crowding = crushStanding
	if _, err := demand.storeys(); err != nil {
		t.Errorf("a crowding at the crush limit was refused: %v", err)
	}
}

func TestADemandOfNoOneOrOnNothingIsRefused(t *testing.T) {
	sound := demand{Occupants: 1, Footprint: 1, PerPerson: 1, Crowding: 1,
		Crush: crushStanding}
	for name, flaw := range map[string]func(*demand){
		"no one":       func(demand *demand) { demand.Occupants = 0 },
		"no footprint": func(demand *demand) { demand.Footprint = 0 },
		"no crowding":  func(demand *demand) { demand.Crowding = 0 },
		"no floor":     func(demand *demand) { demand.PerPerson = 0.5 },
	} {
		demand := sound
		flaw(&demand)
		if _, err := demand.storeys(); err == nil {
			t.Errorf("%s: %+v gave storeys", name, demand)
		}
	}
}
