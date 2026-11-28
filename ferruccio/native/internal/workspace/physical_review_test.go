package workspace

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gogpu/gg"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/render"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/set"
)

// TestPhysicalFilmAtMinimumWindow reaches the complete panel through its controls.
func TestPhysicalFilmAtMinimumWindow(t *testing.T) {
	const pinhole = "film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}"
	const rig = "film: {width: 640, rays: 1, frame: {width: 36, height: 24}, lens: {focal: 50, stop: 8, focus: first}, shutter: 0.5, glimpses: 2}"
	source := strings.Replace(addRecipeFixture, pinhole, rig, 1)
	for _, size := range []float32{16, 18, 20, 22, 24} {
		for _, selected := range []bool{false, true} {
			t.Run(fmt.Sprintf("type-%g-selected-%t", size, selected), func(t *testing.T) {
				work := libraryEventWorkspace(t)
				document, err := draft.Open("rig.yaml", []byte(source))
				if err != nil {
					t.Fatal(err)
				}
				work.state.document = document
				work.editor.SetText(source)
				if selected {
					read, err := set.Read("rig.yaml", []byte(source))
					if err != nil {
						t.Fatal(err)
					}
					work.play.selected = "scene.set.things/first"
					work.working.parsed, work.working.source, work.working.path = &read, source, work.state.path
				}
				work.size = size
				work.rebuild()
				work.ui.Window().HandleResize(int(smallestWindow.Width), int(smallestWindow.Height))
				drawMinimumFilmWindow(t, work)
				for _, id := range []string{"Add…", "Made from", "Describe · D", "Chroma theme", "Type size"} {
					filmControlCentre(t, work, id)
				}
				if selected {
					filmControlCentre(t, work, "Coordinates…")
					filmControlCentre(t, work, "Size…")
				}
				exerciseMinimumFilmPanel(t, work)
			})
		}
	}
}

// exerciseMinimumFilmPanel uses the gauge picker and lens fields in one session.
func exerciseMinimumFilmPanel(t *testing.T, work *Workspace) {
	t.Helper()
	clickLibraryPoint(work, filmControlCentre(t, work, "Physical film…"))
	if work.play.disclosure != "film" {
		t.Fatal("physical film button was unreachable")
	}
	drawMinimumFilmWindow(t, work)
	clickLibraryPoint(work, filmControlCentre(t, work, "Physical frame"))
	drawMinimumFilmWindow(t, work)
	overlays := work.ui.Window().OverlayContentWidgets()
	if len(overlays) != 1 {
		t.Fatalf("gauge menu count %d", len(overlays))
	}
	checkMinimumFilmBounds(t, "gauge menu", overlays[0].(interface{ Bounds() geometry.Rect }).Bounds())
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnd, 0, event.ModNone))
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	if work.play.gaugeChoice != len(physicalGauges()) {
		t.Fatalf("final gauge unreachable: selected %d", work.play.gaugeChoice)
	}
	clickLibraryPoint(work, filmControlCentre(t, work, "Set frame"))
	if size, _ := work.state.document.PhysicalFrame(); size == [2]float64{36, 24} {
		t.Fatal("set frame action was unreachable")
	}
	work.rebuild()
	drawMinimumFilmWindow(t, work)
	for _, id := range []string{"Focal length (mm)", "F-stop"} {
		clickLibraryPoint(work, filmControlCentre(t, work, id))
		work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyA, 0, event.ModSuper))
		work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.Key4, '4', event.ModNone))
	}
	clickLibraryPoint(work, filmControlCentre(t, work, "Set lens"))
	if lens, ok := work.state.document.PhysicalLens(); !ok || lens != (draft.PhysicalLens{FocalMM: 4, FStop: 4}) {
		t.Fatalf("lens controls were unreachable: %+v", lens)
	}
	clickLibraryPoint(work, filmControlCentre(t, work, "Return to world"))
	if work.play.disclosure != "" {
		t.Fatal("exit from physical film panel was unreachable")
	}
}

// drawMinimumFilmWindow uses the same viewport for layout and offscreen drawing.
func drawMinimumFilmWindow(t *testing.T, work *Workspace) {
	t.Helper()
	width, height := int(smallestWindow.Width), int(smallestWindow.Height)
	work.ui.Window().Frame()
	context := gg.NewContext(width, height)
	defer context.Close()
	work.ui.Window().DrawTo(render.NewCanvas(context, width, height))
	if err := context.FlushGPU(); err != nil {
		t.Fatal(err)
	}
}

// filmControlCentre finds a control and verifies its entire pointer target.
func filmControlCentre(t *testing.T, work *Workspace, id string) geometry.Point {
	t.Helper()
	control, ok := libraryWidget(work.root, id).(interface{ ScreenBounds() geometry.Rect })
	if !ok {
		t.Fatalf("missing control %q", id)
	}
	bounds := control.ScreenBounds()
	checkMinimumFilmBounds(t, id, bounds)
	return geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2)
}

// checkMinimumFilmBounds detects clipped and empty pointer targets.
func checkMinimumFilmBounds(t *testing.T, id string, bounds geometry.Rect) {
	t.Helper()
	if bounds.Min.X < 0 || bounds.Min.Y < 0 || bounds.Max.X > smallestWindow.Width ||
		bounds.Max.Y > smallestWindow.Height || bounds.Width() <= 0 || bounds.Height() <= 0 {
		t.Fatalf("%s has bounds %v in %v", id, bounds, smallestWindow)
	}
}
