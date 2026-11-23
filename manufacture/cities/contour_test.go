package cities

import (
	"math"
	"strings"
	"testing"
)

// The hillsides the contour tests lay terraces on, and the room each run
// takes. The numbers are the tests' own: a slope of three in ten, a run
// forty metres long and ten deep, a lane of four.
const (
	hillRise   = 0.3
	runLength  = 40.0
	runDepth   = 10.0
	laneWidth  = 4.0
	townReach  = 500.0
	closeTo    = 1e-6
	westward   = 270.0
	eastward   = 90.0
	coneSlope  = 0.4
	coneHeight = 100.0
	conePeak   = 200.0
)

// climbsEast is a hillside that climbs hillRise for each metre east, so
// that it falls west and its contours run north and south.
func climbsEast(east, _ float64) float64 { return hillRise * east }

// cone is a hill with its peak conePeak metres east of the origin, and the
// contours of it are circles about the peak.
func cone(east, north float64) float64 {
	return coneHeight - coneSlope*math.Hypot(east-conePeak, north)
}

// sized is the room every run of the tests takes.
func sized(int) (Run, error) {
	return Run{Length: runLength, Depth: runDepth}, nil
}

// climbing is a contour ground at the origin on a height, with the rows
// and runs said and the doors looking down the slope or up it.
func climbing(height func(float64, float64) float64, rows, runs int,
	uphill bool) Ground {
	return Ground{Far: townReach, Height: height, Climb: &Climb{Rows: rows,
		Runs: runs, Lane: laneWidth, Uphill: uphill}}
}

// contourOf is the sites a contour lays, or the test's end.
func contourOf(t *testing.T, ground Ground) Laid {
	t.Helper()
	return sitesOf(t, contour{}, ground, Item{Run: sized}, 1, 0)
}

func TestARowDownASlopeRunsAlongItsContourWithItsFrontToTheFall(t *testing.T) {
	laid := contourOf(t, climbing(climbsEast, 2, 3, false))
	if len(laid.Sites) != 6 {
		t.Fatalf("two rows of three runs laid %d", len(laid.Sites))
	}
	for index, site := range laid.Sites {
		if !site.Faced || math.Abs(site.Heading-westward) > closeTo {
			t.Errorf("run %d faces %v, and the ground falls west", index,
				site.Heading)
		}
		row, run := index/3, index%3
		wantEast := float64(row) * (runDepth + laneWidth)
		wantNorth := -float64(run) * runLength
		if math.Abs(site.East-wantEast) > closeTo ||
			math.Abs(site.North-wantNorth) > closeTo {
			t.Errorf("run %d starts at %v e, %v n, not %v e, %v n", index,
				site.East, site.North, wantEast, wantNorth)
		}
	}
}

func TestARowThatLooksUpTheSlopeRunsTheOtherWayAndLeavesItsOwnDepth(t *testing.T) {
	laid := contourOf(t, climbing(climbsEast, 2, 2, true))
	for index, site := range laid.Sites {
		if math.Abs(site.Heading-eastward) > closeTo {
			t.Errorf("run %d faces %v, and its doors look uphill", index,
				site.Heading)
		}
	}
	second := laid.Sites[1]
	if math.Abs(second.North-runLength) > closeTo {
		t.Errorf("the second run starts %v n, and an uphill row runs north",
			second.North)
	}
	above := laid.Sites[2]
	if math.Abs(above.East-(runDepth+laneWidth)) > closeTo {
		t.Errorf("the row above starts %v e, not a depth and a lane up",
			above.East)
	}
}

func TestTheLaneBetweenTwoRowsIsTheWidthTheStandSays(t *testing.T) {
	laid := contourOf(t, climbing(climbsEast, 2, 1, false))
	backOfFirst := laid.Sites[0].East
	frontOfSecond := laid.Sites[1].East - runDepth
	if math.Abs(frontOfSecond-backOfFirst-laneWidth) > closeTo {
		t.Errorf("the lane is %v m wide, not %v", frontOfSecond-backOfFirst,
			laneWidth)
	}
}

