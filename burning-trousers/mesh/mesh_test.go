// Package mesh holds slayer's own mesh (mesh) to scene/shape's Mesh, which
// was first written, by the pictures they make of the same faces. every kind
// of the manufactory, made from a seed on slayer's mesh, is drawn again
// with its faces copied into a shape.Mesh, from three sides, and the two
// pictures are compared.
// Shared groups are expanded into separate world objects, each using the
// reference mesh. Their placement and traversal do not use MeshGroup.
//
// a pixel's light in just-noticeable differences, its cover and its
// depth are all compared. the faces are the same, so a pixel that
// differs is a ray that met them differently.
//
// the pictures see only the faces an eye sees, so every face is also
// met by a ray of its own, from a hair outside its middle, on both
// meshes; a face either mesh dropped lets that ray go on past it. a
// face with another within a millionth of the diagonal along its
// normal, doubled or in a shell that thin, is met by the other, so a
// drop of it passes; no kind made today has one.
//
// what this cannot see: the shape.Mesh copy is made from slayer's own faces, so
// a face slayer's mesh lost in the making is lost from both. slayer's
// own tests hold the making, by a cube, a ball and rays from inside,
// and the manufactory's skin reader here holds every kind's skin.
// Group members also come from the manufactured object: a missing placement
// in construction is missing on both sides. Construction needs its own tests.
//
// a shape.Mesh is no evidence that slayer's mesh is right; it is the other
// side of a change of mesh under the same scenes.
package mesh_test

