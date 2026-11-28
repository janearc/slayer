package workspace

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/uitest"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
)

func TestSourceDrawerKeyboardTraversalSurvivesRebuild(t *testing.T) {
	work := playWorkspace(t)
	win := work.ui.Window()
	win.HandleResize(int(smallestWindow.Width), int(smallestWindow.Height))
	win.Frame()
	ctx := win.Context()
	ctx.RequestFocus(libraryWidget(work.root, "Source"))
	press := func(key event.Key, mods event.Modifiers) {
		work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, key, 0, mods))
		work.ui.HandleEvent(event.NewKeyEvent(event.KeyRelease, key, 0, mods))
	}
	press(event.KeyEnter, event.ModNone)
	if work.play.disclosure != "source" {
		t.Fatal("keyboard activation did not open Source")
	}
	closeButton := libraryWidget(work.root, "Return to world")
	if closeButton == nil || ctx.FocusedWidget() != closeButton {
		t.Fatal("opening Source did not move focus into the drawer")
	}
	work.refreshLayout = true
	work.Tick()
	win.Frame()
	closeButton = libraryWidget(work.root, "Return to world")
	if ctx.FocusedWidget() != closeButton {
		t.Fatal("redrawing Source lost its focused return control")
	}
	press(event.KeyTab, event.ModNone)
	if ctx.FocusedWidget() != work.editor {
		t.Fatal("Tab did not reach the source editor")
	}
	work.refreshLayout = true
	work.Tick()
	win.Frame()
	if ctx.FocusedWidget() != work.editor || !work.editor.IsFocused() {
		t.Fatal("redrawing Source lost its focused editor")
	}
	canvas := &uitest.MockCanvas{}
	work.editor.Draw(ctx, canvas)
	if !hasSourceFocusRing(canvas) {
		t.Fatal("focused source editor drew no Chroma focus outline")
	}
	press(event.KeyTab, event.ModShift)
	closeButton = libraryWidget(work.root, "Return to world")
	if ctx.FocusedWidget() != closeButton {
		t.Fatal("Shift-Tab did not return to the drawer control")
	}
	canvas.Reset()
	work.editor.Draw(ctx, canvas)
	if hasSourceFocusRing(canvas) {
		t.Fatal("unfocused source editor retained its focus outline")
	}
	canvas.Reset()
	closeButton.Draw(ctx, canvas)
	if !hasSourceFocusRing(canvas) {
		t.Fatal("focused Return to world drew no Chroma focus outline")
	}
	press(event.KeyEnter, event.ModNone)
	if work.play.disclosure != "" || ctx.FocusedWidget() != work.viewer {
		t.Fatal("Return to world did not close Source and focus the viewport")
	}
}

func hasSourceFocusRing(canvas *uitest.MockCanvas) bool {
	for _, stroke := range canvas.StrokeRoundRects {
		if stroke.Color == appearance.Cyan && stroke.StrokeWidth == 2 {
			return true
		}
	}
	return false
}
