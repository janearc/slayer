package studio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/janearc/slayer/ferruccio/internal/preview"
)

func TestLoadFileOpensRealJobAndSetAndClearsMissionImage(t *testing.T) {
	for _, fixture := range []struct {
		name string
		kind string
	}{
		{"first-picture.yaml", "job"},
		{"webb-at-l2.yaml", "set"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			workspace := interactionWorkspace(t)
			if workspace.picture.Resource == nil {
				t.Fatal("setup has no mission reference image")
			}
			path := filepath.Join("..", "..", "examples", fixture.name)
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := workspace.LoadFile(path); err != nil {
				t.Fatal(err)
			}
			absolute, err := filepath.Abs(path)
			if err != nil {
				t.Fatal(err)
			}
			if workspace.editor.Text != string(want) || workspace.document.Text() != string(want) {
				t.Error("opened source bytes differ from the Slayer example")
			}
			if workspace.document.Kind() != fixture.kind {
				t.Errorf("document kind = %q; want %q", workspace.document.Kind(), fixture.kind)
			}
			if workspace.sourcePath != absolute {
				t.Errorf("source path = %q; want %q", workspace.sourcePath, absolute)
			}
			if workspace.picture.Resource != nil || workspace.picture.Image != nil || workspace.picture.File != "" {
				t.Error("opening a real document retained the mission image")
			}
			if !workspace.renderButton.Disabled() {
				t.Error("render action enabled without configured tools")
			}
		})
	}
}

func TestConfiguredToolsEnableRenderOnlyForValidOpenedSource(t *testing.T) {
	application := test.NewApp()
	defer application.Quit()
	application.Settings().SetTheme(Theme())
	window := application.NewWindow("Atelier configured tools test")
	workspace, err := New(Options{Window: window, Study: "venus", Tools: preview.Tools{Game: "game5", Slayer: "slayer"}})
	if err != nil || workspace == nil {
		t.Fatalf("New: %v", err)
	}
	window.SetContent(workspace.Content())
	if !workspace.renderButton.Disabled() {
		t.Fatal("render action enabled before a source file was opened")
	}
	if err := workspace.LoadFile(filepath.Join("..", "..", "examples", "first-picture.yaml")); err != nil {
		t.Fatal(err)
	}
	if workspace.renderButton.Disabled() {
		t.Fatal("render action stayed disabled for a valid opened job and configured tools")
	}
	workspace.editor.SetText("scene: [unfinished")
	if !workspace.renderButton.Disabled() || workspace.document.Text() != "scene: [unfinished" {
		t.Fatal("invalid opened source did not disable render while preserving the draft")
	}
}

func TestTextSizeChangesThemeAndCanvasWithoutChangingDocument(t *testing.T) {
	workspace := interactionWorkspace(t)
	workspace.editor.SetText(strings.Replace(workspace.document.Text(), "quality: preview", "quality: exact", 1))
	before := workspace.document.Text()
	wasDirty := workspace.document.Dirty()
	initialTitleSize := workspace.worldTitle.TextSize
	application := fyne.CurrentApp()
	if application == nil {
		t.Fatal("test workspace has no current Fyne app")
	}
	for _, size := range []float32{16, 24, 16, 24} {
		workspace.SetTextSize(size)
		if got := application.Settings().Theme().Size(theme.SizeNameText); got != size {
			t.Errorf("theme body size = %g; want %g", got, size)
		}
		if want := max(float32(16), size*0.9); (codeTheme{}).Size(theme.SizeNameText) != want {
			t.Errorf("code editor body size = %g; want %g", (codeTheme{}).Size(theme.SizeNameText), want)
		}
		if workspace.document.Text() != before || workspace.document.Dirty() != wasDirty {
			t.Error("changing text size changed document bytes or dirty state")
		}
	}
	first24TitleSize := workspace.worldTitle.TextSize
	workspace.SetTextSize(16)
	workspace.SetTextSize(24)
	if got := workspace.worldTitle.TextSize; got != first24TitleSize {
		t.Errorf("repeated 24-point title size = %g; first 24-point size was %g", got, first24TitleSize)
	}
	if first24TitleSize != initialTitleSize*24/18 {
		t.Errorf("24-point title size = %g; want scale from initial %g", first24TitleSize, initialTitleSize)
	}
}

func TestPrimaryActionFocusAndHoverKeepContrast(t *testing.T) {
	workspace := interactionWorkspace(t)
	button := workspace.renderButton
	button.Importance = widget.HighImportance
	selected := actionTheme{button: button}
	variant := theme.VariantDark
	focus := selected.Color(theme.ColorNameFocus, variant)
	hover := selected.Color(theme.ColorNameHover, variant)
	primary := fyne.CurrentApp().Settings().Theme().Color(theme.ColorNamePrimary, variant)
	if colorHex(focus) != colorHex(primary) || colorHex(hover) != colorHex(primary) {
		t.Fatal("high-importance focus and hover do not retain the primary action fill")
	}
	foreground := selected.Color(theme.ColorNameForegroundOnPrimary, variant)
	checkPair(t, foreground, focus, "primary action label on focus")
	checkPair(t, foreground, hover, "primary action label on hover")
}
