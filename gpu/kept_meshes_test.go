package gpu

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// The small world these checks draw: fans of a few faces, a picture of a
// few samples, a few frames, and how far the things move each frame.
// Picked 2026-11-15 to encode in milliseconds.
const (
	fanFaces, smoothFanFaces = 8, 12
	fanAcross, fanDown       = 8, 6
	fanFrames                = 4
	fanStep                  = time.Second / 6
	fanStride                = 0.25
	fanRise                  = 0.1
	fanBack                  = 20
)

// fanCorners and fanNormals are a fan's corners about a centre at height
// lift, each rim corner a little higher than the one before, and its
// normals, all up.
func fanCorners(lift float64, faces int) ([]maths.Vec, [][3]int,
	[]maths.Vec) {
	corners := []maths.Vec{maths.V(0, 0, lift)}
	var normals []maths.Vec
	var made [][3]int
	for index := range faces + 1 {
		turn := 2 * math.Pi * float64(index) / float64(faces+1)
		corners = append(corners, maths.V(math.Cos(turn), math.Sin(turn),
			lift+fanRise*float64(index)))
		if index > 0 {
			made = append(made, [3]int{0, index, index + 1})
		}
	}
	for range corners {
		normals = append(normals, maths.ZAxis)
	}
	return corners, made, normals
}

// shapeFan is a fan as a shape.Mesh, smooth or flat.
func shapeFan(t *testing.T, lift float64, faces int,
	smooth bool) *shape.Mesh {
	t.Helper()
	corners, made, normals := fanCorners(lift, faces)
	fan, err := shape.NewMesh(corners, made, normals, smooth)
	if err != nil {
		t.Fatal(err)
	}
	return fan
}

