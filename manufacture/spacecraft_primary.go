package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// hexagonalPrimary is complete hexagonal rings with an empty centre.
// Each tile has its own backed surface, with an authored sag and gap.
type hexagonalPrimary struct {
	rings                            int
	diameter, fill, depth, thickness float64
}

// skin lays out complete rings without filling their central opening.
func (primary hexagonalPrimary) skin(cut) surface {
	var skin surface
	across := float64(primary.rings)/spaceHalf + primary.fill
	radius := primary.diameter / (across * math.Sqrt(triangleCorners))
	columns := primary.rings*int(spaceOne/spaceHalf) + int(spaceOne)
	for index := range columns {
		column := index - primary.rings
		low := max(-primary.rings, -column-primary.rings)
		high := min(primary.rings, -column+primary.rings)
		for offset := range high - low + int(spaceOne) {
			row := low + offset
			if column == spaceZero && row == spaceZero {
				continue
			}
			x := triangleCorners * spaceHalf * radius *
				float64(column)
			step := float64(row) + float64(column)*spaceHalf
			z := math.Sqrt(triangleCorners) * radius * step
			centre := maths.V(x, spaceZero, z)
			skin = primary.tile(skin, centre, radius*primary.fill)
		}
	}
	return skin
}

// height supplies an authored shallow figure, not a calibrated prescription.
func (primary hexagonalPrimary) height(at maths.Vec) float64 {
	return primary.depth * (at.X*at.X + at.Z*at.Z) /
		(primary.diameter * primary.diameter * spaceHalf * spaceHalf)
}

// tile closes one separated panel around its backing.
func (primary hexagonalPrimary) tile(skin surface, centre maths.Vec,
	radius float64,
) surface {
	top := maths.V(centre.X, primary.height(centre), centre.Z)
	bottom := maths.V(centre.X, -primary.thickness, centre.Z)
	for corner := range hexagonCorners {
		here := primary.corner(centre, corner, radius)
		next := primary.corner(centre, corner+spaceOne, radius)
		lowHere := maths.V(here.X, -primary.thickness, here.Z)
		lowNext := maths.V(next.X, -primary.thickness, next.Z)
		skin = append(skin, face{top, here, next},
			face{bottom, lowNext, lowHere})
		skin = quad(skin, here, lowHere, lowNext, next)
	}
	return skin
}

// corner places a panel vertex on the common shallow figure.
func (primary hexagonalPrimary) corner(centre maths.Vec,
	corner int, radius float64,
) maths.Vec {
	angle := fullTurn * float64(corner%hexagonCorners) / hexagonCorners
	offset := maths.V(radius*math.Cos(angle), spaceZero,
		-radius*math.Sin(angle))
	at := centre.Add(offset)
	at.Y = primary.height(at)
	return at
}
