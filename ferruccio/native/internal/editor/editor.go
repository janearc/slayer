// Package editor provides a small multiline text editor for YAML drafts.
package editor

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
)

const (
	padX              float32 = 12
	padY              float32 = 10
	lineGap           float32 = 1.35
	minFont           float32 = 16
	maxFont           float32 = 36
	defaultFont       float32 = 16
	maxInsertBytes            = 4 << 20
	maxHistoryBytes           = 8 << 20
	maxHistoryEntries         = 100
)

// Widget is a rune-indexed multiline editor. Its fields stay private so text,
// cursor and undo state can only change through the editor's contract.
type Widget struct {
	widget.WidgetBase
	text            []rune
	cursor          int
	anchor          int
	fontSize        float32
	scrollX         float32
	scrollY         float32
	dragging        bool
	onChange        func(string)
	readOnly        bool
	onEscape        func()
	onSettings      func()
	undo            []string
	redo            []string
	lineStartsCache []int
	longestLine     int
	linesCached     bool
}

// New creates an editor initialized with text. Edits call onChange; SetText
// loads a replacement document without calling it or preserving undo history.
func New(text string, onChange func(string)) *Widget {
	return &Widget{
		WidgetBase: *widget.NewWidgetBase(),
		text:       []rune(normalizeNewlines(text)),
		anchor:     -1,
		fontSize:   defaultFont,
		onChange:   onChange,
	}
}

// Text returns the document as UTF-8 text.
func (w *Widget) Text() string { return string(w.text) }

// SetText replaces the document and clears cursor selection and undo history.
func (w *Widget) SetText(text string) {
	w.text = []rune(normalizeNewlines(text))
	w.cursor = len(w.text)
	w.anchor = -1
	w.undo = nil
	w.redo = nil
	w.resetLineCache()
	w.scrollX = 0
	w.scrollY = 0
	w.SetNeedsRedraw(true)
}

// ApplyText applies an external document transformation as one undoable edit.
func (w *Widget) ApplyText(text string) {
	text = normalizeNewlines(text)
	if text == string(w.text) {
		return
	}
	w.undo = appendHistory(w.undo, string(w.text))
	w.redo = nil
	w.text = []rune(text)
	if w.cursor > len(w.text) {
		w.cursor = len(w.text)
	}
	if w.cursor < 0 {
		w.cursor = 0
	}
	if w.anchor > len(w.text) {
		w.anchor = len(w.text)
	}
	if w.anchor < -1 {
		w.anchor = -1
	}
	w.resetLineCache()
	w.clampScroll()
	w.MarkNeedsLayout()
	w.SetNeedsRedraw(true)
	if w.onChange != nil {
		w.onChange(string(w.text))
	}
}

// Undo restores the latest draft snapshot using the editor's normal change path.
func (w *Widget) Undo(ctx widget.Context) { w.undoEdit(ctx) }

// Redo restores the latest undone draft using the editor's normal change path.
func (w *Widget) Redo(ctx widget.Context) { w.redoEdit(ctx) }

// SetFontSize changes the editor's logical font size, clamped to a readable range.
func (w *Widget) SetFontSize(size float32) {
	if size < minFont {
		size = minFont
	}
	if size > maxFont {
		size = maxFont
	}
	if w.fontSize != size {
		w.fontSize = size
		w.clampScroll()
		w.SetNeedsRedraw(true)
	}
}

// IsFocusable reports whether this enabled, visible editor can receive keys.
func (w *Widget) IsFocusable() bool { return w.IsEnabled() && w.IsVisible() }

// Children returns nil because the editor is a leaf widget.
func (w *Widget) Children() []widget.Widget { return nil }

// Layout fills the space its parent offers, with a useful minimum size.
func (w *Widget) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	width, height := c.MaxWidth, c.MaxHeight
	if width >= geometry.Infinity/2 {
		width = 320
	}
	if height >= geometry.Infinity/2 {
		height = 180
	}
	size := c.Constrain(geometry.Sz(width, height))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	w.clampScroll()
	return size
}

