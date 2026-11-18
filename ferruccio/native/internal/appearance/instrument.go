package appearance

import (
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// InstrumentPainter gives the three world intentions a common tactile form.
// Its glyphs are drawn with native vectors so missing font glyphs cannot hide them.
type InstrumentPainter struct {
	ButtonPainter
	Kind string
}

// ButtonHeight supplies a larger target for the primary manipulation controls.
func (p InstrumentPainter) ButtonHeight(size button.Size) float32 { return p.ButtonFontSize(size) + 36 }

// ButtonRadius gives the intention controls a distinct capsule silhouette.
func (InstrumentPainter) ButtonRadius() float32 { return 26 }

// ButtonPadding reserves space for the vector instrument mark and readable label.
func (p InstrumentPainter) ButtonPadding(button.Size) (float32, float32) { return 24, 18 }

// PaintButton retains Chroma state pairs and draws a labelled movement instrument.
func (p InstrumentPainter) PaintButton(canvas widget.Canvas, state button.PaintState) {
	if state.Bounds.IsEmpty() {
		return
	}
	style := p.resolveButton(state)
	radius := p.ButtonRadius()
	canvas.DrawRoundRect(state.Bounds, style.background, radius)
	canvas.StrokeRoundRect(state.Bounds, style.border, radius, style.borderWidth)
	center := geometry.Pt(state.Bounds.Min.X+22, (state.Bounds.Min.Y+state.Bounds.Max.Y)/2)
	p.glyph(canvas, center, style.foreground)
	label := state.Bounds
	label.Min.X += 38
	label.Max.X -= 10
	canvas.DrawText(state.Text, label, p.ButtonFontSize(state.Size), style.foreground, false, widget.TextAlignCenter)
	if state.Focused && !state.Disabled {
		canvas.StrokeRoundRect(state.Bounds.Expand(2), style.focusRing, radius+2, 2)
	}
}

// glyph names an intention through a simple vector mechanism beside its text.
func (p InstrumentPainter) glyph(canvas widget.Canvas, c geometry.Point, ink widget.Color) {
	line := func(x0, y0, x1, y1 float32) {
		canvas.DrawLine(geometry.Pt(c.X+x0, c.Y+y0), geometry.Pt(c.X+x1, c.Y+y1), ink, 1.5)
	}
	switch p.Kind {
	case "move":
		line(-9, 0, 9, 0)
		line(0, -9, 0, 9)
		line(-9, 0, -5, -3)
		line(-9, 0, -5, 3)
		line(9, 0, 5, -3)
		line(9, 0, 5, 3)
		line(0, -9, -3, -5)
		line(0, -9, 3, -5)
		line(0, 9, -3, 5)
		line(0, 9, 3, 5)
	case "turn":
		canvas.StrokeRoundRect(geometry.Rect{Min: geometry.Pt(c.X-8, c.Y-8), Max: geometry.Pt(c.X+8, c.Y+8)}, ink, 8, 1.5)
		line(8, -1, 3, -4)
		line(8, -1, 10, -6)
	case "look":
		canvas.StrokeRoundRect(geometry.Rect{Min: geometry.Pt(c.X-10, c.Y-6), Max: geometry.Pt(c.X+10, c.Y+6)}, ink, 6, 1.5)
		canvas.StrokeRoundRect(geometry.Rect{Min: geometry.Pt(c.X-3, c.Y-3), Max: geometry.Pt(c.X+3, c.Y+3)}, ink, 3, 1.5)
	}
}
