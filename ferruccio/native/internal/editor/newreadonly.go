package editor

// NewReadOnly displays selectable, scrollable source without accepting user edits.
// SetText and ApplyText remain programmatic refresh paths; Tab leaves focus.
func NewReadOnly(text string) *Widget {
	view := New(text, nil)
	view.readOnly = true
	return view
}

// SetEscapeHandler installs a disclosure return action for a focused editor.
// A nil handler restores the ordinary unhandled Escape behavior.
func (w *Widget) SetEscapeHandler(handler func()) { w.onEscape = handler }

// SetSettingsHandler installs the action for command and comma in a focused
// editor. A nil handler leaves the key unhandled.
func (w *Widget) SetSettingsHandler(handler func()) { w.onSettings = handler }
