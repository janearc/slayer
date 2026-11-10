//go:build !darwin

package fuelbed

import "fmt"

// Device is unavailable on platforms without Metal.
type Device struct{}

// NewMetal refuses a GPU path on platforms without Metal.
func NewMetal(Spec) (*Device, error) { return nil, fmt.Errorf("fuelbed: Metal requires Darwin") }

// NewMetalAdvected refuses an accelerated path on platforms without Metal.
func NewMetalAdvected(Spec, Transport) (*Device, error) {
	return nil, fmt.Errorf("fuelbed: Metal requires Darwin")
}

// At refuses a GPU path on platforms without Metal.
func (*Device) At(float64) ([]Cell, error) { return nil, fmt.Errorf("fuelbed: Metal requires Darwin") }

// Close has no resources to release on this platform.
func (*Device) Close() {}
