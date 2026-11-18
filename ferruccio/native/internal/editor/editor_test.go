package editor

import (
	"image"
	"strings"
	"testing"

	"github.com/gogpu/gg"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/render"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
)

// TestDrawUsesCurrentChromaGroundAndInk checks the editor after theme changes.
func TestDrawUsesCurrentChromaGroundAndInk(t *testing.T) {
	previous := appearance.SelectedTheme()
	t.Cleanup(func() { _ = appearance.SelectTheme(previous) })
	var firstGround, firstInk widget.Color
	for index, name := range []string{"vaporwave", "fermi-long"} {
		if err := appearance.SelectTheme(name); err != nil {
			t.Fatalf("SelectTheme(%q): %v", name, err)
		}
		colors := appearance.Theme().Colors
		if index == 0 {
			firstGround, firstInk = colors.Background, colors.OnSurface
		} else if colors.Background == firstGround || colors.OnSurface == firstInk {
			t.Fatal("test themes do not have distinct source ground and ink roles")
		}
		const width, height = 320, 90
		dc := gg.NewContext(width, height)
		canvas := render.NewCanvas(dc, width, height)
		e := New("Readable source ink", nil)
		e.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(width, height)})
		e.Draw(widget.NewContext(), canvas)
		img := dc.Image()
		if !pixelMatches(img.At(250, 70), colors.Background) {
			t.Errorf("%s editor background pixel does not use Chroma ground", name)
		}
		if !containsPixelColor(img, image.Rect(12, 10, 180, 34), colors.OnSurface) {
			t.Errorf("%s editor text does not use Chroma's readable ink", name)
		}
	}
}

// TestSelectionUsesTheThemeCheckedTextPair checks selection's readable pair.
func TestSelectionUsesTheThemeCheckedTextPair(t *testing.T) {
	previous := appearance.SelectedTheme()
	t.Cleanup(func() { _ = appearance.SelectTheme(previous) })
	if err := appearance.SelectTheme("vaporwave"); err != nil {
		t.Fatal(err)
	}
	colors := appearance.Theme().Colors
	const width, height = 320, 90
	dc := gg.NewContext(width, height)
	canvas := render.NewCanvas(dc, width, height)
	e := New("selected text", nil)
	e.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(width, height)})
	e.anchor, e.cursor = 0, len(e.text)
	e.Draw(widget.NewContext(), canvas)
	img := dc.Image()
	area := image.Rect(12, 10, 220, 34)
	if !containsPixelColor(img, area, colors.Primary) {
		t.Error("selection background does not use the theme's checked primary fill")
	}
	if !containsPixelColor(img, area, colors.OnPrimary) {
		t.Error("selected text does not use the matching checked primary ink")
	}
}

// containsPixelColor reports whether a real canvas region contains a color.
func containsPixelColor(img image.Image, area image.Rectangle, want widget.Color) bool {
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			if pixelMatches(img.At(x, y), want) {
				return true
			}
		}
	}
	return false
}

// pixelMatches compares one canvas pixel with a theme role, allowing antialiasing.
func pixelMatches(gotColor interface {
	RGBA() (uint32, uint32, uint32, uint32)
}, want widget.Color) bool {
	r, g, b, a := gotColor.RGBA()
	return a > 0 && abs8(uint8(r>>8), uint8(want.R*255)) <= 4 &&
		abs8(uint8(g>>8), uint8(want.G*255)) <= 4 &&
		abs8(uint8(b>>8), uint8(want.B*255)) <= 4
}

// abs8 returns the distance between two color channels.
func abs8(left, right uint8) uint8 {
	if left > right {
		return left - right
	}
	return right - left
}

func TestNewAndSetText(t *testing.T) {
	var changes []string
	e := New("first:\n  value: 1", func(text string) { changes = append(changes, text) })
	if got := e.Text(); got != "first:\n  value: 1" {
		t.Fatalf("Text() = %q", got)
	}
	if !e.IsFocusable() {
		t.Fatal("new editor is not focusable")
	}
	var _ widget.Widget = e
	var _ widget.Focusable = e

	e.SetText("loaded: true")
	if got := e.Text(); got != "loaded: true" {
		t.Fatalf("Text() after SetText = %q", got)
	}
	if len(changes) != 0 {
		t.Fatalf("SetText called onChange %d times", len(changes))
	}
	e.SetFocused(true)
	press(e, event.KeyZ, event.ModSuper, 0)
	if got := e.Text(); got != "loaded: true" {
		t.Fatalf("undo after SetText = %q, want explicit load to reset history", got)
	}
}

