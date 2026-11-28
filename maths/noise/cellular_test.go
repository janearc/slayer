package noise

import (
	"math"
	"testing"
)

// fieldAround is the steps from a place to every point of a seed's field
// in a block of lattice cells, six either way of the place's own, which is
// as far as any point near enough to matter is.
func fieldAround(seed uint64, at [3]float64) []step {
	var steps []step
	for ix := int64(-6); ix <= 6; ix++ {
		for iy := int64(-6); iy <= 6; iy++ {
			for iz := int64(-6); iz <= 6; iz++ {
				steps = append(steps, pointOf(seed, [3]int64{
					int64(math.Floor(at[0])) + ix,
					int64(math.Floor(at[1])) + iy,
					int64(math.Floor(at[2])) + iz}, at))
			}
		}
	}
	return steps
}

// The nearest two points of a field, found by Cellular, are the nearest
// two found by looking at every point in a block of lattice cells round
// it, and the border is the least distance from the point to the plane
// midway between the nearest point and any other, which plain geometry
// works out from every point of the block. They agree at all but a few
// points, where a point three cells off was nearer than the ones looked
// at; the nearest alone is found at all of them.
func TestCellularFindsTheNearestPointsAndTheBorderOfItsCell(t *testing.T) {
	const seed = 11
	wrong := 0
	for index := range int64(400) {
		at := [3]float64{7*Hash(1, index) - 3.5, 7*Hash(2, index) - 3.5,
			7*Hash(3, index) - 3.5}
		found := Cellular(seed, at[0], at[1], at[2])
		steps := fieldAround(seed, at)
		nearest, next := 0, math.Inf(1)
		for each := range steps {
			if steps[each].long() < steps[nearest].long() {
				nearest = each
			}
		}
		border := math.Inf(1)
		for each := range steps {
			if each != nearest {
				next = math.Min(next, steps[each].long())
				border = math.Min(border, steps[nearest].midway(steps[each]))
			}
		}
		if math.Abs(found.First-steps[nearest].long()) > 1e-9 {
			t.Fatalf("at %v: %v, want %v", at, found.First,
				steps[nearest].long())
		}
		if math.Abs(found.Second-next) > 1e-9 ||
			math.Abs(found.Border-border) > 1e-9 {
			wrong++
		}
		if found.Border < 0 {
			t.Fatalf("at %v the border is %v", at, found.Border)
		}
	}
	if wrong > 4 {
		t.Errorf("%d of 400 second and border were not found", wrong)
	}
}

// The border is a distance: a step of a thousandth of a cell, within a
// cell, moves it by no more than that, wherever the points are, and so a
// seam drawn by it is as wide as it is told to be. The edge, the
// difference of the two distances, is not: it moves slowly where the two
// points are near each other, so a seam drawn by it is wide and soft, and
// is the reason for the border. The anti-test: some step moves the edge
// by under a tenth of the step, which no step moves the border by.
func TestCellularBorderIsADistanceAndTheEdgeIsNot(t *testing.T) {
	const move = 1e-3
	slow := 0
	for index := range int64(3000) {
		at := [3]float64{9 * Hash(7, index), 9 * Hash(8, index),
			9 * Hash(9, index)}
		found := Cellular(5, at[0], at[1], at[2])
		toward := Cellular(5, at[0], at[1], at[2]+move)
		if toward.Identity != found.Identity {
			continue
		}
		if math.Abs(toward.Border-found.Border) > move*(1+1e-6) {
			t.Fatalf("a step of %v moved the border by %v", move,
				toward.Border-found.Border)
		}
		if math.Abs(toward.Edge()-found.Edge()) < 0.1*move {
			slow++
		}
	}
	if slow == 0 {
		t.Errorf("the edge moved by a tenth of the step at no point tried")
	}
}

// The cells are of many shapes and all over: the border is under a twentieth
// of a cell at some points and over a quarter at others, so the cells are
// not one size or a grid; and the anti-test, that a point is not a place,
// is NaN, and not a number that looks like a cell.
func TestCellularHasEdgesAndMiddlesAndRefusesNoPlace(t *testing.T) {
	var near, far int
	for index := range int64(5000) {
		found := Cellular(5, 9*Hash(4, index), 9*Hash(5, index),
			9*Hash(6, index))
		if found.Border < 0.05 {
			near++
		}
		if found.Border > 0.25 {
			far++
		}
		if found.Identity < 0 || found.Identity >= 1 {
			t.Fatalf("an identity of %v", found.Identity)
		}
	}
	if near < 500 || far < 100 {
		t.Errorf("%d points near an edge and %d far, of 5000", near, far)
	}
	for _, nowhere := range []float64{math.NaN(), math.Inf(1)} {
		if found := Cellular(5, nowhere, 0, 0); !math.IsNaN(found.First) ||
			!math.IsNaN(found.Border) || !math.IsNaN(found.Identity) {
			t.Errorf("a point at %v is %+v", nowhere, found)
		}
	}
}

// A point in a cell has that cell's identity, the same at every point
// of it, and two seeds make two fields.
func TestCellularIdentityBelongsToTheCell(t *testing.T) {
	first := Cellular(5, 0.3, 0.3, 0.3)
	near := Cellular(5, 0.3+1e-9, 0.3, 0.3)
	if first.Identity != near.Identity {
		t.Errorf("a step of a billionth changed the cell")
	}
	if other := Cellular(6, 0.3, 0.3, 0.3); other.First == first.First {
		t.Errorf("two seeds are one field")
	}
}
