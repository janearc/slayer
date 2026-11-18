//go:build !darwin

package metalwave

import (
	"fmt"
	"github.com/janearc/slayer/simulation/internal/metalfft"
)

// Plan is unavailable without the native Metal bridge.
type Plan struct{}

// Open reports unsupported hardware without substituting CPU arithmetic.
func Open(shape metalfft.Shape, initial Fields) (*Plan, error) {
	return nil, fmt.Errorf("metalwave: Metal requires Darwin")
}

// Evaluate reports the unavailable backend.
func (plan *Plan) Evaluate(modes []Mode) (Fields, error) { return Fields{}, plan.Usable() }

// Usable reports the unavailable backend.
func (plan *Plan) Usable() error { return fmt.Errorf("metalwave: Metal requires Darwin") }

// Close is harmless when no native resources exist.
func (plan *Plan) Close() {}
