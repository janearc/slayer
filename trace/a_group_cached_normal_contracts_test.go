package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestGroupedLandingKeepsItsRayNormalAfterWorldRounding(t *testing.T) {
	const angle, delta = 1.1, 3.0000000000000004e-08
	solid, err := mesh.New([]maths.Vec{
		maths.V(-2, -2, 0), maths.V(2, -2, 0), maths.V(0, 2, 0),
		maths.V(delta, -2, -1), maths.V(delta, 2, -1), maths.V(delta, 0, 1),
	}, [][3]int{{0, 1, 2}, {3, 4, 5}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	space := maths.Subspace{Origin: maths.V(1e9, 230000000, -370000000),
		Orientation: maths.About(maths.YAxis, angle)}
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: solid, Space: space}})
	if err != nil {
		t.Fatal(err)
	}
	item := thing.Thing{Shape: group}
	placed := placedThing{thing: &item, placement: newPlacement(maths.Subspace{}, nil, 0)}
	ray := maths.Ray{At: space.Out(maths.V(0, 0, 3)),
		Dir: space.Orientation.Out(maths.ZAxis.Neg()).Unit()}
	distance, on, found := placed.reach(ray, 2, 4)
	if !found {
		t.Fatal("ray missed the grouped z plane")
	}
	met := hitRecord(placed, distance, on)
	got := met.normal(ray.Point(distance), ray.Dir)
	want := maths.V(math.Sin(angle), 0, math.Cos(angle))
	if !(got.Sub(want).Len() <= 1e-12) {
		t.Fatalf("group landing normal %v, want ray-face normal %v", got, want)
	}
}
