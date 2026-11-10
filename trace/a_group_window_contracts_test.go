package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestReachPreservesWorldRayWindowsForScaledMeshesAndGroups(t *testing.T) {
	solid, err := mesh.New([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)},
		[][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: solid}})
	if err != nil {
		t.Fatal(err)
	}
	for _, skin := range []shape.Shape{solid, group} {
		for _, scale := range []float64{.25, 1, 3} {
			space := maths.Subspace{Origin: maths.V(3, -2, 5), Scale: scale,
				Orientation: maths.Rotation(maths.V(1, 2, 3), .6)}
			item := thing.Thing{Shape: skin}
			placed := placedThing{thing: &item, placement: newPlacement(space, item.Material, 0)}
			ray := maths.Ray{At: space.Out(maths.V(.2, .2, 2)),
				Dir: space.Orientation.Out(maths.ZAxis.Neg())}
			distance := 2 * scale
			got, landing, hit := placed.reach(ray, .99*distance, 1.01*distance)
			if !hit || !(math.Abs(got-distance) <= 1e-12) {
				t.Fatalf("%T scale %g: distance %g hit %t", skin, scale, got, hit)
			}
			met := hitRecord(placed, got, landing)
			normal := met.normal(ray.Point(got), ray.Dir)
			want := space.Orientation.Out(maths.ZAxis)
			if !(normal.Sub(want).Len() <= 1e-12) {
				t.Fatalf("%T scale %g normal %v, want %v", skin, scale, normal, want)
			}
			for _, limits := range [][2]float64{{.1 * distance, .2 * distance}, {1.1 * distance, 2 * distance}} {
				if _, _, hit := placed.reach(ray, limits[0], limits[1]); hit {
					t.Fatalf("%T scale %g crossed outside %v", skin, scale, limits)
				}
			}
		}
	}
}

func hitRecord(placed placedThing, distance float64, on landing) hit {
	return hit{thing: placed, dist: distance, on: on}
}
