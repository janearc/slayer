package manufacture

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestCrownUsesItsSixExtremaAndClosesEveryFaceOutward(t *testing.T) {
	box := maths.Box{Min: maths.V(2, 3, 4), Max: maths.V(6, 9, 12)}
	faces := (crownShape{box}).skin(cut{})
	if len(faces) != 8 {
		t.Fatalf("crown has %d faces", len(faces))
	}
	expected := map[maths.Vec]bool{
		maths.V(4, 3, 8): false, maths.V(4, 9, 8): false,
		maths.V(6, 6, 8): false, maths.V(4, 6, 4): false,
		maths.V(2, 6, 8): false, maths.V(4, 6, 12): false,
	}
	edges := map[[2]maths.Vec]int{}
	for _, face := range faces {
		middle := face[0].Add(face[1]).Add(face[2]).Scale(1.0 / 3)
		normal := face[1].Sub(face[0]).Cross(face[2].Sub(face[0]))
		if normal.Dot(middle.Sub(maths.V(4, 6, 8))) <= 0 {
			t.Fatal("crown has an inward or degenerate face")
		}
		for i, point := range face {
			if _, found := expected[point]; !found {
				t.Fatalf("unexpected crown vertex %v", point)
			}
			expected[point] = true
			edges[[2]maths.Vec{point, face[(i+1)%3]}]++
		}
	}
	for point, seen := range expected {
		if !seen {
			t.Fatalf("missing crown extremum %v", point)
		}
	}
	for edge, count := range edges {
		if count != 1 || edges[[2]maths.Vec{edge[1], edge[0]}] != 1 {
			t.Fatal("crown has an open or multiply covered edge")
		}
	}
}
