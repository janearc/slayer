package firefront

import (
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
	"time"
)

func frontSpec() Spec {
	return Spec{Origin: maths.V(10, 0, 20), Direction: maths.XAxis, Speed: 2, Start: time.Second, Duration: 3 * time.Second, Cell: 8}
}

func TestAFrontSchedulesDistanceAndBoundsItsGroups(t *testing.T) {
	spec := frontSpec()
	targets := []Target{{At: maths.V(10, 1, 20), Seed: 1}, {At: maths.V(14, 2, 20), Seed: 2}, {At: maths.V(18, 3, 20), Seed: 3}}
	plan, err := New(spec, targets)
	if err != nil {
		t.Fatal(err)
	}
	groups := plan.Groups()
	count := 0
	times := map[uint64]time.Duration{}
	for _, group := range groups {
		if len(group) > 64 {
			t.Fatal("group exceeds GPU field")
		}
		for _, event := range group {
			times[event.Seed] = event.Start
			count++
			if event.Duration != 3*time.Second {
				t.Fatal("duration lost")
			}
		}
	}
	if count != 3 || times[1] != time.Second || times[2] != 3*time.Second || times[3] != 5*time.Second {
		t.Fatal("front did not use distance/speed")
	}
	targets[0].At.X = 999
	groups[0][0].At.X = 888
	if plan.Groups()[0][0].At.X == 888 || plan.Groups()[0][0].At.X == 999 {
		t.Fatal("caller changed plan")
	}
	active := plan.Active(3 * time.Second)
	if len(active) != 2 || active[0].Age != 2*time.Second || active[1].Age != 0 {
		t.Fatalf("wrong active fuel at boundary: %+v", active)
	}
	if got := plan.Active(8 * time.Second); len(got) != 0 {
		t.Fatal("fuel never burns out")
	}
}

func TestFrontGroupingIsDeterministicAndLimitsSimultaneousVolumes(t *testing.T) {
	spec := frontSpec()
	spec.Direction = maths.V(3, 0, 4)
	var targets []Target
	for i := range 400 {
		targets = append(targets, Target{At: maths.V(float64(i%20), 0, float64(i/20)), Seed: uint64(i)})
	}
	a, err := New(spec, targets)
	if err != nil {
		t.Fatal(err)
	}
	b, err := New(spec, targets)
	if err != nil {
		t.Fatal(err)
	}
	flat := 0
	for i, group := range a.Groups() {
		if len(group) > 64 {
			t.Fatal("dense tile not subdivided")
		}
		for j, event := range group {
			flat++
			if event != b.Groups()[i][j] {
				t.Fatal("non-deterministic group order")
			}
		}
	}
	if flat != 400 || a.PeakGroups() < 1 || a.PeakGroups() > len(a.Groups()) {
		t.Fatal("bad volume budget")
	}
}

func TestFrontRejectsUnboundedOrNonphysicalPlans(t *testing.T) {
	for _, change := range []func(*Spec){
		func(s *Spec) { s.Speed = 0 }, func(s *Spec) { s.Speed = math.Inf(1) },
		func(s *Spec) { s.Direction = maths.Vec{} }, func(s *Spec) { s.Direction = maths.YAxis },
		func(s *Spec) { s.Cell = 0 }, func(s *Spec) { s.Duration = 0 },
		func(s *Spec) { s.Origin.X = math.NaN() },
	} {
		spec := frontSpec()
		change(&spec)
		if _, err := New(spec, []Target{{At: maths.Vec{}}}); err == nil {
			t.Fatal("invalid front accepted")
		}
	}
	if _, err := New(frontSpec(), []Target{{At: maths.V(math.Inf(1), 0, 0)}}); err == nil {
		t.Fatal("infinite target accepted")
	}
}
