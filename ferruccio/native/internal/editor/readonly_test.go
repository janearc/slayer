package editor

import (
	"go.uber.org/mock/gomock"
	"strings"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// Native construction text keeps the canonical editor's navigation, not editing.
func TestReadOnlyRefusesTypingDeletionCutPasteAndHistory(t *testing.T) {
	for _, attempt := range []struct {
		name string
		key  event.Key
		mods event.Modifiers
		r    rune
	}{
		{"typing", event.KeyA, event.ModNone, 'a'},
		{"backspace", event.KeyBackspace, event.ModNone, 0},
		{"delete", event.KeyDelete, event.ModNone, 0},
		{"enter", event.KeyEnter, event.ModNone, 0},
		{"cut", event.KeyX, event.ModSuper, 0},
		{"paste", event.KeyV, event.ModSuper, 0},
		{"undo", event.KeyZ, event.ModSuper, 0},
		{"redo", event.KeyZ, event.ModSuper | event.ModShift, 0},
	} {
		t.Run(attempt.name, func(t *testing.T) {
			clipboard := NewMockClipboardProvider(gomock.NewController(t))
			widget.RegisterClipboardProvider(clipboard)
			t.Cleanup(func() { widget.RegisterClipboardProvider(nil) })
			if attempt.name == "cut" {
				clipboard.EXPECT().ClipboardWrite("producer recipe").Return(nil).Times(1)
			}
			if attempt.name == "paste" {
				clipboard.EXPECT().ClipboardRead().Return("pasted replacement", nil).Times(1)
			}
			view := NewReadOnly("producer recipe")
			view.SetFocused(true)
			view.anchor, view.cursor = 0, len(view.text)
			calls := 0
			view.onChange = func(string) { calls++ }
			press(view, attempt.key, attempt.mods, attempt.r)
			if view.Text() != "producer recipe" || calls != 0 || len(view.undo) != 0 {
				t.Fatal("read-only user operation changed text, history or callbacks")
			}
		})
	}
}

func TestReadOnlyPreservesCopySelectionWheelAndKeyboardNavigation(t *testing.T) {
	clipboard := NewMockClipboardProvider(gomock.NewController(t))
	widget.RegisterClipboardProvider(clipboard)
	t.Cleanup(func() { widget.RegisterClipboardProvider(nil) })
	text := strings.Repeat("producer recipe with a sufficiently long line\n", 40)
	view := NewReadOnly(text)
	view.SetBounds(geometry.NewRect(0, 0, 180, 80))
	view.SetFocused(true)
	press(view, event.KeyA, event.ModSuper, 0)
	clipboard.EXPECT().ClipboardWrite(text).Return(nil).Times(1)
	press(view, event.KeyC, event.ModSuper, 0)
	press(view, event.KeyHome, event.ModNone, 0)
	before := view.cursor
	press(view, event.KeyUp, event.ModNone, 0)
	if view.cursor >= before {
		t.Fatal("read-only keyboard row navigation stopped")
	}
	if !view.Event(widget.NewContext(), event.NewWheelEvent(geometry.Pt(0, 50), geometry.Pt(40, 40), geometry.Pt(40, 40), event.ModNone)) || view.scrollY <= 0 {
		t.Fatal("read-only wheel did not scroll canonical editor")
	}
	if view.Event(widget.NewContext(), event.NewKeyEvent(event.KeyPress, event.KeyTab, 0, event.ModNone)) {
		t.Fatal("read-only Tab trapped focus navigation")
	}
	if view.Text() != text {
		t.Fatal("read-only navigation modified recipe")
	}
}

func TestReadOnlySetTextAndEscapeHandler(t *testing.T) {
	view := NewReadOnly("first producer")
	view.SetFocused(true)
	view.ApplyText("programmatic refresh")
	press(view, event.KeyZ, event.ModSuper, 0)
	if view.Text() != "programmatic refresh" {
		t.Fatal("read-only undo changed programmatically refreshed text")
	}
	view.SetText("updated producer")
	if view.Text() != "updated producer" {
		t.Fatal("programmatic producer refresh refused")
	}
	view.SetFocused(true)
	escaped := 0
	view.SetEscapeHandler(func() { escaped++ })
	if !view.Event(widget.NewContext(), event.NewKeyEvent(event.KeyPress, event.KeyEscape, 0, event.ModNone)) || escaped != 1 {
		t.Fatal("Escape did not dispatch the disclosure return handler")
	}
	if view.Text() != "updated producer" {
		t.Fatal("Escape changed recipe")
	}
	editable := New("editable", nil)
	editable.SetFocused(true)
	editable.SetEscapeHandler(func() { escaped++ })
	press(editable, event.KeyEscape, event.ModNone, 0)
	press(editable, event.KeyA, event.ModNone, 'a')
	if escaped != 2 || editable.Text() == "editable" {
		t.Fatal("Escape handler disabled ordinary source editing")
	}
}
