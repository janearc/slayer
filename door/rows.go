package door

// A still drawn on the processor is drawn a row at a time, and each row
// is kept on disk (package diskcache, kind rows) as it is finished, so a
// draw that is stopped, or whose process dies, is picked up from the
// rows it finished when the same job is drawn again.
//
// A row's key is the still's light key, the build and the sources list
// nearest its set, as the pictures' key is, and the row's number: rows
// of another job, size, camera, rays or build are never taken up. An
// entry is whole or not served, as every diskcache entry is. --fresh
// takes up nothing, and keeps what it draws.
//
// The rows are removed once the still is written, so a still drawn
// again is drawn whole, as it was before rows were kept; only a still
// that did not finish leaves rows behind. While the rows are drawn, a
// line at a steady interval says how far the draw is.

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/janearc/slayer/diskcache"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/render/progress"
)

// rowKind keeps the rows of stills not yet finished. A 1600 by 900
// still's rows are about 29 MB, so 2 GiB keeps the rows of about seventy
// stopped stills; past it the least recently used go first.
var rowKind = diskcache.MustKind("rows", rowCap, "a 1600 by 900 "+
	"still's rows are about 29 MB; this keeps about seventy")

// rowCap is the rows' cap in bytes; see rowKind.
const rowCap = 2 << 30

// A kept pixel is frame.Pixel's five float32 words, R, G, B, Cover and
// Depth in that order, each little-endian.
const (
	pixelWords = 5 // defined: frame.Pixel's five fields
	wordBytes  = 4 // defined: a float32 is four bytes
	pixelBytes = pixelWords * wordBytes
)

// Where each of a kept pixel's words stands, in the order encodedRow
// writes them. Defined.
const (
	redWord = iota
	greenWord
	blueWord
	coverWord
	depthWord
)

// keptRows is a still's rows as its draw keeps and takes them up, and
// the count of them its progress lines say. Its methods are trace.Rows.
type keptRows struct {
	// store is nil where rows cannot be kept: no cache home, or a job
	// that is not a set's still. The draw is then counted and said only.
	store *diskcache.Store
	key   diskcache.Key
	// fresh takes up no row.
	fresh bool
	out   io.Writer
	now   func() time.Time
	every time.Duration
	mutex sync.Mutex
	// all is the picture's rows; taken those taken up, drawn those drawn
	all, taken, drawn int
	began             time.Time
	// failed is whether a row could not be kept, which is said once
	failed      bool
	quit, ended chan struct{}
}

// rowsKey is the key a job's rows extend: its light key, the build and
// the sources list nearest its set, or refused where the job is not a
// set's still.
func rowsKey(wanted job.Job) diskcache.Key {
	lightJobKey := wanted.LightKey()
	return stillKey(diskcache.NewKey(rowKind).Built().
		Digest("light", lightJobKey), wanted)
}

// rowsKept is the still's rows: kept under the cache home where the job
// is a set's still and there is a home, and counted either way. Only the
// real workshop's draws have rows; nil is none.
func (run runner) rowsKept() *keptRows {
	if !run.keeps {
		return nil
	}
	out := run.env.progress
	if out == nil {
		out = io.Discard
	}
	kept := &keptRows{key: rowsKey(run.wanted), fresh: run.fresh,
		out: out, now: run.now, every: progress.StillEvery}
	if kept.key.Err() != nil {
		return kept
	}
	home, err := diskcache.Home(run.env.lookup)
	if err != nil {
		return kept
	}
	if store, err := diskcache.Open(home, rowKind); err == nil {
		kept.store = store
	}
	return kept
}

// Begin is told the picture's rows, and starts the progress lines.
func (kept *keptRows) Begin(rows int) {
	kept.mutex.Lock()
	defer kept.mutex.Unlock()
	if kept.quit != nil {
		return
	}
	kept.all, kept.began = rows, kept.now()
	kept.quit, kept.ended = make(chan struct{}), make(chan struct{})
	go kept.telling(kept.quit, kept.ended)
}

