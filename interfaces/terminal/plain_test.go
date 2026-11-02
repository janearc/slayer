package terminal_test

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/slayer/interfaces/terminal"
)

// what the plain check counts in, each defined
const (
	every    = -1 // defined: regexp's count for every match, not the first few
	firstRow = 1  // defined: rows are counted from one
	sextants = 64 // defined: a sextant's six squares make 2 to the 6th patterns
)

// yes is how a table says a case is plain
const yes = "yes"

// These are the pieces of a frame as the standards have them. Byte
// 0x1b is escape. ECMA-48 writes a cursor position as a row and a
// column between a bracket and an H. Colour ends in m. Private mode
// 2026 is synchronised output. Its h begins and its l ends.
const (
	syncBegin  = "\x1b[?2026h"
	frameBegin = syncBegin + "\x1b[H"
	syncEnd    = "\x1b[?2026l"
	frameEnd   = syncEnd
	reset      = "\x1b[0m"
	moveStart  = "\x1b["
	moveEnd    = ";1H"
)

// allowedEscapes matches the only escapes a stream may hold. They are
// synchronised output, cursor home and a move to column one of a row.
// Then 24-bit foreground and background colour. Then reset.
const allowedEscapes = `\x1b\[(\?2026[hl]|H|[0-9]+;1H|` +
	`38;2;[0-9]+;[0-9]+;[0-9]+m|48;2;[0-9]+;[0-9]+;[0-9]+m|0m)`

var allowed = regexp.MustCompile(allowedEscapes)

// errPairs is what a stream with frames that do not pair is told.
var errPairs = errors.New("frames do not begin and end in pairs")

// place is the move to column one of a row, counting from one.
func place(row int) string {
	return moveStart + strconv.Itoa(row) + moveEnd
}

// rgb is a colour as the escapes write it.
type rgb struct {
	red, green, blue int
}

// colours is a foreground and a background colour set, as bytes.
func colours(fore, back rgb) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm",
		fore.red, fore.green, fore.blue, back.red, back.green, back.blue)
}

// plain says why a stream is not plain ANSI. It gives nil if it is.
// Every escape must be an allowed one. Every cursor move must be to a
// row of the frame. Frames begin and end in pairs. All else must be a
// sextant glyph. No newline or other control gets through.
// Kitty graphics and queries are not allowed escapes. None can pass.
func plain(stream string, rows int) error {
	if err := paired(stream, rows); err != nil {
		return err
	}
	return onlyGlyphs(stream)
}

// paired is whether frames begin and end in pairs and every move in
// them is to a row of the frame.
func paired(stream string, rows int) error {
	open := false
	for _, found := range allowed.FindAllString(stream, every) {
		switch {
		case found == syncBegin && open:
			return errPairs
		case found == syncEnd && !open:
			return errPairs
		case found == syncBegin:
			open = true
		case found == syncEnd:
			open = false
		case strings.HasSuffix(found, moveEnd):
			if err := moveWithin(found, rows); err != nil {
				return err
			}
		}
	}
	if open {
		return errPairs
	}
	return nil
}

// moveWithin is whether a move to column one is to a row of the frame.
func moveWithin(move string, rows int) error {
	digits := strings.TrimPrefix(move, moveStart)
	digits = strings.TrimSuffix(digits, moveEnd)
	row, err := strconv.Atoi(digits)
	if err != nil || row < firstRow || row > rows {
		return fmt.Errorf("the move %q is outside a frame of %d rows", move,
			rows)
	}
	return nil
}

// onlyGlyphs is whether all that is not an allowed escape is a glyph a
// sextant picture is drawn with.
func onlyGlyphs(stream string) error {
	known := map[rune]bool{}
	for pattern := range uint8(sextants) {
		known[terminal.Glyph(pattern)] = true
	}
	for _, between := range allowed.Split(stream, every) {
		for _, glyph := range between {
			if !known[glyph] {
				return fmt.Errorf("%q is not a plain escape or a glyph, in %q",
					glyph, between)
			}
		}
	}
	return nil
}

// goodFrame is one plain frame of a single row.
func goodFrame() string {
	return frameBegin + place(firstRow) + "▌" + reset + frameEnd
}

// The check for plain ANSI bites. Each row of the table has text to add
// to a plain frame and says whether the result is plain. Kitty
// graphics and queries are refused. So are titles and screen clears.
// So are newlines and letters. So are unpaired frames. So are moves
// outside the frame.
func TestTheCheckForPlainANSIRefusesWhatIsNot(t *testing.T) {
	for _, row := range casesOf(t, plainnessFile) {
		tail, err := strconv.Unquote(`"` + textIn(t, row, "tail") + `"`)
		if err != nil {
			t.Fatal(err)
		}
		verdict := plain(goodFrame()+tail, wholeIn(t, row, "rows"))
		name := textIn(t, row, "name")
		if (verdict == nil) != (textIn(t, row, "plain") == yes) {
			t.Errorf("%s: the check said %v", name, verdict)
		}
	}
}
