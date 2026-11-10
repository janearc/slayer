package shape

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// The ray meets z=0 while nearly parallel to a nearby x plane. Converting its
// world hit point back to local coordinates loses enough precision to make a
// new nearest-face search choose the x plane. Retain the actual ray hit.
func TestGroupNormalsKeepTheRayFaceAcrossCoordinateRounding(t *testing.T) {
	for _, row := range []struct {
		scale, magnitude, angle, delta float64
	}{
		{0, 1e9, 1.1, 3.0000000000000004e-08},
		{.25, 1e10, .4, -1.0100000000000001e-06},
	} {
		solid, err := mesh.New([]maths.Vec{
			maths.V(-2, -2, 0), maths.V(2, -2, 0), maths.V(0, 2, 0),
			maths.V(row.delta, -2, -1), maths.V(row.delta, 2, -1),
			maths.V(row.delta, 0, 1),
		}, [][3]int{{0, 1, 2}, {3, 4, 5}}, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		space := maths.Subspace{
			Origin:      maths.V(row.magnitude, .23*row.magnitude, -.37*row.magnitude),
			Orientation: maths.About(maths.YAxis, row.angle), Scale: row.scale,
		}
		group, err := NewMeshGroup([]MeshMember{{Mesh: solid, Space: space}})
		if err != nil {
			t.Fatal(err)
		}
		ray := maths.Ray{At: space.Out(maths.V(0, 0, 3)),
			Dir: space.Orientation.Out(maths.ZAxis.Neg()).Unit()}
		unit := row.scale
		if unit == 0 {
			unit = 1
		}
		met, hit := solid.Meet(space.RayIn(ray), 2, 4)
		if !hit || met.Face() != 0 {
			t.Fatal("control ray does not meet the z plane first")
		}
		distance, normal, hit := group.HitNormal(ray, 2*unit, 4*unit)
		want := maths.V(math.Sin(row.angle), 0, math.Cos(row.angle))
		if !hit || !(distance > 2*unit && distance < 4*unit) ||
			!(normal.Sub(want).Len() <= 1e-12) {
			t.Fatalf("scale %g: distance %.17g normal %v hit %t, want z-plane normal %v",
				row.scale, distance, normal, hit, want)
		}
	}
}
