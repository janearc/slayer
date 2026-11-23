package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// orcaTestOpenEdges is how many edges of the faces have no edge running
// the other way along them: none for a closed surface.
func orcaTestOpenEdges(faces [][3]int) int {
	runs := map[[2]int]int{}
	for _, face := range faces {
		for corner := range face {
			runs[[2]int{face[corner], face[(corner+1)%3]}]++
		}
	}
	open := 0
	for edge, count := range runs {
		if runs[[2]int{edge[1], edge[0]}] != count {
			open++
		}
	}
	return open
}

// orcaTestLattice is a lattice of orcaTestSide samples a metre apart on
// each axis, with its lowest corner at the origin.
const orcaTestSide = 7

func orcaTestLattice() orcaLattice {
	return orcaLattice{step: 1, nx: orcaTestSide, ny: orcaTestSide,
		nz: orcaTestSide}
}

// orcaTestInside is a field that is inside at the lattice points given
// and outside at all others.
func orcaTestInside(inside ...maths.Vec) func(maths.Vec) float64 {
	return func(at maths.Vec) float64 {
		for _, point := range inside {
			if point.Sub(at).Len() < .5 {
				return -1
			}
		}
		return 1
	}
}

func TestTwoInsideSamplesTouchingAtAnEdgeContourClosed(t *testing.T) {
	lattice := orcaTestLattice()
	touching := orcaTestInside(maths.V(3, 3, 3), maths.V(4, 4, 3))
	values := lattice.sample(touching)
	if joined := lattice.joined(values); joined != 1 {
		t.Fatalf("%d faces joined, want the one the pair meets at", joined)
	}
	_, faces := orcaContour(touching, lattice)
	if open := orcaTestOpenEdges(faces); open != 0 {
		t.Fatalf("the contour has %d edges with none running back", open)
	}
}

func TestInsideSamplesApartAreLeftAlone(t *testing.T) {
	lattice := orcaTestLattice()
	apart := orcaTestInside(maths.V(2, 2, 3), maths.V(4, 4, 3))
	values := lattice.sample(apart)
	before := append([]float32(nil), values...)
	if joined := lattice.joined(values); joined != 0 {
		t.Fatalf("%d faces joined for samples that do not meet", joined)
	}
	for index := range values {
		if values[index] != before[index] {
			t.Fatalf("sample %d changed", index)
		}
	}
	_, faces := orcaContour(apart, lattice)
	if open := orcaTestOpenEdges(faces); open != 0 || len(faces) == 0 {
		t.Fatalf("%d open edges in %d faces", open, len(faces))
	}
}

func TestJoiningAFaceTurnsInsideTheOutsideCornerWithTheLowerField(t *testing.T) {
	const high, low = .5, .2
	values := []float32{-1, high, low, -1}
	if orcaJoinFace(values, 0, 1, 2) != 1 {
		t.Fatal("an alternating face was not joined")
	}
	if values[2] >= 0 || values[1] != high {
		t.Fatalf("samples %v: the lower outside corner is the one to join",
			values)
	}
	if math.Abs(float64(values[2])+orcaJoinDepth) > 1e-12 {
		t.Fatalf("the joined sample is %v, not just inside", values[2])
	}
}

func TestAFaceThatDoesNotAlternateIsNotJoined(t *testing.T) {
	for name, values := range map[string][]float32{
		"all inside":   {-1, -1, -1, -1},
		"all outside":  {1, 1, 1, 1},
		"an edge":      {-1, -1, 1, 1},
		"one corner":   {-1, 1, 1, 1},
		"three inside": {-1, -1, -1, 1},
	} {
		before := append([]float32(nil), values...)
		if orcaJoinFace(values, 0, 1, 2) != 0 {
			t.Errorf("%s: joined", name)
		}
		for index := range values {
			if values[index] != before[index] {
				t.Errorf("%s: sample %d changed", name, index)
			}
		}
	}
}
