//go:build !darwin

package metalfield

import (
	"fmt"
	"github.com/janearc/slayer/simulation/internal/metalfft"
)

// Plan is unavailable on platforms without the native Metal bridge.
type Plan struct{}

// Open reports the unsupported platform without substituting another backend.
func Open(shape metalfft.Shape, spectrum []complex128) (*Plan, error) {
	return nil, fmt.Errorf("metalfield: Metal requires Darwin")
}

// Evaluate reports the unsupported platform.
func (plan *Plan) Evaluate(factors []complex128) ([]complex128, error) { return nil, plan.Usable() }

// Usable reports the unsupported platform.
func (plan *Plan) Usable() error { return fmt.Errorf("metalfield: Metal requires Darwin") }

// Close is harmless when no native resources exist.
func (plan *Plan) Close() {}
