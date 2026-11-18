//go:build !darwin

package spectral

import "errors"

// errMetalUnavailable is a backend refusal, never a silent CPU fallback.
var errMetalUnavailable = errors.New("spectral: Metal is unavailable " +
	"on this platform")

// MetalPlan is unavailable where the Metal backend is not built.
type MetalPlan struct{}

// OpenMetal refuses where the Metal backend is not built.
func OpenMetal(shape Shape) (*MetalPlan, error) {
	return nil, errMetalUnavailable
}

// Forward refuses where the Metal backend is not built.
func (plan *MetalPlan) Forward(values []complex128) ([]complex128, error) {
	return nil, errMetalUnavailable
}

// Inverse refuses where the Metal backend is not built.
func (plan *MetalPlan) Inverse(values []complex128) ([]complex128, error) {
	return nil, errMetalUnavailable
}

// Close has no device state to release.
func (plan *MetalPlan) Close() {}

// Device is empty where the Metal backend is not built.
func (plan *MetalPlan) Device() string { return "" }