// slayersFan is a fan as one of slayer's meshes.
func slayersFan(t *testing.T, lift float64) *mesh.Mesh {
	t.Helper()
	corners, made, _ := fanCorners(lift, fanFaces)
	fan, err := mesh.New(corners, made, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return fan
}

// fanWorld is the world at a frame: one flat fan standing twice, a
// smooth fan and one of slayer's meshes made again as new pointers with
// the same faces, and a sphere, every one moved on by the frame.
func fanWorld(t *testing.T, flat *shape.Mesh, frame int) *thing.World {
	t.Helper()
	world := &thing.World{}
	shapes := []shape.Shape{flat, shapeFan(t, 2, smoothFanFaces, true),
		flat, slayersFan(t, 3), shape.Sphere{Radius: 1}}
	for index, made := range shapes {
		at := maths.V(float64(index)+fanStride*float64(frame),
			fanStride*float64(frame), 0)
		space := maths.NewSubspace(at, maths.Identity, 1)
		world.Add(thing.Thing{Shape: made, Motion: thing.Still(space)})
	}
	return world
}

// fanEncoded is a frame of the fan world encoded with a keeper, and the
// same frame encoded with none.
func fanEncoded(t *testing.T, world *thing.World, frame int,
	keeper *keptTrees) (got, want *scene) {
	t.Helper()
	eye := maths.NewSubspace(maths.V(0, 0, fanBack), maths.Identity, 1)
	at := time.Duration(frame) * fanStep
	want, err := encode(world, eye, nil, at, fanAcross, fanDown)
	if err != nil {
		t.Fatal(err)
	}
	got, err = encodeKept(world, nil, eye, nil, at, fanAcross, fanDown,
		keeper)
	if err != nil {
		t.Fatal(err)
	}
	return got, want
}

// sameInputs is whether two scenes' inputs are the same to the byte.
func sameInputs(got, want *scene) bool {
	for place, input := range want.inputs() {
		if string(got.inputs()[place]) != string(input) {
			return false
		}
	}
	return true
}

// TestKeptMeshesGiveTheInputsAFullEncodeGives moves every thing each
// frame and makes two meshes again as new pointers: every input is a
// full encode's to the byte, and from the second frame on the faces were
// taken whole from the frame before, so the comparison is of kept ones.
func TestKeptMeshesGiveTheInputsAFullEncodeGives(t *testing.T) {
	keeper := newKeptTrees()
	flat := shapeFan(t, 1, fanFaces, false)
	for frame := range fanFrames {
		got, want := fanEncoded(t, fanWorld(t, flat, frame), frame, keeper)
		keeper.turn()
		if !sameInputs(got, want) {
			t.Fatalf("frame %d: an input differs from a full encode", frame)
		}
	}
	if keeper.reused != fanFrames-1 || keeper.served == 0 {
		t.Fatalf("%d frames of %d took the pool and %d trees were served",
			keeper.reused, fanFrames-1, keeper.served)
	}
}

// TestAMeshChangedInPlaceIsPackedAgain is the anti-test: a shape.Mesh
// whose corner is moved in place, its pointer the same, has new faces
// the next frame. They are a full encode's, not the last frame's, and
// the pool is packed again.
func TestAMeshChangedInPlaceIsPackedAgain(t *testing.T) {
	keeper := newKeptTrees()
	flat := shapeFan(t, 1, fanFaces, false)
	before, _ := fanEncoded(t, fanWorld(t, flat, 0), 0, keeper)
	keeper.turn()
	flat.Vertices[1] = flat.Vertices[1].Add(maths.V(0, 0, fanRise))
	got, want := fanEncoded(t, fanWorld(t, flat, 1), 1, keeper)
	if string(got.triangles.bytes) == string(before.triangles.bytes) {
		t.Fatal("the moved corner did not change the faces")
	}
	if !sameInputs(got, want) || keeper.reused != 0 {
		t.Fatalf("a mesh changed in place was served stale: %d reused",
			keeper.reused)
	}
}

// TestAWrongKeptPoolIsSeen is the anti-test for the comparison: a kept
// pool with one byte changed is served, and the inputs then differ from
// a full encode's.
func TestAWrongKeptPoolIsSeen(t *testing.T) {
	keeper := newKeptTrees()
	flat := shapeFan(t, 1, fanFaces, false)
	fanEncoded(t, fanWorld(t, flat, 0), 0, keeper)
	keeper.turn()
	spoiled := append([]byte(nil), keeper.pool.triangles...)
	spoiled[0] ^= 1
	keeper.pool.triangles = spoiled
	got, want := fanEncoded(t, fanWorld(t, flat, 1), 1, keeper)
	if sameInputs(got, want) {
		t.Fatal("a wrong kept pool went unseen")
	}
}

// TestAShapeMeshKeyIsItsBytes: a mesh made again with the same faces has
// the same key, and a moved corner, a moved normal, a face turned or the
// smooth flag changed each give another.
func TestAShapeMeshKeyIsItsBytes(t *testing.T) {
	key := shapeMeshKey(shapeFan(t, 1, fanFaces, true))
	if shapeMeshKey(shapeFan(t, 1, fanFaces, true)) != key {
		t.Fatal("the same faces made again have another key")
	}
	changes := map[string]func(made *shape.Mesh){
		"a corner moved": func(made *shape.Mesh) {
			made.Vertices[2] = made.Vertices[2].Add(maths.V(fanRise, 0, 0))
		},
		"a normal moved": func(made *shape.Mesh) {
			made.Normals[2] = maths.XAxis
		},
		"a face turned": func(made *shape.Mesh) {
			first := made.Faces[0]
			made.Faces[0] = [3]int{first[1], first[2], first[0]}
		},
		"made flat": func(made *shape.Mesh) { made.Smooth = false },
	}
	for name, change := range changes {
		made := shapeFan(t, 1, fanFaces, true)
		change(made)
		if shapeMeshKey(made) == key {
			t.Errorf("%s: the key did not change", name)
		}
	}
}

// TestASlayersMeshKeyIsItsFaces: two of slayer's meshes made apart with
// the same faces have one key and share a tree, and another mesh has
// another key.
func TestASlayersMeshKeyIsItsFaces(t *testing.T) {
	keeper := newKeptTrees()
	one := keeper.slayersTree(slayersFan(t, 3))
	again := keeper.slayersTree(slayersFan(t, 3))
	other := keeper.slayersTree(slayersFan(t, 4))
	if one.key != again.key || one.key == other.key {
		t.Fatal("slayer's meshes are not keyed by their faces")
	}
	if keeper.held() != 0 || len(keeper.used) != 2 {
		t.Fatalf("%d trees used for two meshes' faces", len(keeper.used))
	}
}
