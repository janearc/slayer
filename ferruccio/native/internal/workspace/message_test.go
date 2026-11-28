package workspace

import (
	"strings"
	"testing"

	"github.com/gogpu/gg"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/render"
	"github.com/gogpu/ui/widget"
)

// TestWrapMessageLinesPreservesLongDiagnostics guards against dropped error text.
func TestWrapMessageLinesPreservesLongDiagnostics(t *testing.T) {
	const width = 16
	input := "schema error: " + strings.Repeat("long_unbroken_token", 5) + "\nexpected: value"
	lines := wrapMessageLines(input, width)
	if len(lines) < 4 {
		t.Fatalf("wrapped into %d lines, want long details split across lines", len(lines))
	}
	for i, line := range lines {
		if got := len([]rune(line)); got > width {
			t.Errorf("line %d has %d runes, want at most %d: %q", i, got, width, line)
		}
	}
	if got := strings.Join(lines, ""); got != strings.ReplaceAll(input, "\n", "") {
		t.Fatalf("wrapping lost or changed diagnostic text: got %q", got)
	}
}

// TestMessagePagesCoverAllWrappedLines checks navigation across the full detail.
func TestMessagePagesCoverAllWrappedLines(t *testing.T) {
	lines := wrapMessageLines(strings.Repeat("validation detail ", 20), 12)
	const pageSize = 5
	var displayed []string
	for page := 0; page < messagePageCount(len(lines), pageSize); page++ {
		displayed = append(displayed, messagePage(lines, page, pageSize)...)
	}
	if strings.Join(displayed, "\n") != strings.Join(lines, "\n") {
		t.Fatal("page navigation skipped or duplicated wrapped diagnostic lines")
	}
}

// TestMessagePageHeightTracksReadingSize keeps pages inside the dialog viewport.
func TestMessagePageHeightTracksReadingSize(t *testing.T) {
	if got := messagePageLines(16, 768); got != 2 {
		t.Fatalf("16px page rows = %d, want 2 rows in the bounded dialog area", got)
	}
	if got := messagePageLines(24, 768); got != 1 {
		t.Fatalf("24px page rows = %d, want 1 row in the bounded dialog area", got)
	}
}

// TestMessageBodyDrawsEveryPageWithinDialogWidth measures actual rendered rows.
func TestMessageBodyDrawsEveryPageWithinDialogWidth(t *testing.T) {
	text := "schema validation failed: " + strings.Repeat("unexpected long path/segment ", 20)
	for _, size := range []float32{16, 24} {
		work := &Workspace{ui: app.New(), size: size}
		width := work.messageLineRunes(size, 560)
		lines := wrapMessageLines(text, width)
		pageSize := messagePageLines(size, 768)
		context := gg.NewContext(560, 160)
		canvas := render.NewCanvas(context, 560, 160)
		var drawn []string
		for page := 0; page < messagePageCount(len(lines), pageSize); page++ {
			body := &messageBody{WidgetBase: *widget.NewWidgetBase(), lines: messagePage(lines, page, pageSize), fontSize: size}
			body.SetBounds(geometry.NewRect(0, 0, 512, messageContentHeight))
			body.Draw(widget.NewContext(), canvas)
			for _, line := range body.lines {
				if width := canvas.MeasureText(line, size, false); width > 512 {
					t.Errorf("%gpx row %q measures %0.1fpx, want at most 512px", size, line, width)
				}
				drawn = append(drawn, line)
			}
		}
		if err := context.FlushGPU(); err != nil {
			context.Close()
			t.Fatalf("flush real diagnostic canvas: %v", err)
		}
		context.Close()
		if got := strings.Join(drawn, ""); got != text {
			t.Errorf("%gpx pages lost or changed diagnostic text", size)
		}
	}
}

// TestShortDiagnosticIsBoundedAndSingleLine prevents the inline preview overflowing.
func TestShortDiagnosticIsBoundedAndSingleLine(t *testing.T) {
	work := &Workspace{size: 24, diagnostic: "schema failed:\n" + strings.Repeat("界long diagnostic ", 100)}
	got := work.shortDiagnostic().Content()
	maxRunes := messageSummaryRunes(work.size)
	if strings.ContainsAny(got, "\r\n") || len([]rune(got)) > maxRunes {
		t.Fatalf("summary is not a bounded single line: %q", got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("truncated summary should signal omitted detail: %q", got)
	}
}

// TestMessageGroupsPreserveLongLimitations keeps the fallback pager lossless.
func TestMessageGroupsPreserveLongLimitations(t *testing.T) {
	groups := []string{"First", "This sentence needs several rows at a larger reading size.", "Last"}
	pages := paginateMessageGroups(groups, 16, 1)
	var shown strings.Builder
	for _, rows := range pages {
		if len(rows) != 1 {
			t.Fatalf("page has %d rows, want one", len(rows))
		}
		shown.WriteString(rows[0])
	}
	if want := strings.Join(groups, ""); shown.String() != want {
		t.Fatalf("pages recovered %q, want %q", shown.String(), want)
	}
}
