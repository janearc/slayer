package cities

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// tolerance is how near two sums of floating-point products must be to be
// the same, in the units of the sum.
const tolerance = 1e-9

// volume is the volume a mesh encloses, by the divergence theorem: a
// positive sum when its faces are wound outward and a negative one when
// they are wound inward.
func volume(mesh *Mesh) float64 {
	var sum float64
	for _, triangle := range mesh.Triangles {
		sum += triangle[0].Dot(triangle[1].Cross(triangle[2])) / 6
	}
	return sum
}

// edge is one side of a triangle, from one corner to the next.
type edge struct{ from, to maths.Vec }

// closed is whether every edge of the mesh is walked once, and once the
// other way by the face next to it.
func closed(mesh *Mesh) bool {
	walked := map[edge]int{}
	for _, triangle := range mesh.Triangles {
		for index, corner := range triangle {
			walked[edge{corner, triangle[(index+1)%len(triangle)]}]++
		}
	}
	for side, count := range walked {
		if count != 1 || walked[edge{side.to, side.from}] != 1 {
			return false
		}
	}
	return true
}

// reversed is the mesh turned inside out.
func reversed(mesh *Mesh) *Mesh {
	turned := &Mesh{}
	for _, triangle := range mesh.Triangles {
		turned.Triangles = append(turned.Triangles,
			Triangle{triangle[0], triangle[2], triangle[1]})
	}
	return turned
}

func TestAPrismOfACircleIsClosedAndWoundOutward(t *testing.T) {
	const radius, height = 2.0, 3.0
	mesh, err := prism{outlineCircle{radius}, outlineCircle{radius}, height}.Mesh("quick")
	if err != nil {
		t.Fatal(err)
	}
	sides, err := sidesAt("quick")
	if err != nil {
		t.Fatal(err)
	}
	count := float64(sides)
	area := count * radius * radius * math.Sin(fullTurn/count) / 2
	if got := volume(mesh); math.Abs(got-area*height) > tolerance {
		t.Errorf("volume %v, want %v", got, area*height)
	}
	if !closed(mesh) {
		t.Error("the prism's walls and caps do not close")
	}
}

func TestTheMeasuresCanTellAnInsideOutPrism(t *testing.T) {
	mesh, err := prism{outlineCircle{1}, outlineCircle{1}, 1}.Mesh("quick")
	if err != nil {
		t.Fatal(err)
	}
	if volume(reversed(mesh)) >= 0 {
		t.Error("a prism turned inside out still has a positive volume")
	}
}

func TestATaperedPrismLeansInAndIsClosed(t *testing.T) {
	mesh, err := prism{outlineCircle{3}, outlineCircle{1}, 1}.Mesh("quick")
	if err != nil {
		t.Fatal(err)
	}
	sides, _ := sidesAt("quick")
	for _, triangle := range mesh.Triangles[:2*sides] {
		normal := triangle[1].Sub(triangle[0]).Cross(triangle[2].Sub(triangle[0]))
		if normal.Y <= 0 {
			t.Fatalf("a wall of a taper faces down: %v", normal)
		}
	}
	if !closed(mesh) || volume(mesh) <= 0 {
		t.Errorf("the taper is not a closed outward solid: %v", volume(mesh))
	}
}

func TestABoxIsAPrismOfARectangleOfTwelveFaces(t *testing.T) {
	mesh, err := Box{2, 3, 5}.Mesh("quick")
	if err != nil {
		t.Fatal(err)
	}
	if len(mesh.Triangles) != 12 || !closed(mesh) {
		t.Errorf("%d triangles, closed %v", len(mesh.Triangles), closed(mesh))
	}
	if got := volume(mesh); math.Abs(got-30) > tolerance {
		t.Errorf("volume %v, want 30", got)
	}
}

func TestABoxWithNoSizeIsRefused(t *testing.T) {
	for _, box := range []Box{{0, 1, 1}, {1, -1, 1}, {1, 1, math.NaN()}} {
		if _, err := box.Mesh("quick"); err == nil {
			t.Errorf("%v was meshed", box)
		}
	}
}

func TestAConcaveOutlineIsRefused(t *testing.T) {
	arrow := polygon{[]outlineCorner{{0, 0}, {2, 1}, {0, 2}, {1, 1}}}
	if _, err := (prism{arrow, arrow, 1}).Mesh("quick"); err == nil {
		t.Error("a prism with a concave cap was meshed")
	}
}

func TestAConvexPolygonIsMeshedFromItsCorners(t *testing.T) {
	triangle := polygon{[]outlineCorner{{0, 0}, {2, 0}, {0, 2}}}
	mesh, err := prism{triangle, triangle, 1}.Mesh("quick")
	if err != nil {
		t.Fatal(err)
	}
	if got := volume(mesh); math.Abs(got-2) > tolerance || !closed(mesh) {
		t.Errorf("volume %v, closed %v", got, closed(mesh))
	}
}

func TestPrismsThatCannotJoinAreRefused(t *testing.T) {
	square := polygon{[]outlineCorner{{0, 0}, {1, 0}, {1, 1}, {0, 1}}}
	triangle := polygon{[]outlineCorner{{0, 0}, {1, 0}, {0, 1}}}
	for name, prism := range map[string]prism{
		"no height":         {square, square, 0},
		"unequal corners":   {square, triangle, 1},
		"too few corners":   {polygon{triangle.Corners[:2]}, square, 1},
		"a circle of none":  {outlineCircle{0}, outlineCircle{1}, 1},
		"an unknown detail": {outlineCircle{1}, outlineCircle{1}, 1},
	} {
		level := Detail("quick")
		if name == "an unknown detail" {
			level = "nowhere"
		}
		if _, err := prism.Mesh(level); err == nil {
			t.Errorf("%s: meshed", name)
		}
	}
}
