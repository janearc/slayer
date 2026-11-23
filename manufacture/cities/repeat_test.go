package cities

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// unitBox is a unit of one box a metre wide, standing a fifth of a
// metre from the start of its space, of a height the test gives, so a
// run of them shows where each one went and which way it faces.
func unitBox(height float64) unit {
	return func(int) ([]placedElement, error) {
		return []placedElement{put("unit", Box{1, height, 1}, plainLook(),
			maths.V(0.2, 0, 0))}, nil
	}
}

// rendered is a placed part drawn, with its place taken out through the
// part's own, so that its corners are where the run put them.
func rendered(t *testing.T, part placedElement) Rendered {
	t.Helper()
	one, err := part.Element.Rendered("quick")
	if err != nil {
		t.Fatal(err)
	}
	one.Place = one.Place.Within(part.Place)
	return one
}

// runOf is the parts of a run, or the test's end.
func runOf(t *testing.T, repeat repeat) []placedElement {
	t.Helper()
	parts, err := repeat.parts()
	if err != nil {
		t.Fatal(err)
	}
	return parts
}

func TestARunPlacesItsUnitCountTimesAtItsSpacing(t *testing.T) {
	parts := runOf(t, repeat{Count: 5, Spacing: 2, Axis: maths.XAxis,
		Unit: unitBox(1)})
	if len(parts) != 5 {
		t.Fatalf("%d parts, want 5", len(parts))
	}
	for index, part := range parts {
		low, _ := reach(rendered(t, part))
		want := 0.2 + 2*float64(index)
		if math.Abs(low.X-want) > tolerance {
			t.Errorf("unit %d starts at %v, want %v", index, low.X, want)
		}
	}
}

func TestARunAlongTheAxisUpStacksStoreys(t *testing.T) {
	parts := runOf(t, repeat{Count: 3, Spacing: 3, Axis: maths.YAxis,
		Unit: unitBox(2)})
	_, high := reach(rendered(t, parts[2]))
	if math.Abs(high.Y-8) > tolerance {
		t.Errorf("the third storey tops out at %v, want 8", high.Y)
	}
}

func TestHandingMirrorsEveryOtherUnit(t *testing.T) {
	parts := runOf(t, repeat{Count: 4, Spacing: 2, Axis: maths.XAxis,
		Handed: true, Unit: unitBox(1)})
	for index, part := range parts {
		low, high := reach(rendered(t, part))
		start := 0.2 + 2*float64(index)
		if index%2 == 1 {
			start = 2*float64(index) + 2 - 1.2
		}
		if math.Abs(low.X-start) > tolerance ||
			math.Abs(high.X-low.X-1) > tolerance {
			t.Errorf("unit %d spans %v to %v, want %v wide from %v", index,
				low.X, high.X, 1.0, start)
		}
	}
}

func TestAReflectedUnitStillFacesOutward(t *testing.T) {
	parts := runOf(t, repeat{Count: 2, Spacing: 2, Axis: maths.XAxis,
		Handed: true, Unit: unitBox(1)})
	for index, part := range parts {
		if got := volume(rendered(t, part).Mesh); math.Abs(got-1) >
			tolerance {
			t.Errorf("unit %d encloses %v, want 1 outward", index, got)
		}
	}
}

func TestARunWithoutHandingMirrorsNothing(t *testing.T) {
	parts := runOf(t, repeat{Count: 2, Spacing: 2, Axis: maths.XAxis,
		Unit: unitBox(1)})
	low, _ := reach(rendered(t, parts[1]))
	if math.Abs(low.X-2.2) > tolerance {
		t.Errorf("the second unit starts at %v, want 2.2", low.X)
	}
}

func TestTheEndUnitsOfARunMayDiffer(t *testing.T) {
	parts := runOf(t, repeat{Count: 4, Spacing: 2, Axis: maths.XAxis,
		Unit: unitBox(1), First: unitBox(5), Last: unitBox(3)})
	heights := make([]float64, len(parts))
	for index, part := range parts {
		_, high := reach(rendered(t, part))
		heights[index] = high.Y
	}
	want := []float64{5, 1, 1, 3}
	for index := range want {
		if math.Abs(heights[index]-want[index]) > tolerance {
			t.Errorf("heights %v, want %v", heights, want)
			break
		}
	}
}

func TestARunOfOneIsItsFirstUnit(t *testing.T) {
	parts := runOf(t, repeat{Count: 1, Spacing: 2, Axis: maths.XAxis,
		Unit: unitBox(1), First: unitBox(5), Last: unitBox(3)})
	if _, high := reach(rendered(t, parts[0])); high.Y != 5 {
		t.Errorf("a run of one is %v high, want the first unit's 5", high.Y)
	}
}

func TestARunThatCannotBeMadeIsRefused(t *testing.T) {
	for name, repeat := range map[string]repeat{
		"no units":   {Count: 0, Spacing: 1, Unit: unitBox(1)},
		"no spacing": {Count: 2, Spacing: 0, Unit: unitBox(1)},
		"no unit":    {Count: 2, Spacing: 1},
	} {
		if _, err := repeat.parts(); err == nil {
			t.Errorf("%s: a run was made", name)
		}
	}
}

func TestAUnitThatFailsFailsTheRun(t *testing.T) {
	failing := func(int) ([]placedElement, error) { return nil, errUnit }
	repeat := repeat{Count: 2, Spacing: 1, Axis: maths.XAxis, Unit: failing}
	if _, err := repeat.parts(); err == nil {
		t.Error("a run of failing units was made")
	}
}
