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
#cgo LDFLAGS: -framework Metal -framework Foundation
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
