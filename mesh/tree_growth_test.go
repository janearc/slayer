package mesh

import (
	"math"
	"sync"
	"testing"

	"github.com/janearc/slayer/maths"
)

// TestTheTreeIsGrownAtTheFirstRayAndKeepsBoundsAndMeets says that New
// grows no tree, that the box it gives is the grown tree's root box to
// the bit, and that rays from many goroutines at once meet the mesh as
// a mesh whose tree was grown first does.
func TestTheTreeIsGrownAtTheFirstRayAndKeepsBoundsAndMeets(t *testing.T) {
	eager, err := New(cubeCorners(), cubeFaces(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	eager.ensureTree()
	lazy, err := New(cubeCorners(), cubeFaces(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lazy.tree.nodes) != nought {
		t.Fatal("tree grown by New")
	}
	if root := eager.tree.nodes[nought].box(); lazy.Bounds() != root {
		t.Fatalf("bounds %v, the tree's root %v", lazy.Bounds(), root)
	}
	if len(lazy.tree.nodes) != nought {
		t.Fatal("bounds grew the tree")
	}
	height := picked(t, "tree-growth-height")
	ray := maths.NewRay(maths.V(nought, nought, height),
		maths.V(nought, nought, -unity))
	want, ok := eager.Meet(ray, nought, math.Inf(plusInfinity))
	if !ok {
		t.Fatal("control ray missed")
	}
	race := racing{test: t, lazy: lazy, ray: ray, want: want,
		rayers: pickedCount(t, "tree-growth-rayers"),
		looks:  pickedCount(t, "tree-growth-storage-looks")}
	race.run()
	if len(lazy.tree.nodes) == nought {
		t.Fatal("tree never grown")
	}
	away := maths.V(picked(t, "tree-growth-away"), nought, nought)
	if lazy.Distance(away) != eager.Distance(away) {
		t.Fatal("distance changed")
	}
}

// racing is a mesh met by many goroutines at once while its storage is
// counted: how many send a ray, the ray, where it must meet the mesh, and
// how many times a goroutine counts the storage.
type racing struct {
	test   *testing.T
	lazy   *Mesh
	ray    maths.Ray
	want   Met
	rayers int
	looks  int
}

// run sends the rays at the mesh at once, and fails the test where one
// does not meet it where want says.
func (race racing) run() {
	race.test.Helper()
	var group sync.WaitGroup
	group.Go(race.count)
	for range race.rayers {
		group.Go(race.meet)
	}
	group.Wait()
}

// count counts the mesh's storage, so many times.
func (race racing) count() {
	for range race.looks {
		_ = race.lazy.Storage()
	}
}

// meet sends the ray at the mesh and fails the test if it is not met
// where the control ray was.
func (race racing) meet() {
	got, ok := race.lazy.Meet(race.ray, nought, math.Inf(plusInfinity))
	if !ok || got != race.want {
		race.test.Errorf("ray: %v %v; want %v", got, ok, race.want)
	}
}

// TestTheBoxOfFewerFacesIsAnotherBox is the anti-test of the box above:
// a cube with a face pair left out has another box, so a box worked
// from the faces is compared to a root box that could differ.
func TestTheBoxOfFewerFacesIsAnotherBox(t *testing.T) {
	every := spansFaces(cubeFaces())
	few := spansFaces(cubeFaces()[:pair])
	all := boxOf(cubeCorners(), every)
	part := boxOf(cubeCorners(), few)
	if all == part {
		t.Errorf("a cube and two of its faces have one box, %v", all)
	}
}
