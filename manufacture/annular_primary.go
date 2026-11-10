package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// annularPrimary closes an exact panel inventory around a central opening.
// Diameter is the circumscribed planar extent, not an optical collecting area.
type annularPrimary struct {
	count, hole                      int
	diameter, fill, depth, thickness float64
}

// skin scales the complete annular inventory to its declared aperture.
func (primary annularPrimary) skin(cut) surface {
	centres := annularPanels(primary.count, primary.hole)
	if len(centres) == spaceZero {
		return nil
	}
	radius := panelEnvelope(centres, primary.fill)
	scale := primary.diameter * spaceHalf / radius
	figure := hexagonalPrimary{
		diameter:  primary.diameter,
		depth:     primary.depth,
		thickness: primary.thickness}
	var skin surface
	for _, centre := range centres {
		skin = figure.tile(skin, centre.Scale(scale),
			primary.fill*scale)
	}
	return skin
}

// panelEnvelope fits the outermost panel corner to the nominal aperture.
func panelEnvelope(centres []maths.Vec, fill float64) float64 {
	flat := hexagonalPrimary{diameter: spaceOne}
	radius := float64(spaceZero)
	for _, centre := range centres {
		for corner := range hexagonCorners {
			point := flat.corner(centre, corner, fill)
			radius = math.Max(radius, math.Hypot(point.X, point.Z))
		}
	}
	return radius
}
