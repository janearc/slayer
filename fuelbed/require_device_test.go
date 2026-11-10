//go:build darwin

package fuelbed_test

import (
	"errors"
	"github.com/janearc/slayer/metal"
	"testing"
)

// requireFuelDevice skips only absence of Metal, never a failed stencil,
// allocation or command on the available test device.
func requireFuelDevice(t *testing.T, err error) {
	t.Helper()
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
}
