package trace

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// the counts of the scratch tests, each a choice of the tests
const (
	// manyLamps and moreLamps are two scenes past lampsEvery, the second
	// far larger, as a street of lanterns is.
	manyLamps = 40
	moreLamps = 400
	// askRuns is how many asks an allocation count is averaged over.
	askRuns = 200
)

// asksAllocate is how many allocations one ask of a scene of so many
// suns makes, once the scratch has been used, averaged over askRuns.
func asksAllocate(count int) float64 {
	scene := stage(suns(count), 0)
	point, up := maths.V(0.3, 0, -0.7), maths.V(0, 1, 0)
	var room [lampsEvery]lit
	scene.asked(room[:], point, up, byLight)
	return testingAllocs(func() {
		scene.asked(room[:], point, up, byLight)
	})
}

// testingAllocs is testing.AllocsPerRun over askRuns.
func testingAllocs(run func()) float64 {
	return testing.AllocsPerRun(askRuns, run)
}

// TestAskingDoesNotAllocateByTheLamps says that an ask of a scene of
// four hundred lamps allocates no more than one of forty: the lamps
// weighed are held in scratch kept between asks, not grown each time.
func TestAskingDoesNotAllocateByTheLamps(t *testing.T) {
	few, many := asksAllocate(manyLamps), asksAllocate(moreLamps)
	if many > few {
		t.Errorf("%v allocations an ask of %d lamps, %v of %d", many,
			moreLamps, few, manyLamps)
	}
}

// TestAnAskFromARoomAllocatesNothing says that with the caller's room
// big enough for its lamps, an ask of a large scene allocates nothing:
// the weighing is in scratch and the picks go into the room.
func TestAnAskFromARoomAllocatesNothing(t *testing.T) {
	if many := asksAllocate(moreLamps); many != 0 {
		t.Errorf("%v allocations an ask of %d lamps, want none", many,
			moreLamps)
	}
}

// TestScratchKeptBetweenAsksChangesNoAnswer says that the same point
// asked again and again, with the scratch reused between, and with
// another point asked between, gets the same lamps with the same counts.
func TestScratchKeptBetweenAsksChangesNoAnswer(t *testing.T) {
	scene := stage(suns(moreLamps), 0)
	point, other := maths.V(0.3, 0, -0.7), maths.V(-4, 0, 2)
	up := maths.V(0, 1, 0)
	first := append([]lit(nil), scene.asked(nil, point, up, byLight)...)
	for range askRuns {
		scene.asked(nil, other, up, byLight)
		again := scene.asked(nil, point, up, byLight)
		if len(again) != len(first) {
			t.Fatalf("%d lamps asked, then %d", len(first), len(again))
		}
		for index := range first {
			if again[index] != first[index] {
				t.Fatalf("ask %d: %v, then %v", index, first[index],
					again[index])
			}
		}
	}
}

// TestGrowingEachAskAllocatesByTheLamps is the anti-test of the first:
// the weighing as it was, its two lists grown from nothing at each ask,
// allocates more for four hundred lamps than for forty, so the first
// test can tell scratch from none.
func TestGrowingEachAskAllocatesByTheLamps(t *testing.T) {
	grown := func(count int) float64 {
		return testingAllocs(func() {
			var rest []lit
			var worths []float64
			for index := range count {
				rest = append(rest, lit{index: index})
				worths = append(worths, 1)
			}
			if len(rest) != count || len(worths) != count {
				t.Fatal("grew", len(rest), "and", len(worths), "of", count)
			}
		})
	}
	if few, many := grown(manyLamps), grown(moreLamps); !(many > few) {
		t.Errorf("%v allocations growing %d, %v growing %d", many,
			moreLamps, few, manyLamps)
	}
}
