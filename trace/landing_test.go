package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// aCube is slayer's mesh of a cube of side one about its middle, each
// face turning counterclockwise seen from outside.
func aCube(t *testing.T) *mesh.Mesh {
	t.Helper()
	corners := make([]maths.Vec, 8)
	for i := range corners {
		corners[i] = maths.V(float64(i&1)-0.5, float64(i>>1&1)-0.5,
			float64(i>>2&1)-0.5)
	}
	made, err := mesh.New(corners, [][3]int{{0, 4, 6}, {0, 6, 2},
		{1, 3, 7}, {1, 7, 5}, {0, 1, 5}, {0, 5, 4}, {2, 6, 7}, {2, 7, 3},
		{0, 2, 3}, {0, 3, 1}, {4, 5, 7}, {4, 7, 6}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// A ray that lands on slayer's mesh carries the face it landed on out
// of the crowd, and the hit's normal is that face's, put out in the
// world and turned against the ray: on a cube turned and doubled, rays
// from its +x side come back with its turned +x for a normal.
func TestALandingOnAMeshKnowsItsFace(t *testing.T) {
	world := thing.NewWorld()
	turn := maths.About(maths.YAxis, 0.5)
	world.Add(thing.Thing{Shape: aCube(t), Motion: thing.Still(
		maths.Subspace{Orientation: turn, Scale: 2})})
	seen := gathered(place(world, 0))
	side := turn.Out(maths.XAxis).Unit()
	for _, off := range []maths.Vec{maths.V(0, 0, 0), maths.V(0, 0.4, 0.3),
		maths.V(0, -0.6, 0.2)} {
		from := side.Scale(5).Add(turn.Out(off))
		ray := maths.Ray{At: from, Dir: side.Neg()}
		met, found := seen.first(ray, 0)
		if !found || !met.on.meshed || math.Abs(met.dist-4) > 1e-12 {
			t.Fatalf("from %v: found %v, landed %v, at %v", from, found,
				met.on.meshed, met.dist)
		}
		normal := met.normal(ray.Point(met.dist), ray.Dir)
		if normal.Sub(side).Len() > 1e-12 {
			t.Errorf("from %v the cube faces %v, not %v", from, normal, side)
		}
	}
}
