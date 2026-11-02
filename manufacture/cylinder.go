package manufacture

import (
	"math"

	"github.com/janearc/slayer/geom"
)

// Cylinder is the wall of a tube of sides standing on base, a radius at the
// foot and one at the top, with no caps. A top radius of zero makes a cone.
func Cylinder(base geom.Vec3, height float64, baseRadius float64, topRadius float64, sides int, material int) []geom.Triangle {
	if sides < 3 {
		sides = 3
	}
	if height <= 0 || (baseRadius <= 0 && topRadius <= 0) {
		return nil
	}

	isCone := topRadius == 0
	var triangles []geom.Triangle
	if isCone {
		triangles = make([]geom.Triangle, 0, sides)
	} else {
		triangles = make([]geom.Triangle, 0, sides*2)
	}

	apex := base.Add(geom.Vec3{0, 0, height})

	for i := 0; i < sides; i++ {
		theta0 := 2 * math.Pi * float64(i) / float64(sides)
		theta1 := 2 * math.Pi * float64(i+1) / float64(sides)

		cos0, sin0 := math.Cos(theta0), math.Sin(theta0)
		cos1, sin1 := math.Cos(theta1), math.Sin(theta1)

		// base vertices
		b0 := base.Add(geom.Vec3{cos0 * baseRadius, sin0 * baseRadius, 0})
		b1 := base.Add(geom.Vec3{cos1 * baseRadius, sin1 * baseRadius, 0})

		if isCone {
			// cone: one triangle per side (b0, b1, apex)
			triangles = append(triangles, geom.Triangle{
				V:        [3]geom.Vec3{b0, b1, apex},
				Material: material,
				Clear:    false,
			})
		} else {
			// cylinder: two triangles per side
			t0 := base.Add(geom.Vec3{cos0 * topRadius, sin0 * topRadius, height})
			t1 := base.Add(geom.Vec3{cos1 * topRadius, sin1 * topRadius, height})

			triangles = append(triangles, geom.Triangle{
				V:        [3]geom.Vec3{b0, b1, t0},
				Material: material,
				Clear:    false,
			})
			triangles = append(triangles, geom.Triangle{
				V:        [3]geom.Vec3{b1, t1, t0},
				Material: material,
				Clear:    false,
			})
		}
	}

	return triangles
}

// Cone is a cylinder that comes to a point.
func Cone(base geom.Vec3, height float64, baseRadius float64, sides int, material int) []geom.Triangle {
	return Cylinder(base, height, baseRadius, 0, sides, material)
}
