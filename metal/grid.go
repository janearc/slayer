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

// what Grid refuses
var (
	errNoNumbers = errors.New("metal: a pixel needs a number at least")
	errTooBig    = errors.New("metal: an input over two gigabytes")
)

// Grid runs the kernel once for every pixel of a width by height picture and
// gives back per numbers a pixel, row by row. The inputs are the kernel's
// buffers from 0, as given; the picture is its next buffer, and the band's
// first row the one after that.
//
// It draws the picture in bands of at most rows rows, one command each, so no
// single command runs long enough for the system to stop it; the inputs are
// made into buffers once for all the bands. Rows of zero or less is the whole
// picture at once.
func (k *Kernel) Grid(width, height, rows, per int,
	inputs [][]byte) ([]float32, error) {
	if k.ctx == nil {
		return nil, errors.New("metal: the kernel is closed")
	}
	if per <= 0 {
		return nil, errNoNumbers
	}
	for _, input := range inputs {
		if len(input) > math.MaxInt32 {
			return nil, errTooBig
		}
	}
	out := make([]float32, width*height*per)
	if width <= 0 || height <= 0 {
		return out, nil
	}
	if rows <= 0 {
		rows = height
	}
	pointers, lengths, free := pinned(inputs)
	defer free()
	var why [1024]C.char
	failed := C.slayer_metal_grid(k.ctx, C.int(width), C.int(height),
		C.int(rows), C.int(per), pointers, lengths, C.int(len(inputs)),
		(*C.float)(unsafe.Pointer(&out[0])), &why[0], C.int(len(why)))
	// k is not read after k.ctx, so without this its finalizer could
	// free the context while the gpu still draws with it
	runtime.KeepAlive(k)
	if failed != 0 {
		return nil, errors.New("metal: " + C.GoString(&why[0]))
	}
	return out, nil
}

// pinned copies the inputs into C memory, which cgo lets C hold a
// pointer to, and gives back the pointers, their lengths, and how to
// let them go.
func pinned(inputs [][]byte) (*unsafe.Pointer, *C.long, func()) {
	if len(inputs) == 0 {
		return nil, nil, func() {}
	}
	count := C.size_t(len(inputs))
	pointerSize := C.size_t(unsafe.Sizeof(uintptr(0)))
	pointers := unsafe.Slice((*unsafe.Pointer)(C.malloc(count*pointerSize)),
		len(inputs))
	longSize := C.size_t(unsafe.Sizeof(C.long(0)))
	lengths := unsafe.Slice((*C.long)(C.malloc(count*longSize)),
		len(inputs))
	for index, input := range inputs {
		lengths[index] = C.long(len(input))
		pointers[index] = nil
		if len(input) > 0 {
			pointers[index] = C.CBytes(input)
		}
	}
	free := func() {
		for _, pointer := range pointers {
			C.free(pointer)
		}
		C.free(unsafe.Pointer(&pointers[0]))
		C.free(unsafe.Pointer(&lengths[0]))
	}
	return &pointers[0], &lengths[0], free
}
