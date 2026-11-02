package mesh

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// rectangle is the solid angle a rectangle fills seen from a height
// straight above its middle: four of the corner's quarters, each the
// angle of a rectangle a half as long and a half as wide, seen from
// above a corner of it, which is atan(ab / (h * sqrt(a2 + b2 + h2))).
func rectangle(long, wide, height float64) float64 {
	a, b := long/2, wide/2
	return 4 * math.Atan(a*b/(height*math.Sqrt(a*a+b*b+height*height)))
}

// a triangle that is half a rectangle, cut along the diagonal through
// its middle, fills half the rectangle's angle seen from above the
// middle; seen from behind it is that angle, and from in front, less
// that: a point is inside the turn of the face it is behind.
func TestAFaceFillsTheAngleOfHalfItsRectangle(t *testing.T) {
	made := mustMesh(t, []maths.Vec{maths.V(-2, -1.5, 0), maths.V(2, -1.5, 0),
		maths.V(2, 1.5, 0)}, [][3]int{{0, 1, 2}}, nil, false)
	for _, each := range []struct{ height, sign float64 }{
		{2, -1}, {0.5, -1}, {7, -1}, {-2, 1}, {-0.5, 1}, {-7, 1}} {
		want := each.sign * rectangle(4, 3, math.Abs(each.height)) / 2
		got := made.solidAngle(0, maths.V(0, 0, each.height))
		if math.Abs(got-want) > 1e-12 {
			t.Errorf("at %v the face fills %v, not %v", each.height, got,
				want)
		}
	}
}

// regular is a solid with its corners and faces turned out.
func regular(t *testing.T, corners []maths.Vec, faces [][3]int) *Mesh {
	outward(corners, faces)
	return mustMesh(t, corners, faces, nil, false)
}

// tetrahedron is the regular one, its corners at alternate corners of
// the cube from minus one to one.
func tetrahedron(t *testing.T) *Mesh {
	return regular(t, []maths.Vec{maths.V(1, 1, 1), maths.V(1, -1, -1),
		maths.V(-1, 1, -1), maths.V(-1, -1, 1)},
		[][3]int{{0, 1, 2}, {0, 3, 1}, {0, 2, 3}, {1, 3, 2}})
}

// octahedron is the regular one, its corners a unit out on each axis.
func octahedron(t *testing.T) *Mesh {
	corners := []maths.Vec{maths.V(1, 0, 0), maths.V(-1, 0, 0),
		maths.V(0, 1, 0), maths.V(0, -1, 0), maths.V(0, 0, 1),
		maths.V(0, 0, -1)}
	var faces [][3]int
	for x := range 2 {
		for y := range 2 {
			for z := range 2 {
				faces = append(faces, [3]int{x, 2 + y, 4 + z})
			}
		}
	}
	return regular(t, corners, faces)
}

// seen from its middle, each of a regular solid's faces fills an equal
// share of the whole sphere's four pi: a quarter for the tetrahedron's
// four, an eighth for the octahedron's eight.
func TestRegularFacesFillEqualSharesFromTheMiddle(t *testing.T) {
	for name, each := range map[string]struct {
		made  *Mesh
		share float64
	}{"tetrahedron": {tetrahedron(t), math.Pi},
		"octahedron": {octahedron(t), math.Pi / 2}} {
		for i := range each.made.faces {
			got := each.made.solidAngle(i, maths.Vec{})
			if math.Abs(got-each.share) > 1e-12 {
				t.Errorf("%s face %d fills %v, not %v", name, i, got,
					each.share)
			}
		}
	}
}

// the faces of a closed solid fill the whole sphere, four pi, from any
// point inside it, and nothing from any point outside, where the near
// faces' angles are taken from the far ones': whatever the shape and
// wherever the point, asymmetric as the tetrahedron with corners at
// the origin and 4, 3 and 2 out on the axes.
func TestTheFacesOfASolidFillTheSphereOnlyFromInside(t *testing.T) {
	skew := regular(t, []maths.Vec{maths.V(0, 0, 0), maths.V(4, 0, 0),
		maths.V(0, 3, 0), maths.V(0, 0, 2)},
		[][3]int{{0, 2, 1}, {0, 1, 3}, {0, 3, 2}, {1, 2, 3}})
	for _, each := range []struct {
		made   *Mesh
		point  maths.Vec
		inside bool
	}{
		{skew, maths.V(0.5, 0.5, 0.5), true},
		{skew, maths.V(1, 1, 0.01), true},
		{skew, maths.V(0.01, 0.01, 0.01), true},
		{skew, maths.V(2, 0.5, 0.2), true},
		{skew, maths.V(0.1, 2.2, 0.1), true},
		{skew, maths.V(1, 1, -0.01), false},
		{skew, maths.V(5, 5, 5), false},
		{skew, maths.V(-0.01, 0.5, 0.5), false},
		{skew, maths.V(3, 2, 1), false},
		{skew, maths.V(0.5, 0.5, 2), false},
		{tetrahedron(t), maths.V(0, 0, 0), true},
		{tetrahedron(t), maths.V(0.3, 0.2, -0.1), true},
		{tetrahedron(t), maths.V(1, 1, 1.01), false},
		{octahedron(t), maths.V(0.2, -0.3, 0.4), true},
		{octahedron(t), maths.V(0.4, 0.4, 0.4), false},
		{octahedron(t), maths.V(0, 0, -1.1), false},
	} {
		want := 0.0
		if each.inside {
			want = 4 * math.Pi
		}
		got := 0.0
		for i := range each.made.faces {
			got += each.made.solidAngle(i, each.point)
		}
		if math.Abs(got-want) > 1e-12 {
			t.Errorf("from %v the faces fill %v, not %v", each.point, got,
				want)
		}
		if each.made.Inside(each.point) != each.inside {
			t.Errorf("%v is inside: %v", each.point, !each.inside)
		}
	}
}

// a solid with a face missing has no inside, though its other faces
// fill more than half the sphere from the middle: ten of the cube's
// twelve fill ten thirds of pi.
func TestAnOpenSolidHasNoInside(t *testing.T) {
	open := mustMesh(t, cubeCorners(), cubeFaces()[2:], nil, false)
	sum := 0.0
	for i := range open.faces {
		sum += open.solidAngle(i, maths.Vec{})
	}
	if math.Abs(sum-10*math.Pi/3) > 1e-12 {
		t.Fatalf("ten faces fill %v", sum)
	}
	if open.Inside(maths.Vec{}) {
		t.Error("an open mesh has an inside")
	}
}

// a point's distance less than zero is how deep it is: in the skew tetrahedron,
// the nearest side to a point is the one it is least far from, signed by which
// side of the surface it is on.
func TestDistanceIsBelowNoughtDeepInsideASolid(t *testing.T) {
	skew := regular(t, []maths.Vec{maths.V(0, 0, 0), maths.V(4, 0, 0),
		maths.V(0, 3, 0), maths.V(0, 0, 2)},
		[][3]int{{0, 2, 1}, {0, 1, 3}, {0, 3, 2}, {1, 2, 3}})
	for _, each := range []struct {
		point maths.Vec
		want  float64
	}{
		{maths.V(1, 1, 0.25), -0.25},
		{maths.V(0.1, 1, 1), -0.1},
		{maths.V(1, 1, -0.5), 0.5},
		{maths.V(-2, 1, 1), 2},
	} {
		if got := skew.Distance(each.point); math.Abs(got-each.want) >
			1e-12 {
			t.Errorf("at %v the distance is %v, not %v", each.point, got,
				each.want)
		}
	}
}
