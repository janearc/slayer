package primates

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// thighColumn keeps one meridian on a fixed ray of the transverse section
// until the thigh profile takes over. Height and radius are ordered
// independently, so the saddle cannot swing the meridian across its neighbour.
type thighColumn struct {
	origin, radial               maths.Vec
	radius, fromRadius, toRadius float64
	top, bottom, fromY, toY      float64
}

func (column thighColumn) point(share float64) maths.Vec {
	radius := hermite(column.radius, 1, column.fromRadius, column.toRadius, share)
	point := column.origin.Add(column.radial.Scale(radius))
	point.Y = hermite(column.top, column.bottom, column.fromY, column.toY, share)
	return point
}

// Ordered endpoint derivatives keep the four Bézier controls between
// their neighbours and preserve the distal derivative where possible.
func orderedSlopes(start, end, from, to float64) (float64, float64) {
	distance := math.Abs(end - start)
	direction := math.Copysign(1, end-start)
	endSpeed := math.Min(3*distance, math.Max(0, direction*to))
	startSpeed := math.Min(3*distance-endSpeed, math.Max(0, direction*from))
	return direction * startSpeed, direction * endSpeed
}

func meridianTangent(wall, across, radial maths.Vec) maths.Vec {
	cross := across.X*radial.Z - across.Z*radial.X
	if math.Abs(cross) <= 1e-10 {
		return wall
	}
	drift := (wall.X*radial.Z - wall.Z*radial.X) / cross
	// Remove the azimuth component within the existing tangent plane.
	return wall.Sub(across.Scale(drift))
}

func thighColumns(model *humanMesh, limb limbSweep, boundary skinBoundary, angles rimAngles) []thighColumn {
	joinAt := limb.blendEnd
	section := limbSectionAt(limb.stations, joinAt)
	origin := section.centre
	const step = .0004
	before := limbSectionAt(limb.stations, joinAt-step)
	after := limbSectionAt(limb.stations, joinAt+step)
	columns := make([]thighColumn, len(boundary.rim))
	for i, index := range boundary.rim {
		root := model.vertices[index].point
		angle := angles.near[i]
		sine, cosine := math.Sincos(angle)
		radial := limb.across(section.bend).Scale(section.width * sine).
			Add(limb.forward(section.bend).Scale(section.depth * cosine))
		lengthSquared := radial.Dot(radial)
		radius := root.Sub(origin).Dot(radial) / lengthSquared
		target := origin.Add(radial)
		count := len(boundary.rim)
		prior := model.vertices[boundary.rim[(i+count-1)%count]].point
		next := model.vertices[boundary.rim[(i+1)%count]].point
		across := next.Sub(prior)
		enter := meridianTangent(boundary.inward[i], across, radial).Unit().
			Scale(math.Min(.037, .67*target.Sub(root).Len()))
		leave := limb.point(after, angle).Sub(limb.point(before, angle)).Scale(joinAt / (2 * step))
		across = limb.across(section.bend).Scale(section.width * cosine).
			Sub(limb.forward(section.bend).Scale(section.depth * sine))
		leave = meridianTangent(leave, across, radial)
		fromRadius, toRadius := orderedSlopes(radius, 1, enter.Dot(radial)/lengthSquared,
			leave.Dot(radial)/lengthSquared)
		fromY, toY := orderedSlopes(root.Y, target.Y, enter.Y, leave.Y)
		columns[i] = thighColumn{origin: origin, radial: radial, radius: radius,
			fromRadius: fromRadius, toRadius: toRadius, top: root.Y, bottom: target.Y,
			fromY: fromY, toY: toY}
	}
	return columns
}

// femaleThighSweep shares the existing graft and then grows downward in a
// common transverse chart before reverting to the normal leg sections.
func (model *humanMesh) femaleThighSweep(limb limbSweep, boundary skinBoundary) []int {
	joinAt := limb.blendEnd
	section := limbSectionAt(limb.stations, joinAt)
	angles := model.angles(boundary.rim, section.centre, limb.across(section.bend),
		limb.forward(section.bend), section.width, section.depth)
	columns := thighColumns(model, limb, boundary, angles)
	previous := boundary.rim
	for row := 1; row <= limb.rows; row++ {
		at := float64(row) / float64(limb.rows)
		section := limbSectionAt(limb.stations, at)
		round := maths.Smooth(joinAt, .52, at)
		site := limb.site
		site.attachment = maths.Smooth(0, joinAt, at)
		next := make([]int, len(previous))
		for i, column := range columns {
			point := limb.point(section, angles.angle(i, round))
			if at < joinAt {
				point = column.point(at / joinAt)
			}
			root := model.vertices[boundary.rim[i]]
			next[i] = model.limbPoint(point, limb.weight(at, point, root.weights), site)
		}
		model.connect(previous, next)
		previous = next
	}
	return turnRim(previous, angles.front)
}
