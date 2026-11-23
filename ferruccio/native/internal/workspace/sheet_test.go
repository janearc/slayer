package workspace

import (
	"testing"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
)

// column is a column of text, as a settings panel is made.
func column() *primitives.BoxWidget {
	return primitives.VBox(primitives.Text("Cache folder"), primitives.Text("not set")).Gap(14).Padding(10).Width(300)
}

// smallestWindow is the smallest the window may be made, as main.go has it.
var smallestWindow = geometry.Sz(1280, 800)

// holds reports whether the tree under w holds a widget for which is true.
func holds(w widget.Widget, is func(widget.Widget) bool) bool {
	if is(w) {
		return true
	}
	for _, child := range w.Children() {
		if holds(child, is) {
			return true
		}
	}
	return false
}

func TestAColumnGivenOnlyBoundsAndDrawnDrawsNothingWhichIsWhyTheDialogCannotHoldOne(t *testing.T) {
	panel := column()
	panel.SetBounds(geometry.NewRect(100, 100, 400, 48))
	canvas := &uitest.MockCanvas{}
	panel.Draw(uitest.NewMockContext(), canvas)
	if len(canvas.Texts) != 0 {
		t.Fatalf("a column that was never laid out drew %d texts; the dialog's way of drawing content would have worked", len(canvas.Texts))
	}
}

func TestASheetFillsTheWindowAndDrawsAColumnOfText(t *testing.T) {
	ctx := uitest.NewMockContext()
	panel := column()
	made := newSheet(panel, nil)
	made.Layout(ctx, geometry.Loose(ctx.WindowSizeVal))
	if panel.Bounds() != geometry.NewRect(0, 0, ctx.WindowSizeVal.Width, ctx.WindowSizeVal.Height) {
		t.Fatalf("the content is at %v in a window of %v", panel.Bounds(), ctx.WindowSizeVal)
	}
	canvas := &uitest.MockCanvas{}
	made.Draw(ctx, canvas)
	uitest.AssertDrawnText(t, canvas, "Cache folder")
	uitest.AssertDrawnText(t, canvas, "not set")
}

func TestAButtonInASheetCanBePressed(t *testing.T) {
	ctx := uitest.NewMockContext()
	pressed := 0
	control := button.New(button.TextOpt("Choose"), button.OnClick(func() { pressed++ }))
	panel := primitives.VBox(control).Padding(10).Width(300)
	made := newSheet(panel, nil)
	made.Layout(ctx, geometry.Loose(ctx.WindowSizeVal))
	at := control.Bounds()
	centre := geometry.Pt((at.Min.X+at.Max.X)/2, (at.Min.Y+at.Max.Y)/2)
	made.Event(ctx, uitest.Click(centre.X, centre.Y))
	made.Event(ctx, uitest.Release(centre.X, centre.Y))
	if pressed != 1 {
		t.Fatalf("the button in the sheet was pressed %d times", pressed)
	}
}

func TestEscapeDismissesASheetAndNoOtherKeyDoes(t *testing.T) {
	ctx := uitest.NewMockContext()
	dismissed := 0
	made := newSheet(column(), func() { dismissed++ })
	made.Layout(ctx, geometry.Loose(ctx.WindowSizeVal))
	made.Event(ctx, uitest.KeyPress(event.KeyA, 0))
	if dismissed != 0 {
		t.Fatal("a letter dismissed the sheet")
	}
	if !made.Event(ctx, uitest.KeyPress(event.KeyEscape, 0)) || dismissed != 1 {
		t.Fatalf("Escape dismissed the sheet %d times", dismissed)
	}
}

func TestASheetWithNoDismissLeavesEscapeAlone(t *testing.T) {
	ctx := uitest.NewMockContext()
	made := newSheet(column(), nil)
	made.Layout(ctx, geometry.Loose(ctx.WindowSizeVal))
	if made.Event(ctx, uitest.KeyPress(event.KeyEscape, 0)) {
		t.Fatal("Escape was taken by a sheet that cannot be dismissed")
	}
}

