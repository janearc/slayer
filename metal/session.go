//go:build darwin

package metal

/*
#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import (
	"errors"
	"runtime"
	"time"
	"unsafe"
)

// A Session is many kernels compiled once from one source, buffers that
// live on the GPU as long as it does, and dispatches gathered into a
// batch and run as one command: for work that steps many times without
// coming back, as a fluid does.
//
// Its buffers are memory the CPU and the GPU share, so Go reads and writes
// them where they lie.
type Session struct {
	ctx     unsafe.Pointer
	kernels map[string]int
}

// A Buffer is a session's buffer of numbers.
type Buffer struct {
	index   int
	numbers []float32
}

// Numbers is the buffer's numbers, where they lie: what Go writes there
// the GPU reads, and what the GPU wrote there Go reads, between
// batches.
func (buffer Buffer) Numbers() []float32 { return buffer.numbers }

// OpenSession compiles a source and each named kernel in it.
func OpenSession(source string, kernels ...string) (*Session, error) {
	src := C.CString(source)
	defer C.free(unsafe.Pointer(src))
	names := make([]*C.char, len(kernels))
	for k, name := range kernels {
		names[k] = C.CString(name)
		defer C.free(unsafe.Pointer(names[k]))
	}
	listed := (**C.char)(C.malloc(C.size_t(len(names)) *
		C.size_t(unsafe.Sizeof(uintptr(0)))))
	defer C.free(unsafe.Pointer(listed))
	copy(unsafe.Slice(listed, len(names)), names)
	var why [1024]C.char
	var noDevice C.int
	ctx := C.slayer_session_open(src, listed, C.int(len(names)), &noDevice,
		&why[0], C.int(len(why)))
	if noDevice != 0 {
		return nil, ErrNoDevice
	}
	if ctx == nil {
		return nil, errors.New("metal: " + C.GoString(&why[0]))
	}
	made := &Session{ctx: ctx, kernels: map[string]int{}}
	for k, name := range kernels {
		made.kernels[name] = k
	}
	runtime.SetFinalizer(made, (*Session).Close)
	return made, nil
}

// Buffer is a new buffer of so many numbers, all zero.
func (s *Session) Buffer(count int) (Buffer, error) {
	if s.ctx == nil {
		return Buffer{}, errors.New("metal: the session is closed")
	}
	var why [1024]C.char
	index := C.slayer_session_buffer(s.ctx, C.long(count*4), &why[0],
		C.int(len(why)))
	if index < 0 {
		return Buffer{}, errors.New("metal: " + C.GoString(&why[0]))
	}
	at := C.slayer_session_contents(s.ctx, index)
	return Buffer{index: int(index),
		numbers: unsafe.Slice((*float32)(at), count)}, nil
}

// Dispatch adds a kernel to the batch, over width by height threads,
// its buffers bound in order and its params after them.
func (s *Session) Dispatch(kernel string, width, height int,
	params []byte, buffers ...Buffer) error {
	return s.dispatch(kernel, width, height, false, params, buffers)
}

// Grouped adds a kernel to the batch as height groups of width threads,
// each group sharing its threadgroup memory: a row to a group, for work
// a row does together.
func (s *Session) Grouped(kernel string, width, height int,
	params []byte, buffers ...Buffer) error {
	return s.dispatch(kernel, width, height, true, params, buffers)
}

// dispatch adds a kernel to the batch, grouped or not.
func (s *Session) dispatch(kernel string, width, height int, grouped bool,
	params []byte, buffers []Buffer) error {
	which, known := s.kernels[kernel]
	if !known || s.ctx == nil {
		return errors.New("metal: no kernel " + kernel + " in the session")
	}
	bound := make([]C.int, len(buffers)+1)
	for b, buffer := range buffers {
		bound[b] = C.int(buffer.index)
	}
	var pp unsafe.Pointer
	if len(params) > 0 {
		pp = C.CBytes(params)
		defer C.free(pp)
	}
	flag := C.int(0)
	if grouped {
		flag = 1
	}
	var why [1024]C.char
	failed := C.slayer_session_dispatch(s.ctx, C.int(which), &bound[0],
		C.int(len(buffers)), pp, C.int(len(params)), C.int(width),
		C.int(height), flag, &why[0], C.int(len(why)))
	if failed != 0 {
		return errors.New("metal: " + C.GoString(&why[0]))
	}
	return nil
}

// Commit runs the batch, in the order it was given, and waits for it.
func (s *Session) Commit() error {
	if s.ctx == nil {
		return errors.New("metal: the session is closed")
	}
	var why [1024]C.char
	failed := C.slayer_session_commit(s.ctx, &why[0], C.int(len(why)))
	runtime.KeepAlive(s)
	if failed != 0 {
		return errors.New("metal: " + C.GoString(&why[0]))
	}
	return nil
}

// GPUTime is how long the GPU has spent on the session's batches.
func (s *Session) GPUTime() time.Duration {
	if s.ctx == nil {
		return 0
	}
	return time.Duration(float64(C.slayer_session_gpu(s.ctx)) *
		float64(time.Second))
}

// Close lets the session and its buffers go; their numbers are not to be
// read after.
func (s *Session) Close() {
	if s.ctx != nil {
		C.slayer_session_close(s.ctx)
		s.ctx = nil
	}
}
