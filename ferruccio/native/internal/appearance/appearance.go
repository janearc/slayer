// Package appearance contains Atelier's GoGPU theme and painters.
package appearance

import (
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/core/dialog"
	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/theme/devtools"
	"github.com/gogpu/ui/widget"
)

// ButtonPainter draws clearly outlined primary and secondary buttons.
type ButtonPainter struct {
	Size    float32
	Primary bool
}

type buttonAppearance struct {
	background  widget.Color
	foreground  widget.Color
	border      widget.Color
	focusRing   widget.Color
	borderWidth float32
	radius      float32
}

// resolveButton chooses fills, labels and edges for the current button state.
func (p ButtonPainter) resolveButton(state button.PaintState) buttonAppearance {
	accent, foreground := Cyan, Cyan
	background := Ground
	if p.Primary {
		accent, foreground, background = Pink, selected.primaryInk, selected.primaryFill
	}

	switch state.Variant {
	case button.Outlined:
		background = Ground
		foreground = accent
	case button.TextOnly:
		background = Ground
		foreground = accent
	case button.Tonal:
		background = Ground
		foreground = accent
	}

	border, borderWidth := accent, float32(1.4)
	if p.Primary && state.Variant == button.Filled {
		border = Cyan
	}
	focusRing := Cyan
	if state.Disabled {
		background, foreground, border = Ground, selected.dim, Border
		borderWidth = 1.2
	} else if state.Pressed {
		if p.Primary {
			background = primaryPressed
		}
		border = secondaryPress
	} else if state.Hovered {
		if p.Primary {
			background = primaryHover
		}
		border = secondaryHover
	}
	if state.Focused && !state.Disabled {
		borderWidth = 1.8
	}
	if state.Background != nil && !state.Disabled {
		background = *state.Background
	}
	radius := float32(8)
	if state.Radius != nil {
		radius = *state.Radius
	}
	return buttonAppearance{background: background, foreground: foreground, border: border, focusRing: focusRing, borderWidth: borderWidth, radius: radius}
}

// PaintButton draws the fill, a contrasting outline, label, and optional focus ring.
func (p ButtonPainter) PaintButton(canvas widget.Canvas, state button.PaintState) {
	if state.Bounds.IsEmpty() {
		return
	}
	style := p.resolveButton(state)
	canvas.DrawRoundRect(state.Bounds, style.background, style.radius)
	canvas.StrokeRoundRect(state.Bounds, style.border, style.radius, style.borderWidth)
	canvas.DrawText(state.Text, state.Bounds, p.ButtonFontSize(state.Size), style.foreground, false, widget.TextAlignCenter)
	if state.Focused && !state.Disabled {
		canvas.StrokeRoundRect(state.Bounds.Expand(2), style.focusRing, style.radius+2, 2)
	}
}

// ButtonHeight gives each button variant enough room for its type size.
func (p ButtonPainter) ButtonHeight(size button.Size) float32 {
	padding := float32(16)
	switch size {
	case button.Small:
		padding = 12
	case button.Large:
		padding = 20
	}
	return p.ButtonFontSize(size) + padding
}

// ButtonPadding returns the horizontal and vertical inset for each button size.
func (p ButtonPainter) ButtonPadding(size button.Size) (float32, float32) {
	horizontal, vertical := float32(12), float32(6)
	switch size {
	case button.Small:
		horizontal, vertical = 10, 6
	case button.Large:
		horizontal, vertical = 16, 10
	}
	return horizontal, vertical
}

// ButtonFontSize scales labels from the configured reader size.
func (p ButtonPainter) ButtonFontSize(size button.Size) float32 {
	base := normalizedSize(p.Size)
	switch size {
	case button.Small:
		return max(minimumSize, base-2)
	case button.Large:
		return base + 2
	default:
		return base
	}
}

// ButtonRadius keeps button corners consistent across sizes.
func (ButtonPainter) ButtonRadius() float32 { return 8 }

// DropdownPainter replaces the upstream DevTools painter's fixed 13px labels.
type DropdownPainter struct {
	Size  float32
	Theme *devtools.Theme
}

// PaintTrigger draws a dropdown trigger with the configured type size.
func (p DropdownPainter) PaintTrigger(canvas widget.Canvas, state *dropdown.TriggerPaintState) {
	devtools.DropdownPainter{Theme: p.theme()}.PaintTrigger(fontCanvas{Canvas: canvas, size: normalizedSize(p.Size)}, state)
}

// PaintMenu draws dropdown choices with the configured type size.
func (p DropdownPainter) PaintMenu(canvas widget.Canvas, state *dropdown.MenuPaintState) {
	devtools.DropdownPainter{Theme: p.theme()}.PaintMenu(fontCanvas{Canvas: canvas, size: normalizedSize(p.Size)}, state)
}

// theme returns the supplied GoGPU theme or Atelier's default.
func (p DropdownPainter) theme() *devtools.Theme {
	if p.Theme != nil {
		return p.Theme
	}
	return Theme()
}

// DialogPainter replaces the upstream 13px action labels with the selected size.
type DialogPainter struct {
	Size  float32
	Theme *devtools.Theme
}

// PaintDialog draws a dialog using the configured type size.
func (p DialogPainter) PaintDialog(canvas widget.Canvas, state dialog.PaintState) {
	devtools.DialogPainter{Theme: p.theme()}.PaintDialog(fontCanvas{Canvas: canvas, size: normalizedSize(p.Size)}, state)
}

// DialogPadding returns comfortable inset spacing for dialog content.
func (p DialogPainter) DialogPadding() float32 { return 20 }

// DialogTitleHeight reserves room for a title at the current type size.
func (p DialogPainter) DialogTitleHeight() float32 { return normalizedSize(p.Size) * 1.6 }

// DialogMaxWidth bounds the default width of dialogs.
func (p DialogPainter) DialogMaxWidth() float32 { return 560 }

// theme returns the supplied GoGPU theme or Atelier's default.
func (p DialogPainter) theme() *devtools.Theme {
	if p.Theme != nil {
		return p.Theme
	}
	return Theme()
}

type fontCanvas struct {
	widget.Canvas
	size float32
}

// DrawText replaces upstream fixed label sizes while forwarding to the canvas.
func (c fontCanvas) DrawText(text string, bounds geometry.Rect, _ float32, color widget.Color, bold bool, align widget.TextAlign) {
	c.Canvas.DrawText(text, bounds, c.size, color, bold, align)
}

var (
	_ button.Painter       = ButtonPainter{}
	_ button.LayoutMetrics = ButtonPainter{}
	_ dropdown.Painter     = DropdownPainter{}
	_ dialog.Painter       = DialogPainter{}
	_ dialog.LayoutMetrics = DialogPainter{}
)
