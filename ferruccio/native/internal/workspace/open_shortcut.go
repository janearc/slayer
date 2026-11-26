package workspace

import (
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// shortcutRoot is a window root that intercepts open before its child.
type shortcutRoot struct {
	widget.WidgetBase
	content widget.Widget
	open    func()
	save    func()
}

// newShortcutRoot makes a window root that routes Open before its content.
func newShortcutRoot(content widget.Widget, open, save func()) *shortcutRoot {
	root := &shortcutRoot{content: content, open: open, save: save}
	root.SetVisible(true)
	root.SetEnabled(true)
	root.AddChild(content)
	return root
}

// Layout sizes the content and the shortcut root together.
func (root *shortcutRoot) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := root.content.Layout(ctx, constraints)
	root.SetBounds(geometry.NewRect(0, 0, size.Width, size.Height))
	return size
}

// Draw paints the content through the widget drawing pipeline.
func (root *shortcutRoot) Draw(ctx widget.Context, canvas widget.Canvas) {
	widget.DrawChild(root.content, ctx, canvas)
}

// Event handles the Open shortcut and forwards other input to the content.
func (root *shortcutRoot) Event(ctx widget.Context, input event.Event) bool {
	if key, ok := input.(*event.KeyEvent); ok && key.KeyType == event.KeyPress && key.Key == event.KeyO && key.Modifiers().Has(event.ModSuper) && root.open != nil {
		root.open()
		return true
	}
	if key, ok := input.(*event.KeyEvent); ok && key.KeyType == event.KeyPress && key.Key == event.KeyS && key.Modifiers().Has(event.ModSuper) && !key.Modifiers().HasAny(event.ModCtrl|event.ModAlt|event.ModShift) && root.save != nil {
		root.save()
		return true
	}
	// GoGPU text fields currently recognize Control for editing shortcuts.
	// On macOS those shortcuts use Command; translate only for a focused
	// text field, leaving the source editor and the app's shortcuts alone.
	if key, ok := input.(*event.KeyEvent); ok && key.KeyType == event.KeyPress && key.Modifiers().Has(event.ModSuper) && !key.Modifiers().HasAny(event.ModCtrl|event.ModAlt|event.ModShift) {
		if field, ok := ctx.FocusedWidget().(*textfield.Widget); ok {
			switch key.Key {
			case event.KeyA, event.KeyC, event.KeyX, event.KeyV:
				translated := event.NewKeyEvent(key.KeyType, key.Key, 0, event.ModCtrl)
				return field.Event(ctx, translated)
			}
		}
	}
	return root.content.Event(ctx, input)
}

// Children returns the content for the widget tree.
func (root *shortcutRoot) Children() []widget.Widget { return []widget.Widget{root.content} }
