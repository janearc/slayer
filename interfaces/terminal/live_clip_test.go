package terminal

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

// anImage is a picture of cols by rows cells, each cell different enough
// that a cut can be told from a shift.
func someCells(cols, rows int) *Cells {
	img := image.NewRGBA(image.Rect(0, 0, cols*2, rows*3))
	for y := 0; y < rows*3; y++ {
		for x := 0; x < cols*2; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(x * 9), uint8(y * 9), uint8(x * y), 255})
		}
	}
	return Chop(img)
}

// a cut keeps the top left and no more than the screen holds: 5 by 4 cells
// cut to 3 by 2 are the first 3 of each of the first 2 rows.
func TestClippedKeepsTheTopLeft(t *testing.T) {
	cells := someCells(5, 4)
	got := clipped(cells, 3, 2)
	if got.W != 3 || got.H != 2 || len(got.Cells) != 6 {
		t.Fatalf("got %d by %d, %d cells", got.W, got.H, len(got.Cells))
	}
	for y := range 2 {
		for x := range 3 {
			if got.At(x, y) != cells.At(x, y) {
				t.Errorf("cell %d,%d moved", x, y)
			}
		}
	}
}

// anti-test: cells that already fit are not copied or changed, and a screen
// larger than the picture does not make the picture larger.
func TestClippedLeavesWhatFits(t *testing.T) {
	cells := someCells(3, 2)
	if clipped(cells, 3, 2) != cells {
		t.Error("cells that fit were copied")
	}
	if got := clipped(cells, 80, 24); got != cells || got.W != 3 || got.H != 2 {
		t.Errorf("a big screen changed the picture to %d by %d", got.W, got.H)
	}
	if got := clipped(cells, 80, 1); got.W != 3 || got.H != 1 {
		t.Errorf("one row: %d by %d", got.W, got.H)
	}
}

// the status line is: no wrapping, go to the last row, default colours, the
// text, erase the rest of the row, wrapping back on.
func TestStatusLineWritesTheLastRowAlone(t *testing.T) {
	const want = "\x1b[?7l\x1b[24;1H\x1b[0mframe 1 of 3\x1b[K\x1b[?7h"
	if got := statusLine("frame 1 of 3", 80, 24); got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// text is cut to the width, by characters; text that fits is not cut.
// anti-test: a width of nought writes no text, and a row is never beyond
// the one asked for.
func TestStatusLineIsClippedToTheWidth(t *testing.T) {
	for what, row := range map[string]struct {
		text string
		cols int
		want string
	}{
		"longer":  {"abcdefgh", 5, "abcde"},
		"exact":   {"abcde", 5, "abcde"},
		"shorter": {"abc", 5, "abc"},
		"runes":   {"éééé", 2, "éé"},
		"none":    {"abc", 0, ""},
		"empty":   {"", 5, ""},
	} {
		line := statusLine(row.text, row.cols, 3)
		inner := strings.TrimSuffix(strings.TrimPrefix(line,
			"\x1b[?7l\x1b[3;1H\x1b[0m"), "\x1b[K\x1b[?7h")
		if inner != row.want {
			t.Errorf("%s: wrote %q, want %q", what, inner, row.want)
		}
	}
}

// anti-test: no escape or other control byte is written from the text, so
// text cannot move the cursor or change a colour.
func TestStatusLineWritesNoControlBytes(t *testing.T) {
	line := statusLine("a\x1b[31mb\r\nc\x00\u0085d", 80, 5)
	inner := strings.TrimSuffix(strings.TrimPrefix(line,
		"\x1b[?7l\x1b[5;1H\x1b[0m"), "\x1b[K\x1b[?7h")
	if inner != "a[31mbcd" {
		t.Errorf("wrote %q", inner)
	}
}

// the events a view's tests make are the events Live sends: a key, a
// resize with its screen, and the end. anti-test: each is of its own kind
// and carries nothing of the others.
func TestEventsMadeForTestsAreLives(t *testing.T) {
	var decoder Decoder
	key := decoder.Feed([]byte("q"))[0]
	if got := KeyEvent(key); got.Kind() != EventKey || got.Key() != key ||
		got.Screen() != (Screen{}) {
		t.Errorf("a key event is %+v", got)
	}
	screen := Screen{Cols: 3, Rows: 2, W: 6, H: 6}
	if got := ResizeEvent(screen); got.Kind() != EventResize ||
		got.Screen() != screen || got.Key() != (Key{}) {
		t.Errorf("a resize event is %+v", got)
	}
	if got := EndEvent(); got.Kind() != EventEnd || got.Key() != (Key{}) {
		t.Errorf("an end event is %+v", got)
	}
}
