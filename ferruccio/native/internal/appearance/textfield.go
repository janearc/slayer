package appearance

import (
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/widget"
)

// TextFieldPainter fits single-line input to the current colourway and size.
type TextFieldPainter struct {
	textfield.DefaultPainter
	Size float32
}

// TextFieldFontSize uses the workspace's selected type size.
func (p TextFieldPainter) TextFieldFontSize() float32 { return p.Size }

// PaintTextField supplies the current colourway to the standard field painter.
func (p TextFieldPainter) PaintTextField(canvas widget.Canvas, state *textfield.PaintState) {
	shown := *state
	shown.FontSize = p.Size
	shown.ColorScheme = textfield.TextFieldColorScheme{
		Background: Ground, Border: Border, FocusBorder: Cyan, ErrorBorder: Pink,
		TextColor: Ink, Placeholder: Ink, CursorColor: Cyan, DisabledBg: Panel,
		DisabledFg: Border, SelectionBg: Raised, ErrorText: Pink,
	}
	p.DefaultPainter.PaintTextField(canvas, &shown)
}
