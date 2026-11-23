package thing

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
)

// MaximumAxisSpinRPM is the authored motion safety bound, not a physical
// limit.
const MaximumAxisSpinRPM = 1e6

// The numbers an axis spin is worked in, each true by definition.
const (
	spinNought = 0           // defined: no amount
	spinUnity  = 1           // defined: one whole
	spinMinute = 60          // defined: seconds in a minute
	spinTurn   = 2 * math.Pi // defined: radians in one full turn
)

// axisSpin rotates native geometry about a local pivot inside its parent.
type axisSpin struct {
	parent      Motion
	pivot, axis maths.Vec
	rpm, phase  float64
}

// NewAxisSpin composes an immutable local-axis rotation with a parent
// motion. Pivot uses the geometry's native coordinates; RPM is signed
// revolutions/minute.
func NewAxisSpin(parent Motion, pivot, axis maths.Vec,
	rpm, phase float64) (Motion, error) {
	finite := spinFinite(pivot.X, pivot.Y, pivot.Z,
		axis.X, axis.Y, axis.Z, rpm, phase)
	if parent == nil || !finite || math.Abs(rpm) > MaximumAxisSpinRPM {
		return nil, fmt.Errorf("thing: axis spin needs a parent, finite "+
			"pivot/axis/phase and RPM in [-%g,%g]",
			MaximumAxisSpinRPM, MaximumAxisSpinRPM)
	}
	rest := math.Max(math.Abs(axis.Y), math.Abs(axis.Z))
	largest := math.Max(math.Abs(axis.X), rest)
	if largest == spinNought {
		return nil, fmt.Errorf("thing: axis spin needs a nonzero axis")
	}
	scaled := maths.V(axis.X/largest, axis.Y/largest, axis.Z/largest)
	wrapped := math.Remainder(phase, spinTurn)
	return axisSpin{parent, pivot, scaled.Unit(), rpm, wrapped}, nil
}

// spinFinite reports whether every value is a real number.
func spinFinite(values ...float64) bool {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, spinNought) {
			return false
		}
	}
	return true
}

// At evaluates the same frame clock as the parent and preserves exact rest.
func (spin axisSpin) At(at time.Duration) maths.Subspace {
	parent := spin.parent.At(at)
	turns := spinTurn * (spin.rpm / spinMinute) * at.Seconds()
	angle := math.Remainder(spin.phase+turns, spinTurn)
	if angle == spinNought {
		return parent
	}
	turn := maths.Rotation(spin.axis, angle)
	offset := spin.pivot.Sub(turn.Out(spin.pivot))
	local := maths.NewSubspace(offset, turn, spinUnity)
	posed := local.Within(parent)
	posed.Scale = parent.Scale
	return posed
}
