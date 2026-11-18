//go:build draw && darwin

package gpu

import (
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// sparkMesh is a small triangle, a spark of the films these checks
// stand in for.
func sparkMesh(t *testing.T) *shape.Mesh {
	t.Helper()
	made, err := shape.NewMesh([]maths.Vec{maths.V(0, 0, 0),
		maths.V(0.2, 0, 0), maths.V(0, 0.2, 0)}, [][3]int{{0, 1, 2}}, nil,
		false)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// sparkFrame is a world of a mesh group and sparks standing at the
// given places, in the given order, as a film's frames add, remove and
// reorder them.
func sparkFrame(t *testing.T, spark *shape.Mesh, group *shape.MeshGroup,
	places []maths.Vec) *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: group,
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, -3, 0)})})
	for _, place := range places {
		world.Add(thing.Thing{Shape: spark,
			Motion: thing.Still(maths.Subspace{Origin: place})})
	}
	return world
}

// sparkFrames are a film's frames: sparks added, removed, moved, and
// the same sparks in the other order.
func sparkFrames() [][]maths.Vec {
	three := []maths.Vec{maths.V(-4, 1, 0), maths.V(3, 2, 1),
		maths.V(0, 5, -2)}
	five := append(slices.Clone(three), maths.V(6, -1, 3),
		maths.V(-2, -2, 4))
	reversed := slices.Clone(five)
	slices.Reverse(reversed)
	moved := []maths.Vec{maths.V(1, 1, 1), maths.V(-5, 0, 2)}
	return [][]maths.Vec{three, five, moved, reversed}
}

// readings is what the three readers of a scene's standing order made:
// the instances and their tree, the native first faces, and the native
// hierarchy's layout.
type readings struct {
	instances, tops, firsts []byte
}

// read has the three readers read the scene's order as it is now. The
// instances are packed again into fresh buffers, as instancesOf packs
// them in an encode.
func read(t *testing.T, encoded *scene) readings {
	t.Helper()
	encoded.instances, encoded.tops = packer{}, packer{}
	if err := encoded.instancesOf(); err != nil {
		t.Fatal(err)
	}
	return readings{instances: encoded.instances.bytes,
		tops: encoded.tops.bytes, firsts: encoded.nativeFirsts()}
}

// planLayout is the layout of the scene's native hierarchy now, and its
// root.
func planLayout(t *testing.T, encoded *scene) (metal.Layout, int) {
	t.Helper()
	plan, root, err := encoded.accelerationPlan()
	if err != nil {
		t.Fatal(err)
	}
	return plan.Layout(), root
}

// sameReadings is whether two readings are byte for byte one.
func sameReadings(one, other readings) bool {
	return string(one.instances) == string(other.instances) &&
		string(one.tops) == string(other.tops) &&
		string(one.firsts) == string(other.firsts)
}

// TestEveryReaderGetsToppedsOrderForItsFrame encodes each frame of a
// film whose sparks are added, removed, moved and reordered: the order
// handed to the instances, the first faces and the hierarchy is the
// one topped makes of that frame's standings, and what each reader
// makes is what it makes from an order worked afresh.
func TestEveryReaderGetsToppedsOrderForItsFrame(t *testing.T) {
	spark := sparkMesh(t)
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: spark},
		{Mesh: sparkMesh(t), Space: maths.Subspace{
			Origin: maths.V(1, 0, 0)}}})
	if err != nil {
		t.Fatal(err)
	}
	for index, places := range sparkFrames() {
		world := sparkFrame(t, spark, group, places)
		encoded, err := encode(world, maths.Subspace{}, nil, 0, 4, 3)
		if err != nil {
			t.Fatal(err)
		}
		handed := encoded.ordered()
		want, _ := topped(encoded.standings)
		if !slices.Equal(handed.order, want) || len(handed.local) != 1 {
			t.Fatalf("frame %d: handed %v, topped makes %v", index,
				handed.order, want)
		}
		members, _ := topped(encoded.groups[0].members)
		if !slices.Equal(handed.local[0], members) {
			t.Fatalf("frame %d: group order %v, topped makes %v", index,
				handed.local[0], members)
		}
		kept := read(t, encoded)
		keptLayout, keptRoot := planLayout(t, encoded)
		encoded.order = nil
		fresh := read(t, encoded)
		encoded.order = nil
		freshLayout, freshRoot := planLayout(t, encoded)
		if !sameReadings(kept, fresh) || !keptLayout.Same(freshLayout) ||
			keptRoot != freshRoot {
			t.Fatalf("frame %d: a reader made something else from the "+
				"order handed it", index)
		}
	}
}

// TestAStaleOrderIsNeverHanded is the anti-test: the previous frame's
// order, put in this frame's scene, is not handed to a reader; a
// standing added after an order was worked makes another; and a wrong
// order made to look current shows in what the readers make, so the
// comparison above would catch a stale one.
func TestAStaleOrderIsNeverHanded(t *testing.T) {
	spark := sparkMesh(t)
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: spark}})
	if err != nil {
		t.Fatal(err)
	}
	frames := sparkFrames()
	before, err := encode(sparkFrame(t, spark, group, frames[1]),
		maths.Subspace{}, nil, 0, 4, 3)
	if err != nil {
		t.Fatal(err)
	}
	now, err := encode(sparkFrame(t, spark, group, frames[3]),
		maths.Subspace{}, nil, 0, 4, 3)
	if err != nil {
		t.Fatal(err)
	}
	stale := before.ordered()
	now.order = stale
	if now.ordered() == stale {
		t.Fatal("the previous frame's order was handed to this frame")
	}
	worked := now.ordered()
	now.stand(spark, func() pendingMesh { return now.shapeTree(spark) },
		maths.Subspace{Origin: maths.V(9, 9, 9)}, 0, false)
	if again := now.ordered(); again == worked ||
		len(again.order) != len(now.standings) {
		t.Fatal("an order worked before a standing was added was handed")
	}
	fresh := read(t, now)
	forced := *now.ordered()
	forced.order = slices.Clone(forced.order)
	slices.Reverse(forced.order)
	now.order = &forced
	wrong := read(t, now)
	if sameReadings(fresh, wrong) {
		t.Fatal("a stale order forced on the readers went unseen")
	}
}
