package workspace

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/geometry"
)

func TestPhotoCompositionsCanOpenTheWorkingView(t *testing.T) {
	for _, composition := range []string{"compose", "observe"} {
		t.Run(composition, func(t *testing.T) {
			path := checkingFile(t, checkingSource)
			work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{
				File: path, Composition: composition, SuppressAutomaticPreview: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			defer work.Close()
			work.editor.ApplyText(checkingSource + "# Keep this unsaved edit.\n")
			edited := work.state.document.Text()
			if err := work.Navigate(5, 0, 0); err == nil || !strings.Contains(err.Error(), "Choose Play") {
				t.Fatalf("photo camera message = %v", err)
			}
			work.ui.Window().HandleResize(1600, 1000)
			drawLibraryWindow(t, work)
			if libraryWidget(work.root, "Zoom +") != nil {
				t.Fatal("photograph advertised an unavailable camera control")
			}
			play, ok := libraryWidget(work.root, "Play").(interface{ ScreenBounds() geometry.Rect })
			if !ok {
				t.Fatal("composition has no Play action")
			}
			bounds := play.ScreenBounds()
			clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
			if work.composition != "play" || work.working.mode != "solid" || work.viewer.selectObject == nil {
				t.Fatal("Play did not start the working viewport")
			}
			if work.state.path != path || work.state.document.Text() != edited || !work.state.document.Dirty() {
				t.Fatal("entering Play replaced the unsaved draft")
			}
		})
	}
}

func TestCatalogueKeepsYAMLEditingWithoutPlay(t *testing.T) {
	path, err := filepath.Abs("../../../examples/subaru-settings.yaml")
	if err != nil {
		t.Fatal(err)
	}
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{
		File: path, Composition: "compose", SuppressAutomaticPreview: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer work.Close()
	before := work.state.document.Text()
	work.enterPlay()
	if work.composition != "compose" {
		t.Fatal("catalogue entered the unsupported working view")
	}
	work.ui.Window().HandleResize(1600, 1000)
	drawLibraryWindow(t, work)
	if libraryWidget(work.root, "Play") != nil {
		t.Fatal("Compose offers Play for a catalogue")
	}
	edit, ok := libraryWidget(work.root, "Edit YAML").(interface{ ScreenBounds() geometry.Rect })
	if !ok {
		t.Fatal("catalogue lost its Edit YAML action")
	}
	bounds := edit.ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	if work.composition != "observe" || work.state.document.Text() != before {
		t.Fatal("Edit YAML did not open the same catalogue in Observe")
	}
	if libraryWidget(work.root, "Play") != nil {
		t.Fatal("Observe offers Play for a catalogue")
	}
}
