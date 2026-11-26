package branching

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

func TestWhorlsPlaceAndSizeBranchesFromOrderedDraws(t *testing.T) {
	habit := Habit{Turn: measure.Radians(1), Start: .1, Top: .9, Fewest: 1,
		Reach: .5, Girth: .2, Low: measure.Degrees(90)}
	whorls, err := NewWhorls(2, 2, habit)
	if err != nil {
		t.Fatal(err)
	}
	draws := &ringDraws{}
	got, err := whorls.At(Size{Length: measure.Metres(10), Girth: measure.Metres(1)}, 0, 0, draws)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.Space.Origin.Y-3) > 1e-12 || math.Abs(got.Size.Length.InMetres()-5) > 1e-12 || math.Abs(got.Size.Girth.InMetres()-.14) > 1e-12 {
		t.Fatalf("unexpected placement: %+v", got)
	}
	if !reflect.DeepEqual(draws.calls, []int{0, 1, 2}) {
		t.Fatalf("draw order changed: %v", draws.calls)
	}
}

func TestWhorlsRejectInvalidCountsAndStem(t *testing.T) {
	habit := Habit{Start: .1, Top: .9, Fewest: 1, Reach: .5, Girth: .2}
	if _, err := NewWhorls(0, 1, habit); !errors.Is(err, ErrWhorls) {
		t.Fatalf("zero count: %v", err)
	}
	if _, err := NewWhorls(1, 0, habit); !errors.Is(err, ErrWhorls) {
		t.Fatalf("zero branch count: %v", err)
	}
	bad := habit
	bad.Reach = 0
	if _, err := NewWhorls(1, 1, bad); !errors.Is(err, ErrHabit) {
		t.Fatalf("bad habit: %v", err)
	}
	whorls, err := NewWhorls(1, 1, habit)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := whorls.At(Size{Length: measure.Metres(0), Girth: measure.Metres(1)}, 0, 0, &ringDraws{}); !errors.Is(err, ErrStem) {
		t.Fatalf("zero stem: %v", err)
	}
	whorls.Habit.Low = measure.Radians(math.NaN())
	if _, err := whorls.At(Size{Length: measure.Metres(1), Girth: measure.Metres(1)}, 0, 0, &ringDraws{}); !errors.Is(err, ErrNoRun) {
		t.Fatalf("invalid run: %v", err)
	}
	if _, err := SpaceAlong(maths.Vec{}, maths.Vec{}); !errors.Is(err, ErrNoRun) {
		t.Fatalf("zero direction: %v", err)
	}
}
