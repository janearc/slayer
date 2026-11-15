package lay_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/crystal"
	"github.com/janearc/slayer/ninjafilm/curve"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/lay"
	"github.com/janearc/slayer/ninjafilm/record"
	"pgregory.net/rapid"
)

// what the clearest tests choose, each picked in picked-numbers.yml
var (
	clearFloorMost  = data.Picked("lay-clear-floor-most")
	clearFloorLeast = data.Picked("lay-clear-floor-least")
	toleranceTight  = data.Picked("tolerance-tight")
)

// lamps is each of the three lamps by itself, in order.
var lamps = []maths.Light{maths.RGB(whole, none, none),
	maths.RGB(none, whole, none), maths.RGB(none, none, whole)}

// clearing is records each seeing and blocking one lamp, red then green
// then blue, as many as asked, each with its curve's least density
// drawn.
func clearing(t *rapid.T, many int) ([]record.Record, []float64) {
	grain, err := crystal.New(grained)
	if err != nil {
		t.Fatal(err)
	}
	var made []record.Record
	var floors []float64
	for index := range many {
		floor := rapid.Float64Range(none, clearFloorMost).Draw(t, "floor")
		tone, err := curve.New(floor, floor+curveMost, curveSlope)
		if err != nil {
			t.Fatal(err)
		}
		lamp := lamps[index%len(lamps)]
		each, err := record.New(lamp, lamp, grain, tone)
		if err != nil {
			t.Fatal(err)
		}
		made, floors = append(made, each), append(floors, floor)
	}
	return made, floors
}

// tenth is ten to the minus a density: the share it passes.
func tenth(density float64) float64 { return math.Pow(tenfold, -density) }

// tenfold is the base of a density. It is defined: a density of one
// passes a tenth.
const tenfold = 10

// near is whether two lights are within a tolerance, lamp by lamp.
func near(one, other maths.Light, tolerance float64) bool {
	return math.Abs(one.R-other.R) <= tolerance &&
		math.Abs(one.G-other.G) <= tolerance &&
		math.Abs(one.B-other.B) <= tolerance
}

// Records one over another add their densities: a stacked stock at its
// clearest passes ten to the minus the sum of the floors in each lamp.
func TestAStackedLayAtItsClearestAddsTheFloors(t *testing.T) {
	rapid.Check(t, stackedClearestAdds)
}

func stackedClearestAdds(t *rapid.T) {
	held, floors := clearing(t, len(lamps)+whole)
	var want maths.Light
	for index, floor := range floors {
		want = want.Add(lamps[index%len(lamps)].Scale(floor))
	}
	got := stackedLay(t).Clearest(held)
	expected := maths.RGB(tenth(want.R), tenth(want.G), tenth(want.B))
	if !near(got, expected, toleranceTight) {
		t.Fatalf("stacked floors %v pass %v, want %v", floors, got,
			expected)
	}
}

// A mosaic whose records clear to nothing passes, at its clearest, the
// share its plate is made to pass: no more.
func TestAMosaicAtItsClearestPassesItsPlatesShare(t *testing.T) {
	rapid.Check(t, mosaicClearestIsPasses)
}

func mosaicClearestIsPasses(t *rapid.T) {
	passes := rapid.Float64Range(passesLeast, whole).Draw(t, "passes")
	scatter := laid(t, "mosaic", elementsSample, passes)
	held := clearToNothing(t)
	got := scatter.Clearest(held)
	if !near(got, maths.Grey(passes), toleranceTight) {
		t.Fatalf("a plate passing %v passes %v at its clearest", passes,
			got)
	}
}

// clearToNothing is one record for each lamp, each clearing to no
// density at all.
func clearToNothing(t *rapid.T) []record.Record {
	grain, err := crystal.New(grained)
	if err != nil {
		t.Fatal(err)
	}
	tone, err := curve.New(none, curveMost, curveSlope)
	if err != nil {
		t.Fatal(err)
	}
	var made []record.Record
	for _, lamp := range lamps {
		each, err := record.New(lamp, lamp, grain, tone)
		if err != nil {
			t.Fatal(err)
		}
		made = append(made, each)
	}
	return made
}

// A lay's clearest is what its sites pass on the mean, each record at
// its floor: over a field of sites the two agree within six standard
// errors of what the field can show.
func TestAClearestIsTheMeanOfItsSites(t *testing.T) {
	rapid.Check(t, clearestIsTheMean)
}

func clearestIsTheMean(t *rapid.T) {
	held, _ := clearing(t, len(lamps))
	for _, scatter := range []lay.Lay{stackedLay(t),
		laid(t, "mosaic", elementsSample, examplePass)} {
		seed := rapid.Uint64().Draw(t, "seed")
		var total maths.Light
		for index := range fieldSites {
			column, row := int64(index%fieldWidth), int64(index/fieldWidth)
			here, filter := scatter.Site(held, seed, column, row, none)
			var dense maths.Light
			for _, one := range here {
				dense = dense.Add(one.Clearest())
			}
			total = total.Add(filter.Times(record.Passed(dense)))
		}
		mean := total.Scale(whole / float64(fieldSites))
		bound := sigmas * examplePass * float64(len(lamps)) /
			math.Sqrt(float64(fieldSites))
		if got := scatter.Clearest(held); !near(got, mean, bound) {
			t.Fatalf("clearest %v, the sites pass %v", got, mean)
		}
	}
}

// Anti: a stacked stock is not as clear as its clearest record. A
// second record with any floor at all over the same lamp passes less.
func TestAStackedLayIsNotAsClearAsOneOfItsRecords(t *testing.T) {
	rapid.Check(t, stackedIsNotOneRecord)
}

func stackedIsNotOneRecord(t *rapid.T) {
	held, floors := clearing(t, len(lamps)+whole)
	if floors[len(lamps)] < clearFloorLeast {
		return
	}
	all := stackedLay(t).Clearest(held)
	alone := stackedLay(t).Clearest(held[:len(lamps)])
	if !(all.R < alone.R) || all.G != alone.G || all.B != alone.B {
		t.Fatalf("a second red record moved %v to %v", alone, all)
	}
}

// Anti: a mosaic whose records keep some density at their clearest
// passes less than its plate's share, in every lamp.
func TestAMosaicWithFloorsPassesLessThanItsPlate(t *testing.T) {
	rapid.Check(t, mosaicFloorsPassLess)
}

func mosaicFloorsPassLess(t *rapid.T) {
	held, floors := clearing(t, len(lamps))
	scatter := laid(t, "mosaic", elementsSample, examplePass)
	got := scatter.Clearest(held)
	for index, lamp := range []float64{got.R, got.G, got.B} {
		if floors[index] >= clearFloorLeast && !(lamp < examplePass) {
			t.Fatalf("a floor of %v passes %v of the lamp, the plate "+
				"passes %v", floors[index], lamp, examplePass)
		}
	}
}