// Draw paints a clipped, scrollable text surface with selection and caret.
func (w *Widget) Draw(_ widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	colors := appearance.Theme().Colors
	canvas.DrawRoundRect(bounds, appearance.DecorativePanel, 6)
	canvas.DrawRoundRect(geometry.Rect{
		Min: geometry.Pt(bounds.Min.X+1, bounds.Min.Y+1),
		Max: geometry.Pt(bounds.Max.X-1, bounds.Max.Y-1),
	}, appearance.Ground, 5)
	canvas.StrokeRoundRect(bounds, appearance.Border, 6, 1)
	inner := geometry.Rect{
		Min: geometry.Pt(bounds.Min.X+1, bounds.Min.Y+1),
		Max: geometry.Pt(bounds.Max.X-1, bounds.Max.Y-1),
	}
	canvas.PushClip(inner)
	defer canvas.PopClip()

	starts := w.lineStarts()
	lineHeight := w.lineHeight()
	firstRow := int(w.scrollY / lineHeight)
	lastRow := int((w.scrollY+bounds.Height())/lineHeight) + 1
	if firstRow < 0 {
		firstRow = 0
	}
	if lastRow >= len(starts) {
		lastRow = len(starts) - 1
	}
	for row := firstRow; row <= lastRow; row++ {
		line := w.lineAt(row)
		y := bounds.Min.Y + padY + float32(row)*lineHeight - w.scrollY
		lineBounds := geometry.Rect{Min: geometry.Pt(bounds.Min.X+padX-w.scrollX, y), Max: geometry.Pt(bounds.Max.X-padX, y+lineHeight)}
		if y+lineHeight < inner.Min.Y || y > inner.Max.Y {
			continue
		}
		if lo, hi, ok := w.selectionOnLine(row, starts[row], len(line)); ok {
			origin := bounds.Min.X + padX - w.scrollX
			left := origin + w.measure(canvas, line[:lo])
			right := origin + w.measure(canvas, line[:hi])
			canvas.DrawRect(geometry.Rect{Min: geometry.Pt(left, y), Max: geometry.Pt(right, y+lineHeight)}, colors.Primary)
			if lo > 0 {
				beforeBounds := geometry.Rect{Min: lineBounds.Min, Max: geometry.Pt(left, lineBounds.Max.Y)}
				canvas.DrawText(string(line[:lo]), beforeBounds, w.fontSize, colors.OnSurface, false, widget.TextAlignLeft)
			}
			selectedBounds := geometry.Rect{Min: geometry.Pt(left, y), Max: geometry.Pt(right, y+lineHeight)}
			canvas.DrawText(string(line[lo:hi]), selectedBounds, w.fontSize, colors.OnPrimary, false, widget.TextAlignLeft)
			if hi < len(line) {
				afterBounds := geometry.Rect{Min: geometry.Pt(right, y), Max: lineBounds.Max}
				canvas.DrawText(string(line[hi:]), afterBounds, w.fontSize, colors.OnSurface, false, widget.TextAlignLeft)
			}
		} else {
			canvas.DrawText(string(line), lineBounds, w.fontSize, colors.OnSurface, false, widget.TextAlignLeft)
		}
	}

	if w.IsFocused() {
		row, col := w.rowColumn(w.cursor)
		if row >= 0 && row < len(starts) {
			line := w.lineAt(row)
			x := bounds.Min.X + padX + w.measure(canvas, line[:col]) - w.scrollX
			y := bounds.Min.Y + padY + float32(row)*lineHeight - w.scrollY
			canvas.DrawLine(geometry.Pt(x, y+2), geometry.Pt(x, y+lineHeight-2), colors.OnSurfaceSecondary, 1.5)
		}
	}
}

