package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// ultraviolet was looked at for this and not used: it draws a grid of
// cells for a program that owns the screen, it has no tagged release to
// pin, and two lines that are redrawn in place need only a carriage
// return, an erase and a cursor up. so the status is plain ANSI.

// spinning is the frames of the spinner. they are ASCII, so a spinner
// reads as one in a log, a pager or a terminal with no fancy glyphs.
const spinning = `|/-\`

// refresh is how often the status is drawn again, so the spinner turns
// even while a slow server sends nothing.
const refresh = 100 * time.Millisecond

// fallbackColumns is the width taken when the terminal will not say.
const fallbackColumns = 80

// the control sequences the status is drawn with.
const (
	hideCursor = "\x1b[?25l"
	showCursor = "\x1b[?25h"
	eraseLine  = "\x1b[2K"
	cursorUp   = "\x1b[A"
	cursorDown = "\x1b[B"
	lineStart  = "\r"
)

// decimal is the size of a step between units of bytes.
const decimal = 1000

// unitNames are the units of bytes, from bytes up.
var unitNames = []string{"B", "KB", "MB", "GB", "TB", "PB"}

// bigUnit is the first unit that is said with two decimals, not one.
const bigUnit = 3

// humanBytes is a count of bytes in decimal units, as 812.4 MB or
// 1.71 GB.
func humanBytes(count int64) string {
	if count < decimal {
		return fmt.Sprintf("%d B", count)
	}
	value, unit := float64(count), 0
	for value >= decimal && unit < len(unitNames)-1 {
		value /= decimal
		unit++
	}
	if unit >= bigUnit {
		return fmt.Sprintf("%.2f %s", value, unitNames[unit])
	}
	return fmt.Sprintf("%.1f %s", value, unitNames[unit])
}

// tally is how the sources have gone so far.
type tally struct {
	sources int
	fetched int
	kept    int
	never   int
	failed  int
	bytes   int64
}

// done is how many sources have been dealt with.
func (count tally) done() int {
	return count.fetched + count.kept + count.never + count.failed
}

// percent is how much of the total has come, never over a hundred.
func percent(received, total int64) int {
	const whole = 100
	return int(min(received*whole/total, whole))
}

// cut is a line cut to the columns, less one so that it never reaches
// the last column and is never wrapped.
func cut(line string, columns int) string {
	letters := []rune(line)
	if limit := max(columns-1, 0); len(letters) > limit {
		return string(letters[:limit])
	}
	return line
}

// statusLines are the two lines of the live status for a frame of the
// spinner: what is being fetched and how much of it has come, and how
// many sources are done. each is cut to the columns.
func statusLines(frame int, key string, received, total int64,
	count tally, columns int) [2]string {
	first := fmt.Sprintf("%c fetching %s  %s", spinning[frame%len(spinning)],
		key, humanBytes(received))
	if total > 0 {
		first += fmt.Sprintf(" of %s  %d%%", humanBytes(total),
			percent(received, total))
	}
	second := fmt.Sprintf("  %d of %d: %d fetched, %d already here, %d failed",
		count.done(), count.sources, count.fetched, count.kept, count.failed)
	return [2]string{cut(first, columns), cut(second, columns)}
}

// summary is the one line that replaces the status when fetching is
// over: what came, what was here, what failed, and the shelf.
func summary(count tally, shelf string, stopped bool) string {
	said := fmt.Sprintf("fetched %d (%s), %d already here, %d failed",
		count.fetched, humanBytes(count.bytes), count.kept, count.failed)
	if count.never > 0 {
		said += fmt.Sprintf(", %d never fetched", count.never)
	}
	if stopped {
		said = "stopped early: " + said
	}
	return said + "; shelf " + shelf
}

// display is the live status, two lines kept below what is printed,
// and drawn again in place. it is safe to use from the fetching loop and
// from the timer that turns the spinner.
type display struct {
	mu       sync.Mutex
	out      io.Writer
	columns  func() int
	frame    int
	key      string
	received int64
	total    int64
	count    tally
	drawn    bool
	stop     chan struct{}
	stopped  sync.WaitGroup
}

// newDisplay is a display writing to out, asking columns how wide the
// terminal is, for a number of sources.
func newDisplay(out io.Writer, columns func() int, sources int) *display {
	return &display{out: out, columns: columns,
		count: tally{sources: sources}, stop: make(chan struct{})}
}

// terminalColumns says whether out is a terminal that can be drawn on,
// and if so how to ask how wide it is. a dumb terminal is not one.
func terminalColumns(out io.Writer) (func() int, bool) {
	file, ok := out.(*os.File)
	if !ok || os.Getenv("TERM") == "dumb" || !term.IsTerminal(int(file.Fd())) {
		return nil, false
	}
	return func() int {
		width, _, err := term.GetSize(int(file.Fd()))
		if err != nil || width <= 0 {
			return fallbackColumns
		}
		return width
	}, true
}

// begin hides the cursor, draws the status, and starts the timer that
// draws it again.
func (screen *display) begin() {
	screen.mu.Lock()
	io.WriteString(screen.out, hideCursor)
	screen.draw()
	screen.mu.Unlock()
	screen.stopped.Add(1)
	go screen.turn()
}

// turn draws the status again each refresh until it is stopped.
func (screen *display) turn() {
	defer screen.stopped.Done()
	clock := time.NewTicker(refresh)
	defer clock.Stop()
	for {
		select {
		case <-screen.stop:
			return
		case <-clock.C:
			screen.mu.Lock()
			screen.frame++
			screen.draw()
			screen.mu.Unlock()
		}
	}
}

// draw writes the two lines, and leaves the cursor at the start of the
// first. the caller holds the lock.
func (screen *display) draw() {
	lines := statusLines(screen.frame, screen.key, screen.received,
		screen.total, screen.count, screen.columns())
	fmt.Fprint(screen.out, lineStart+eraseLine+lines[0]+"\n"+eraseLine+
		lines[1]+cursorUp+lineStart)
	screen.drawn = true
}

// wipe takes the two lines off the screen, if they are there. the caller
// holds the lock.
func (screen *display) wipe() {
	if screen.drawn {
		fmt.Fprint(screen.out, lineStart+eraseLine+cursorDown+eraseLine+
			cursorUp+lineStart)
		screen.drawn = false
	}
}

// above prints a line above the status, and draws the status again.
func (screen *display) above(line string) {
	screen.mu.Lock()
	defer screen.mu.Unlock()
	screen.wipe()
	fmt.Fprintln(screen.out, line)
	screen.draw()
}

// start says which source is being fetched now.
func (screen *display) start(key string) {
	screen.mu.Lock()
	defer screen.mu.Unlock()
	screen.key, screen.received, screen.total = key, 0, 0
}

// report is the shelf's progress function.
func (screen *display) report(key string, received, total int64) {
	screen.mu.Lock()
	defer screen.mu.Unlock()
	screen.key, screen.received, screen.total = key, received, total
}

// record counts a source as dealt with, from what Fetch said and
// whether it failed. a failure is printed above the status.
func (screen *display) record(key, said string, failure error) {
	if failure != nil {
		screen.mu.Lock()
		screen.count.failed++
		screen.mu.Unlock()
		screen.above(failure.Error())
		return
	}
	screen.mu.Lock()
	defer screen.mu.Unlock()
	switch {
	case strings.HasPrefix(said, key+": kept, "):
		screen.count.kept++
	case strings.HasPrefix(said, key+": never fetched"):
		screen.count.never++
	default:
		screen.count.fetched++
		screen.count.bytes += screen.received
	}
}

// finish stops the timer, takes the status off the screen, shows the
// cursor, and prints the summary line in its place.
func (screen *display) finish(shelf string, stopped bool) {
	close(screen.stop)
	screen.stopped.Wait()
	screen.mu.Lock()
	defer screen.mu.Unlock()
	screen.wipe()
	io.WriteString(screen.out, showCursor)
	fmt.Fprintln(screen.out, summary(screen.count, shelf, stopped))
}
