package workspace

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

// sheet is the whole window while a sheet of settings or the like is up: it
// is the window's root, and holds the one panel of ordinary widgets, text,
// buttons, dropdowns and a scroll view among them.
//
// It is the root, and not an overlay, for two reasons found by reading and by
// picture. dialog.Widget cannot hold controls: it never lays out its content,
// sizes itself from its title and actions alone, and takes every mouse event.
//
// And an overlay is painted into the window with the rest, but the toolkit
// paints repaint boundaries, such as the window's own buttons and the picture,
// as textures after the immediate drawing, so a panel drawn as an overlay had
// the window's controls drawn over it and the window behind it undimmed.
//
// As the root, nothing else is in the tree: nothing can be drawn over the
// sheet and nothing else can take the pointer. The window's own widgets come
// back, as they were, when the layout is built again without it.
type sheet struct {
	widget.WidgetBase
	content *primitives.BoxWidget
	// dismiss is called on Escape.
	dismiss func()
}

// newSheet is a sheet around content. dismiss is what Escape does.
func newSheet(content *primitives.BoxWidget, dismiss func()) *sheet {
	made := &sheet{content: content, dismiss: dismiss}
	made.SetVisible(true)
	made.SetEnabled(true)
	made.AddChild(content)
	return made
}

// Layout fills the window with the content.
func (made *sheet) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(ctx.WindowSize())
	made.SetBounds(geometry.NewRect(0, 0, size.Width, size.Height))
	made.content.Layout(ctx, geometry.Tight(size))
	made.content.SetBounds(geometry.NewRect(0, 0, size.Width, size.Height))
	return size
}

// Draw draws the content.
func (made *sheet) Draw(ctx widget.Context, canvas widget.Canvas) {
	if canvas == nil {
		return
	}
	made.content.Draw(ctx, canvas)
}

// Event gives the content the event, and closes on Escape.
func (made *sheet) Event(ctx widget.Context, e event.Event) bool {
	if made.content.Event(ctx, e) {
		return true
	}
	if key, ok := e.(*event.KeyEvent); ok && key.KeyType == event.KeyPress && key.Key == event.KeyEscape && made.dismiss != nil {
		made.dismiss()
		return true
	}
	return false
}

// Children is the content.
func (made *sheet) Children() []widget.Widget { return []widget.Widget{made.content} }
