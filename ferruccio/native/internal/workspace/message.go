package workspace

import (
	"strconv"
	"strings"

	"github.com/gogpu/ui/core/dialog"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
)

const (
	messageDefaultDialogHeight float32 = 160
	workingDetailsDialogWidth  float32 = 1040
	messageMinDialogHeight     float32 = 120
	messageContentWidthInset   float32 = 48 // pinned dialog surface uses 24px on each side
	messageContentHeight       float32 = 48 // pinned dialog surface reserves 52px title and 60px actions
	messageReservedHeight      float32 = messageDefaultDialogHeight - messageContentHeight
	messageLineHeightRatio     float32 = 1.2
	messageGlyphWidthRatio     float32 = 1.2 // conservative upper bound for ordinary diagnostic glyphs
	messageSummaryInset        float32 = 36  // right pane uses 18px padding on both sides
)

// message presents complete error details in short wrapped pages with copy access.
func (work *Workspace) message(title, text string, extra ...dialog.Action) {
	work.showMessagePage(title, text, "", 0, extra)
}

// showMessagePage builds the selected page and its copy/navigation actions.
func (work *Workspace) showMessagePage(title, text, notice string, page int, extra []dialog.Action) {
	fontSize := work.size
	if fontSize <= 0 {
		fontSize = 16
	}
	pageLines := messagePageLines(fontSize, work.ui.Window().WindowSize().Height)
	dialogWidth := appearance.DialogPainter{Theme: work.theme, Size: work.size}.DialogMaxWidth()
	copyText := text
	groups := []string{text}
	if notice != "" {
		dialogWidth = workingDetailsDialogWidth
		groups = strings.SplitAfter(notice, " · ")
		if text != "" {
			groups = append([]string{text + "\n"}, groups...)
			copyText += "\n\n"
		}
		copyText += notice
	}
	maxRunes := work.messageLineRunes(fontSize, dialogWidth)
	pageRows := paginateMessageGroups(groups, maxRunes, pageLines)
	pages := len(pageRows)
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	pageTitle := title
	if pages > 1 {
		pageTitle += " · " + messagePageLabel(page, pages)
	}
	actions := make([]dialog.Action, 0, 4+len(extra))
	// Keep navigation in the same right-aligned positions when a dialog
	// adds actions. GoGPU estimates click widths from label byte counts.
	actions = append(actions, extra...)
	requestedPage := page
	if page > 0 {
		actions = append(actions, dialog.Action{Label: "Back", Variant: dialog.VariantOutlined, OnClick: func() {
			requestedPage = page - 1
		}})
	}
	actions = append(actions, dialog.Action{Label: "Copy", Variant: dialog.VariantOutlined, OnClick: func() {
		widget.ClipboardWrite(copyText)
	}})
	if page+1 < pages {
		actions = append(actions, dialog.Action{Label: "Next", Variant: dialog.VariantOutlined, OnClick: func() {
			requestedPage = page + 1
		}})
	}
	actions = append(actions, dialog.Action{Label: "Close", Variant: dialog.VariantOutlined})

	body := &messageBody{
		WidgetBase: *widget.NewWidgetBase(),
		lines:      pageRows[page],
		fontSize:   fontSize,
	}
	dialog.New(
		dialog.Title(pageTitle),
		dialog.MaxWidth(dialogWidth),
		dialog.Content(body),
		dialog.Actions(actions...),
		dialog.OnClose(func() {
			// Removing the old dialog removes overlays above it. Show the
			// requested page only after the old dialog closes.
			if requestedPage != page {
				work.showMessagePage(title, text, notice, requestedPage, extra)
			}
		}),
		dialog.PainterOpt(appearance.DialogPainter{Theme: work.theme, Size: work.size}),
	).Show(work.ui.Window().Context())
	work.host.RequestRedraw()
}

// messageLineRunes keeps each rendered row within the dialog's content width.
func (work *Workspace) messageLineRunes(fontSize, dialogWidth float32) int {
	windowWidth := work.ui.Window().WindowSize().Width
	availableWidth := windowWidth * 0.9
	if dialogWidth > availableWidth {
		dialogWidth = availableWidth
	}
	contentWidth := dialogWidth - messageContentWidthInset
	width := int(contentWidth / (fontSize * messageGlyphWidthRatio))
	if width < 1 {
		return 1
	}
	return width
}

// messagePageLines fits whole text rows inside the dialog's bounded content area.
func messagePageLines(fontSize, windowHeight float32) int {
	dialogHeight := messageDefaultDialogHeight
	maxHeight := windowHeight * 0.9
	if dialogHeight > maxHeight {
		dialogHeight = maxHeight
	}
	if dialogHeight < messageMinDialogHeight {
		dialogHeight = messageMinDialogHeight
	}
	contentHeight := dialogHeight - messageReservedHeight
	lineHeight := fontSize * messageLineHeightRatio
	lines := int(contentHeight / lineHeight)
	if lines < 1 {
		return 1
	}
	return lines
}

