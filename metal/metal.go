//go:build darwin

// Package metal bridges Go and Apple's Metal API through cgo. Kernels compile
// from source when opened, so building Slayer needs no separate Metal compiler
// toolchain.
//
// CPU implementations provide the reference for kernel tests. Each test states
// and measures the tolerance allowed for its GPU result.
package metal

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Metal -framework Foundation -framework IOKit
#cgo LDFLAGS: -framework CoreGraphics
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

// ErrNoDevice is a machine with no GPU Metal can use. A test that needs
// the GPU skips on this, and only this: a kernel that fails to compile
// must fail its test, not skip it.
var ErrNoDevice = errors.New("metal: no metal device")

// Kernel is one compute kernel on the system's GPU.
type Kernel struct {
	ctx    unsafe.Pointer
	device string
	// fixed, when not nought, is the one height every grid band is drawn
	// at, in place of the caller's
	fixed int
	// pretendStops is how many band waits from now report the system
	// stopping their band for the display, after waiting it out; for the
	// tests of what follows a stop, which the system cannot be asked for
	pretendStops int
}

// FixBands has every grid band drawn rows tall, as far as the band
// buffers hold, instead of as tall as the caller's rows; nought restores
// them. A picture does not change with its bands; this is how that is
// shown, and how a slow band is drawn again at a known size.
func (k *Kernel) FixBands(rows int) {
	k.fixed = max(0, rows)
	if k.fixed == 0 {
		k.fixed = max(0, environmentBands)
	}
}

// Open compiles a kernel from Metal source and readies it to run.
func Open(source, kernel string) (*Kernel, error) {
	src, fn := C.CString(source), C.CString(kernel)
	defer C.free(unsafe.Pointer(src))
	defer C.free(unsafe.Pointer(fn))
	var why [1024]C.char
	var noDevice C.int
	ctx := C.slayer_metal_open(src, fn, &noDevice, &why[0], C.int(len(why)))
	if noDevice != 0 {
		return nil, ErrNoDevice
	}
	if ctx == nil {
		return nil, errors.New("metal: " + C.GoString(&why[0]))
	}
	k := &Kernel{ctx: ctx, device: C.GoString(C.slayer_metal_name(ctx))}
	k.SetLimit(defaultLimit)
	k.FixBands(0)
	runtime.SetFinalizer(k, (*Kernel).Close)
	return k, nil
}

// Device is the GPU's own name.
func (k *Kernel) Device() string { return k.device }

// Points runs the kernel once for each point, with params as its third
// buffer, and gives back one number a point.
func (k *Kernel) Points(points [][3]float32, params []byte) ([]float32, error) {
	if k.ctx == nil {
		return nil, errors.New("metal: the kernel is closed")
	}
	out := make([]float32, len(points))
	if len(points) == 0 {
		return out, nil
	}
	var pp unsafe.Pointer
	if len(params) > 0 {
		pp = unsafe.Pointer(&params[0])
	}
	var buf [1024]C.char
	rc := C.slayer_metal_points(k.ctx,
		(*C.float)(unsafe.Pointer(&points[0][0])), C.int(len(points)),
		pp, C.int(len(params)),
		(*C.float)(unsafe.Pointer(&out[0])), &buf[0], C.int(len(buf)))
	runtime.KeepAlive(points)
	runtime.KeepAlive(params)
	runtime.KeepAlive(k)
	if rc != 0 {
		return nil, errors.New("metal: " + C.GoString(&buf[0]))
	}
	return out, nil
}

// GPUTime is how long the gpu has spent on everything the kernel has run,
// by the system's own timestamps on its commands: the time the work took
// on the gpu, which is less than the time a caller waited for it.
func (k *Kernel) GPUTime() time.Duration {
	if k.ctx == nil {
		return 0
	}
	seconds := float64(C.slayer_metal_gpu(k.ctx))
	runtime.KeepAlive(k)
	return time.Duration(seconds * float64(time.Second))
}

// Close lets the kernel go. Closing twice is harmless.
func (k *Kernel) Close() {
	if k.ctx != nil {
		C.slayer_metal_close(k.ctx)
		k.ctx = nil
	}
}

// defaultLimit is how long the wait for one command lasts before it gives
// up, unless SetLimit says otherwise. It is measured and picked: the longest
// single command on Melancholia at 4096 by 3002 samples ran 1.2 s on the
// device (the `timing` lines of a build that printed each command's GPU time,
// from `hillside -set sets/melancholia.yaml -gpu -w 2048 -n 2`), and this is
// fifty times that, since the device is often shared with other work.
const defaultLimit = 60 * time.Second

// SetLimit sets how long the wait for any one command of the kernel lasts
// before it gives up with an error saying what it waited for. The command
// is not cancelled: it may keep the device busy, and the kernel refuses
// every later command, so a caller that sees the error closes the kernel
// and does not reopen one in the same process on the same work.
func (k *Kernel) SetLimit(limit time.Duration) {
	if k.ctx == nil {
		return
	}
	C.slayer_metal_limit(k.ctx, C.double(limit.Seconds()))
	runtime.KeepAlive(k)
}
