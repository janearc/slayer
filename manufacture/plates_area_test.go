package manufacture

import (
	"fmt"
	"testing"

	"github.com/janearc/slayer/maths"
)

// harnessSine is the cutoff of shape.Mesh's check, written again here so
// that this test does not stand on the rule it tests. read:
// scene/shape/mesh.go, check.
const harnessSine = 1e-12

// platesSeeds is how many seeds, from zero, the plates are made from.
const platesSeeds = 64

// hasArea is whether three corners are a triangle shape.Mesh would
// accept, by its own rule: the cross of the edges, at the edges' own
// size, has a square above sineLeast squared times the edges' squares.
func hasArea(one, two, three maths.Vec) bool {
	first, second := two.Sub(one), three.Sub(one)
	across := first.Cross(second)
	return across.Dot(across) >
		harnessSine*harnessSine*first.Dot(first)*second.Dot(second)
}

// withoutArea is the places of the faces of a surface that have none.
func withoutArea(skin surface) []int {
	var bare []int
	for index, one := range skin {
		if !hasArea(one[0], one[1], one[2]) {
			bare = append(bare, index)
		}
	}
	return bare
}

// A slab or a chip, from any of the first sixty-four seeds, has no
// triangle without area.
func TestNoPlateHasATriangleWithoutArea(t *testing.T) {
	for name, row := range plateRows() {
		for number := range uint64(platesSeeds) {
			made, err := row.made(Sow(number))
			if err != nil {
				t.Fatalf("%s %d: %v", name, number, err)
			}
			skin := made.skins()[body]
			if bare := withoutArea(skin); len(bare) > 0 {
				t.Errorf("%s %d: faces %v have no area: %v",
					name, number, bare, fmt.Sprint(skin[bare[0]]))
			}
		}
	}
}

// Anti-test: the rule does find a triangle with a repeated corner, one with
// its corners in a line, and one a hair off a line, and lets a plain one by.
func TestAnAreaLessTriangleIsFound(t *testing.T) {
	origin, along := maths.V(0, 0, 0), maths.V(1, 0, 0)
	for name, skin := range map[string]surface{
		"repeated": {{origin, along, along}},
		"in line":  {{origin, along, maths.V(2, 0, 0)}},
		"a hair":   {{origin, along, maths.V(2, 1e-13, 0)}},
	} {
		if len(withoutArea(skin)) != 1 {
			t.Errorf("%s: a face without area was let by", name)
		}
	}
	plain := surface{{origin, along, maths.V(0, 1, 0)}}
	if len(withoutArea(plain)) != 0 {
		t.Error("a plain triangle was found to have no area")
	}
}