import (
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// inRefMesh is the same things with each of slayer's meshes copied face
// for face into a shape.Mesh.
func inRefMesh(t *testing.T, things []thing.Thing) []thing.Thing {
	t.Helper()
	var copied []thing.Thing
	cache := map[shape.Shape]shape.Shape{}
	converted := func(solid shape.Shape) shape.Shape {
		if known, ok := cache[solid]; ok {
			return known
		}
		var other shape.Shape = solid
		switch held := solid.(type) {
		case *mesh.Mesh:
			other = refMesh(t, held.Points(), held.Faces())
		}
		cache[solid] = other
		return other
	}
	for _, each := range things {
		if group, ok := each.Shape.(*shape.MeshGroup); ok {
			for _, member := range group.Members() {
				if _, ok := member.Mesh.(*mesh.Mesh); !ok {
					t.Fatalf("group member is not a production mesh: %T", member.Mesh)
				}
				copy := each
				copy.Shape = converted(member.Mesh)
				// These pictures are taken at time zero, for both renderers.
				copy.Motion = thing.Still(member.Space.Within(each.At(0)))
				copied = append(copied, copy)
			}
			continue
		}
		each.Shape = converted(each.Shape)
		copied = append(copied, each)
	}
	return copied
}

// Group comparisons must not route both pictures through MeshGroup.Hit.
func TestReferenceGroupUsesSeparatePlacedMeshes(t *testing.T) {
	prototype, err := mesh.New([]maths.Vec{
		maths.V(0, 0, 0),
		maths.V(1, 0, 0), maths.V(0, 1, 0),
	}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	group, err := shape.NewMeshGroup([]shape.MeshMember{
		{Mesh: prototype, Space: maths.Subspace{Origin: maths.V(2, 0, 0), Scale: 2}},
		{Mesh: prototype, Space: maths.Subspace{Origin: maths.V(-3, 0, 0)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	copies := inRefMesh(t, []thing.Thing{{Shape: group}})
	if len(copies) != 2 {
		t.Fatalf("group stayed one reference object: %d", len(copies))
	}
	for i, copy := range copies {
		if _, ok := copy.Shape.(*shape.Mesh); !ok {
			t.Fatalf("reference member %d shares group traversal: %T", i, copy.Shape)
		}
	}
	if copies[0].At(0).Out(maths.XAxis) != maths.V(4, 0, 0) ||
		copies[1].At(0).Out(maths.XAxis) != maths.V(-2, 0, 0) {
		t.Fatal("reference members lost their independent placements")
	}
}

// refMesh is a shape.Mesh of some faces, flat.
func refMesh(t *testing.T, corners []maths.Vec, faces [][3]int) *shape.Mesh {
	t.Helper()
	theirs, err := shape.NewMesh(corners, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return theirs
}

// A face slayer's mesh was not given is found by its own ray, so the
// test above would see a face either mesh dropped.
func TestADroppedFaceIsFound(t *testing.T) {
	corners, faces := ball(1)
	last := len(faces) - 1
	ours, err := mesh.New(corners, faces[:last], nil, false)
	if err != nil {
		t.Fatal(err)
	}
	missed := unmet(ours, refMesh(t, corners, faces), corners, faces)
	if !slices.Equal(missed, []int{last}) {
		t.Errorf("one face left out: %v not met, want [%d]",
			missed, last)
	}
}

// meeter is a mesh a ray can meet: mesh.Mesh or shape.Mesh.
type meeter interface {
	Hit(ray maths.Ray, near, far float64) (float64, bool)
}

// unmet is the faces, of those given, that a ray from a hair outside a
// face's middle, along its normal inward, does not meet within that hair
// on both meshes. a face of no area has no normal and is passed over. a
// face met nearer than the hair, by another lying close over it, is met.
func unmet(ours, theirs meeter, corners []maths.Vec, faces [][3]int) []int {
	box := maths.Box{Min: corners[0], Max: corners[0]}
	for _, corner := range corners {
		box = joined(box, maths.Box{Min: corner, Max: corner})
	}
	hair := 1e-6 * box.Max.Sub(box.Min).Len()
	var missed []int
	for i, at := range faces {
		one, two := corners[at[0]], corners[at[1]]
		three := corners[at[2]]
		across := two.Sub(one).Cross(three.Sub(one))
		if across.Len() == 0 {
			continue
		}
		out := across.Unit()
		middle := one.Add(two).Add(three).Scale(1.0 / 3)
		ray := maths.Ray{
			At:  middle.Add(out.Scale(hair)),
			Dir: out.Scale(-1),
		}
		_, oursMet := ours.Hit(ray, 0, 1.001*hair)
		_, theirsMet := theirs.Hit(ray, 0, 1.001*hair)
		if !oursMet || !theirsMet {
			missed = append(missed, i)
		}
	}
	return missed
}

// joined is the box around two boxes.
func joined(one, other maths.Box) maths.Box {
	return maths.Box{
		Min: maths.V(math.Min(one.Min.X, other.Min.X),
			math.Min(one.Min.Y, other.Min.Y), math.Min(one.Min.Z, other.Min.Z)),
		Max: maths.V(math.Max(one.Max.X, other.Max.X),
			math.Max(one.Max.Y, other.Max.Y), math.Max(one.Max.Z, other.Max.Z)),
	}
}

// ball is an icosahedron's faces cut in four so many times, each cut's
// new corners pushed out to the unit sphere, every face turning out.
func ball(times int) ([]maths.Vec, [][3]int) {
	gold := (1 + math.Sqrt(5)) / 2
	corners := []maths.Vec{
		maths.V(-1, gold, 0), maths.V(1, gold, 0),
		maths.V(-1, -gold, 0), maths.V(1, -gold, 0),
		maths.V(0, -1, gold), maths.V(0, 1, gold), maths.V(0, -1, -gold),
		maths.V(0, 1, -gold), maths.V(gold, 0, -1), maths.V(gold, 0, 1),
		maths.V(-gold, 0, -1), maths.V(-gold, 0, 1),
	}
	for i, corner := range corners {
		corners[i] = corner.Unit()
	}
	faces := [][3]int{
		{0, 11, 5},
		{0, 5, 1},
		{0, 1, 7},
		{0, 7, 10},
		{0, 10, 11},
		{1, 5, 9},
		{5, 11, 4},
		{11, 10, 2},
		{10, 7, 6},
		{7, 1, 8},
		{3, 9, 4},
		{3, 4, 2},
		{3, 2, 6},
		{3, 6, 8},
		{3, 8, 9},
		{4, 9, 5},
		{2, 4, 11},
		{6, 2, 10},
		{8, 6, 7},
		{9, 8, 1},
	}
	for range times {
		corners, faces = quartered(corners, faces)
	}
	return corners, faces
}

// quartered is each face cut in four at the middles of its edges, the
// middles pushed out to the unit sphere and shared by the faces either
// side.
func quartered(corners []maths.Vec, faces [][3]int) ([]maths.Vec,
	[][3]int,
) {
	middles := map[[2]int]int{}
	middle := func(one, other int) int {
		key := [2]int{min(one, other), max(one, other)}
		if at, seen := middles[key]; seen {
			return at
		}
		corners = append(corners,
			corners[one].Add(corners[other]).Scale(0.5).Unit())
		middles[key] = len(corners) - 1
		return len(corners) - 1
	}
	var cut [][3]int
	for _, at := range faces {
		ab, bc, ca := middle(at[0], at[1]), middle(at[1], at[2]),
			middle(at[2], at[0])
		cut = append(cut, [3]int{at[0], ab, ca}, [3]int{at[1], bc, ab},
			[3]int{at[2], ca, bc}, [3]int{ab, bc, ca})
	}
	return corners, cut
}