func TestApplyTextIsUndoableAndPreservesSelectionAndScroll(t *testing.T) {
	var changes []string
	e := New("before camera move\n  position: 1", func(text string) { changes = append(changes, text) })
	e.cursor = 5
	e.anchor = 2
	e.scrollX = 17
	e.scrollY = 23
	before := e.Text()
	updated := "after camera move\n  position: 2\n  orbit: true"
	e.ApplyText(updated)
	if e.Text() != updated || e.cursor != 5 || e.anchor != 2 || e.scrollX != 17 || e.scrollY != 23 {
		t.Fatalf("ApplyText state text=%q cursor=%d anchor=%d scroll=(%v,%v)", e.Text(), e.cursor, e.anchor, e.scrollX, e.scrollY)
	}
	if len(changes) != 1 || changes[0] != updated {
		t.Fatalf("ApplyText callbacks = %q, want one updated document", changes)
	}
	e.SetFocused(true)
	press(e, event.KeyZ, event.ModSuper, 0)
	if e.Text() != before {
		t.Fatalf("undo after ApplyText = %q, want %q", e.Text(), before)
	}
	press(e, event.KeyZ, event.ModSuper|event.ModShift, 0)
	if e.Text() != updated {
		t.Fatalf("redo after ApplyText = %q, want %q", e.Text(), updated)
	}

	undoCount, redoCount, callbackCount := len(e.undo), len(e.redo), len(changes)
	e.ApplyText(e.Text())
	if len(e.undo) != undoCount || len(e.redo) != redoCount || len(changes) != callbackCount {
		t.Fatalf("no-op ApplyText changed history/callbacks: undo %d->%d redo %d->%d callbacks %d->%d", undoCount, len(e.undo), redoCount, len(e.redo), callbackCount, len(changes))
	}
}

func TestKeyboardEditingAndUndo(t *testing.T) {
	var changes []string
	e := New("a", func(text string) { changes = append(changes, text) })
	e.SetFocused(true)

	press(e, event.KeyRight, 0, 0)
	press(e, event.KeyB, 0, 'b')
	if got := e.Text(); got != "ab" {
		t.Fatalf("text after typing = %q", got)
	}
	press(e, event.KeyZ, event.ModSuper, 0)
	if got := e.Text(); got != "a" {
		t.Fatalf("text after undo = %q", got)
	}
	press(e, event.KeyZ, event.ModSuper|event.ModShift, 0)
	if got := e.Text(); got != "ab" {
		t.Fatalf("text after redo = %q", got)
	}
	if len(changes) != 3 {
		t.Fatalf("onChange called %d times, want edit, undo, redo", len(changes))
	}
}

func TestUnicodeAndMultilineNavigation(t *testing.T) {
	e := New("λ\n猫", nil)
	e.SetFocused(true)
	press(e, event.KeyHome, 0, 0)
	press(e, event.KeyUp, 0, 0)
	press(e, event.KeyEnter, 0, 0)
	press(e, event.KeyUnknown, 0, '🙂')
	if got := e.Text(); got != "\n🙂λ\n猫" {
		t.Fatalf("text after Unicode/multiline input = %q", got)
	}
}

func TestShiftSelectionAndDelete(t *testing.T) {
	e := New("abcd", nil)
	e.SetFocused(true)
	press(e, event.KeyHome, 0, 0)
	press(e, event.KeyRight, event.ModShift, 0)
	press(e, event.KeyRight, event.ModShift, 0)
	press(e, event.KeyDelete, 0, 0)
	if got := e.Text(); got != "cd" {
		t.Fatalf("text after selected delete = %q, want %q", got, "cd")
	}
}

func TestMouseFocusAndUnicodeRuneChange(t *testing.T) {
	var changed string
	e := New("ab", func(text string) { changed = text })
	e.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(200, 100)})
	ctx := widget.NewContext()
	click := event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, geometry.Pt(15, 15), geometry.Pt(15, 15), 0)
	if !e.Event(ctx, click) {
		t.Fatal("mouse press was not consumed")
	}
	if ctx.FocusedWidget() != e || !e.IsFocused() {
		t.Fatal("mouse press did not request focus through UI context")
	}
	press(e, event.KeyUnknown, 0, '猫')
	if got := e.Text(); got != "猫ab" || changed != got {
		t.Fatalf("rune input text/callback = %q/%q, want %q", got, changed, "猫ab")
	}
}

