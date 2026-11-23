package cities

import (
	"math"
	"testing"
)

// The footprint and the slope the founding tests stand on: five metres
// across and ten deep, on ground that falls three in ten toward its front
// wall, which is at the footprint's far z.
const (
	footWidth  = 5.0
	footDepth  = 10.0
	fallToDoor = 0.3
)

// designReach is the least and greatest height of everything a design draws.
func designReach(t *testing.T, design Design) (low, high float64) {
	t.Helper()
	drawn, err := design.Rendered("quick")
	if err != nil {
		t.Fatal(err)
	}
	low, high = math.Inf(1), math.Inf(-1)
	for _, one := range drawn {
		bottom, top := worldReach(one)
		low, high = math.Min(low, bottom[1]), math.Max(high, top[1])
	}
	return low, high
}

func TestADoorFoundsTheFloorAtTheGroundOutsideIt(t *testing.T) {
	footprint := rect{0, 0, footWidth, footDepth}
	falling := incline(0, -fallToDoor)
	highest, err := footingFounded(footprint, falling, FoundHighest)
	if err != nil {
		t.Fatal(err)
	}
	door, err := footingFounded(footprint, falling, FoundAtDoor)
	if err != nil {
		t.Fatal(err)
	}
	front := -fallToDoor * footDepth
	if highest.Base != 0 || math.Abs(highest.Low-front) > heightTol {
		t.Errorf("the highest founding is %+v, and the ground runs from 0 "+
			"at the back to %v at the front", highest, front)
	}
	if math.Abs(door.Base-front) > heightTol || door.drop() > heightTol {
		t.Errorf("a door founding is %+v, and the floor should be the "+
			"front's %v with no plinth under it", door, front)
	}
}

func TestOnGroundThatRisesToTheDoorTheTwoFoundationsAreTheSame(t *testing.T) {
	footprint := rect{0, 0, footWidth, footDepth}
	rising := incline(0, fallToDoor)
	highest, _ := footingFounded(footprint, rising, FoundHighest)
	door, _ := footingFounded(footprint, rising, FoundAtDoor)
	if highest != door {
		t.Errorf("ground that rises to the door founds %+v and %+v",
			highest, door)
	}
}

func TestFootingOnIsTheHighestFoundingAndNothingElse(t *testing.T) {
	footprint := rect{0, 0, footWidth, footDepth}
	ground := incline(0.1, -fallToDoor)
	was, err := footingOn(footprint, ground)
	if err != nil {
		t.Fatal(err)
	}
	now, err := footingFounded(footprint, ground, FoundHighest)
	if err != nil || was != now {
		t.Errorf("FootingOn is %+v and the highest founding %+v (%v)", was,
			now, err)
	}
}

func TestATerraceFoundedAtItsDoorsStandsLowerWhereTheGroundFallsToThem(t *testing.T) {
	ground := incline(0, -fallToDoor)
	highest, err := TerraceFounded("portofino", 3, ground, "", FoundHighest)
	if err != nil {
		t.Fatal(err)
	}
	door, err := TerraceFounded("portofino", 3, ground, "", FoundAtDoor)
	if err != nil {
		t.Fatal(err)
	}
	asBefore, err := terraceAtDetail("portofino", 3, ground, "")
	if err != nil {
		t.Fatal(err)
	}
	_, before := designReach(t, asBefore)
	_, same := designReach(t, highest)
	_, lower := designReach(t, door)
	if before != same {
		t.Errorf("TerraceFounded at the highest ground is %v high and "+
			"TerraceAt %v", same, before)
	}
	if !(lower < same-1) {
		t.Errorf("a terrace founded at its door is %v high, and %v founded "+
			"at the highest ground: the door should be at the ground in "+
			"front, a few metres below", lower, same)
	}
}

func TestATerracesRunIsTheRoomItsPlanTakes(t *testing.T) {
	run, err := TerraceRun("portofino", 3)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := TerraceRun("portofino", 3)
	if run != again {
		t.Errorf("the same seed took %+v and then %+v", run, again)
	}
	const bays, narrowest, widest = 14, 5.4, 5.8
	if run.Length < bays*narrowest || run.Length > bays*widest {
		t.Errorf("a run of fourteen bays is %v m long", run.Length)
	}
	if run.Depth < 10 {
		t.Errorf("a run is %v m deep, and its bays are ten to twelve", run.Depth)
	}
}

func TestARunOfACultureNoOneKnowsIsRefused(t *testing.T) {
	if _, err := TerraceRun("atlantis", 3); err == nil {
		t.Error("the room of a terrace of no culture was given")
	}
}