// Event handles keyboard editing, mouse caret selection and two-axis scrolling.
func (w *Widget) Event(ctx widget.Context, e event.Event) bool {
	if !w.IsEnabled() || !w.IsVisible() {
		return false
	}
	switch ev := e.(type) {
	case *event.KeyEvent:
		if !w.IsFocused() || (ev.KeyType != event.KeyPress && ev.KeyType != event.KeyRepeat) {
			return false
		}
		return w.key(ctx, ev)
	case *event.MouseEvent:
		if ev.MouseType == event.MousePress && ev.Button == event.ButtonLeft {
			if !w.Bounds().Contains(ev.Position) {
				return false
			}
			if ctx != nil {
				ctx.RequestFocus(w)
			}
			w.SetFocused(true)
			w.cursor = w.positionToOffset(ev.Position)
			w.anchor = -1
			w.dragging = true
			w.invalidate(ctx)
			return true
		}
		if ev.MouseType == event.MouseDrag && w.dragging {
			if w.anchor < 0 {
				w.anchor = w.cursor
			}
			w.cursor = w.positionToOffset(ev.Position)
			w.invalidate(ctx)
			return true
		}
		if ev.MouseType == event.MouseRelease && w.dragging {
			w.dragging = false
			return true
		}
	case *event.WheelEvent:
		if !w.Bounds().Contains(ev.Position) {
			return false
		}
		// GoGPU's macOS bridge forwards smooth logical-pixel deltas unchanged.
		// WheelEvent has no delta-mode field, so preserve the provided movement.
		if ev.Modifiers().Has(event.ModShift) && ev.Delta.X == 0 {
			w.scrollX += ev.Delta.Y
		} else {
			w.scrollX += ev.Delta.X
			w.scrollY += ev.Delta.Y
		}
		w.clampScroll()
		w.invalidate(ctx)
		return true
	}
	return false
}

// key applies editor shortcuts, cursor movement and text input.
func (w *Widget) key(ctx widget.Context, ev *event.KeyEvent) bool {
	if ev.Key == event.KeyEscape && w.onEscape != nil {
		w.onEscape()
		return true
	}
	if w.readOnly && ev.Key == event.KeyTab {
		return false
	}
	mods := ev.Modifiers()
	command := mods.HasAny(event.ModSuper | event.ModCtrl)
	if command {
		switch ev.Key {
		case event.KeyComma:
			if w.onSettings != nil {
				w.onSettings()
				return true
			}
		case event.KeyA:
			w.anchor, w.cursor = 0, len(w.text)
			w.invalidate(ctx)
			return true
		case event.KeyC:
			if start, end, ok := w.selection(); ok {
				widget.ClipboardWrite(string(w.text[start:end]))
			}
			return true
		case event.KeyX:
			if start, end, ok := w.selection(); ok {
				widget.ClipboardWrite(string(w.text[start:end]))
				w.replaceSelection(nil, ctx)
			}
			return true
		case event.KeyV:
			clipboard := widget.ClipboardRead()
			if len(clipboard) <= maxInsertBytes {
				if pasted := normalizeNewlines(clipboard); pasted != "" {
					w.replaceSelection([]rune(pasted), ctx)
				}
			}
			return true
		case event.KeyZ:
			if mods.Has(event.ModShift) {
				w.redoEdit(ctx)
			} else {
				w.undoEdit(ctx)
			}
			return true
		}
	}

	shift := mods.Has(event.ModShift)
	switch ev.Key {
	case event.KeyLeft:
		w.moveHorizontal(-1, shift, command)
	case event.KeyRight:
		w.moveHorizontal(1, shift, command)
	case event.KeyUp:
		w.moveVertical(-1, shift)
	case event.KeyDown:
		w.moveVertical(1, shift)
	case event.KeyHome:
		w.moveToLineEdge(false, shift)
	case event.KeyEnd:
		w.moveToLineEdge(true, shift)
	case event.KeyBackspace:
		if start, end, ok := w.selection(); ok {
			w.replaceRange(start, end, nil, ctx)
		} else if w.cursor > 0 {
			w.replaceRange(w.cursor-1, w.cursor, nil, ctx)
		}
	case event.KeyDelete:
		if start, end, ok := w.selection(); ok {
			w.replaceRange(start, end, nil, ctx)
		} else if w.cursor < len(w.text) {
			w.replaceRange(w.cursor, w.cursor+1, nil, ctx)
		}
	case event.KeyEnter, event.KeyNumpadEnter:
		w.replaceSelection([]rune{'\n'}, ctx)
	case event.KeyTab:
		w.replaceSelection([]rune{'\t'}, ctx)
	default:
		if ev.Rune >= ' ' && ev.Rune != 0x7f && !command {
			w.replaceSelection([]rune{ev.Rune}, ctx)
			return true
		}
		return false
	}
	w.ensureCursorVisible()
	w.invalidate(ctx)
	return true
}

