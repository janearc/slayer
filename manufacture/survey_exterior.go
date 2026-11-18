package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// surveyTier slopes a service wing's long sides and outer end inward. The
// inner end stays vertical beside the separate telescope enclosure.
type surveyTier struct{ width, height, depth, inset float64 }

// skin deforms the eight corners of a closed box, preserving winding.
func (tier surveyTier) skin(fine cut) surface {
	skin := boxShape(tier.width, tier.height, tier.depth).skin(fine)
	for face := range skin {
		for corner, point := range skin[face] {
			point.X += tier.width * spaceHalf
			point.Z += tier.depth * spaceHalf
			if point.Y > 0 {
				if point.X > tier.width*spaceHalf {
					point.X -= tier.inset
				}
				if point.Z > tier.depth*spaceHalf {
					point.Z -= tier.inset
				} else {
					point.Z += tier.inset
				}
			}
			skin[face][corner] = point
		}
	}
	return skin
}

// panelDetails follows the actual polygon facets, leaving the observing slit
// open. Seams, louvers and shutter housings are photographic scenery cues.
func (survey surveyOptics) panelDetails(assembly *instrumentAssembly,
	radius, base, top float64,
) {
	defer assembly.detailSince(len(assembly.object.parts))
	opening := groundValue("opening-deg") * maths.Degree
	sides := int(facilityValue("facets"))
	seam := facilityValue("panel-seam")
	roof := top - facilityValue("roof-height")
	point := func(index int) maths.Vec {
		angle := opening*spaceHalf + (fullTurn-opening)*float64(index)/float64(sides)
		return maths.V(radius*math.Sin(angle), 0, -radius*math.Cos(angle))
	}
	for side := range sides {
		left, right := point(side), point(side+1)
		assembly.rod(left.Add(maths.YAxis.Scale(base+seam)), left.Add(maths.YAxis.Scale(roof-seam)), seam, surveyCladding)
		middle := left.Add(right).Scale(spaceHalf)
		tangent := right.Sub(left).Scale(facilityValue("vent-width-share") * spaceHalf)
		rows := int(facilityValue("vent-rows"))
		for row := range rows {
			y := base + (roof-base)*float64(row+1)/float64(rows+1)
			for slat := range int(facilityValue("vent-slats")) {
				centre := middle.Add(maths.YAxis.Scale(y + float64(slat)*facilityValue("vent-pitch")))
				assembly.rod(centre.Sub(tangent), centre.Add(tangent), facilityValue("vent-radius"), surveyCladding)
			}
		}
	}
	width, depth := facilityValue("shutter-width"), facilityValue("shutter-depth")
	for _, side := range []int{0, sides} {
		edge := point(side)
		x := edge.X
		if x < 0 {
			x -= width
		}
		assembly.box(width, roof-base, depth, maths.V(x, base, edge.Z-depth*spaceHalf), surveyCladding)
	}
}
