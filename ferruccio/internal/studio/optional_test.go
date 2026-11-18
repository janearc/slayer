package studio

import (
	"strings"
	"testing"
)

func TestUnspecifiedSettingsDoNotShowStaleValues(t *testing.T) {
	workspace := interactionWorkspace(t)
	before := workspace.document.Text()
	workspace.editor.SetText(strings.Replace(before, "quality: preview\n", "", 1))
	if workspace.quality.Selected != "" {
		t.Fatal("omitted setting displayed a stale value")
	}
	if strings.Contains(workspace.document.Text(), "quality:") {
		t.Fatal("display sync inserted an omitted setting")
	}
	workspace.renderer.SetSelected("gpu-native-shadows")
	if !strings.Contains(workspace.document.Text(), "renderer: gpu-native-shadows") {
		t.Fatal("schema-supported renderer could not be selected")
	}
	if strings.Contains(workspace.document.Text(), "quality:") {
		t.Fatal("editing renderer invented a quality setting")
	}
}