// wrapMessageLines breaks long paragraphs without dropping their characters.
func wrapMessageLines(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	paragraphs := strings.Split(text, "\n")
	lines := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		runes := []rune(paragraph)
		if len(runes) == 0 {
			lines = append(lines, "")
			continue
		}
		for len(runes) > width {
			cut := width
			for i := width - 1; i > 0; i-- {
				if runes[i] == ' ' || runes[i] == '\t' {
					cut = i + 1
					break
				}
			}
			lines = append(lines, string(runes[:cut]))
			runes = runes[cut:]
		}
		lines = append(lines, string(runes))
	}
	return lines
}

// paginateMessageGroups keeps each limitation on one page when it fits.
func paginateMessageGroups(groups []string, width, rowsPerPage int) [][]string {
	pages := [][]string{nil}
	for _, group := range groups {
		lines := wrapMessageLines(group, width)
		current := len(pages) - 1
		if len(pages[current])+len(lines) <= rowsPerPage {
			pages[current] = append(pages[current], lines...)
			continue
		}
		if len(pages[current]) == 0 {
			pages = pages[:current]
		}
		for i := 0; i < messagePageCount(len(lines), rowsPerPage); i++ {
			pages = append(pages, messagePage(lines, i, rowsPerPage))
		}
	}
	return pages
}

// messagePageCount returns the number of bounded pages needed for wrapped rows.
func messagePageCount(lineCount, pageSize int) int {
	if lineCount <= 0 || pageSize <= 0 {
		return 0
	}
	return (lineCount + pageSize - 1) / pageSize
}

// messagePage returns one page's rows, clamped to the available line range.
func messagePage(lines []string, page, pageSize int) []string {
	if page < 0 || pageSize <= 0 {
		return nil
	}
	start := page * pageSize
	if start >= len(lines) {
		return nil
	}
	end := start + pageSize
	if end > len(lines) {
		end = len(lines)
	}
	return lines[start:end]
}

// messagePageLabel formats a human-readable one-based page counter.
func messagePageLabel(page, pages int) string {
	return strconv.Itoa(page+1) + "/" + strconv.Itoa(pages)
}

// shortDiagnostic returns a one-line preview while full details stay in the dialog.
func (work *Workspace) shortDiagnostic() *primitives.TextWidget {
	return primitives.TextFn(func() string {
		return summarizeDiagnostic(work.diagnostic, work.size)
	}).FontSize(work.size).Color(appearance.Ink).MaxLines(1)
}

// summarizeDiagnostic makes a changing diagnostic safe for the narrow inline preview.
func summarizeDiagnostic(diagnostic string, size float32) string {
	maxRunes := messageSummaryRunes(size)
	text := strings.Join(strings.Fields(diagnostic), " ")
	runes := []rune(text)
	if len(runes) > maxRunes {
		if maxRunes == 1 {
			text = "…"
		} else {
			text = string(runes[:maxRunes-1]) + "…"
		}
	}
	return text
}

// messageSummaryRunes leaves conservative space for one line in the right pane.
func messageSummaryRunes(size float32) int {
	if size <= 0 {
		size = 16
	}
	max := int((480 - messageSummaryInset) / (size * messageGlyphWidthRatio))
	if max < 1 {
		return 1
	}
	return max
}

// messageBody draws one bounded page of wrapped diagnostic lines.
type messageBody struct {
	widget.WidgetBase
	lines    []string
	fontSize float32
}

// Layout fills the supplied viewport with its bounded page.
func (body *messageBody) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(constraints.MaxWidth, constraints.MaxHeight))
	body.SetBounds(geometry.FromPointSize(body.Position(), size))
	return size
}

// Draw clips the page to its viewport and draws each row separately.
func (body *messageBody) Draw(_ widget.Context, canvas widget.Canvas) {
	bounds := body.Bounds()
	canvas.PushClip(bounds)
	defer canvas.PopClip()
	lineHeight := body.fontSize * messageLineHeightRatio
	for row, line := range body.lines {
		top := bounds.Min.Y + float32(row)*lineHeight
		if top >= bounds.Max.Y {
			break
		}
		lineBounds := geometry.Rect{Min: geometry.Pt(bounds.Min.X, top), Max: geometry.Pt(bounds.Max.X, top+lineHeight)}
		canvas.DrawText(line, lineBounds, body.fontSize, appearance.Ink, false, widget.TextAlignLeft)
	}
}

// Event leaves page navigation to dialog actions.
func (*messageBody) Event(_ widget.Context, _ event.Event) bool { return false }

// Children returns nil because a page draws its rows directly.
func (*messageBody) Children() []widget.Widget { return nil }
