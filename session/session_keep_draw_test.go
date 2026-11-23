//go:build draw

package session

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

// The rows a stopped frame is let finish, and the one worker that makes
// that count exact.
const (
	heldStopAfter = 5 // defined: some rows, fewer than the frame's
	heldOneWorker = 1 // defined: rows drawn one after another
)

// heldRows is a frame's rows kept in memory, counting the rows drawn and
// taken up, which can stop the draw once so many rows are drawn.
type heldRows struct {
	mutex        sync.Mutex
	kept         map[int][]frame.Pixel
	drawn, taken int
	stopAt       int
	stop         context.CancelFunc
}

// newHeldRows is rows that hold nothing yet and never stop a draw.
func newHeldRows() *heldRows {
	return &heldRows{kept: map[int][]frame.Pixel{}}
}

// Begin is told the frame's rows, which the held rows do not need.
func (rows *heldRows) Begin(int) {}

// Taken is a row kept before, counted.
func (rows *heldRows) Taken(row, _ int) ([]frame.Pixel, bool) {
	rows.mutex.Lock()
	defer rows.mutex.Unlock()
	pixels, found := rows.kept[row]
	if found {
		rows.taken++
	}
	return pixels, found
}

// Drawn keeps a row, counted, and stops the draw once stopAt are drawn.
func (rows *heldRows) Drawn(row int, pixels []frame.Pixel) {
	rows.mutex.Lock()
	defer rows.mutex.Unlock()
	rows.kept[row] = append([]frame.Pixel(nil), pixels...)
	rows.drawn++
	if rows.stop != nil && rows.drawn == rows.stopAt {
		rows.stop()
	}
}

// a still drawn with rows to keep keeps every row and is the picture
// drawn without; drawn again it takes every row up and draws none, and
// is the same picture to the byte.
func TestSessionStillKeepsItsRowsAndTakesThemUp(t *testing.T) {
	wanted := stillOf(t, sessionWidow, "", "")
	want := sessionPicture(t, wanted, Ask{})
	rows := newHeldRows()
	first := sessionPicture(t, wanted, Ask{}.Keeping(rows))
	if !bytes.Equal(first, want) || rows.drawn != sessionHeight {
		t.Fatalf("kept, the still drew %d rows of %d and was the "+
			"same picture: %t", rows.drawn, sessionHeight,
			bytes.Equal(first, want))
	}
	again := sessionPicture(t, wanted, Ask{}.Keeping(rows))
	if !bytes.Equal(again, want) || rows.taken != sessionHeight {
		t.Errorf("drawn again it took %d rows of %d and was the same "+
			"picture: %t", rows.taken, sessionHeight,
			bytes.Equal(again, want))
	}
}

// anti: a frame whose context is done partway through its rows stops: it
// gives the context's error and keeps exactly the rows it finished. One
// worker draws, so those are the rows it was let finish.
func TestSessionFrameStoppedPartwayKeepsWhatItFinished(t *testing.T) {
	workers := runtime.GOMAXPROCS(heldOneWorker)
	t.Cleanup(func() { runtime.GOMAXPROCS(workers) })
	session := opening(t, stillOf(t, sessionWidow, "", ""))
	stop, cancel := context.WithCancel(context.Background())
	defer cancel()
	rows := newHeldRows()
	rows.stopAt, rows.stop = heldStopAfter, cancel
	_, err := session.Frame(stop, nothing, Ask{}.Keeping(rows))
	if !errors.Is(err, context.Canceled) || rows.drawn != heldStopAfter {
		t.Errorf("a frame stopped partway gave %v and kept %d rows, "+
			"not %d", err, rows.drawn, heldStopAfter)
	}
}

// a still through a rig of two glimpses keeps no rows, since a row's key
// holds no instant, and is the picture drawn without them; drawn again it
// takes none up.
func TestSessionRiggedStillKeepsNoRows(t *testing.T) {
	rigged := onFixtureRigged(t, "0.5", false)
	wanted := jobFrom(t, "scene: {set: "+rigged+"}\n")
	want := sessionPicture(t, wanted, Ask{})
	rows := newHeldRows()
	first := sessionPicture(t, wanted, Ask{}.Keeping(rows))
	again := sessionPicture(t, wanted, Ask{}.Keeping(rows))
	if !bytes.Equal(first, want) || !bytes.Equal(again, want) {
		t.Error("a rigged still kept is not the picture drawn without")
	}
	if rows.drawn != nothing || rows.taken != nothing {
		t.Errorf("a rigged still kept %d rows and took %d up",
			rows.drawn, rows.taken)
	}
}
