package frame_test

import (
	"sync"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

// What the race test chooses, each picked in testdata/numbers.yaml.
var (
	writers  = pickedWhole("writers")
	raceWide = pickedWhole("race-wide")
	raceTall = pickedWhole("race-tall")
)

// numbered is the pixel that only one place holds: the place counted from
// one, in red. Its depth is zero, so no pixel it fills is a miss.
func numbered(width int, at place) frame.Pixel {
	return frame.Pixel{R: float32(rowMajor(width, at) + one)}
}

// painter is one of the goroutines that share a frame. It writes the rows of
// the frame that are its own, and no other.
type painter struct {
	grid *frame.Frame
	own  int // the remainder of its rows' numbers, divided by the painters
	of   int // how many painters there are
	done *sync.WaitGroup
}

// paint writes every pixel of the painter's rows, then says it is done.
func (each painter) paint() {
	defer each.done.Done()
	for y := each.own; y < each.grid.H; y += each.of {
		for x := none; x < each.grid.W; x++ {
			at := place{x: x, y: y}
			each.grid.Set(x, y, numbered(each.grid.W, at))
		}
	}
}

// shared is a frame written by every painter at once, each on its own rows,
// and waited for. Run under -race, a write that shared state between places
// shows here.
func shared() *frame.Frame {
	grid := frame.New(raceWide, raceTall)
	var done sync.WaitGroup
	for own := range writers {
		each := painter{grid: grid, own: own, of: writers, done: &done}
		done.Add(one)
		go each.paint()
	}
	done.Wait()
	return grid
}

// goroutines that set different pixels of one frame at once leave every pixel
// as its own goroutine set it. Under -race it also finds a shared field.
func TestWritersOnDifferentPixelsDoNotCollide(t *testing.T) {
	grid := shared()
	for y := none; y < grid.H; y++ {
		for x := none; x < grid.W; x++ {
			at := place{x: x, y: y}
			if got, want := grid.At(x, y), numbered(grid.W, at); got != want {
				t.Errorf("the pixel at %v is %+v, want %+v", at, got, want)
			}
		}
	}
}
