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
	"unsafe"
)

// Function selects another entry point, preserving resources and GPU time.
// Calls on a Kernel are sequential; selection waits on no pending dispatch.
func (k *Kernel) Function(name string) error {
	if k.ctx == nil {
		return errors.New("metal: the kernel is closed")
	}
	function := C.CString(name)
	defer C.free(unsafe.Pointer(function))
	var why [1024]C.char
	failed := C.slayer_metal_function(k.ctx, function, &why[0], C.int(len(why)))
	runtime.KeepAlive(k)
	if failed != 0 {
		return errors.New("metal: " + C.GoString(&why[0]))
	}
	return nil
}
