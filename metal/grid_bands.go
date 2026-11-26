//go:build darwin

package metal

/*
#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"time"
	"unsafe"
)

// GridBands runs the same globally addressed grid with one reusable readback
// band. Inputs remain on Metal for the call. A callback must consume or copy
// its slice before returning; later bands overwrite that storage.
//
// rows must be positive: it is the most rows a band holds, and so the
// readback's size; how many a band does hold its time decides
// (band_heights.go). Calls and
// callbacks must not reenter this Kernel. Callback errors stop dispatch,
// release the buffers and propagate unchanged.
func (kernel *Kernel) GridBands(width, height, rows, per int,
	inputs [][]byte, consume func(first, down int, samples []float32) error,
) error {
	return kernel.GridBandsSent(width, height, rows, per, inputs, nil,
		consume)
}

// GridBandsSent is GridBands that calls sent, if it is not nil, once,
// when every input has been copied to Metal and before the first band is
// drawn. The inputs are not read again after it: a caller that holds the
// only other copy of one may let it go there.
func (kernel *Kernel) GridBandsSent(width, height, rows, per int,
	inputs [][]byte, sent func(),
	consume func(first, down int, samples []float32) error,
) error {
	return kernel.gridBandsSent(0, 0, width, height, rows, per, inputs, sent, consume)
}

// RegionBandsSent draws a compact grid at a full-grid origin. The kernel sees
// that origin plus each band's local first row; callbacks receive local rows.
func (kernel *Kernel) RegionBandsSent(x, y, width, height, rows, per int,
	inputs [][]byte, sent func(),
	consume func(first, down int, samples []float32) error,
) error {
	if !regionOriginFits(x, y, width, height) {
		return errGridSize
	}
	return kernel.gridBandsSent(x, y, width, height, rows, per, inputs, sent, consume)
}

// regionOriginFits checks that a dispatch fits its integer coordinates.
func regionOriginFits(x, y, width, height int) bool {
	return x >= 0 && y >= 0 && width > 0 && height > 0 &&
		width <= math.MaxInt32 && height <= math.MaxInt32 &&
		x <= math.MaxInt32-width && y <= math.MaxInt32-height
}

// gridBandsSent dispatches a grid in compact bands from the given origin.
func (kernel *Kernel) gridBandsSent(x, y, width, height, rows, per int,
	inputs [][]byte, sent func(),
	consume func(first, down int, samples []float32) error,
) error {
	if err := kernel.validBands(width, height, rows, per, inputs, consume); err != nil {
		return err
	}
	if width <= 0 || height <= 0 {
		return nil
	}
	rows = min(rows, height)
	stream, err := kernel.openBands(width, height, rows, per, inputs)
	if err != nil {
		return err
	}
	defer C.slayer_metal_grid_close(stream)
	var why [1024]C.char
	if C.slayer_metal_grid_origin(stream, C.int(x), C.int(y), &why[0], C.int(len(why))) != 0 {
		return errors.New("metal: " + C.GoString(&why[0]))
	}
	if sent != nil {
		sent()
	}
	defer runtime.KeepAlive(kernel)
	samples := make([]float32, width*rows*per)
	place := func(_, down int) []float32 { return samples[:width*down*per] }
	drawing := kernel.drawingOf(stream, height, rows)
	return drawing.draw(place, consume)
}

// bandDrawing is one picture height rows tall drawn through a stream of
// the kernel in bands of at most rows rows, the next down rows tall
// (band_heights.go). ahead is the band committed behind the one being
// waited out, so the device draws one band while the caller reads the
// last; it is no band at the picture's foot.
type bandDrawing struct {
	kernel *Kernel
	stream unsafe.Pointer
	height int
	rows   int
	down   int
	// fixed is whether every band keeps its height whatever its time
	fixed bool
	// least is the least time any band of the picture took on the device,
	// taken as what a command costs before its rows; nought before any
	least time.Duration
	// said is whether the drawing has said a row is over the aim
	said   bool
	ahead  span
	memory *firstBandMemory
}

// draw draws the picture. Each band is waited out, the band after it
// committed, if it was not already, and the band waited out copied into
// the slice place gives it and handed to consume while the next draws.
func (drawing *bandDrawing) draw(place func(first, down int) []float32,
	consume func(first, down int, samples []float32) error) (err error) {
	marks := newFirstBandMemory()
	drawing.memory = &marks
	defer func() { drawing.memory = nil }()
	defer func() {
		// a drawing that stops early, by its consumer or a failure,
		// leaves no band in flight for the stream's next drawing, and
		// says what became of any it waited out
		if err != nil {
			err = errors.Join(err, drawing.settle())
		}
	}()
	band := drawing.after(span{})
	if err := drawing.commit(band); err != nil {
		return err
	}
	marks.mark(bandMemoryCommit)
	for band.down > 0 {
		drawn, spent, err := drawing.await(band)
		if err != nil {
			return err
		}
		marks.mark(bandMemoryWait)
		samples := place(drawn.first, drawn.down)
		if band, err = drawing.next(samples, drawn, spent); err != nil {
			return err
		}
		if err := consume(drawn.first, drawn.down, samples); err != nil {
			return err
		}
		marks.mark(bandMemoryConsume)
	}
	return nil
}

// settle waits out and forgets every band in flight, giving back the
// first failure among them.
func (drawing *bandDrawing) settle() error {
	drawing.ahead = span{}
	var why [1024]C.char
	failed := C.slayer_metal_grid_settle(drawing.stream, &why[0],
		C.int(len(why)))
	runtime.KeepAlive(drawing.kernel)
	if failed != 0 {
		return errors.New("metal: " + C.GoString(&why[0]))
	}
	return nil
}

// commit commits a band, and nothing for no band.
func (drawing *bandDrawing) commit(band span) error {
	if band.down == 0 {
		return nil
	}
	return drawing.kernel.commitBand(drawing.stream, band.first, band.down)
}

// next sizes the bands after drawn by its time, has the band after it
// committed, if it was not already, copies drawn into samples while that
// one draws, and commits the band after that into drawn's buffer. It
// gives back the band to wait out next.
func (drawing *bandDrawing) next(samples []float32, drawn span,
	spent time.Duration) (span, error) {
	drawing.timed(drawn, spent)
	following := drawing.ahead
	drawing.ahead = span{}
	if following.down == 0 {
		following = drawing.after(drawn)
		if err := drawing.commit(following); err != nil {
			return span{}, err
		}
	}
	receipt(drawn.first, drawn.down, spent)
	err := drawing.kernel.copyBand(drawing.stream, drawn.down, samples)
	if err == nil && drawing.memory != nil {
		drawing.memory.mark(bandMemoryCopy)
	}
	if err == nil && following.down > 0 {
		drawing.ahead = drawing.after(following)
		err = drawing.commit(drawing.ahead)
	}
	return following, err
}

// await waits out a band. When the system stopped it for the display,
// any band ahead is waited out and dropped, and the same rows are sent
// again half as tall, a row at least, after a longer turn, up to
// mostStops times. It gives back the band drawn and its time on the
// device.
func (drawing *bandDrawing) await(band span) (span, time.Duration, error) {
	for stops := 1; ; stops++ {
		before := drawing.kernel.GPUTime()
		stopped, err := drawing.kernel.waitBand(drawing.stream)
		if err == nil {
			return band, drawing.kernel.GPUTime() - before, nil
		}
		if !stopped || stops >= mostStops {
			return span{}, 0, stoppedTooOften(band, stops, stopped, err)
		}
		if err := drawing.drop(); err != nil {
			return span{}, 0, err
		}
		band = drawing.halved(band)
		stopReceipt(band.first, band.down, err)
		time.Sleep(stopTurn)
		if err := drawing.commit(band); err != nil {
			return span{}, 0, err
		}
	}
}

// drop waits out the band ahead, if there is one, and forgets it: it is
// drawn again in its turn. A stop for the display is no failure here.
func (drawing *bandDrawing) drop() error {
	if drawing.ahead.down == 0 {
		return nil
	}
	drawing.ahead = span{}
	if stopped, err := drawing.kernel.waitBand(drawing.stream); !stopped {
		return err
	}
	return nil
}

// stoppedTooOften is the error of a band that failed: its own when the
// system did not stop it for the display, or one naming its rows and its
// stops when it was stopped too often to send again.
func stoppedTooOften(band span, stops int, stopped bool, err error) error {
	if !stopped {
		return err
	}
	return fmt.Errorf("metal: the system stopped the rows from %d for "+
		"the display %d times, the last %d rows tall (%s); %w",
		band.first, stops, band.down, presence(), err)
}

// validBands checks dimensions and host limits before narrowing or allocation.
func (kernel *Kernel) validBands(width, height, rows, per int, inputs [][]byte,
	consume func(int, int, []float32) error,
) error {
	if kernel.ctx == nil {
		return errors.New("metal: the kernel is closed")
	}
	if per <= 0 {
		return errNoNumbers
	}
	if len(inputs) > math.MaxInt32 {
		return errTooBig
	}
	if rows <= 0 || consume == nil {
		return errors.New("metal: grid bands need rows and a consumer")
	}
	for _, input := range inputs {
		if len(input) > math.MaxInt32 {
			return errTooBig
		}
	}
	if width > 0 && height > 0 && !gridFits(width, height, per) {
		return errGridSize
	}
	return nil
}

// openBands copies each input directly into its retained Metal buffer. The C
// call completes its copy before returning and never retains the Go pointer.
func (kernel *Kernel) openBands(width, height, rows, per int,
	inputs [][]byte) (unsafe.Pointer, error) {
	defer runtime.KeepAlive(kernel)
	var why [1024]C.char
	stream := C.slayer_metal_grid_begin(kernel.ctx, C.int(width), C.int(height),
		C.int(rows), C.int(per),
		C.int(len(inputs)), &why[0], C.int(len(why)))
	if stream == nil {
		return nil, errors.New("metal: " + C.GoString(&why[0]))
	}
	for _, input := range inputs {
		var pointer unsafe.Pointer
		if len(input) > 0 {
			pointer = unsafe.Pointer(&input[0])
		}
		failed := C.slayer_metal_grid_add(stream, pointer, C.long(len(input)), &why[0], C.int(len(why)))
		runtime.KeepAlive(input)
		if failed != 0 {
			C.slayer_metal_grid_close(stream)
			return nil, errors.New("metal: " + C.GoString(&why[0]))
		}
	}
	return stream, nil
}

// errClosedDuring is a kernel closed while a grid was drawn with it.
var errClosedDuring = errors.New(
	"metal: the kernel closed during grid consumption")

// commitBand commits the band of down rows from first without waiting.
func (kernel *Kernel) commitBand(stream unsafe.Pointer, first,
	down int) error {
	if kernel.ctx == nil {
		return errClosedDuring
	}
	var why [1024]C.char
	failed := C.slayer_metal_grid_commit(stream, C.int(first), C.int(down),
		&why[0], C.int(len(why)))
	runtime.KeepAlive(kernel)
	if failed != 0 {
		return errors.New("metal: " + C.GoString(&why[0]))
	}
	return nil
}

// errPretendStop is the stop a test asked for in place of the system's.
var errPretendStop = errors.New(
	"metal: stopped for the display, as the test asked")

// waitStopped is what slayer_metal_grid_wait returns when StopOn ended
// the wait, and not a failure of the band.
const waitStopped = 2

// waitBand waits out the oldest band in flight, and says whether a
// failure was the system stopping it for the display. A wait StopOn
// ended gives the context's error, and is never that.
func (kernel *Kernel) waitBand(stream unsafe.Pointer) (bool, error) {
	if kernel.ctx == nil {
		return false, errClosedDuring
	}
	var why [1024]C.char
	var stopped C.int
	failed := C.slayer_metal_grid_wait(stream, &stopped, &why[0],
		C.int(len(why)))
	runtime.KeepAlive(kernel)
	if failed == waitStopped {
		return false, stoppedError(C.GoString(&why[0]))
	}
	if failed != 0 {
		return stopped != 0, errors.New("metal: " + C.GoString(&why[0]))
	}
	if kernel.pretendStops > 0 {
		kernel.pretendStops--
		return true, errPretendStop
	}
	return false, nil
}

// copyBand copies the band last waited out, down rows, into samples.
func (kernel *Kernel) copyBand(stream unsafe.Pointer, down int,
	samples []float32) error {
	var why [1024]C.char
	failed := C.slayer_metal_grid_copy(stream, C.int(down),
		(*C.float)(unsafe.Pointer(&samples[0])), &why[0], C.int(len(why)))
	runtime.KeepAlive(kernel)
	if failed != 0 {
		return errors.New("metal: " + C.GoString(&why[0]))
	}
	return nil
}