func TestWheelScrollAndFontSizeStayBounded(t *testing.T) {
	e := New(strings.Repeat("wide line\n", 80)+strings.Repeat("x", 200), nil)
	e.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(180, 100)})
	e.Event(nil, event.NewWheelEvent(geometry.Pt(1, 2), geometry.Pt(50, 50), geometry.Pt(50, 50), 0))
	if e.scrollY <= 0 || e.scrollX <= 0 {
		t.Fatalf("wheel scroll = (%v, %v), want both axes to move", e.scrollX, e.scrollY)
	}
	e.SetFontSize(100)
	if e.fontSize > maxFont {
		t.Fatalf("font size = %v, want at most %v", e.fontSize, maxFont)
	}
	e.SetFontSize(8)
	if e.fontSize < minFont {
		t.Fatalf("font size = %v, want at least %v", e.fontSize, minFont)
	}
}

// TestTrackpadWheelDeltasUseLogicalPixelsAndNaturalDirection guards smooth scrolling.
func TestTrackpadWheelDeltasUseLogicalPixelsAndNaturalDirection(t *testing.T) {
	e := New(strings.Repeat("wide line\n", 80)+strings.Repeat("x", 200), nil)
	e.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(180, 100)})
	ctx := widget.NewContext()

	if !e.Event(ctx, event.NewWheelEvent(geometry.Pt(4.5, 5.25), geometry.Pt(50, 50), geometry.Pt(50, 50), 0)) {
		t.Fatal("in-bounds trackpad scroll was not consumed")
	}
	if e.scrollX != 4.5 || e.scrollY != 5.25 {
		t.Fatalf("smooth deltas moved offsets to (%v, %v), want raw logical pixels (4.5, 5.25)", e.scrollX, e.scrollY)
	}

	e.Event(ctx, event.NewWheelEvent(geometry.Pt(-1.5, -2.25), geometry.Pt(50, 50), geometry.Pt(50, 50), 0))
	if e.scrollX != 3 || e.scrollY != 3 {
		t.Fatalf("negative smooth deltas moved offsets to (%v, %v), want (3, 3)", e.scrollX, e.scrollY)
	}
	e.Event(ctx, event.NewWheelEvent(geometry.Pt(0, 2.5), geometry.Pt(50, 50), geometry.Pt(50, 50), event.ModShift))
	if e.scrollX != 5.5 || e.scrollY != 3 {
		t.Fatalf("Shift+vertical scroll moved offsets to (%v, %v), want (5.5, 3)", e.scrollX, e.scrollY)
	}
}

// TestLayoutClampsScrollToResizedViewport prevents stale viewport bounds.
func TestLayoutClampsScrollToResizedViewport(t *testing.T) {
	e := New(strings.Repeat("line\n", 100), nil)
	e.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(180, 100)})
	e.scrollY = 1900

	size := e.Layout(nil, geometry.Tight(geometry.Sz(180, 600)))
	if size.Height != 600 || e.Bounds().Height() != 600 {
		t.Fatalf("layout size/bounds = %v/%v, want 180x600", size, e.Bounds().Size())
	}
	maxScroll := float32(len(e.lineStarts()))*e.lineHeight() - (e.Bounds().Height() - 2*padY)
	if e.scrollY > maxScroll {
		t.Fatalf("scroll after viewport resize = %v, exceeds new maximum %v", e.scrollY, maxScroll)
	}
	e.cursor = 0
	e.scrollY = 500
	e.Layout(nil, geometry.Tight(geometry.Sz(180, 600)))
	if e.scrollY != 500 {
		t.Fatalf("layout moved manual scroll to %v with cursor at document start", e.scrollY)
	}
}

func TestLayoutHonorsAvailableBounds(t *testing.T) {
	e := New("a", nil)
	got := e.Layout(nil, geometry.Constraints{MinWidth: 90, MaxWidth: 180, MinHeight: 60, MaxHeight: 120})
	if got.Width != 180 || got.Height != 120 {
		t.Fatalf("Layout() = %v, want largest bounded size 180x120", got)
	}
}

func TestInsertionAndUndoStorageAreBounded(t *testing.T) {
	e := New("", nil)
	e.replaceRange(0, 0, []rune(strings.Repeat("x", maxInsertBytes+1)), nil)
	if got := e.Text(); got != "" {
		t.Fatalf("oversized insertion changed text (bytes=%d)", len(got))
	}
	for range maxHistoryEntries + 5 {
		e.replaceRange(len(e.text), len(e.text), []rune{'x'}, nil)
	}
	if len(e.undo) > maxHistoryEntries {
		t.Fatalf("undo entries = %d, want at most %d", len(e.undo), maxHistoryEntries)
	}
}

func press(e *Widget, key event.Key, mods event.Modifiers, r rune) {
	e.Event(nil, event.NewKeyEvent(event.KeyPress, key, r, mods))
}
