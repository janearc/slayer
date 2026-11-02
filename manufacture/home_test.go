package manufacture

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// mustBox is a box of so many metres wide, tall and deep.
func mustBox(t *testing.T, wide, tall, deep float64) box {
	t.Helper()
	made, err := newBox(measure.Metres(wide), measure.Metres(tall),
		measure.Metres(deep))
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// a box is closed and wound outward, slayer's mesh judging, every face
// looks away from its middle, and it holds its width times its height
// times its depth, which is worked out and not measured.
func TestABoxIsClosedOutwardAndHoldsItsVolume(t *testing.T) {
	t.Parallel()
	made := mustBox(t, 3, 2, 5)
	skin := made.skin(mustCut(t, 4))
	if flaw := unsound(skin); flaw != "" {
		t.Fatalf("a box: %s", flaw)
	}
	if index := inward(skin, maths.V(0, 1, 0)); index >= 0 {
		t.Errorf("face %d of a box looks inward", index)
	}
	if got := skin.volume(); math.Abs(got-30) > 1e-12 {
		t.Errorf("a box of 3 by 2 by 5 holds %v, want 30", got)
	}
	wide, tall, deep, lowest := extent(skin)
	if wide != 3 || tall != 2 || deep != 5 || lowest != 0 {
		t.Errorf("a box is %v by %v by %v from %v, want 3 by 2 by 5 "+
			"from the ground", wide, tall, deep, lowest)
	}
}

// a box of no width, height or depth is no box.
func TestABoxOfNothingIsRefused(t *testing.T) {
	t.Parallel()
	for _, size := range [][3]float64{{0, 1, 1}, {1, 0, 1}, {1, 1, -1},
		{math.NaN(), 1, 1}, {1, math.Inf(1), 1}} {
		_, err := newBox(measure.Metres(size[0]), measure.Metres(size[1]),
			measure.Metres(size[2]))
		if !errors.Is(err, errBox) {
			t.Errorf("a box of %v: %v, not %v", size, err, errBox)
		}
	}
}

// every storey above the ground stands on the one below it: they meet
// across its width, and its back keeps over the floor below while its
// front reaches no further past that floor's front than the row
// allows.
func TestEveryStoreyStandsOnTheOneBelow(t *testing.T) {
	t.Parallel()
	row := homeRow()
	for number := range uint64(40) {
		plan := row.planned(sow(number))
		for level := 1; level < len(plan.storeys); level++ {
			upper, lower := plan.storeys[level], plan.storeys[level-1]
			if overlap(upper, lower, 0) <= 0 {
				t.Errorf("seed %d: storey %d meets none of the one "+
					"below across", number, level)
			}
			past := upper.front() - lower.front()
			if past > row.past.most*upper.deep+1e-9 {
				t.Errorf("seed %d: storey %d reaches %v m past the "+
					"floor below, more than %v of its depth",
					number, level, past, row.past.most)
			}
			if upper.back() >= lower.front() {
				t.Errorf("seed %d: storey %d stands on nothing",
					number, level)
			}
			if upper.low < lower.top() {
				t.Errorf("seed %d: storey %d is sunk into the one "+
					"below's room", number, level)
			}
		}
	}
}

// a pane stands just off its wall: never in the wall's plane, where a
// ray would meet glass and wall at one depth and the tracers break the
// tie apart, and never far enough off to be seen standing free.
func TestEveryPaneStandsJustOffItsWall(t *testing.T) {
	t.Parallel()
	for number := range uint64(40) {
		plan := homeRow().planned(sow(number))
		if len(plan.panes) == 0 {
			t.Errorf("seed %d: a home of no glass", number)
		}
		for index, pane := range plan.panes {
			wall := plan.storeys[pane.on]
			off, flaw := standsOff(pane.block, wall)
			if flaw != "" {
				t.Fatalf("seed %d, pane %d: %s", number, index, flaw)
			}
			if off < 0.001 || off > 0.05 {
				t.Errorf("seed %d, pane %d stands %v m off its wall, "+
					"want a millimetre to five centimetres", number,
					index, off)
			}
		}
	}
}

// the glass on a wall is no more of it than the row allows.
func TestAWallIsNoMoreGlassThanItsRowAllows(t *testing.T) {
	t.Parallel()
	row := homeRow()
	for number := range uint64(40) {
		plan := row.planned(sow(number))
		glass := map[[2]int]float64{}
		for _, pane := range plan.panes {
			glass[[2]int{pane.on, pane.wall}] += pane.across()
		}
		for at, across := range glass {
			wall := plan.storeys[at[0]]
			long := wall.wide
			if at[1] == sideWall {
				long = wall.deep
			}
			if across > row.glazed.most*long+1e-9 {
				t.Errorf("seed %d: storey %d's wall %d is %v m of glass "+
					"in %v", number, at[0], at[1], across, long)
			}
		}
	}
}

// overlap is how far two blocks meet along an axis: x for 0, y for 1, z for 2.
// a gap is less than zero.
func overlap(one, other block, axis int) float64 {
	low := [3]float64{one.at.X - one.wide/2, one.low,
		one.at.Z - one.deep/2}
	high := [3]float64{one.at.X + one.wide/2, one.top(),
		one.at.Z + one.deep/2}
	otherLow := [3]float64{other.at.X - other.wide/2, other.low,
		other.at.Z - other.deep/2}
	otherHigh := [3]float64{other.at.X + other.wide/2, other.top(),
		other.at.Z + other.deep/2}
	return math.Min(high[axis], otherHigh[axis]) -
		math.Max(low[axis], otherLow[axis])
}

// standsOff is how far a pane stands off a wall: apart along one axis
// and lying wholly on the wall along the other two, or what is wrong.
func standsOff(pane, wall block) (float64, string) {
	apart := -1.0
	for axis := range 3 {
		met := overlap(pane, wall, axis)
		size := [3]float64{pane.wide, pane.tall, pane.deep}[axis]
		switch {
		case met < 0 && apart < 0:
			apart = -met
		case met < 0:
			return 0, "apart from its wall two ways"
		case met < size-1e-9:
			return 0, "past the edge of its wall"
		}
	}
	if apart < 0 {
		return 0, "inside its wall"
	}
	return apart, ""
}
