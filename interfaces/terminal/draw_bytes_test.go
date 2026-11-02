package terminal

import (
	"errors"
	"image/color"
	"strconv"
	"strings"
	"testing"
)

// the pieces of a frame, from the standards and not from Draw. an escape
// is byte 0x1b. ECMA-48 gives CSI as escape and "[", cursor position as
// CSI row ; column H (CUP), CSI H alone for the top left, and select
// graphic rendition as CSI parameters m, with 0 for reset.
//
// xterm's 24-bit colour is SGR 38;2;R;G;B for the foreground and
// 48;2;R;G;B for the background. synchronised output is private mode
// 2026: CSI ? 2026 h begins a frame and CSI ? 2026 l ends it.
const (
	frameBegin = "\x1b[?2026h\x1b[H"
	frameEnd   = "\x1b[?2026l"
	reset      = "\x1b[0m"
)

// place is CUP to the first column of a row, counting from 1.
func place(row int) string { return "\x1b[" + strconv.Itoa(row) + ";1H" }

func fgOf(r, g, b int) string {
	return "\x1b[38;2;" + strconv.Itoa(r) + ";" + strconv.Itoa(g) + ";" + strconv.Itoa(b) + "m"
}

func bgOf(r, g, b int) string {
	return "\x1b[48;2;" + strconv.Itoa(r) + ";" + strconv.Itoa(g) + ";" + strconv.Itoa(b) + "m"
}

// the whole frame for three cells across and two down, byte for byte.
// a colour is sent when it is the first on its row or has changed from
// the one before, the fg and the bg apart, fg first, so a cell that
// changes one sends only that one.
//
// row 0: A on fg 10,20,30 and bg 40,50,60 sends both. then U+1FB07
// changes only the bg to 41,50,60, and U+258C only the fg to 11,20,30.
//
// row 1 starts over: D is the colours row 0 ended on and sends both
// again, E sends none, and F changes the fg's blue to 31.
func TestADrawnFrameIsEveryByteAsTheStandardsHaveIt(t *testing.T) {
	f0, b0 := color.RGBA{10, 20, 30, 255}, color.RGBA{40, 50, 60, 255}
	b1 := color.RGBA{41, 50, 60, 255}
	f1 := color.RGBA{11, 20, 30, 255}
	f2 := color.RGBA{11, 20, 31, 255}
	c := &Cells{W: 3, H: 2, Cells: []Cell{
		{Rune: 'A', Fg: f0, Bg: b0}, {Rune: 0x1FB07, Fg: f0, Bg: b1}, {Rune: 0x258C, Fg: f1, Bg: b1},
		{Rune: 'D', Fg: f1, Bg: b1}, {Rune: 'E', Fg: f1, Bg: b1}, {Rune: 'F', Fg: f2, Bg: b1},
	}}
	want := frameBegin +
		place(1) + fgOf(10, 20, 30) + bgOf(40, 50, 60) + "A" + bgOf(41, 50, 60) + "\U0001FB07" + fgOf(11, 20, 30) + "▌" + reset +
		place(2) + fgOf(11, 20, 30) + bgOf(41, 50, 60) + "DE" + fgOf(11, 20, 31) + "F" + reset +
		frameEnd
	var b strings.Builder
	ends(t, "draw", func() { Draw(&b, c) })
	if got := b.String(); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// a first cell of zero, black on black, still sends its colours: the terminal
// may have anything set, so nothing is taken as already so.
func TestTheFirstCellOfARowSendsBlackToo(t *testing.T) {
	var b strings.Builder
	if err := Draw(&b, &Cells{W: 1, H: 1, Cells: []Cell{{Rune: ' '}}}); err != nil {
		t.Fatal(err)
	}
	want := frameBegin + place(1) + fgOf(0, 0, 0) + bgOf(0, 0, 0) + " " + reset + frameEnd
	if got := b.String(); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// a grid with no cells across or none down is a frame with only its
// begin and end, and one with rows but no cells sends each row's place
// and reset and nothing between. neither draws a colour.
func TestEmptyGridsAreFramesOfNothing(t *testing.T) {
	for _, w := range []struct {
		c    *Cells
		want string
	}{
		{&Cells{}, frameBegin + frameEnd},
		{&Cells{W: 3}, frameBegin + frameEnd},
		{&Cells{H: 2}, frameBegin + place(1) + reset + place(2) + reset + frameEnd},
	} {
		var b strings.Builder
		if err := Draw(&b, w.c); err != nil {
			t.Fatal(err)
		}
		if got := b.String(); got != w.want {
			t.Errorf("%d by %d: got %q, want %q", w.c.W, w.c.H, got, w.want)
		}
	}
}

// a grid larger than any screen is drawn as given, not clipped, as the
// doc says: 200 across and 50 down is 50 rows each placed at its own row,
// each sending its one colour pair once and 200 cells, then reset.
func TestAGridBeyondAnyScreenIsDrawnWhole(t *testing.T) {
	c := &Cells{W: 200, H: 50, Cells: make([]Cell, 200*50)}
	col := color.RGBA{1, 2, 3, 255}
	for i := range c.Cells {
		c.Cells[i] = Cell{Rune: 'x', Fg: col, Bg: col}
	}
	var b strings.Builder
	ends(t, "draw", func() { Draw(&b, c) })
	s := b.String()
	for _, w := range []struct {
		of   string
		want int
	}{
		{"x", 200 * 50}, {reset, 50}, {fgOf(1, 2, 3), 50}, {bgOf(1, 2, 3), 50},
		{"1H", 50}, {"\x1b[50;1H", 1}, {"\x1b[51;1H", 0},
	} {
		if got := strings.Count(s, w.of); got != w.want {
			t.Errorf("%q appears %d times, want %d", w.of, got, w.want)
		}
	}
	if !strings.HasPrefix(s, frameBegin+"\x1b[1;1H") || !strings.HasSuffix(s, reset+frameEnd) {
		t.Error("the frame does not begin and end as it should")
	}
}

// failing is a writer that takes nothing and says why.
type failing struct{ err error }

func (f failing) Write([]byte) (int, error) { return 0, f.err }

// a writer that fails is told to the caller, as it came, not dropped.
func TestADrawToAWriterThatFailsReturnsItsError(t *testing.T) {
	boom := errors.New("the pipe is gone")
	err := Draw(failing{boom}, &Cells{W: 1, H: 1, Cells: make([]Cell, 1)})
	if !errors.Is(err, boom) {
		t.Errorf("got %v, want %v", err, boom)
	}
	var b strings.Builder
	if err := Draw(&b, &Cells{}); err != nil {
		t.Errorf("a writer that works gave %v", err)
	}
}
