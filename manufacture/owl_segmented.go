package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// owlSegmented has an authored annular layout and a backed panel at each cell.
// Curvature zero is the flat secondary; a positive radius is a sphere.
type owlSegmented struct {
	count, hole                          int
	diameter, curvature, fill, thickness float64
}

// skin changes neither panel inventory nor its physical figure with detail.
func (mirror owlSegmented) skin(cut) surface {
	centres := annularPanels(mirror.count, mirror.hole)
	if len(centres) == spaceZero {
		return nil
	}
	scale := mirror.diameter * spaceHalf / panelEnvelope(centres, mirror.fill)
	var skin surface
	for _, centre := range centres {
		skin = mirror.panel(skin, centre.Scale(scale), mirror.fill*scale)
	}
	return skin
}

// height rationalizes the spherical sag to avoid subtracting close radii.
// The source radius does not turn this display surface into a ray prescription.
func (mirror owlSegmented) height(point maths.Vec) float64 {
	if mirror.curvature == spaceZero {
		return spaceZero
	}
	radial := point.X*point.X + point.Z*point.Z
	inside := mirror.curvature*mirror.curvature - radial
	return radial / (mirror.curvature + math.Sqrt(inside))
}

// point uses the shared planar hex layout but this mirror's figure.
func (mirror owlSegmented) point(centre maths.Vec,
	corner int, radius float64,
) maths.Vec {
	flat := hexagonalPrimary{}
	point := flat.corner(centre, corner, radius)
	point.Y = mirror.height(point)
	return point
}

// panel backs every corner at a fixed axial thickness below its own figure.
// The backing therefore follows the sphere, rather than filling to a flat slab.
func (mirror owlSegmented) panel(skin surface, centre maths.Vec,
	radius float64,
) surface {
	top := centre
	top.Y = mirror.height(top)
	down := maths.YAxis.Scale(-mirror.thickness)
	bottom := top.Add(down)
	for corner := range hexagonCorners {
		here := mirror.point(centre, corner, radius)
		next := mirror.point(centre, corner+spaceOne, radius)
		lowHere, lowNext := here.Add(down), next.Add(down)
		skin = append(skin, face{top, here, next},
			face{bottom, lowNext, lowHere})
		skin = quad(skin, here, lowHere, lowNext, next)
	}
	return skin
}
