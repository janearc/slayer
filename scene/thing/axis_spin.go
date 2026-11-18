package thing

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"math"
	"time"
)

// MaximumAxisSpinRPM is the authored motion safety bound, not a physical limit.
const MaximumAxisSpinRPM = 1e6

// axisSpin rotates native geometry about a local pivot inside its parent.
type axisSpin struct {
	parent      Motion
	pivot, axis maths.Vec
	rpm, phase  float64
}

// NewAxisSpin composes an immutable local-axis rotation with a parent motion.
// Pivot uses the geometry's native coordinates; RPM is signed revolutions/minute.
func NewAxisSpin(parent Motion, pivot, axis maths.Vec, rpm, phase float64) (Motion, error) {
	if parent == nil || !spinFinite(pivot.X, pivot.Y, pivot.Z, axis.X, axis.Y, axis.Z, rpm, phase) || math.Abs(rpm) > MaximumAxisSpinRPM {
		return nil, fmt.Errorf("thing: axis spin needs a parent, finite pivot/axis/phase and RPM in [-%g,%g]", MaximumAxisSpinRPM, MaximumAxisSpinRPM)
	}
	largest := math.Max(math.Abs(axis.X), math.Max(math.Abs(axis.Y), math.Abs(axis.Z)))
	if largest == 0 {
		return nil, fmt.Errorf("thing: axis spin needs a nonzero axis")
	}
	scaled := maths.V(axis.X/largest, axis.Y/largest, axis.Z/largest)
	return axisSpin{parent, pivot, scaled.Unit(), rpm, math.Remainder(phase, 2*math.Pi)}, nil
}

// spinFinite reports whether every value is a real number.
func spinFinite(values ...float64) bool {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}

// At evaluates the same frame clock as the parent and preserves exact rest.
func (spin axisSpin) At(at time.Duration) maths.Subspace {
	parent := spin.parent.At(at)
	angle := math.Remainder(spin.phase+2*math.Pi*(spin.rpm/60)*at.Seconds(), 2*math.Pi)
	if angle == 0 {
		return parent
	}
	turn := maths.Rotation(spin.axis, angle)
	local := maths.NewSubspace(spin.pivot.Sub(turn.Out(spin.pivot)), turn, 1)
	posed := local.Within(parent)
	posed.Scale = parent.Scale
	return posed
}