// replaceSelection replaces the active selection, or inserts at the cursor.
func (w *Widget) replaceSelection(insert []rune, ctx widget.Context) {
	start, end, ok := w.selection()
	if !ok {
		start, end = w.cursor, w.cursor
	}
	w.replaceRange(start, end, insert, ctx)
}

// replaceRange records one bounded edit and replaces a half-open rune range.
func (w *Widget) replaceRange(start, end int, insert []rune, ctx widget.Context) {
	if w.readOnly {
		return
	}
	if start < 0 {
		start = 0
	}
	if end > len(w.text) {
		end = len(w.text)
	}
	if start > end || start == end && len(insert) == 0 {
		return
	}
	if runeBytes(insert) > maxInsertBytes {
		return
	}
	w.undo = appendHistory(w.undo, string(w.text))
	w.redo = nil
	changed := make([]rune, 0, len(w.text)-(end-start)+len(insert))
	changed = append(changed, w.text[:start]...)
	changed = append(changed, insert...)
	changed = append(changed, w.text[end:]...)
	w.text = changed
	w.resetLineCache()
	w.cursor = start + len(insert)
	w.anchor = -1
	w.afterEdit(ctx)
}

// afterEdit updates scrolling, redraw state and the document change callback.
func (w *Widget) afterEdit(ctx widget.Context) {
	w.clampScroll()
	w.ensureCursorVisible()
	w.invalidate(ctx)
	if w.onChange != nil {
		w.onChange(string(w.text))
	}
}

// undoEdit restores the most recent snapshot and saves the current text for redo.
func (w *Widget) undoEdit(ctx widget.Context) {
	if w.readOnly {
		return
	}
	if len(w.undo) == 0 {
		return
	}
	w.redo = appendHistory(w.redo, string(w.text))
	w.text = []rune(w.undo[len(w.undo)-1])
	w.undo[len(w.undo)-1] = ""
	w.undo = w.undo[:len(w.undo)-1]
	w.resetLineCache()
	w.cursor = len(w.text)
	w.anchor = -1
	w.afterEdit(ctx)
}

// redoEdit restores the next snapshot and saves the current text for undo.
func (w *Widget) redoEdit(ctx widget.Context) {
	if w.readOnly {
		return
	}
	if len(w.redo) == 0 {
		return
	}
	w.undo = appendHistory(w.undo, string(w.text))
	w.text = []rune(w.redo[len(w.redo)-1])
	w.redo[len(w.redo)-1] = ""
	w.redo = w.redo[:len(w.redo)-1]
	w.resetLineCache()
	w.cursor = len(w.text)
	w.anchor = -1
	w.afterEdit(ctx)
}