// Taken is row number row as an earlier draw kept it, where it was kept
// whole at this width, and the draw is not fresh.
func (kept *keptRows) Taken(row, width int) ([]frame.Pixel, bool) {
	if kept.store == nil || kept.fresh {
		return nil, false
	}
	hit, err := kept.store.Get(kept.keyOf(row))
	if err != nil || !hit.Found() {
		return nil, false
	}
	pixels, whole := decodedRow(hit.Payload(), width)
	if whole {
		kept.mutex.Lock()
		kept.taken++
		kept.mutex.Unlock()
	}
	return pixels, whole
}

// Drawn counts a row just drawn and keeps it; a row that cannot be kept
// is said once and the draw goes on, since keeping only saves time.
func (kept *keptRows) Drawn(row int, pixels []frame.Pixel) {
	kept.mutex.Lock()
	kept.drawn++
	kept.mutex.Unlock()
	if kept.store == nil {
		return
	}
	err := kept.store.Put(kept.keyOf(row), encodedRow(pixels))
	if err == nil {
		return
	}
	kept.mutex.Lock()
	first := !kept.failed
	kept.failed = true
	kept.mutex.Unlock()
	if first {
		kept.say(fmt.Sprintf("the rows are not kept: %v", err))
	}
}

// keyOf is row number row's key.
func (kept *keptRows) keyOf(row int) diskcache.Key {
	return kept.key.Text("row", strconv.Itoa(row))
}

// telling says a progress line each interval until quit is closed, and
// closes ended once it says no more.
func (kept *keptRows) telling(quit <-chan struct{}, ended chan<- struct{}) {
	defer close(ended)
	ticker := time.NewTicker(kept.every)
	defer ticker.Stop()
	for {
		select {
		case <-quit:
			return
		case <-ticker.C:
			kept.say(kept.line())
		}
	}
}

// line is the draw's progress, as progress.StillLine says it: the rows
// finished, taken up or drawn, of all, and the time taken and left.
func (kept *keptRows) line() string {
	kept.mutex.Lock()
	defer kept.mutex.Unlock()
	return progress.StillLine(kept.taken+kept.drawn, kept.all, kept.taken,
		kept.now().Sub(kept.began))
}

// say writes one line where progress goes; a line that cannot be written
// is let go, since it only says how far the draw is.
func (kept *keptRows) say(line string) {
	_, _ = fmt.Fprintln(kept.out, line)
}

// close stops the progress lines and says the last one. Nil rows, or
// rows never begun, say nothing.
func (kept *keptRows) close() {
	if kept == nil {
		return
	}
	kept.mutex.Lock()
	quit, ended := kept.quit, kept.ended
	kept.quit = nil
	kept.mutex.Unlock()
	if quit == nil {
		return
	}
	close(quit)
	<-ended
	kept.say(kept.line())
}

// forget removes every row kept of the still, once it is written. A row
// that cannot be removed is said once, and the rest are left.
func (kept *keptRows) forget() {
	if kept == nil || kept.store == nil {
		return
	}
	for row := nothing; row < kept.all; row++ {
		if err := kept.store.Remove(kept.keyOf(row)); err != nil {
			kept.say(fmt.Sprintf("the rows are not removed: %v", err))
			return
		}
	}
}

// encodedRow is a row of pixels as kept: each pixel's five words.
func encodedRow(pixels []frame.Pixel) []byte {
	raw := make([]byte, nothing, len(pixels)*pixelBytes)
	for _, pixel := range pixels {
		for _, word := range []float32{pixel.R, pixel.G, pixel.B,
			pixel.Cover, pixel.Depth} {
			raw = binary.LittleEndian.AppendUint32(raw,
				math.Float32bits(word))
		}
	}
	return raw
}

// decodedRow is a kept row read back, and whether it is a whole row of
// width pixels; a row of another length is not one.
func decodedRow(raw []byte, width int) ([]frame.Pixel, bool) {
	if width < nothing || len(raw) != width*pixelBytes {
		return nil, false
	}
	pixels := make([]frame.Pixel, width)
	for x := range pixels {
		words := raw[x*pixelBytes:]
		pixels[x] = frame.Pixel{R: wordAt(words, redWord),
			G: wordAt(words, greenWord), B: wordAt(words, blueWord),
			Cover: wordAt(words, coverWord),
			Depth: wordAt(words, depthWord)}
	}
	return pixels, true
}

// wordAt is the index'th float32 word of raw.
func wordAt(raw []byte, index int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(
		raw[index*wordBytes:]))
}
