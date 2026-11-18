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
// rows must be positive: it is the most rows a band holds. Calls and
// callbacks must not reenter this Kernel. Callback errors stop dispatch,
// release the buffers and propagate unchanged.
func (kernel *Kernel) GridBands(width, height, rows, per int,
	inputs [][]byte, consume func(first, down int, samples []float32) error,
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
	defer runtime.KeepAlive(kernel)
	samples := make([]float32, width*rows*per)
	place := func(_, down int) []float32 { return samples[:width*down*per] }
	drawing := bandDrawing{kernel: kernel, stream: stream, height: height,
		rows: kernel.bandRows(rows)}
	return drawing.draw(place, consume)
}

// bandRows is the height of a grid's bands whose buffers hold most rows:
// the kernel's fixed height, as far as the buffers hold, or the most.
func (kernel *Kernel) bandRows(most int) int {
	if kernel.fixed > 0 {
		return min(kernel.fixed, most)
	}
	return most
}

// span is a band of a picture: down rows from first. No rows is no band.
type span struct{ first, down int }

// bandDrawing is one picture height rows tall drawn through a stream of
// the kernel in bands rows tall. ahead is the band committed behind the
// one being waited out, so the device draws one band while the caller
// reads the last; it is no band at the picture's foot.
type bandDrawing struct {
	kernel *Kernel
	stream unsafe.Pointer
	height int
	rows   int
	ahead  span
}

// draw draws the picture. Each band is waited out, the band after it
// committed, if it was not already, and the band waited out copied into
// the slice place gives it and handed to consume while the next draws.
func (drawing *bandDrawing) draw(place func(first, down int) []float32,
	consume func(first, down int, samples []float32) error) (err error) {
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
	for band.down > 0 {
		drawn, spent, err := drawing.await(band)
		if err != nil {
			return err
		}
		samples := place(drawn.first, drawn.down)
		if band, err = drawing.next(samples, drawn, spent); err != nil {
			return err
		}
		if err := consume(drawn.first, drawn.down, samples); err != nil {
			return err
		}
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

// after is the band that follows one in the picture, or no band at the
// picture's foot.
func (drawing *bandDrawing) after(band span) span {
	from := band.first + band.down
	if from >= drawing.height {
		return span{}
	}
	return span{from, min(drawing.rows, drawing.height-from)}
}

// commit commits a band, and nothing for no band.
func (drawing *bandDrawing) commit(band span) error {
	if band.down == 0 {
		return nil
	}
	return drawing.kernel.commitBand(drawing.stream, band.first, band.down)
}

// next has the band after drawn committed, if it was not already, copies
// drawn into samples while that one draws, and commits the band after
// that into drawn's buffer. It gives back the band to wait out next.
func (drawing *bandDrawing) next(samples []float32, drawn span,
	spent time.Duration) (span, error) {
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
		band.down = max(1, band.down/2)
		drawing.rows = min(drawing.rows, band.down)
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

// waitBand waits out the oldest band in flight, and says whether a
// failure was the system stopping it for the display.
func (kernel *Kernel) waitBand(stream unsafe.Pointer) (bool, error) {
	if kernel.ctx == nil {
		return false, errClosedDuring
	}
	var why [1024]C.char
	var stopped C.int
	failed := C.slayer_metal_grid_wait(stream, &stopped, &why[0],
		C.int(len(why)))
	runtime.KeepAlive(kernel)
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
