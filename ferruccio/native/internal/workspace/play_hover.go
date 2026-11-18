package workspace

import (
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
)

// updateHover names only a current native surface under the pointer.
func (view *imageView) updateHover(point geometry.Point) {
	name := ""
	if view.navigable && view.hoverObject != nil {
		if at, ok := view.imagePoint(point); ok {
			name = view.hoverObject(float64(at.X), float64(at.Y))
		}
	}
	if name != view.hoverName || point != view.hoverPoint {
		view.hoverName = name
		view.hoverPoint = point
		view.SetNeedsRedraw(true)
	}
}

// drawHover makes the invitation local to the object rather than a lesson panel.
func (view *imageView) drawHover(canvas widget.Canvas) {
	if view.dragging || view.hoverName == "" {
		return
	}
	label := view.hoverName + " · drag to " + view.mode
	if view.mode == "look" {
		label = view.hoverName + " · drag to look around"
	}
	bounds := view.Bounds()
	width := min(bounds.Width()-24, canvas.MeasureText(label, view.fontSize, false)+24)
	if width <= 0 {
		return
	}
	x := max(bounds.Min.X+12, min(view.hoverPoint.X+18, bounds.Max.X-width-12))
	y := max(bounds.Min.Y+12, min(view.hoverPoint.Y+22, bounds.Max.Y-view.fontSize-28))
	box := geometry.Rect{Min: geometry.Pt(x, y), Max: geometry.Pt(x+width, y+view.fontSize+20)}
	// The label's own clip, as each camera card has, puts it in a scissor
	// group after the world image's. Sharing the image's clip, gg's GPU
	// renderer drew the card under the image and only the text above it.
	// Two points of room keep the outline's stroke inside the clip.
	canvas.PushClip(box.Expand(2))
	defer canvas.PopClip()
	canvas.DrawRoundRect(box, appearance.Ground, 12)
	canvas.StrokeRoundRect(box, appearance.Cyan, 12, 1.5)
	canvas.DrawText(label, box, view.fontSize, appearance.Ink, false, widget.TextAlignCenter)
}
