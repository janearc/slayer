package main

/*
#include "viewer.h"
*/
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/janearc/slayer/internal/appicon"
)

// installIcons gives Cocoa owned image data before the viewer starts.
func installIcons() error {
	light, err := appicon.PNG(false)
	if err != nil {
		return err
	}
	dark, err := appicon.PNG(true)
	if err != nil {
		return err
	}
	return nativeIcons(light, dark)
}

// nativeIcons copies the bytes into native images before the Go call returns.
func nativeIcons(light, dark []byte) error {
	why := C.viewer_icons(
		(*C.uint8_t)(unsafe.Pointer(unsafe.SliceData(light))),
		C.size_t(len(light)),
		(*C.uint8_t)(unsafe.Pointer(unsafe.SliceData(dark))),
		C.size_t(len(dark)))
	if why != nil {
		return fmt.Errorf("viewer icon: %s", C.GoString(why))
	}
	return nil
}
