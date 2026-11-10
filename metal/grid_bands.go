//go:build darwin

package metal

/*
#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import (
	"errors"
	"math"
	"runtime"
	"unsafe"
)

// GridBands runs the same globally addressed grid with one reusable readback
// band. Inputs remain on Metal for the call. A callback must consume or copy
// its slice before returning; later bands overwrite that storage.
//
// rows must be positive. Calls and callbacks must not reenter this Kernel.
// Callback errors stop dispatch, release the buffers and propagate unchanged.
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
	stream, err := kernel.openBands(width, rows, per, inputs)
	if err != nil {
		return err
	}
	defer C.slayer_metal_grid_close(stream)
	defer runtime.KeepAlive(kernel)
	samples := make([]float32, width*rows*per)
	for first := 0; first < height; {
		down := min(rows, height-first)
		band := samples[:width*down*per]
		if err := kernel.readBand(stream, first, down, band); err != nil {
			return err
		}
		if err := consume(first, down, band); err != nil {
			return err
		}
		first += down
	}
	return nil
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
func (kernel *Kernel) openBands(width, rows, per int, inputs [][]byte) (unsafe.Pointer, error) {
	defer runtime.KeepAlive(kernel)
	var why [1024]C.char
	stream := C.slayer_metal_grid_begin(kernel.ctx, C.int(width), C.int(rows), C.int(per),
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

// readBand keeps the kernel live and copies only the current rows into Go.
func (kernel *Kernel) readBand(stream unsafe.Pointer, first, down int, samples []float32) error {
	if kernel.ctx == nil {
		return errors.New("metal: the kernel closed during grid consumption")
	}
	var why [1024]C.char
	failed := C.slayer_metal_grid_band(stream, C.int(first), C.int(down),
		(*C.float)(unsafe.Pointer(&samples[0])), &why[0], C.int(len(why)))
	runtime.KeepAlive(kernel)
	if failed != 0 {
		return errors.New("metal: " + C.GoString(&why[0]))
	}
	return nil
}