// moveHorizontal moves by one rune or to a line edge, optionally extending selection.
func (w *Widget) moveHorizontal(direction int, extend, byLine bool) {
	if !extend {
		if start, end, ok := w.selection(); ok {
			if direction < 0 {
				w.cursor = start
			} else {
				w.cursor = end
			}
			w.anchor = -1
			return
		}
	} else if w.anchor < 0 {
		w.anchor = w.cursor
	}
	if byLine {
		row, _ := w.rowColumn(w.cursor)
		starts := w.lineStarts()
		if direction < 0 {
			w.cursor = starts[row]
		} else {
			w.cursor = w.lineEnd(row)
		}
	} else {
		w.cursor += direction
	}
	if w.cursor < 0 {
		w.cursor = 0
	}
	if w.cursor > len(w.text) {
		w.cursor = len(w.text)
	}
}

// moveVertical keeps the column where possible while changing document rows.
func (w *Widget) moveVertical(direction int, extend bool) {
	if extend && w.anchor < 0 {
		w.anchor = w.cursor
	} else if !extend {
		w.anchor = -1
	}
	row, col := w.rowColumn(w.cursor)
	starts := w.lineStarts()
	row += direction
	if row < 0 {
		row = 0
	}
	if row >= len(starts) {
		row = len(starts) - 1
	}
	lineLength := w.lineEnd(row) - starts[row]
	if col > lineLength {
		col = lineLength
	}
	w.cursor = starts[row] + col
}

// moveToLineEdge moves to the current row's start or end.
func (w *Widget) moveToLineEdge(end, extend bool) {
	if extend && w.anchor < 0 {
		w.anchor = w.cursor
	} else if !extend {
		w.anchor = -1
	}
	row, _ := w.rowColumn(w.cursor)
	starts := w.lineStarts()
	if end {
		w.cursor = w.lineEnd(row)
	} else {
		w.cursor = starts[row]
	}
}

// selection returns the selected rune range in document order.
func (w *Widget) selection() (int, int, bool) {
	if w.anchor < 0 || w.anchor == w.cursor {
		return 0, 0, false
	}
	if w.anchor < w.cursor {
		return w.anchor, w.cursor, true
	}
	return w.cursor, w.anchor, true
}

// selectionOnLine intersects the document selection with one visible row.
func (w *Widget) selectionOnLine(row, start, length int) (int, int, bool) {
	lo, hi, ok := w.selection()
	if !ok {
		return 0, 0, false
	}
	lineLo, lineHi := start, start+length
	if lo > lineHi || hi < lineLo {
		return 0, 0, false
	}
	lo -= lineLo
	hi -= lineLo
	if lo < 0 {
		lo = 0
	}
	if hi > length {
		hi = length
	}
	return lo, hi, lo <= hi
}

// lineAt returns the rune slice for a row without copying the document.
func (w *Widget) lineAt(row int) []rune {
	starts := w.lineStarts()
	if row < 0 || row >= len(starts) {
		return nil
	}
	return w.text[starts[row]:w.lineEnd(row)]
}

// lineStarts caches row offsets and the longest row length for this document.
func (w *Widget) lineStarts() []int {
	if w.linesCached {
		return w.lineStartsCache
	}
	starts := []int{0}
	lineStart := 0
	w.longestLine = 0
	for i, r := range w.text {
		if r == '\n' {
			if i-lineStart > w.longestLine {
				w.longestLine = i - lineStart
			}
			lineStart = i + 1
			starts = append(starts, lineStart)
		}
	}
	if len(w.text)-lineStart > w.longestLine {
		w.longestLine = len(w.text) - lineStart
	}
	w.lineStartsCache = starts
	w.linesCached = true
	return starts
}

// resetLineCache invalidates derived row offsets after document changes.
func (w *Widget) resetLineCache() {
	w.lineStartsCache = nil
	w.longestLine = 0
	w.linesCached = false
}

// rowColumn maps a rune offset to its row and column.
func (w *Widget) rowColumn(offset int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(w.text) {
		offset = len(w.text)
	}
	starts := w.lineStarts()
	row := sort.Search(len(starts), func(i int) bool { return starts[i] > offset }) - 1
	if row < 0 {
		row = 0
	}
	return row, offset - starts[row]
}

