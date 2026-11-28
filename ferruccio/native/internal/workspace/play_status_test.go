package workspace

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
)

func TestRenderLogIncludesCompleteFooterStatus(t *testing.T) {
	work := libraryEventWorkspace(t)
	work.composition = "compose"
	work.status = "Draft saved with unchecked dependencies"
	_, _ = work.log.Write([]byte("producer transcript"))
	work.rebuild()
	work.ui.Window().HandleResize(int(smallestWindow.Width), int(smallestWindow.Height))
	work.ui.Window().Frame()
	work.showRenderLog()
	work.ui.Window().Frame()
	overlays := work.ui.Window().OverlayContentWidgets()
	if len(overlays) != 1 {
		t.Fatalf("Render log opened %d overlays", len(overlays))
	}
	canvas := &uitest.MockCanvas{}
	overlays[0].Draw(work.ui.Window().Context(), canvas)
	var drawn []string
	for _, row := range canvas.Texts {
		drawn = append(drawn, strings.TrimSpace(row.Text))
	}
	if !strings.Contains(strings.Join(drawn, " "), work.status) {
		t.Fatalf("render log omitted the footer status: %q", drawn)
	}
}

func TestPlayStatusFitsBesideAuthoringControls(t *testing.T) {
	for _, size := range []float32{16, 18, 20, 22, 24} {
		t.Run(fmt.Sprintf("type-%g", size), func(t *testing.T) {
			work := libraryEventWorkspace(t)
			work.size = size
			work.rebuild()
			work.ui.Window().HandleResize(int(smallestWindow.Width), int(smallestWindow.Height))
			work.working.failure = strings.Repeat("W", 300)
			work.reportWorkingFailure()
			work.invalidate()
			work.ui.Window().Frame()
			canvas := &uitest.MockCanvas{}
			work.root.Draw(work.ui.Window().Context(), canvas)
			filmControlCentre(t, work, "Add…")
			filmControlCentre(t, work, "Describe · D")
			assertStatusFits(t, canvas)
			clickLibraryPoint(work, filmControlCentre(t, work, "View details…"))
			if len(work.ui.Window().OverlayContentWidgets()) != 1 {
				t.Fatal("the full staging diagnostic was unreachable")
			}
		})
	}
}

func TestViewDetailsButtonsKeepTheCompleteWorkingNotice(t *testing.T) {
	work := libraryEventWorkspace(t)
	const notice = volumeBoundsNotice + ` · Light "North · Wide" is hidden`
	work.working.notice = notice
	work.showWorkingDetails()

	first, next := detailsPage(t, work, "Working view limitations · 1/3", "Next")
	clickLibraryPoint(work, next)
	middle, next := detailsPage(t, work, "Working view limitations · 2/3", "Next")
	clickLibraryPoint(work, next)
	last, back := detailsPage(t, work, "Working view limitations · 3/3", "Back")
	if !strings.Contains(first, "clipped to the working frame") || first+middle+last != notice {
		t.Fatalf("details pages recovered %q, %q, and %q, want all text with bounds together", first, middle, last)
	}
	clickLibraryPoint(work, back)
	_, back = detailsPage(t, work, "Working view limitations · 2/3", "Back")
	clickLibraryPoint(work, back)
	returned, close := detailsPage(t, work, "Working view limitations · 1/3", "Close")
	if returned != first {
		t.Fatalf("Back returned %q, want %q", returned, first)
	}
	clickLibraryPoint(work, close)
	if len(work.ui.Window().OverlayContentWidgets()) != 0 {
		t.Fatal("Close left a details page open")
	}
}

func detailsPage(t *testing.T, work *Workspace, title, action string) (string, geometry.Point) {
	t.Helper()
	window := work.ui.Window()
	window.Frame()
	overlays := window.OverlayContentWidgets()
	if len(overlays) != 1 {
		t.Fatalf("%s: got %d overlays, want one", title, len(overlays))
	}
	canvas := &uitest.MockCanvas{}
	overlays[0].Draw(window.Context(), canvas)
	var body strings.Builder
	foundTitle := title == ""
	var foundAction bool
	var centre geometry.Point
	for _, row := range canvas.Texts {
		switch row.Text {
		case title:
			foundTitle = true
		case action:
			foundAction = true
			centre = geometry.Pt((row.Bounds.Min.X+row.Bounds.Max.X)/2, (row.Bounds.Min.Y+row.Bounds.Max.Y)/2)
		case "Back", "Next", "Copy", "Close":
		default:
			body.WriteString(row.Text)
		}
	}
	if !foundTitle || !foundAction {
		t.Fatalf("wanted title %q and action %q, got %+v", title, action, canvas.Texts)
	}
	return body.String(), centre
}

func assertStatusFits(t *testing.T, canvas *uitest.MockCanvas) {
	t.Helper()
	for _, row := range canvas.Texts {
		if !strings.HasPrefix(row.Text, "Staging failed:") {
			continue
		}
		if width := canvas.MeasureText(row.Text, row.FontSize, false); width > row.Bounds.Width() {
			t.Fatalf("readout needs %.1f pixels in a %.1f pixel slot", width, row.Bounds.Width())
		}
		if !strings.HasSuffix(row.Text, "…") {
			t.Fatal("the readout does not indicate the omitted text")
		}
		return
	}
	t.Fatal("staging error summary was not drawn")
}

func TestStatusLineKeepsTextThatFits(t *testing.T) {
	canvas := &uitest.MockCanvas{}
	measure := func(text string) float32 { return canvas.MeasureText(text, 18, false) }
	const text = "E 7.00 m · N 4.00 m"
	if got := fitStatusLine(text, measure(text), measure); got != text {
		t.Fatalf("short readout was changed to %q", got)
	}
}
