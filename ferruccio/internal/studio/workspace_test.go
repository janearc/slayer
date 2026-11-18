package studio

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
)

// The inspector and text editor must operate on the same source document.
func TestInspectorEditsDocumentAndCannotDiscardBrokenText(t *testing.T) {
	application := test.NewApp()
	defer application.Quit()
	application.Settings().SetTheme(Theme())
	window := application.NewWindow("Atelier test")
	workspace, err := New(Options{Window: window, Study: "venus"})
	if err != nil || workspace == nil {
		t.Fatalf("New: %v", err)
	}
	window.SetContent(workspace.Content())
	workspace.quality.SetSelected("exact")
	if !strings.Contains(workspace.editor.Text, "quality: exact") {
		t.Fatal("dropdown did not update the editor")
	}
	workspace.editor.SetText("scene: [unfinished")
	workspace.quality.SetSelected("preview")
	if workspace.document.Text() != "scene: [unfinished" || workspace.editor.Text != "scene: [unfinished" {
		t.Fatal("a control silently discarded invalid work")
	}
}
