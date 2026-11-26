package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/ninjafilm"
)

func TestStockPanelEditsJobAndFreezesProducerInput(t *testing.T) {
	raw, err := os.ReadFile("../../../examples/first-picture.yaml")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "job.yaml")
	doc, err := draft.Open(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	work := libraryEventWorkspace(t)
	work.state.document, work.state.path = doc, path
	work.editor.SetText(string(raw))
	work.rebuild()
	if libraryWidget(work.root, "Stock…") == nil {
		t.Fatal("stock control missing")
	}
	work.toggleDisclosure("stock")
	drawLibraryWindow(t, work)
	control := libraryWidget(work.root, "Development stock")
	bounds := control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	drawLibraryWindow(t, work)
	overlays := work.ui.Window().OverlayContentWidgets()
	if len(overlays) != 1 {
		t.Fatalf("stock menu count %d", len(overlays))
	}
	menu := overlays[0].(interface{ Bounds() geometry.Rect }).Bounds()
	clickLibraryPoint(work, geometry.Pt(menu.Min.X+20, menu.Min.Y+60))
	if work.play.stockChoice != 1 {
		t.Fatalf("stock choice %d", work.play.stockChoice)
	}
	control = libraryWidget(work.root, "Set stock")
	bounds = control.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	names, err := ninjafilm.Stocks()
	if err != nil {
		t.Fatal(err)
	}
	authored, err := job.ReadSource(path, []byte(doc.Text()))
	if err != nil {
		t.Fatal(err)
	}
	stock, ok := authored.Stock()
	if !ok || stock.Name() != names[0] {
		t.Fatalf("producer stock %q %v", stock.Name(), ok)
	}
	plan, err := preview.Prepare(preview.Request{Path: path, Text: doc.Text(), Profile: preview.Profile{Full: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	frozen, err := os.ReadFile(plan.Input)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := job.ReadSource(plan.Input, frozen)
	if err != nil {
		t.Fatal(err)
	}
	saved, ok := parsed.Stock()
	if !ok || saved.Name() != names[0] {
		t.Fatalf("frozen stock %q %v", saved.Name(), ok)
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if doc.Text() != string(raw) {
		t.Fatal("stock undo failed")
	}
}

func TestStockControlsFitSmallestWindow(t *testing.T) {
	raw, err := os.ReadFile("../../../examples/first-picture.yaml")
	if err != nil {
		t.Fatal(err)
	}
	document, err := draft.Open("job.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	work := libraryEventWorkspace(t)
	work.state.document = document
	work.editor.SetText(string(raw))
	work.rebuild()
	ctx := uitest.NewMockContext()
	ctx.WindowSizeVal = smallestWindow
	work.root.Layout(ctx, geometry.Loose(smallestWindow))
	for _, id := range []string{"Stock…", "Add…", "Made from", "Describe · D"} {
		point, found := centreOf(work.root, id, geometry.Point{})
		if !found || point.X < 0 || point.Y < 0 || point.X > smallestWindow.Width || point.Y > smallestWindow.Height {
			t.Fatalf("%s found %v at %v", id, found, point)
		}
	}
	work.toggleDisclosure("stock")
	work.root.Layout(ctx, geometry.Loose(smallestWindow))
	for _, id := range []string{"Development stock", "Set stock"} {
		point, found := centreOf(work.root, id, geometry.Point{})
		if !found || point.X < 0 || point.Y < 0 || point.X > smallestWindow.Width || point.Y > smallestWindow.Height {
			t.Fatalf("%s found %v at %v", id, found, point)
		}
	}
}
