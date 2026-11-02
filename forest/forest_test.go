package forest

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestTheGroundHasTwoTrianglesToACell(t *testing.T) {
	ground, err := Ground(5, 10)
	if err != nil {
		t.Fatal(err)
	}
	points := ground.Points()
	if ground.FaceCount() != 50 || len(points) != 36 {
		t.Errorf("%d faces and %d vertices, want 50 and 36",
			ground.FaceCount(), len(points))
	}
	if ground.Smooth() {
		t.Errorf("the ground is smooth, and it is flat")
	}
	far := points[len(points)-1]
	if far.X != 10 || far.Y != 0 || far.Z != 10 {
		t.Errorf("the far corner is %v, want 10, 0, 10", far)
	}
}

func TestTheGroundRefusesWhatItCannotBe(t *testing.T) {
	cases := map[string][2]float64{"no cells": {0, 1}, "no size": {4, 0},
		"a size below nought":  {4, -1},
		"a size of nan":        {4, math.NaN()},
		"an endless size":      {4, math.Inf(1)},
		"a size endless below": {4, math.Inf(-1)},
		"too many cells":       {mostSide + 1, 1}}
	for name, one := range cases {
		if _, err := Ground(int(one[0]), one[1]); err != errGround {
			t.Errorf("a ground of %s: refused with %v", name, err)
		}
	}
}

func TestTheSameSeedMakesTheSameForest(t *testing.T) {
	one, _ := Placements(7, 20, 2, 0.8)
	again, _ := Placements(7, 20, 2, 0.8)
	other, _ := Placements(8, 20, 2, 0.8)
	same := 0
	for index := range one {
		if one[index] != again[index] {
			t.Fatalf("place %d differs for one seed", index)
		}
		if one[index] == other[index] {
			same++
		}
	}
	if len(one) != 400 || same == len(one) {
		t.Errorf("%d places, %d the same under another seed", len(one),
			same)
	}
}

func TestEveryPlaceStaysNearItsGridPoint(t *testing.T) {
	spacing, jitter := 2.0, 0.8
	placed, _ := Placements(3, 12, spacing, jitter)
	for index, one := range placed {
		column, row := float64(index%12), float64(index/12)
		off := math.Max(math.Abs(one.At().X-column*spacing),
			math.Abs(one.At().Z-row*spacing))
		if off > jitter*spacing/2+1e-9 || one.At().Y != 0 {
			t.Errorf("place %d is %v off, at %v", index, off,
				one.At())
		}
		if one.Size() < 0.8 || one.Size() > 1.2 || one.Turn() < 0 ||
			one.Turn() >= 2*math.Pi {
			t.Errorf("place %d has size %v and turn %v", index,
				one.Size(), one.Turn())
		}
	}
}

func TestPlacementsRefuseWhatTheyCannotBe(t *testing.T) {
	for _, bad := range [][4]float64{{0, 1, 0.5, 0}, {4, 0, 0.5, 0},
		{4, 1, -1, 0}, {4, 1, 2, 0}, {4, math.NaN(), 0.5, 0},
		{4, math.Inf(1), 0.5, 0}, {4, 1, math.NaN(), 0},
		{mostSide + 1, 1, 0.5, 0}} {
		_, err := Placements(1, int(bad[0]), bad[1], bad[2])
		if err != errPlaces {
			t.Errorf("placements of %v were not refused", bad)
		}
	}
}

func TestEveryVertexOfTheGroundIsOnItsGrid(t *testing.T) {
	side, size := 4, 8.0
	ground, _ := Ground(side, size)
	cell := size / float64(side)
	for index, at := range ground.Points() {
		column, row := float64(index%(side+1)), float64(index/(side+1))
		if math.Abs(at.X-column*cell) > 1e-12 || at.Y != 0 ||
			math.Abs(at.Z-row*cell) > 1e-12 {
			t.Errorf("vertex %d is %v, want %v, 0, %v", index, at,
				column*cell, row*cell)
		}
	}
}

// each cell is two triangles, together the cell, every one facing up
func TestEveryTriangleOfTheGroundIsHalfACellAndFacesUp(t *testing.T) {
	side, size := 4, 8.0
	ground, _ := Ground(side, size)
	cell := size / float64(side)
	for index := range ground.FaceCount() {
		corner, b, c := ground.Corners(index)
		first, second := b.Sub(corner), c.Sub(corner)
		turn := first.Cross(second)
		if math.Abs(turn.Len()/2-cell*cell/2) > 1e-9 {
			t.Errorf("face %d has area %v, want %v", index,
				turn.Len()/2, cell*cell/2)
		}
		if turn.Y <= 0 || turn.X != 0 || turn.Z != 0 {
			t.Errorf("face %d faces %v, want up", index, turn)
		}
	}
}

func TestTheLargestGroundAndForestAreAccepted(t *testing.T) {
	if _, err := Ground(mostSide, 1); err != nil {
		t.Errorf("the largest ground was refused: %v", err)
	}
	if _, err := Placements(1, mostSide, 1, 0.5); err != nil {
		t.Errorf("the largest forest was refused: %v", err)
	}
}

func TestTheSmallestGroundAndForestAreAccepted(t *testing.T) {
	ground, err := Ground(1, 0.001)
	if err != nil || ground.FaceCount() != 2 {
		t.Errorf("a ground of one cell: %v, %v", ground, err)
	}
	for _, jitter := range []float64{0, 1} {
		placed, err := Placements(1, 1, 0.001, jitter)
		if err != nil || len(placed) != 1 {
			t.Errorf("one place, jitter %v: %v, %v", jitter, placed,
				err)
		}
	}
}

func TestWithNoJitterEveryPlaceIsOnItsGridPoint(t *testing.T) {
	side, spacing := 5, 3.0
	placed, _ := Placements(9, side, spacing, 0)
	for index, one := range placed {
		want := maths.V(float64(index%side)*spacing, 0,
			float64(index/side)*spacing)
		if one.At() != want {
			t.Errorf("place %d is %v, not %v", index, one.At(),
				want)
		}
	}
}

// the forest of one seed is pinned, so that the same seed is the same
// forest from one version to the next
func TestOneSeedMakesTheForestItAlwaysDid(t *testing.T) {
	placed, _ := Placements(7, 4, 2, 0.8)
	pinned := map[int]placement{
		0: {maths.V(-0.7794463948964258, 0, 0.3246799058952702),
			1.1138703531632872, 0.9405313864944742},
		5: {maths.V(2.114853131213434, 0, 1.6731366547982502),
			2.7826257854749428, 1.1876137780007743},
	}
	for index, want := range pinned {
		got := placed[index]
		if math.Abs(got.At().X-want.At().X) > 1e-12 ||
			math.Abs(got.At().Z-want.At().Z) > 1e-12 ||
			math.Abs(got.Turn()-want.Turn()) > 1e-12 ||
			math.Abs(got.Size()-want.Size()) > 1e-12 {
			t.Errorf("place %d is %v, was %v", index, got, want)
		}
	}
}
