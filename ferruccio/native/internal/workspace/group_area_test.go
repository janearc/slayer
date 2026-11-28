package workspace

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gogpu/gg"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/render"
	"github.com/gogpu/ui/uitest"
	"github.com/janearc/slayer/ferruccio/internal/draft"
)

func checkRegionGuidance(canvas *uitest.MockCanvas, measure func(string, float32, bool) float32) error {
	var lines []string
	for _, row := range canvas.Texts {
		if len(lines) == 0 && !strings.HasPrefix(row.Text, "Metres east") {
			continue
		}
		if row.Text == "West" {
			break
		}
		width := measure(row.Text, row.FontSize, false)
		if width > 450 {
			return fmt.Errorf("%q needs %g points in a 450-point drawer", row.Text, width)
		}
		lines = append(lines, row.Text)
	}
	if !strings.HasSuffix(strings.Join(lines, ""), "fewer members.") {
		return fmt.Errorf("region guidance was incomplete: %q", lines)
	}
	return nil
}

func TestRegionGuidanceFitsTheDrawer(t *testing.T) {
	work := libraryEventWorkspace(t)
	source := strings.Replace(checkingSource, "setting: space", "setting: ground", 1)
	source = strings.Replace(source, "  film:", "    stands:\n      grove: {kind: yurt, count: 4, seed: 2, habit: scatter, area: {west: 20, south: 20, east: 80, north: 80}}\n  film:", 1)
	document, err := draft.Open("scene.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = document
	work.editor.SetText(source)
	work.play.selected = "scene.set.stands/grove"
	work.toggleDisclosure("region")
	work.ui.Window().HandleResize(int(smallestWindow.Width), int(smallestWindow.Height))
	metrics := gg.NewContext(450, 100)
	defer metrics.Close()
	measure := render.NewCanvas(metrics, 450, 100).MeasureText
	for _, size := range []float32{16, 18, 20, 22, 24} {
		work.size = size
		work.rebuild()
		work.ui.Window().Frame()
		canvas := &uitest.MockCanvas{}
		work.root.Draw(work.ui.Window().Context(), canvas)
		if err := checkRegionGuidance(canvas, measure); err != nil {
			t.Errorf("%g pt: %v", size, err)
		}
		filmControlCentre(t, work, "Set region")
	}
}

func TestGroupRegionPanelEditsWithoutStartingRender(t *testing.T) {
	source := strings.Replace(checkingSource, "setting: space", "setting: ground", 1)
	source = strings.Replace(source, "  film:", "    stands:\n      grove: {kind: yurt, count: 4, seed: 2, habit: scatter, area: {west: 20, south: 20, east: 80, north: 80}}\n  film:", 1)
	document, err := draft.Open("scene.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work := libraryEventWorkspace(t)
	work.state.document = document
	work.editor.SetText(source)
	work.play.selected = "scene.set.stands/grove"
	work.rebuild()
	if libraryWidget(work.root, "Region…") == nil {
		t.Fatal("region control missing")
	}
	work.toggleDisclosure("region")
	drawLibraryWindow(t, work)
	field, ok := libraryWidget(work.root, "East metres").(*textfield.Widget)
	if !ok {
		t.Fatal("east field missing")
	}
	field.SetText("100")
	control := libraryWidget(work.root, "Set region")
	if control == nil {
		t.Fatal("set region button missing")
	}
	bounds := control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	area, ok := document.GroupArea(work.play.selected)
	if !ok || area[2] != 100 || work.pending.document != nil || work.Running() {
		t.Fatalf("region = %v, editable %v", area, ok)
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != source {
		t.Fatal("undo did not restore source")
	}
	work.rebuild()
	field, ok = libraryWidget(work.root, "East metres").(*textfield.Widget)
	if !ok || field.Text() != "80" {
		t.Fatal("undo left stale region coordinates in the panel")
	}
	if err := document.SetGroupArea(work.play.selected, [4]float64{20, 20, 120, 80}); err != nil {
		t.Fatal(err)
	}
	work.rebuild()
	field, ok = libraryWidget(work.root, "East metres").(*textfield.Widget)
	if !ok || field.Text() != "120" {
		t.Fatal("source change left stale region coordinates in the panel")
	}
}
