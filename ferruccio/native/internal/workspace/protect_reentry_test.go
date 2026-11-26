package workspace

import (
	"testing"

	"github.com/gogpu/ui/event"
)

func TestDirtyProtectionKeepsOnePromptAndAllowsAnotherAfterCancel(t *testing.T) {
	work := libraryEventWorkspace(t)
	work.state.document.Replace(work.state.document.Text() + "\n# unsaved\n")
	called := 0
	work.protect(func() { called++ })
	work.protect(func() { called += 10 })
	if got := len(work.ui.Window().OverlayContentWidgets()); got != 1 {
		t.Fatalf("repeated request opened %d prompts, want one", got)
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEscape, 0, event.ModNone))
	if called != 0 || len(work.ui.Window().OverlayContentWidgets()) != 0 {
		t.Fatal("cancel did not leave the draft and action alone")
	}
	work.protect(func() { called++ })
	if got := len(work.ui.Window().OverlayContentWidgets()); got != 1 {
		t.Fatalf("request after cancel opened %d prompts, want one", got)
	}
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyTab, 0, event.ModShift))
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	if called != 1 || len(work.ui.Window().OverlayContentWidgets()) != 0 {
		t.Fatalf("discard called the action %d times or left a prompt open", called)
	}
	work.protect(func() {})
	if got := len(work.ui.Window().OverlayContentWidgets()); got != 1 {
		t.Fatalf("request after discard opened %d prompts, want one", got)
	}
}

func TestCleanProtectionRunsTheAction(t *testing.T) {
	work := libraryEventWorkspace(t)
	called := false
	work.protect(func() { called = true })
	if !called || len(work.ui.Window().OverlayContentWidgets()) != 0 {
		t.Fatal("a clean draft did not continue without a prompt")
	}
}

func TestRepeatedOpenRequestsKeepOneDirtyDraftPrompt(t *testing.T) {
	work := libraryEventWorkspace(t)
	work.state.document.Replace(work.state.document.Text() + "\n# unsaved\n")
	work.requestOpen()
	work.Tick()
	work.requestOpen()
	work.Tick()
	if got := len(work.ui.Window().OverlayContentWidgets()); got != 1 {
		t.Fatalf("repeated open requests left %d prompts, want one", got)
	}
}
