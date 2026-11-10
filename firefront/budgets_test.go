package firefront

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

func TestGroupWindowsBudgetGapsAndHalfOpenTargetWindows(t *testing.T) {
	spec := Spec{Direction: maths.XAxis, Speed: 1, Cell: 10, Duration: 2 * time.Second}
	var targets []Target
	for i, x := range []float64{0, 4, 2, 6} {
		z := 0.0
		if i >= 2 {
			z = 20
		}
		targets = append(targets, Target{At: maths.V(x, 0, z), Seed: uint64(i)})
	}
	plan, err := New(spec, targets)
	if err != nil {
		t.Fatal(err)
	}
	// Group windows are [0,6) and [2,8), including their dark gaps.
	// Targets occupy [0,2), [2,4), [4,6), [6,8), with no overlap.
	if len(plan.Groups()) != 2 || plan.PeakGroups() != 2 {
		t.Fatalf("group budget: %d in %d groups", plan.PeakGroups(), len(plan.Groups()))
	}
	for _, test := range []struct {
		tail time.Duration
		want int
	}{
		{0, 1}, {-time.Second, 1}, {time.Second, 2},
		{time.Duration(math.MaxInt64), 4},
	} {
		if got := plan.PeakTargets(test.tail); got != test.want {
			t.Fatalf("tail %s: peak %d, want %d", test.tail, got, test.want)
		}
	}
	for second := 0; second <= 8; second++ {
		want := 1
		if second == 8 {
			want = 0
		}
		if got := len(plan.Active(time.Duration(second) * time.Second)); got != want {
			t.Fatalf("at %ds: %d active, want %d", second, got, want)
		}
	}
}

func TestFrontRejectsTargetAndClockRangesBeforeConversion(t *testing.T) {
	for _, targets := range [][]Target{
		nil,
		make([]Target, 100001),
		{{At: maths.V(200000, 0, 0)}},
		{{At: maths.V(0, 0, 1e20)}},
		{{At: maths.V(math.MaxFloat64, 0, 0)}},
	} {
		_, err := New(frontSpec(), targets)
		if err == nil || !strings.Contains(err.Error(), "invalid prescribed ignition plan") {
			t.Fatalf("bad targets: %v", err)
		}
	}
	for _, change := range []func(*Spec){
		func(s *Spec) { s.Start = -25 * time.Hour },
		func(s *Spec) { s.Start = 25 * time.Hour },
		func(s *Spec) { s.Duration = 25 * time.Hour },
		func(s *Spec) { s.Direction = maths.V(math.MaxFloat64, 0, math.MaxFloat64) },
	} {
		spec := frontSpec()
		change(&spec)
		if _, err := New(spec, []Target{{}}); err == nil {
			t.Fatal("unbounded plan accepted")
		}
	}
}
