package studio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
)

func interactionWorkspace(t *testing.T) *Workspace {
	t.Helper()
	application := test.NewApp()
	t.Cleanup(application.Quit)
	application.Settings().SetTheme(Theme())
	window := application.NewWindow("Atelier interaction test")
	workspace, err := New(Options{Window: window, Study: "venus"})
	if err != nil || workspace == nil {
		t.Fatalf("New: %v", err)
	}
	window.SetContent(workspace.Content())
	return workspace
}

func TestRawYAMLEditAndInspectorSelectionsStayReciprocal(t *testing.T) {
	workspace := interactionWorkspace(t)
	raw := strings.Replace(workspace.document.Text(), "renderer: gpu\nquality: preview", "renderer: cpu\nquality: exact", 1)
	workspace.editor.SetText(raw)
	if workspace.quality.Selected != "exact" || workspace.renderer.Selected != "cpu" {
		t.Fatalf("raw YAML did not update inspector: quality=%q renderer=%q", workspace.quality.Selected, workspace.renderer.Selected)
	}
	workspace.quality.SetSelected("preview")
	workspace.renderer.SetSelected("gpu")
	if !strings.Contains(workspace.editor.Text, "quality: preview") || !strings.Contains(workspace.editor.Text, "renderer: gpu") {
		t.Fatalf("inspector selections did not update raw YAML: %s", workspace.editor.Text)
	}
	if workspace.document.Text() != workspace.editor.Text {
		t.Fatal("editor and document contain different source")
	}
}

func TestReservedAreaNavigationPreservesDraftAndSetPicture(t *testing.T) {
	workspace := interactionWorkspace(t)
	workspace.editor.SetText(strings.Replace(workspace.document.Text(), "quality: preview", "quality: exact", 1))
	before := workspace.document.Text()
	wasDirty := workspace.document.Dirty()
	picture := workspace.picture
	if !wasDirty {
		t.Fatal("setup did not create an unsaved draft")
	}
	for _, area := range []string{"cameras", "sequence", "music"} {
		test.Tap(workspace.navigationButtons[area])
		if len(workspace.view.Objects) != 1 || workspace.view.Objects[0] == workspace.setContent {
			t.Fatalf("%s navigation did not show its reserved view", area)
		}
		test.Tap(workspace.navigationButtons["set"])
		if len(workspace.view.Objects) != 1 || workspace.view.Objects[0] != workspace.setContent {
			t.Fatalf("return from %s did not restore set view", area)
		}
		if workspace.picture != picture {
			t.Fatalf("return from %s replaced the set picture", area)
		}
		if workspace.document.Text() != before || workspace.editor.Text != before || workspace.document.Dirty() != wasDirty {
			t.Fatalf("%s navigation changed draft text or dirty state", area)
		}
	}
}

func TestInvalidRawYAMLDisablesInspectorWithoutDiscardingText(t *testing.T) {
	workspace := interactionWorkspace(t)
	invalid := "# Keep this broken draft\nscene: [unfinished"
	workspace.editor.SetText(invalid)
	if workspace.document.Text() != invalid || workspace.editor.Text != invalid {
		t.Fatal("invalid YAML was discarded")
	}
	for name, control := range map[string]interface{ Disabled() bool }{
		"quality":  workspace.quality,
		"renderer": workspace.renderer,
		"preset":   workspace.preset,
	} {
		if !control.Disabled() {
			t.Errorf("%s control remained enabled for invalid YAML", name)
		}
	}
}

func TestLoadFileOverLimitLeavesOriginalWorkspaceUntouched(t *testing.T) {
	workspace := interactionWorkspace(t)
	workspace.editor.SetText(strings.Replace(workspace.document.Text(), "quality: preview", "quality: exact", 1))
	document := workspace.document
	before := document.Text()
	wasDirty := document.Dirty()
	path := filepath.Join(t.TempDir(), "oversized.yaml")
	if err := os.WriteFile(path, []byte(strings.Repeat(" ", (4<<20)+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := workspace.LoadFile(path); err == nil || !strings.Contains(err.Error(), "4 MiB") {
		t.Fatalf("LoadFile error = %v; want a 4 MiB size refusal", err)
	}
	if workspace.document != document || workspace.document.Text() != before || workspace.editor.Text != before || workspace.document.Dirty() != wasDirty {
		t.Fatal("oversized file changed the original document or editor")
	}
}
