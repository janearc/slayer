package primates

import (
	"math"

	"github.com/janearc/slayer/bioforms/hair"
	"github.com/janearc/slayer/maths"
)

// flow turns a rearward course in the scalp's tangent plane. On the
// forehead, the course leaves the hairline towards the crown before
// branching gently along the part. At the nape it runs downward.
func (spec ScalpHair) flow(site hair.Site, high, part, partWeight, jitter float64) maths.Vec {
	point, normal := site.Chart, site.ChartNormal
	along := scalpCourse(normal)
	across := along.Cross(normal).Unit()
	turn := .14 + partWeight*(.53*math.Tanh(part/.009)+.08)
	turn += jitter
	turn += spec.crownTurn(point, across, high)
	sine, cosine := math.Sincos(turn)
	return along.Scale(cosine).Add(across.Scale(sine))
}

// scalpCourse meets the front, dome and back as a continuous meridian.
// Near the ears the turning axis tilts to let hair settle at the
// sides. A posterior component keeps this lateral course defined
// where the axis approaches the local normal.
func scalpCourse(normal maths.Vec) maths.Vec {
	side := maths.Smooth(.56, .96, math.Abs(normal.X))
	axis := maths.V(1-.59*side, 0, math.Copysign(.88*side, normal.X))
	posterior := maths.V(0, 0, -1).Add(normal.Scale(normal.Z))
	return normal.Cross(axis).Add(posterior.Scale(.43 * side)).Unit()
}

// crownTurn bends the flow around a posterior crown without
// reversing it or removing roots at the centre of the bend.
func (spec ScalpHair) crownTurn(point, across maths.Vec, high float64) float64 {
	delta := maths.V(point.X-.016-spec.Part*.24, 0, point.Z+.036)
	distance := delta.Len()
	weight := high * bell(distance/.036) * maths.Smooth(0, .011, distance)
	hand := 1.
	if spec.CounterClockwise {
		hand = -1
	}
	around := maths.V(-delta.Z, 0, delta.X).Unit()
	return .63 * hand * weight * across.Dot(around)
}
