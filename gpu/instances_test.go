package gpu

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// A mesh standing in a hundred places is in the pool once: its faces
// and its tree's boxes are written one time, each place is a standing
// of its own, and the standings have a tree over them, each in exactly
// one of its leaves.
func TestAMeshStandingOftenIsPooledOnce(t *testing.T) {
	face := meshOf(t, false, false)
	world := thing.NewWorld()
	for index := range 100 {
		at := maths.Subspace{Origin: maths.V(float64(index%10)*3, 0,
			float64(index/10)*3), Orientation: maths.About(maths.YAxis,
			float64(index)), Scale: 1 + float64(index%3)}
		world.Add(thing.Thing{Shape: face.Shape, Motion: thing.Still(at)})
	}
	encoded := newScene()
	if err := encoded.things(world, 0); err != nil {
		t.Fatal(err)
	}
	if encoded.counts.triangles != 1 || encoded.counts.instances != 100 {
		t.Fatalf("%d faces and %d standings", encoded.counts.triangles,
			encoded.counts.instances)
	}
	order, nodes := topped(encoded.standings)
	held := map[int]int{}
	for _, each := range nodes {
		if each.count == 0 {
			continue
		}
		for _, index := range order[each.first : each.first+each.count] {
			held[index]++
		}
	}
	for index := range 100 {
		if held[index] != 1 {
			t.Errorf("standing %d is in %d leaves", index, held[index])
		}
	}
}

// A standing's box in the world holds its mesh wherever the mesh's
// corners go: every corner of a turned, moved and doubled face is in
// its standing's box.
func TestAStandingsBoxHoldsItsMesh(t *testing.T) {
	space := maths.Subspace{Origin: maths.V(4, -2, 7),
		Orientation: maths.About(maths.V(1, 2, 3).Unit(), 1.1), Scale: 2}
	local := empty().fit(maths.V(0, 0, 0)).fit(maths.V(1, 0, 0)).
		fit(maths.V(0, 1, 0))
	bounds := worldBox(local, space)
	for _, corner := range []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(0, 1, 0)} {
		at := space.Out(corner)
		if !bounds.holds(empty().fit(at)) {
			t.Errorf("%v stands at %v, outside %v", corner, at, bounds)
		}
	}
}
