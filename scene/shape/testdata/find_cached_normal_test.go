package shape

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestDiagnosticFindCachedNormalRoundingCases(t *testing.T) {
	for _, scale := range []float64{0, .25} {
		found := false
		for _, magnitude := range []float64{1e8, 1e9, 1e10, 1e11, 1e12} {
			for _, angle := range []float64{.1, .2, .3, .4, .7, 1.1} {
				space := maths.Subspace{
					Origin:      maths.V(magnitude, .23*magnitude, -.37*magnitude),
					Orientation: maths.About(maths.YAxis, angle), Scale: scale,
				}
				unit := scale
				if unit == 0 {
					unit = 1
				}
				ray := maths.Ray{
					At:  space.Out(maths.V(0, 0, 3)),
					Dir: space.Orientation.Out(maths.ZAxis.Neg()).Unit(),
				}
				for index := -128; index <= 128; index++ {
					delta := float64(index) * 1e-8
					solid, err := mesh.New([]maths.Vec{
						maths.V(-2, -2, 0), maths.V(2, -2, 0), maths.V(0, 2, 0),
						maths.V(delta, -2, -1), maths.V(delta, 2, -1), maths.V(delta, 0, 1),
					}, [][3]int{{0, 1, 2}, {3, 4, 5}}, nil, false)
					if err != nil {
						t.Fatal(err)
					}
					met, hit := solid.Meet(space.RayIn(ray), 2, 4)
					if !hit || met.Face() != 0 {
						continue
					}
					group, err := NewMeshGroup([]MeshMember{{Mesh: solid, Space: space}})
					if err != nil {
						t.Fatal(err)
					}
					distance, normal, hit := group.HitNormal(ray, 2*unit, 4*unit)
					if !hit {
						continue
					}
					expected := space.Orientation.Out(maths.ZAxis).Unit()
					point := space.In(ray.Point(distance))
					nearest := space.Orientation.Out(solid.Normal(point)).Unit()
					if normal.Dot(expected) > .999 && math.Abs(nearest.Dot(expected)) < .5 {
						t.Logf("scale=%g magnitude=%.17g angle=%.17g delta=%.17g distance=%.17g point=%#v cached=%#v nearest=%#v",
							scale, magnitude, angle, delta, distance, point, normal, nearest)
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Logf("scale %g: no fixed rounding example found", scale)
		}
	}
}
