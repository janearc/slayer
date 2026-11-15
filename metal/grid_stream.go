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

// GridStream is a grid whose inputs stay on the device between
// pictures: open it once, draw, change only the inputs that changed,
// and draw again. An animation's still scene uploads once; each frame
// uploads its moment.
//
// Calls on a stream are sequential, as on its Kernel, and the stream
// belongs to the Kernel that opened it: close the stream first.
//
// Same says whether an input already holds given bytes, read in place
// on the device's side, so a caller keeps no copy of what it sent, and
// Refresh copies only the parts of an input that changed.
//
// Reshape draws later pictures at another size, or of another entry
// point's numbers, over the same inputs.
type GridStream struct {
	kernel                   *Kernel
	stream                   unsafe.Pointer
	width, height, rows, per int
	// lengths is each input's bytes as last given, which an empty
	// input's buffer, as many zeros as the kernel declares, cannot say
	lengths []int
}

// OpenGrid retains the inputs on the device for pictures of a fixed
// size, drawn in bands of at most rows rows as Grid draws them.
func (k *Kernel) OpenGrid(width, height, rows, per int,
	inputs [][]byte) (*GridStream, error) {
	if k.ctx == nil {
		return nil, errors.New("metal: the kernel is closed")
	}
	if per <= 0 {
		return nil, errNoNumbers
	}
	if width <= 0 || height <= 0 || rows <= 0 {
		return nil, errors.New("metal: a stream needs a size and rows")
	}
	for _, input := range inputs {
		if len(input) > math.MaxInt32 {
			return nil, errTooBig
		}
	}
	if !gridFits(width, height, per) {
		return nil, errGridSize
	}
	rows = min(rows, height)
	stream, err := k.openBands(width, height, rows, per, inputs)
	if err != nil {
		return nil, err
	}
	lengths := make([]int, len(inputs))
	for index, input := range inputs {
		lengths[index] = len(input)
	}
	return &GridStream{kernel: k, stream: stream, width: width,
		height: height, rows: rows, per: per, lengths: lengths}, nil
}

// Update replaces one input's bytes. Nothing else is touched, so a
// frame pays for what it changed and no more.
func (s *GridStream) Update(index int, input []byte) error {
	if s.stream == nil || s.kernel.ctx == nil {
		return errors.New("metal: the stream is closed")
	}
	if index < 0 || index >= len(s.lengths) {
		return errors.New("metal: no such input to update")
	}
	if len(input) > math.MaxInt32 {
		return errTooBig
	}
	defer runtime.KeepAlive(s.kernel)
	var pointer unsafe.Pointer
	if len(input) > 0 {
		pointer = unsafe.Pointer(&input[0])
	}
	var why [1024]C.char
	failed := C.slayer_metal_grid_update(s.stream, C.int(index), pointer,
		C.long(len(input)), &why[0], C.int(len(why)))
	runtime.KeepAlive(input)
	if failed != 0 {
		return errors.New("metal: " + C.GoString(&why[0]))
	}
	s.lengths[index] = len(input)
	return nil
}

// Same is whether input index holds exactly these bytes. It compares
// them where the device reads them and copies nothing; an empty input
// is the same as an empty one and as nothing else.
func (s *GridStream) Same(index int, input []byte) (bool, error) {
	if s.stream == nil || s.kernel.ctx == nil {
		return false, errors.New("metal: the stream is closed")
	}
	if index < 0 || index >= len(s.lengths) {
		return false, errors.New("metal: no such input to compare")
	}
	if len(input) != s.lengths[index] {
		return false, nil
	}
	if len(input) == 0 {
		return true, nil
	}
	defer runtime.KeepAlive(s.kernel)
	var why [1024]C.char
	var same C.int
	failed := C.slayer_metal_grid_same(s.stream, C.int(index),
		unsafe.Pointer(&input[0]), C.long(len(input)), &same, &why[0],
		C.int(len(why)))
	runtime.KeepAlive(input)
	if failed != 0 {
		return false, errors.New("metal: " + C.GoString(&why[0]))
	}
	return same != 0, nil
}

