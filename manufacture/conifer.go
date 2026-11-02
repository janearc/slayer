package manufacture

import (
	"math"

	"github.com/janearc/slayer/geom"
)

type ConiferParams struct {
	Height      float64
	BaseRadius  float64
	Depth       int
	TaperRate   float64
	DepthScale  float64
	BranchStart float64
	Sides       int
}

// DefaultConiferParams is the shape of the plain recursive conifer, ten high.
func DefaultConiferParams() ConiferParams {
	return ConiferParams{
		Height:      10,
		BaseRadius:  2,
		Depth:       5,
		TaperRate:   0.7,
		DepthScale:  0.7,
		BranchStart: 0.3,
		Sides:       8,
	}
}

// Conifer is the recursive conifer at base: a cone trunk and, on it, shorter
// ones leaning out, to the depth the params ask.
func Conifer(base geom.Vec3, params ConiferParams) []geom.Triangle {
	if params.Depth < 0 {
		return nil
	}
	return coniferRecursive(base, params, 0)
}

// coniferRecursive is the cone at one depth and the branches that grow from its
// top.
func coniferRecursive(base geom.Vec3, params ConiferParams, depth int) []geom.Triangle {
	triangles := make([]geom.Triangle, 0)

	if depth > params.Depth {
		return triangles
	}

	scale := math.Pow(params.DepthScale, float64(depth))
	height := params.Height * scale
	radius := params.BaseRadius * math.Pow(params.TaperRate, float64(depth))

	material := 2
	trunk := Cone(base, height, radius, params.Sides, material)
	triangles = append(triangles, trunk...)

	if depth < params.Depth {
		top := base.Add(geom.Vec3{0, 0, height})
		numBranches := 3 + depth
		for b := 0; b < numBranches; b++ {
			theta := 2 * math.Pi * float64(b) / float64(numBranches)
			phi := params.BranchStart + float64(depth)*0.15

			cos_phi := math.Cos(phi)
			sin_phi := math.Sin(phi)

			branchDir := geom.Vec3{
				cos_phi * math.Cos(theta),
				cos_phi * math.Sin(theta),
				sin_phi,
			}

			branchLen := height * 0.4
			branchBase := top.Add(branchDir.Scale(branchLen * 0.3))

			branchParams := params
			branchParams.Height *= 0.6
			branchParams.BaseRadius *= 0.5

			branch := coniferRecursive(branchBase, branchParams, depth+1)
			triangles = append(triangles, branch...)
		}
	}

	return triangles
}