func TestARowOnAHillThatBendsIsTurnedToTheContourWhereEachRunStands(t *testing.T) {
	laid := contourOf(t, climbing(cone, 1, 3, false))
	first, last := laid.Sites[0].Heading, laid.Sites[2].Heading
	if !(first-last > 5) || last < 180 {
		t.Errorf("the runs of a row round a cone face %v then %v, and the "+
			"fall turns away from the peak as the row goes south", first,
			last)
	}
	for index := 1; index < len(laid.Sites); index++ {
		if laid.Sites[index].Heading >= laid.Sites[index-1].Heading {
			t.Errorf("run %d does not turn on from the one before it", index)
		}
	}
}

func TestAContourLaysNoMoreThanItIsAskedForAndNoneBeyondItsReach(t *testing.T) {
	ground := climbing(climbsEast, 3, 4, false)
	few := sitesOf(t, contour{}, ground, Item{Run: sized}, 1, 5)
	if len(few.Sites) != 5 {
		t.Errorf("a contour asked for five laid %d", len(few.Sites))
	}
	ground.Far = 100
	near := contourOf(t, ground)
	for index, site := range near.Sites {
		if math.Hypot(site.East, site.North) > ground.Far {
			t.Errorf("run %d starts %v e, %v n, beyond %v m", index,
				site.East, site.North, ground.Far)
		}
	}
	if !strings.Contains(near.Said, "past its reach") {
		t.Errorf("a contour that left runs out said %q", near.Said)
	}
}

func TestTheSameGroundMakesTheSameContour(t *testing.T) {
	ground := climbing(cone, 2, 2, false)
	first, again := contourOf(t, ground), contourOf(t, ground)
	if len(first.Sites) != len(again.Sites) {
		t.Fatalf("a contour laid %d then %d", len(first.Sites),
			len(again.Sites))
	}
	for index := range first.Sites {
		if first.Sites[index] != again.Sites[index] {
			t.Errorf("run %d is %+v then %+v", index, first.Sites[index],
				again.Sites[index])
		}
	}
}

func TestAContourOnGroundThatIsFlatIsRefused(t *testing.T) {
	flat := func(float64, float64) float64 { return 0 }
	_, err := contour{}.Sites(climbing(flat, 1, 1, false),
		Item{Run: sized}, 1, 0)
	if err == nil || !strings.Contains(err.Error(), "contour") {
		t.Errorf("a contour on flat ground was laid, or refused as %v", err)
	}
}

func TestAContourThatIsNotToldWhatItNeedsIsRefused(t *testing.T) {
	whole := climbing(climbsEast, 1, 1, false)
	for name, ground := range map[string]Ground{
		"no height": {Far: townReach, Climb: whole.Climb},
		"no climb":  {Far: townReach, Height: climbsEast},
		"no rows": {Far: townReach, Height: climbsEast,
			Climb: &Climb{Rows: 0, Runs: 1}},
		"no runs": {Far: townReach, Height: climbsEast,
			Climb: &Climb{Rows: 1, Runs: 0}},
	} {
		if _, err := (contour{}).Sites(ground, Item{Run: sized}, 1, 0); err == nil {
			t.Errorf("a contour with %s was laid", name)
		}
	}
	if _, err := (contour{}).Sites(whole, Item{}, 1, 0); err == nil {
		t.Error("a contour that is not told the room a run takes was laid")
	}
}

func TestTheContourIsAHabitAStandMayName(t *testing.T) {
	habit, err := HabitNamed(HabitContour)
	if err != nil {
		t.Fatal(err)
	}
	if _, isContour := habit.(contour); !isContour {
		t.Errorf("the habit %q is a %T", HabitContour, habit)
	}
}
