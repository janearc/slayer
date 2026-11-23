package trace

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/render/frame"
)

// The size of the pictures the kept draws are held to, and the rows a
// stopped draw is let finish before it is told to stop.
const (
	keptSide  = 16 // defined: small enough to draw at once
	keptRays  = 2  // defined: so rows are averaged from fine rows
	stopAfter = 5  // defined: some rows, fewer than the picture's
	oneWorker = 1  // defined: rows drawn one after another
)

// never is the stop of a draw that does not stop: no context at all,
// which NewKeeping says it takes.
var never context.Context

// memoryRows is rows kept in memory, which can stop a draw after so many
// rows are drawn, and can be told to hand back rows of the wrong width.
type memoryRows struct {
	mutex  sync.Mutex
	kept   map[int][]frame.Pixel
	begun  int
	drawn  int
	stopAt int
	stop   context.CancelFunc
	short  bool
}

// newMemoryRows is rows that keep nothing yet and never stop a draw.
func newMemoryRows() *memoryRows {
	return &memoryRows{kept: map[int][]frame.Pixel{}}
}

// Begin counts the picture's rows.
func (rows *memoryRows) Begin(count int) {
	rows.mutex.Lock()
	defer rows.mutex.Unlock()
	rows.begun = count
}

// Taken is a kept row, or one pixel short of it where the rows are short.
func (rows *memoryRows) Taken(row, width int) ([]frame.Pixel, bool) {
	rows.mutex.Lock()
	defer rows.mutex.Unlock()
	pixels, found := rows.kept[row]
	if found && rows.short {
		return pixels[1:], true
	}
	return pixels, found
}

// Drawn keeps a row, and stops the draw once stopAt rows are drawn.
func (rows *memoryRows) Drawn(row int, pixels []frame.Pixel) {
	rows.mutex.Lock()
	defer rows.mutex.Unlock()
	rows.kept[row] = append([]frame.Pixel(nil), pixels...)
	rows.drawn++
	if rows.stop != nil && rows.drawn == rows.stopAt {
		rows.stop()
	}
}

// keptDraw is SampledKept of the smoking scene at the test's size.
func keptDraw(keeping Keeping) (*frame.Frame, error) {
	return SampledKept(smoking(), overTheScene, film.Pinhole{}, 0,
		keptSide, keptSide, keptRays, nil, keeping)
}

// A draw by rows that keeps nothing is Sampled's picture to the bit.
func TestAKeptDrawIsSampled(t *testing.T) {
	want := sampled(smoking(), overTheScene, film.Pinhole{}, 0, keptSide,
		keptSide, keptRays)
	got := must(keptDraw(Keeping{}))
	samePixels(t, got, want, "drawn by rows,")
}

// At one ray a pixel a draw by rows is Sampled's too, whose fine frame
// is the picture.
func TestAKeptDrawOfOneRayIsSampled(t *testing.T) {
	want := sampled(smoking(), overTheScene, film.Pinhole{}, 0, keptSide,
		keptSide, oneRay)
	smoke := smoking()
	got := must(SampledKept(smoke, overTheScene, film.Pinhole{}, 0,
		keptSide, keptSide, oneRay, nil, Keeping{}))
	samePixels(t, got, want, "drawn by rows at one ray,")
}

// A draw stopped partway gives the stop's error and keeps what it
// finished; asked again it takes those up, draws the rest, and the
// picture is Sampled's to the bit. One worker draws, so the rows
// finished before the stop are exactly those it was let finish.
func TestAStoppedDrawPicksUpWhereItStopped(t *testing.T) {
	workers := runtime.GOMAXPROCS(oneWorker)
	t.Cleanup(func() { runtime.GOMAXPROCS(workers) })
	rows := newMemoryRows()
	stop, cancel := context.WithCancel(context.Background())
	defer cancel()
	rows.stopAt, rows.stop = stopAfter, cancel
	_, err := keptDraw(NewKeeping(stop, rows))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("a stopped draw gave %v, not the stop's error", err)
	}
	if len(rows.kept) != stopAfter {
		t.Fatalf("a stopped draw kept %d rows, not the %d it finished",
			len(rows.kept), stopAfter)
	}
	taken := len(rows.kept)
	rows.stop, rows.drawn = nil, 0
	keeping := NewKeeping(context.Background(), rows)
	got := must(keptDraw(keeping))
	if rows.drawn != keptSide-taken {
		t.Errorf("asked again it drew %d rows, not the %d left",
			rows.drawn, keptSide-taken)
	}
	want := sampled(smoking(), overTheScene, film.Pinhole{}, 0, keptSide,
		keptSide, keptRays)
	samePixels(t, got, want, "picked up,")
}

// Rows are told how many the picture has before any is drawn.
func TestKeptRowsAreToldThePicturesRows(t *testing.T) {
	rows := newMemoryRows()
	keeping := NewKeeping(never, rows)
	must(keptDraw(keeping))
	if rows.begun != keptSide || len(rows.kept) != keptSide {
		t.Errorf("rows were told %d and kept %d, not %d of each",
			rows.begun, len(rows.kept), keptSide)
	}
}

// Anti: a draw told to stop before it begins draws and keeps no row,
// and gives the stop's error.
func TestADrawStoppedFirstKeepsNothing(t *testing.T) {
	rows := newMemoryRows()
	stop, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := keptDraw(NewKeeping(stop, rows))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("a draw stopped first gave %v, not the stop's error",
			err)
	}
	if len(rows.kept) != 0 {
		t.Errorf("a draw stopped first kept %d rows", len(rows.kept))
	}
}

// Anti: a kept row of another width is not taken up; it is drawn again,
// and the picture is still Sampled's.
func TestARowOfAnotherWidthIsDrawnAgain(t *testing.T) {
	rows := newMemoryRows()
	keeping := NewKeeping(never, rows)
	must(keptDraw(keeping))
	rows.short, rows.drawn = true, 0
	again := NewKeeping(never, rows)
	got := must(keptDraw(again))
	if rows.drawn != keptSide {
		t.Errorf("with every kept row short it drew %d rows, not %d",
			rows.drawn, keptSide)
	}
	want := sampled(smoking(), overTheScene, film.Pinhole{}, 0, keptSide,
		keptSide, keptRays)
	samePixels(t, got, want, "with short rows,")
}
