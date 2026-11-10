package firefront

import (
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
	"time"
)

func TestCoordinatePredicatesRejectEachNonfiniteComponent(t *testing.T) {
	for _, bad := range []float64{math.Inf(-1), math.Inf(1), math.NaN()} {
		if finite(bad) {
			t.Fatal("nonfinite scalar accepted")
		}
		for axis := range 3 {
			p := maths.Vec{}
			switch axis {
			case 0:
				p.X = bad
			case 1:
				p.Y = bad
			case 2:
				p.Z = bad
			}
			if pointOK(p) {
				t.Fatalf("nonfinite component %d accepted", axis)
			}
		}
	}
	if !pointOK(maths.V(1, -2, 3)) || !finite(-math.MaxFloat64) {
		t.Fatal("finite input refused")
	}
}

func TestGroupBudgetCountsBirthsAfterEarlierDeaths(t *testing.T) {
	spec := Spec{Direction: maths.XAxis, Speed: 1, Cell: 1, Duration: time.Second}
	for _, test := range []struct {
		targets []Target
		peak    int
	}{
		{[]Target{{At: maths.Vec{}}, {At: maths.V(1, 0, 10)}}, 1},
		{[]Target{{At: maths.Vec{}}, {At: maths.V(1, 0, 10)}, {At: maths.V(1.5, 0, 20)}}, 2},
	} {
		plan, err := New(spec, test.targets)
		if err != nil {
			t.Fatal(err)
		}
		if got := plan.PeakGroups(); got != test.peak {
			t.Fatalf("half-open group occupancy %d want %d", got, test.peak)
		}
	}
}
