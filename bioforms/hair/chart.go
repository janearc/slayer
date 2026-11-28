package hair

import (
	"fmt"

	"github.com/janearc/slayer/maths"
)

// ChartFlow carries a heading from the authored chart into the rest
// triangle. The triangle corners supply the local change of coordinates.
// Without a chart, or at a collapsed chart face, the heading is already
// interpreted in the skin's rest space.
func (site Site) ChartFlow(direction maths.Vec) maths.Vec {
	from, to := site.chart.authored, site.chart.rest
	if !(from.area().Len() > 0) {
		return direction
	}
	return to.flow(from.coordinates(direction))
}

type chartFrame struct{ authored, rest triangle }

func (surface Surface) checkChart() error {
	if surface.Chart == nil {
		return nil
	}
	if len(surface.Chart) != len(surface.Points) {
		return fmt.Errorf("hair: need one chart point per skin vertex")
	}
	for i, point := range surface.Chart {
		if !finite(point.Len()) {
			return fmt.Errorf("hair: chart point %d is not finite", i)
		}
	}
	return nil
}

func (surface Surface) chartCorners(face [3]int) [3]maths.Vec {
	points := surface.Chart
	if points == nil {
		points = surface.Points
	}
	return [3]maths.Vec{points[face[0]], points[face[1]], points[face[2]]}
}

func (surface Surface) chartSite(site Site, face [3]int, rest triangle, normals []maths.Vec) Site {
	if surface.Chart == nil {
		return site
	}
	authored := edges(surface.Chart, face)
	site.Chart = surface.Chart[face[0]].
		Add(authored.first.Scale(site.Weights[1])).
		Add(authored.second.Scale(site.Weights[2]))
	site.chart = chartFrame{authored, rest}
	if normal := authored.normal(normals, face, site.Weights); normal != (maths.Vec{}) {
		site.ChartNormal = normal
	}
	return site
}

// ChartVector maps a displacement within the chart triangle and carries
// its perpendicular component along the rest skin's normal. This moves
// a nearby guide offset around one root. It does not retarget a full
// path across all the triangles it passes.
func (site Site) ChartVector(direction maths.Vec) maths.Vec {
	from, to := site.chart.authored, site.chart.rest
	area := from.area()
	magnitude := area.Len()
	if !(magnitude > 0 && finite(magnitude)) {
		return direction
	}
	normal := area.Scale(1 / magnitude)
	coefficients := [2]float64{
		direction.Cross(from.second).Dot(normal) / magnitude,
		from.first.Cross(direction).Dot(normal) / magnitude,
	}
	return to.flow(coefficients).Add(site.Normal.Scale(direction.Dot(normal)))
}
