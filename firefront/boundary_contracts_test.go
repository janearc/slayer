package firefront

import (
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
	"time"
)

func TestFrontChecksEveryTargetAndOriginComponent(t *testing.T) {
	for _, bad := range []float64{math.Inf(-1), math.Inf(1), math.NaN()} {
		for axis := range 3 {
			point := maths.Vec{}
			switch axis {
			case 0:
				point.X = bad
			case 1:
				point.Y = bad
			case 2:
				point.Z = bad
			}
			spec := frontSpec()
			spec.Origin = point
			if _, err := New(spec, []Target{{At: maths.Vec{}}}); err == nil {
				t.Fatalf("bad origin axis %d accepted", axis)
			}
			if _, err := New(frontSpec(), []Target{{At: point}}); err == nil {
				t.Fatalf("bad target axis %d accepted", axis)
			}
		}
	}
}

func TestFrontAcceptsExactTimeAndTargetLimits(t *testing.T) {
	for _, start := range []time.Duration{-24 * time.Hour, 24 * time.Hour} {
		for _, duration := range []time.Duration{time.Nanosecond, 24 * time.Hour} {
			spec := frontSpec()
			spec.Start = start
			spec.Duration = duration
			plan, err := New(spec, []Target{{At: spec.Origin}})
			if err != nil {
				t.Fatal(err)
			}
			event := plan.Groups()[0][0]
			if event.Start != start || event.Duration != duration {
				t.Fatal("time endpoint changed")
			}
		}
	}
	targets := make([]Target, 100000)
	for i := range targets {
		targets[i] = Target{At: frontSpec().Origin, Seed: uint64(i)}
	}
	plan, err := New(frontSpec(), targets)
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, group := range plan.Groups() {
		if len(group) > 64 {
			t.Fatal("group exceeded 64")
		}
		for _, event := range group {
			if event.Seed != uint64(seen) {
				t.Fatalf("lost or duplicated target %d", seen)
			}
			seen++
		}
	}
	if seen != len(targets) {
		t.Fatalf("lost targets: %d", seen)
	}
	if _, err := New(frontSpec(), append(targets, Target{})); err == nil {
		t.Fatal("100001 targets accepted")
	}
}

func TestFrontTileOrderUsesMetresAndBothCoordinates(t *testing.T) {
	spec := frontSpec()
	spec.Cell = .5
	spec.Origin = maths.Vec{}
	targets := []Target{
		{At: maths.V(.1, 0, .6), Seed: 4}, {At: maths.V(-.1, 0, .1), Seed: 2},
		{At: maths.V(.6, 0, 0), Seed: 5}, {At: maths.V(.1, 0, .1), Seed: 3},
		{At: maths.V(-.1, 0, -.1), Seed: 1},
	}
	plan, err := New(spec, targets)
	if err != nil {
		t.Fatal(err)
	}
	groups := plan.Groups()
	if len(groups) != 5 {
		t.Fatalf("half-metre tiles: %d groups", len(groups))
	}
	for i, group := range groups {
		if len(group) != 1 || group[0].Seed != uint64(i+1) {
			t.Fatalf("tile %d out of lexicographic order: %+v", i, group)
		}
	}
}

func TestFrontConversionBoundsAreClosedAndRefuseTheNextInterval(t *testing.T) {
	for _, axis := range []maths.Vec{maths.XAxis, maths.ZAxis} {
		for _, sign := range []float64{-1, 1} {
			for _, step := range []float64{0, 1} {
				spec := frontSpec()
				spec.Origin = maths.Vec{}
				spec.Speed = 1e9
				spec.Cell = 1
				_, err := New(spec, []Target{{At: axis.Scale(sign * (1e9 + step))}})
				if (err == nil) != (step == 0) {
					t.Fatalf("tile bound %v sign %g step %g: %v", axis, sign, step, err)
				}
			}
		}
	}
	for _, sign := range []float64{-1, 1} {
		for _, step := range []float64{0, .5} {
			spec := frontSpec()
			spec.Origin = maths.Vec{}
			spec.Speed = 1
			plan, err := New(spec, []Target{{At: maths.V(sign*(86400+step), 0, 0)}})
			if (err == nil) != (step == 0) {
				t.Fatalf("travel bound sign %g step %g: %v", sign, step, err)
			}
			if err == nil && plan.Groups()[0][0].Start != spec.Start+time.Duration(sign)*24*time.Hour {
				t.Fatal("travel endpoint changed")
			}
		}
	}
}