// refreshChunk is the parts Refresh compares and copies an input in,
// picked 2026-11-12: small enough that a bake that rewrites part of a
// large input sends about that part, large enough that comparing costs
// no more than reading the bytes.
const refreshChunk = 1 << 16

// Refresh makes input index hold these bytes, copying only the parts
// that differ from what it holds, and says how many bytes it copied. An
// input of another length is sent whole, as Update sends it.
func (s *GridStream) Refresh(index int, input []byte) (int64, error) {
	if s.stream == nil || s.kernel.ctx == nil {
		return 0, errors.New("metal: the stream is closed")
	}
	if index < 0 || index >= len(s.lengths) {
		return 0, errors.New("metal: no such input to refresh")
	}
	if len(input) != s.lengths[index] || len(input) == 0 {
		if same, err := s.Same(index, input); err != nil || same {
			return 0, err
		}
		return int64(len(input)), s.Update(index, input)
	}
	defer runtime.KeepAlive(s.kernel)
	var why [1024]C.char
	var sent C.long
	failed := C.slayer_metal_grid_refresh(s.stream, C.int(index),
		unsafe.Pointer(&input[0]), C.long(len(input)), refreshChunk, &sent,
		&why[0], C.int(len(why)))
	runtime.KeepAlive(input)
	if failed != 0 {
		return 0, errors.New("metal: " + C.GoString(&why[0]))
	}
	return int64(sent), nil
}

// Reshape draws later pictures width by height, per numbers a pixel,
// in bands of at most rows rows, over the inputs already held. Anything
// drawn ahead is waited out first. A shape the stream cannot take is
// refused and the old one kept.
func (s *GridStream) Reshape(width, height, rows, per int) error {
	if s.stream == nil || s.kernel.ctx == nil {
		return errors.New("metal: the stream is closed")
	}
	if per <= 0 {
		return errNoNumbers
	}
	if width <= 0 || height <= 0 || rows <= 0 {
		return errors.New("metal: a stream needs a size and rows")
	}
	if !gridFits(width, height, per) {
		return errGridSize
	}
	rows = min(rows, height)
	defer runtime.KeepAlive(s.kernel)
	var why [1024]C.char
	failed := C.slayer_metal_grid_shape(s.stream, C.int(width),
		C.int(height), C.int(rows), C.int(per), &why[0], C.int(len(why)))
	if failed != 0 {
		return errors.New("metal: " + C.GoString(&why[0]))
	}
	s.width, s.height, s.rows, s.per = width, height, rows, per
	return nil
}

// Draw draws the whole picture, band by band, into one slice, as Grid
// gives it back.
func (s *GridStream) Draw() ([]float32, error) {
	out := make([]float32, s.width*s.height*s.per)
	err := s.DrawBands(func(first, down int, samples []float32) error {
		copy(out[first*s.width*s.per:], samples)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DrawBands draws the picture through one reusable band, as GridBands
// does: a callback must consume or copy its slice before returning.
func (s *GridStream) DrawBands(
	consume func(first, down int, samples []float32) error) error {
	if s.stream == nil || s.kernel.ctx == nil {
		return errors.New("metal: the stream is closed")
	}
	if consume == nil {
		return errors.New("metal: a drawing needs a consumer")
	}
	defer runtime.KeepAlive(s.kernel)
	samples := make([]float32, s.width*s.rows*s.per)
	for first := 0; first < s.height; {
		down := min(s.rows, s.height-first)
		band := samples[:s.width*down*s.per]
		if err := s.kernel.readBand(s.stream, first, down, band); err != nil {
			return err
		}
		if err := consume(first, down, band); err != nil {
			return err
		}
		first += down
	}
	return nil
}

// Close lets the stream's buffers go. Closing twice is harmless.
func (s *GridStream) Close() {
	if s.stream != nil {
		C.slayer_metal_grid_close(s.stream)
		s.stream = nil
	}
	runtime.KeepAlive(s.kernel)
}
