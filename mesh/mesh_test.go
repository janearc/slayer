package mesh

import (
	"errors"
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
)

// what cannot be a mesh is refused, each for its reason: no faces, the
// wrong count of normals, a corner that is not there or not a point, a
// normal pointing nowhere, a face with no area, an edge two faces run
// the same way, and a closed cube turned inside out.
func TestWhatCannotBeAMeshIsRefused(t *testing.T) {
	corners, faces := cubeCorners(), cubeFaces()
	reversed := slices.Clone(faces)
	for i, at := range reversed {
		reversed[i][1], reversed[i][2] = at[2], at[1]
	}
	nowhere := slices.Clone(corners)
	nowhere[3] = maths.V(math.NaN(), 0, 0)
	flat := slices.Clone(cubeNormals())
	flat[2] = maths.Vec{}
	for name, made := range map[string]func() (*Mesh, error){
		"no faces": func() (*Mesh, error) {
			return New(corners, nil, nil, false)
		},
		"normals": func() (*Mesh, error) {
			return New(corners, faces, corners[:3], true)
		},
		"a corner not there": func() (*Mesh, error) {
			return New(corners, append(slices.Clone(faces),
				[3]int{0, 1, 8}), nil, false)
		},
		"a corner not a point": func() (*Mesh, error) {
			return New(nowhere, faces, nil, false)
		},
		"a normal nowhere": func() (*Mesh, error) {
			return New(corners, faces, flat, true)
		},
		"no area": func() (*Mesh, error) {
			return New(corners, [][3]int{{0, 1, 1}}, nil, false)
		},
		"an edge run twice": func() (*Mesh, error) {
			return New(corners, [][3]int{{0, 1, 3}, {0, 1, 2}}, nil, false)
		},
		"inside out": func() (*Mesh, error) {
			return New(corners, reversed, nil, false)
		},
	} {
		if _, err := made(); !errors.Is(err, ErrMesh) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// cubeNormals is a normal at each of the cube's corners, out from its
// middle.
func cubeNormals() []maths.Vec {
	corners := cubeCorners()
	for i, corner := range corners {
		corners[i] = corner.Unit()
	}
	return corners
}

// a cube and a ball of faces are closed; one face of a cube is not.
func TestAClosedMeshKnowsItIs(t *testing.T) {
	if !cube(t).Closed() {
		t.Error("the cube is not closed")
	}
	if made, _, _ := ball(t, 2); !made.Closed() {
		t.Error("the ball is not closed")
	}
	if mustMesh(t, cubeCorners(), cubeFaces()[:2], nil, false).Closed() {
		t.Error("one side of a cube is closed")
	}
}

// slabs is where a ray first meets the cube's box, by the slab
// arithmetic of a box, and whether it does: the true answer the mesh's
// faces are held to.
func slabs(ray maths.Ray) (float64, bool) {
	enter, leave := math.Inf(-1), math.Inf(1)
	for axis, at := range parts(ray.At) {
		way := parts(ray.Dir)[axis]
		low, high := (-0.5-at)/way, (0.5-at)/way
		enter = math.Max(enter, math.Min(low, high))
		leave = math.Min(leave, math.Max(low, high))
	}
	return enter, enter <= leave && enter > 0
}

// a ray from outside meets the cube where a box's slabs say, to a part
// in a hundred million million, facing the way of the side it lands on:
// ten thousand rays from all round toward points near it.
func TestARayMeetsTheCubeWhereItIs(t *testing.T) {
	made := cube(t)
	random := rand.New(rand.NewPCG(1, 2))
	for range 10000 {
		from := maths.V(random.NormFloat64(), random.NormFloat64(),
			random.NormFloat64()).Unit().Scale(3)
		toward := maths.V(random.Float64()-0.5, random.Float64()-0.5,
			random.Float64()-0.5).Scale(1.4)
		ray := maths.Ray{At: from, Dir: toward.Sub(from).Unit()}
		want, wanted := slabs(ray)
		met, got := made.Meet(ray, 0, math.Inf(1))
		if got != wanted {
			t.Fatalf("from %v toward %v: met %v, a box %v", from, toward,
				got, wanted)
		}
		if !got {
			continue
		}
		if math.Abs(met.Along()-want) > 1e-14*want {
			t.Fatalf("met at %v, the box at %v", met.Along(), want)
		}
		point := ray.Point(met.Along())
		normal := made.NormalAt(met)
		if math.Abs(normal.Dot(point)*2-maxPart(point)*2) > 1e-9 ||
			math.Abs(normal.Len()-1) > 1e-12 {
			t.Fatalf("at %v the cube faces %v", point, normal)
		}
	}
}

// maxPart is the largest of a point's parts by size.
func maxPart(point maths.Vec) float64 {
	return math.Max(math.Abs(point.X), math.Max(math.Abs(point.Y),
		math.Abs(point.Z)))
}

// no ray slips between faces: from inside a closed ball of faces,
// every ray leaves through a face, aimed exactly at a corner, at the
// middle of an edge, or anywhere at all, from the middle and from off
// it. a test that is not watertight loses some of these.
func TestNoRaySlipsBetweenFaces(t *testing.T) {
	made, corners, faces := ball(t, 3)
	random := rand.New(rand.NewPCG(3, 4))
	starts := []maths.Vec{maths.V(0, 0, 0), maths.V(0.1, -0.2, 0.05),
		maths.V(-0.3, 0.3, 0.2)}
	var aims []maths.Vec
	aims = append(aims, corners...)
	for _, at := range faces {
		aims = append(aims, corners[at[0]].Add(corners[at[1]]).Scale(0.5))
	}
	for range 20000 {
		aims = append(aims, maths.V(random.NormFloat64(),
			random.NormFloat64(), random.NormFloat64()))
	}
	missed := 0
	for _, from := range starts {
		for _, aim := range aims {
			ray := maths.Ray{At: from, Dir: aim.Sub(from).Unit()}
			if _, ok := made.Hit(ray, 0, math.Inf(1)); !ok {
				missed++
			}
		}
	}
	if missed > 0 {
		t.Errorf("%d of %d rays from inside slipped out", missed,
			len(starts)*len(aims))
	}
}

// a ball of faces round the unit sphere is met from outside between
// the sphere its flattest faces reach in to and the sphere itself.
func TestABallOfFacesIsMetAtTheSphere(t *testing.T) {
	made, corners, faces := ball(t, 3)
	inner := 1.0
	for _, at := range faces {
		middle := corners[at[0]].Add(corners[at[1]]).Add(corners[at[2]])
		inner = math.Min(inner, middle.Scale(1.0/3).Len())
	}
	random := rand.New(rand.NewPCG(5, 6))
	for range 5000 {
		from := maths.V(random.NormFloat64(), random.NormFloat64(),
			random.NormFloat64()).Unit().Scale(4)
		ray := maths.Ray{At: from, Dir: from.Neg().Unit()}
		along, ok := made.Hit(ray, 0, math.Inf(1))
		if !ok || along < 3-1e-12 || along > 4-inner+1e-12 {
			t.Fatalf("straight in from %v: %v at %v, inner %v", from, ok,
				along, inner)
		}
	}
}

// a corner is a point only if each part of it is a finite number: not
// nowhere, and not endlessly far either way, in any one of its parts.
func TestACornerIsAPointOnlyIfEveryPartIsFinite(t *testing.T) {
	for axis := range 3 {
		for _, bad := range []float64{math.NaN(), math.Inf(1),
			math.Inf(-1)} {
			corners := cubeCorners()
			part := parts(corners[3])
			part[axis] = bad
			corners[3] = vectorOf(part)
			if _, err := New(corners, cubeFaces(), nil, false); !errors.Is(
				err, ErrMesh) {
				t.Errorf("part %d of %v: %v", axis, bad, err)
			}
			if finite(corners[3]) {
				t.Errorf("part %d of %v is a point", axis, bad)
			}
		}
	}
	if !finite(maths.V(1e300, -1e300, 0)) {
		t.Error("a far point is not a point")
	}
}

// a mesh blends its corners' normals only if it is asked to and has
// them: asked without them, or given them and not asked, it faces
// each face's own way.
func TestASmoothMeshNeedsNormalsAndTheAsk(t *testing.T) {
	normals := []maths.Vec{maths.V(1, 0, 0), maths.V(0, 1, 0),
		maths.V(0, 0, 1), maths.V(0, 0, 1)}
	flat := [][3]int{{0, 1, 2}}
	for name, each := range map[string]struct {
		normals []maths.Vec
		smooth  bool
		want    bool
	}{"both": {normals, true, true}, "no normals": {nil, true, false},
		"no ask": {normals, false, false}, "neither": {nil, false, false}} {
		made := mustMesh(t, aSquare(0), flat, each.normals, each.smooth)
		if made.Smooth() != each.want {
			t.Errorf("%s: smooth is %v", name, made.Smooth())
		}
		if !each.want {
			got := made.facing(0, weights{0.2, 0.3, 0.5})
			if got != maths.V(0, 0, 1) {
				t.Errorf("%s: it faces %v", name, got)
			}
		}
	}
}

// the blend is each corner's normal by its weight, made one long: a
// fifth of the first, three tenths of the second and half the third,
// the normals along x, y and z, which is (0.2, 0.3, 0.5) over its
// length, the root of 0.38.
func TestTheBlendWeighsEachCornersNormalByItsOwnWeight(t *testing.T) {
	normals := []maths.Vec{maths.V(1, 0, 0), maths.V(0, 1, 0),
		maths.V(0, 0, 1), maths.V(0, 0, 1)}
	made := mustMesh(t, aSquare(0), [][3]int{{0, 1, 2}}, normals, true)
	long := math.Sqrt(0.38)
	want := maths.V(0.2/long, 0.3/long, 0.5/long)
	if got := made.facing(0, weights{0.2, 0.3, 0.5}); got.Sub(want).Len() >
		1e-15 {
		t.Errorf("the blend faces %v, not %v", got, want)
	}
}

// the mesh's bounds are the box round its corners, whatever the shape
// of the tree: the cube's, a tetrahedron's, and a mesh of two faces
// that is one leaf.
func TestTheBoundsAreTheBoxRoundTheCorners(t *testing.T) {
	skew := mustMesh(t, []maths.Vec{maths.V(0, 0, 0), maths.V(4, 0, 0),
		maths.V(0, 3, 0), maths.V(0, 0, 2)},
		[][3]int{{0, 2, 1}, {0, 1, 3}, {0, 3, 2}, {1, 2, 3}}, nil, false)
	two := mustMesh(t, aSquare(2), [][3]int{{0, 1, 2}, {0, 2, 3}}, nil,
		false)
	for name, each := range map[string]struct {
		made     *Mesh
		low, top maths.Vec
	}{"cube": {cube(t), maths.V(-0.5, -0.5, -0.5), maths.V(0.5, 0.5, 0.5)},
		"tetrahedron": {skew, maths.V(0, 0, 0), maths.V(4, 3, 2)},
		"two faces":   {two, maths.V(0, 0, 2), maths.V(1, 1, 2)}} {
		if got := each.made.Bounds(); got.Min != each.low ||
			got.Max != each.top {
			t.Errorf("%s: bounds %v to %v", name, got.Min, got.Max)
		}
	}
}