func TestTheSettingsPageDrawsItsTitleFileAndEverySettingsLabel(t *testing.T) {
	work := settingsWorkspace(t, "renderer :: gpu\n")
	work.size, work.theme = 16, appearance.Theme()
	ctx := uitest.NewMockContext()
	ctx.WindowSizeVal = geometry.Sz(1600, 1000)
	made := newSheet(work.settingsPage(), nil)
	made.Layout(ctx, geometry.Loose(ctx.WindowSizeVal))
	canvas := &uitest.MockCanvas{}
	made.Draw(ctx, canvas)
	for _, want := range []string{"Settings", "Cache folder", "Source shelf", "Slayer", "Examples folder", "Theme", "Renderer", "Processors", "Close"} {
		uitest.AssertDrawnText(t, canvas, want)
	}
}

func TestWhileSettingsAreOpenTheWindowIsTheSheetAloneWithNoWidgetOfTheWindow(t *testing.T) {
	work := settingsWorkspace(t, "")
	work.size, work.theme = 16, appearance.Theme()
	work.settingsOpen = true
	root := work.layout()
	if _, isSheet := root.(*sheet); !isSheet {
		t.Fatalf("the window's root is %T, not the sheet", root)
	}
	isPicture := func(w widget.Widget) bool { _, is := w.(*imageView); return is }
	if holds(root, isPicture) {
		t.Fatal("the picture is in the tree under the sheet, where it could be drawn over it or take the pointer")
	}
}

func TestCloseAndTheTitleStayInsideTheSmallestWindowAtEveryTypeSize(t *testing.T) {
	for _, size := range []float32{16, 18, 20, 24} {
		work := settingsWorkspace(t, "cache :: /a/path\nshelf :: /a/shelf\nslayer :: /a/slayer\nexamples :: /an/examples\n")
		work.size, work.theme = size, appearance.Theme()
		ctx := uitest.NewMockContext()
		ctx.WindowSizeVal = smallestWindow
		made := newSheet(work.settingsPage(), nil)
		made.Layout(ctx, geometry.Loose(smallestWindow))
		canvas := &uitest.MockCanvas{}
		made.Draw(ctx, canvas)
		for _, drawn := range canvas.Texts {
			if (drawn.Text == "Close" || drawn.Text == "Settings") && (drawn.Bounds.Max.Y > smallestWindow.Height || drawn.Bounds.Min.Y < 0) {
				t.Fatalf("size %v: %q is at %v in a window of %v", size, drawn.Text, drawn.Bounds, smallestWindow)
			}
		}
	}
}

func TestEveryControlOfTheSheetCanBeFoundByItsIDAndLiesInsideTheWindow(t *testing.T) {
	work := settingsWorkspace(t, "shelf :: /a/shelf\n")
	work.size, work.theme = 16, appearance.Theme()
	ctx := uitest.NewMockContext()
	ctx.WindowSizeVal = geometry.Sz(1600, 1000)
	root := newSheet(work.settingsPage(), nil)
	root.Layout(ctx, geometry.Loose(ctx.WindowSizeVal))
	work.root, work.settingsOpen = root, true
	for _, id := range []string{"Close", "Clear source shelf", "Choose slayer", "Setting processors", "Setting renderer", "Setting theme"} {
		at, found := centreOf(work.root, id, geometry.Point{})
		if !found || at.X < 0 || at.Y < 0 || at.X > 1600 || at.Y > 1000 {
			t.Fatalf("%s: found %v at %v", id, found, at)
		}
	}
	if _, found := centreOf(work.root, "no such control", geometry.Point{}); found {
		t.Fatal("a control that is not there was found")
	}
}

func TestProofSettingsClickRefusesWhenTheSheetIsNotOpen(t *testing.T) {
	work := settingsWorkspace(t, "")
	if err := work.ProofSettingsClick("Close"); err == nil {
		t.Fatal("a press was sent to a sheet that is not open")
	}
}
