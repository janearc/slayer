// Package terminal converts images to terminal cells, writes them to the
// screen, and measures the available display size.
package terminal

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// Screen is how big the terminal is, in cells and in the sextant pixels a
// render should be drawn at.
type Screen struct {
	Cols, Rows int
	W, H       int // pixels: two a cell across, three down
	// Pixel is one pixel's width over its height, for the lens.
	Pixel float64
	// Measured is false when the terminal did not report its cell size in
	// pixels, and Pixel is a guess from a cell twice as tall as it is wide.
	Measured bool
}

// Measure asks the terminal on f how big it is. no file, and a terminal that
// says it is zero cells across or down, as a fresh pty may, are errors that say
// so, not a screen of no size.
func Measure(f *os.File) (Screen, error) {
	if f == nil {
		return Screen{}, fmt.Errorf("no file to measure: a terminal " +
			"is a file, such as os.Stdout")
	}
	ws, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return Screen{}, fmt.Errorf("%s is not a terminal that says "+
			"its size: %w", f.Name(), err)
	}
	if ws.Col == 0 || ws.Row == 0 {
		return Screen{}, fmt.Errorf("%s says it is %d by %d cells: "+
			"nothing can be drawn; resize it, or set its size "+
			"with stty cols and rows", f.Name(), ws.Col, ws.Row)
	}
	s := Screen{Cols: int(ws.Col), Rows: int(ws.Row), Pixel: 0.75}
	s.W, s.H = s.Cols*2, s.Rows*3
	if ws.Xpixel > 0 && ws.Ypixel > 0 && ws.Col > 0 && ws.Row > 0 {
		cellW := float64(ws.Xpixel) / float64(ws.Col)
		cellH := float64(ws.Ypixel) / float64(ws.Row)
		s.Pixel, s.Measured = (cellW/2)/(cellH/3), true
	}
	return s, nil
}

// Draw writes the cells from the top left of the screen as one synchronised
// frame, so a terminal that keeps mode 2026 shows it all at once; one that
// does not may draw it as it comes.
//
// the cells are drawn as given: wider or taller than the screen, they wrap
// or pile on its last line, the caller's to clip.
func Draw(out io.Writer, c *Cells) error {
	var b strings.Builder
	b.WriteString("\x1b[?2026h\x1b[H")
	for y := 0; y < c.H; y++ {
		fmt.Fprintf(&b, "\x1b[%d;1H", y+1)
		var fg, bg [3]uint8
		first := true
		for x := 0; x < c.W; x++ {
			cell := c.At(x, y)
			f := [3]uint8{cell.Fg.R, cell.Fg.G, cell.Fg.B}
			g := [3]uint8{cell.Bg.R, cell.Bg.G, cell.Bg.B}
			if first || f != fg {
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm",
					f[0], f[1], f[2])
				fg = f
			}
			if first || g != bg {
				fmt.Fprintf(&b, "\x1b[48;2;%d;%d;%dm",
					g[0], g[1], g[2])
				bg = g
			}
			first = false
			b.WriteRune(cell.Rune)
		}
		b.WriteString("\x1b[0m")
	}
	b.WriteString("\x1b[?2026l")
	_, err := io.WriteString(out, b.String())
	return err
}

// Session is the terminal taken over for a picture: the alternate screen,
// no cursor, keys read one at a time. End gives it all back.
type Session struct {
	in, out *os.File
	old     *term.State
}

// Check says whether Open would take in and out. It touches neither.
// Ask it before a long job. Open the terminal when the job is done.
func Check(in, out *os.File) error {
	if !term.IsTerminal(int(in.Fd())) || !term.IsTerminal(int(out.Fd())) {
		return fmt.Errorf("needs a terminal on both input and output")
	}
	return nil
}

// Open takes over the terminal on in and out.
func Open(in, out *os.File) (*Session, error) {
	if err := Check(in, out); err != nil {
		return nil, err
	}
	old, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return nil, err
	}
	if _, err := io.WriteString(out, "\x1b[?1049h\x1b[?25l"); err != nil {
		// a terminal left raw, with nothing on it, is worse than none
		term.Restore(int(in.Fd()), old)
		return nil, fmt.Errorf("could not take over the screen: %w",
			err)
	}
	return &Session{in: in, out: out, old: old}, nil
}

// Wait blocks until a key is pressed.
func (s *Session) Wait() error {
	var b [16]byte
	_, err := s.in.Read(b[:])
	return err
}

// Next waits for a key and says whether it asks to stop: q, escape or
// ctrl-c, which raw mode delivers as a key rather than a signal. Any other
// key, enter among them, is a request for the next picture: an arrow or a
// function key too, which arrives as escape and more, not escape alone.
func (s *Session) Next() (stop bool, err error) {
	var b [16]byte
	n, err := s.in.Read(b[:])
	if err != nil || n == 0 {
		return true, err
	}
	if n > 1 && b[0] == 0x1b {
		return false, nil
	}
	switch b[0] {
	case 'q', 0x1b, 0x03:
		return true, nil
	}
	return false, nil
}

// End gives the terminal back: colour reset, the cursor shown, the main
// screen, raw mode undone. Calling it twice is harmless. it says nothing
// of a write or a restore that fails, and a caller that panics without
// it leaves the terminal raw: defer it.
func (s *Session) End() {
	if s.old == nil {
		return
	}
	io.WriteString(s.out, "\x1b[0m\x1b[?25h\x1b[?1049l")
	term.Restore(int(s.in.Fd()), s.old)
	s.old = nil
}
