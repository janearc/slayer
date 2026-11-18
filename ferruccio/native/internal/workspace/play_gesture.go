package workspace

import (
	"github.com/gogpu/ui/geometry"
	"math"
)

// imagePoint maps a pointer to the actual uncropped image, excluding letterbox.
func (view *imageView) imagePoint(point geometry.Point) (geometry.Point, bool) {
	if view.source == nil || !finitePoint(point) {
		return geometry.Point{}, false
	}
	rect := view.imageBounds()
	if rect.IsEmpty() || !rect.Contains(point) {
		return geometry.Point{}, false
	}
	return geometry.Pt((point.X-rect.Min.X)/rect.Width(), (point.Y-rect.Min.Y)/rect.Height()), true
}

// imageBounds is shared by selection and presentation; it never crops pixels.
func (view *imageView) imageBounds() geometry.Rect {
	bounds := view.Bounds()
	aspect := view.aspect
	if aspect <= 0 && view.source != nil && view.source.Bounds().Dy() > 0 {
		aspect = float64(view.source.Bounds().Dx()) / float64(view.source.Bounds().Dy())
	}
	if bounds.IsEmpty() || aspect <= 0 || math.IsNaN(aspect) || math.IsInf(aspect, 0) {
		return geometry.Rect{}
	}
	width, height := float64(bounds.Width()), float64(bounds.Height())
	if width/height > aspect {
		width = height * aspect
	} else {
		height = width / aspect
	}
	w, h := float32(max(1, int(width))), float32(max(1, int(height)))
	origin := geometry.Pt(bounds.Min.X+(bounds.Width()-w)/2, bounds.Min.Y+(bounds.Height()-h)/2)
	return geometry.Rect{Min: origin, Max: geometry.Pt(origin.X+w, origin.Y+h)}
}
