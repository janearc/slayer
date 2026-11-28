package workspace

import (
	"sort"
	"strings"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
)

// statusLine measures changing text against the width the toolbar assigns it.
type statusLine struct {
	*primitives.TextWidget
}

func (work *Workspace) statusLine(text func() string) *statusLine {
	return &statusLine{work.live(text).MaxLines(1)}
}

func (line *statusLine) Draw(_ widget.Context, canvas widget.Canvas) {
	bounds := line.Bounds()
	if !line.IsVisible() || bounds.IsEmpty() {
		return
	}
	text := strings.Join(strings.Fields(line.Content()), " ")
	size := line.Style().FontSize
	measure := func(text string) float32 { return canvas.MeasureText(text, size, false) }
	text = fitStatusLine(text, bounds.Width(), measure)
	if text == "" {
		return
	}
	canvas.PushClip(bounds)
	defer canvas.PopClip()
	canvas.DrawText(text, bounds, size, appearance.Ink, false, widget.TextAlignLeft)
}

func fitStatusLine(text string, width float32, measure func(string) float32) string {
	const ellipsis = "…"
	if measure(text) <= width {
		return text
	}
	if measure(ellipsis) > width {
		return ""
	}
	runes := []rune(text)
	overflow := sort.Search(len(runes), func(i int) bool {
		return measure(string(runes[:i+1])+ellipsis) > width
	})
	return strings.TrimRight(string(runes[:overflow]), " ") + ellipsis
}
