package workspace

import (
	"strings"
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/janearc/slayer/ferruccio/internal/draft"
)

func TestPhysicalFilmPanelEditsRealGaugeAndUndoes(t *testing.T) {
	old := "film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}"
	rig := "film: {width: 640, rays: 1, frame: {width: 36, height: 24}, lens: {focal: 50, stop: 8, focus: first}, shutter: 0.5, glimpses: 2}"
	source := strings.Replace(addRecipeFixture, old, rig, 1)
	document, err := draft.Open("rig.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work := libraryEventWorkspace(t)
	work.state.document = document
	work.editor.SetText(source)
	work.rebuild()
	if libraryWidget(work.root, "Physical film…") == nil {
		t.Fatal("physical film control missing")
	}
	work.toggleDisclosure("film")
	drawLibraryWindow(t, work)
	control := libraryWidget(work.root, "Physical frame")
	bounds := control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	drawLibraryWindow(t, work)
	overlays := work.ui.Window().OverlayContentWidgets()
	if len(overlays) != 1 {
		t.Fatalf("gauge menu count %d", len(overlays))
	}
	menu := overlays[0].(interface{ Bounds() geometry.Rect }).Bounds()
	clickLibraryPoint(work, geometry.Pt(menu.Min.X+20, menu.Min.Y+60))
	if work.play.gaugeChoice != 1 {
		t.Fatalf("gauge choice %d", work.play.gaugeChoice)
	}
	control = libraryWidget(work.root, "Set frame")
	bounds = control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	if size, ok := document.PhysicalFrame(); !ok || size != [2]float64{5.79, 4.01} || work.pending.document != nil || work.Running() {
		t.Fatalf("frame %v %v", size, ok)
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != source {
		t.Fatal("physical frame undo failed")
	}
}

func TestPhysicalFilmControlFitsSmallestWindow(t *testing.T) {
	old := "film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}"
	rig := "film: {width: 640, rays: 1, frame: {width: 36, height: 24}, lens: {focal: 50, stop: 8, focus: first}, shutter: 0.5, glimpses: 2}"
	source := strings.Replace(addRecipeFixture, old, rig, 1)
	document, err := draft.Open("rig.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	work := libraryEventWorkspace(t)
	work.state.document = document
	work.editor.SetText(source)
	work.rebuild()
	ctx := uitest.NewMockContext()
	ctx.WindowSizeVal = smallestWindow
	work.root.Layout(ctx, geometry.Loose(smallestWindow))
	for _, id := range []string{"Physical film…", "Add…", "Made from", "Describe · D"} {
		point, found := centreOf(work.root, id, geometry.Point{})
		if !found || point.X < 0 || point.Y < 0 || point.X > smallestWindow.Width || point.Y > smallestWindow.Height {
			t.Fatalf("%s found %v at %v", id, found, point)
		}
	}
	work.toggleDisclosure("film")
	work.root.Layout(ctx, geometry.Loose(smallestWindow))
	for _, id := range []string{"Physical frame", "Set frame"} {
		p, ok := centreOf(work.root, id, geometry.Point{})
		if !ok || p.X < 0 || p.Y < 0 || p.X > smallestWindow.Width || p.Y > smallestWindow.Height {
			t.Fatalf("%s found %v at %v", id, ok, p)
		}
	}
}
