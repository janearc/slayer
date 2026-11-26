package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

func TestOpenShortcutAcrossRoot(t *testing.T) {
	called := 0
	root := newShortcutRoot(primitives.Box(), func() { called++ })
	if !root.Event(widget.NewContext(), event.NewKeyEvent(event.KeyPress, event.KeyO, 0, event.ModSuper)) || called != 1 {
		t.Fatal("command O did not request open")
	}
	root.Event(widget.NewContext(), event.NewKeyEvent(event.KeyPress, event.KeyO, 'o', event.ModNone))
	if called != 1 {
		t.Fatal("ordinary O requested open")
	}
}

func TestOpenShortcutReachesWorkspace(t *testing.T) {
	ui := app.New()
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), ui, Options{Examples: "../../../examples", Composition: "play", SuppressAutomaticPreview: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { work.clearNavigation(); os.RemoveAll(filepath.Dir(work.state.path)) })
	ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyO, 'o', event.ModNone))
	if work.wantOpen.Load() {
		t.Fatal("ordinary O requested open")
	}
	ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyO, 0, event.ModSuper))
	if !work.wantOpen.Load() {
		t.Fatal("command O did not reach the workspace")
	}
	work.wantOpen.Store(false)
	ui.Window().Context().RequestFocus(work.editor)
	ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyO, 0, event.ModSuper))
	if !work.wantOpen.Load() {
		t.Fatal("command O did not reach the workspace with the editor focused")
	}
}

func TestOpenShortcutRootSupportsRepaintBoundary(t *testing.T) {
	root := newShortcutRoot(primitives.Box(), nil)
	if _, ok := any(root).(interface{ SetRepaintBoundary(bool) }); !ok {
		t.Fatal("shortcut root cannot be made a repaint boundary")
	}
}
