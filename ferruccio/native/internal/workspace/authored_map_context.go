package workspace

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
)

func (plan *authoredPlan) caption() string {
	span, unit := plan.span, "m"
	if span >= 1000 {
		span, unit = span/1000, "km"
	}
	direction := "north up"
	if plan.angle != 0 {
		direction = "rotated"
	}
	caption := fmt.Sprintf("Authored map · %.3g %s across · %s", span, unit, direction)
	if plan.readOnly {
		return caption + " · job’s scene · open scene to edit"
	}
	return caption + " · elevation unavailable"
}

// drawContext paints authored bounds behind marks. A stand area remains a
// footprint of the region in source, independent of what it generates.
func (plan *authoredPlan) drawContext(pixels *image.RGBA, selected string) {
	if plan.terrain != nil {
		plan.drawExtent(pixels, *plan.terrain, mapColour(appearance.Ink).C)
	}
	standInk := mapColour(appearance.Border).C
	for id, area := range plan.areas {
		if id != selected {
			plan.drawExtent(pixels, *area, standInk)
		}
	}
	if area := plan.areas[selected]; area != nil {
		plan.drawExtent(pixels, *area, mapColour(appearance.Pink).C)
	}
}

func (plan *authoredPlan) drawExtent(pixels *image.RGBA, extent draft.MapRectangle, ink color.Color) {
	x0, y0 := plan.project(extent.West, extent.South)
	x1, y1 := plan.project(extent.West, extent.North)
	x2, y2 := plan.project(extent.East, extent.North)
	x3, y3 := plan.project(extent.East, extent.South)
	drawMapLine(pixels, x0, y0, x1, y1, ink)
	drawMapLine(pixels, x1, y1, x2, y2, ink)
	drawMapLine(pixels, x2, y2, x3, y3, ink)
	drawMapLine(pixels, x3, y3, x0, y0, ink)
}

// drawMapLine walks only the visible part of the dominant axis, leaving
// image bounds to clip the other coordinate. Work cannot grow with zoom.
func drawMapLine(pixels *image.RGBA, x0, y0, x1, y1 float64, ink color.Color) {
	size := float64(pixels.Rect.Dx())
	if math.IsNaN(x0+y0+x1+y1) || math.IsInf(x0+y0+x1+y1, 0) {
		return
	}
	if max(x0, x1) < 0 || min(x0, x1) > 1 || max(y0, y1) < 0 || min(y0, y1) > 1 {
		return
	}
	steep := math.Abs(y1-y0) > math.Abs(x1-x0)
	if steep {
		x0, y0, x1, y1 = y0, x0, y1, x1
	}
	if x0 > x1 {
		x0, y0, x1, y1 = x1, y1, x0, y0
	}
	x0, x1 = math.Round(x0*size), math.Round(x1*size)
	start, end := max(x0, 0), min(x1, size-1)
	if start > end {
		return
	}
	rise := 0.0
	if x1 != x0 {
		rise = (y1 - y0) / (x1 - x0)
	}
	for x := start; x <= end; x++ {
		y := math.Round((y0 + (x-x0)*rise) * size)
		if y < 0 || y >= size || math.IsNaN(y) {
			continue
		}
		if steep {
			pixels.Set(int(y), int(x), ink)
		} else {
			pixels.Set(int(x), int(y), ink)
		}
	}
}