// lineEnd returns the rune offset immediately before a row's newline.
func (w *Widget) lineEnd(row int) int {
	starts := w.lineStarts()
	if row < 0 {
		return 0
	}
	if row+1 < len(starts) {
		return starts[row+1] - 1
	}
	return len(w.text)
}

// lineHeight returns the vertical step used to place editor rows.
func (w *Widget) lineHeight() float32 { return w.fontSize * lineGap }

// measure asks the active canvas for the rendered width of a rune slice.
func (w *Widget) measure(canvas widget.Canvas, text []rune) float32 {
	if len(text) == 0 {
		return 0
	}
	return canvas.MeasureText(string(text), w.fontSize, false)
}

// positionToOffset maps a pointer position to the nearest rune column.
func (w *Widget) positionToOffset(position geometry.Point) int {
	starts := w.lineStarts()
	row := int((position.Y - w.Bounds().Min.Y - padY + w.scrollY) / w.lineHeight())
	if row < 0 {
		row = 0
	}
	if row >= len(starts) {
		row = len(starts) - 1
	}
	col := int((position.X - w.Bounds().Min.X - padX + w.scrollX) / (w.fontSize * 0.6))
	if col < 0 {
		col = 0
	}
	lineLength := w.lineEnd(row) - starts[row]
	if col > lineLength {
		col = lineLength
	}
	return starts[row] + col
}

// ensureCursorVisible adjusts scrolling to keep the insertion point in view.
func (w *Widget) ensureCursorVisible() {
	b := w.Bounds()
	row, col := w.rowColumn(w.cursor)
	lineHeight := w.lineHeight()
	y := float32(row) * lineHeight
	if y < w.scrollY {
		w.scrollY = y
	}
	if y+lineHeight > w.scrollY+b.Height()-2*padY {
		w.scrollY = y + lineHeight - b.Height() + 2*padY
	}
	x := float32(col) * w.fontSize * 0.6
	if x < w.scrollX {
		w.scrollX = x
	}
	if x > w.scrollX+b.Width()-2*padX {
		w.scrollX = x - b.Width() + 2*padX
	}
	w.clampScroll()
}

// clampScroll restricts both offsets to the content and viewport extents.
func (w *Widget) clampScroll() {
	b := w.Bounds()
	maxLines := len(w.lineStarts())
	maxY := float32(maxLines)*w.lineHeight() - (b.Height() - 2*padY)
	if maxY < 0 {
		maxY = 0
	}
	if w.scrollY < 0 {
		w.scrollY = 0
	}
	if w.scrollY > maxY {
		w.scrollY = maxY
	}
	maxX := float32(w.longestLine)*w.fontSize*0.6 - (b.Width() - 2*padX)
	if maxX < 0 {
		maxX = 0
	}
	if w.scrollX < 0 {
		w.scrollX = 0
	}
	if w.scrollX > maxX {
		w.scrollX = maxX
	}
}

// appendHistory stores a snapshot while respecting count and byte limits.
func appendHistory(history []string, text string) []string {
	if len(text) > maxHistoryBytes {
		return nil
	}
	history = append(history, text)
	bytes := 0
	for _, item := range history {
		bytes += len(item)
	}
	for len(history) > maxHistoryEntries || bytes > maxHistoryBytes {
		bytes -= len(history[0])
		history[0] = ""
		history = history[1:]
	}
	return history
}

// runeBytes counts UTF-8 storage bytes and stops once insertion exceeds its cap.
func runeBytes(runes []rune) int {
	bytes := 0
	for _, r := range runes {
		bytes += utf8.RuneLen(r)
		if bytes > maxInsertBytes {
			return bytes
		}
	}
	return bytes
}

// invalidate marks the widget dirty and notifies the active UI context.
func (w *Widget) invalidate(ctx widget.Context) {
	w.SetNeedsRedraw(true)
	if ctx != nil {
		ctx.Invalidate()
	}
}

// normalizeNewlines converts CRLF and CR line endings to LF.
func normalizeNewlines(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
}
