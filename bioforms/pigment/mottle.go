package pigment

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Mottle varies pigment in rest space without a wrapped coordinate seam.
// Scale is the coarsest lattice spacing in chart units. Strength bounds
// the signed shift. The zero value gives no variation.
type Mottle struct {
	Seed            uint64
	Scale, Strength float64
}

// Valid reports whether the strength and scale can describe a mottle.
func (m Mottle) Valid() bool {
	return share(m.Strength) && (positive(m.Scale) || m.Scale == 0 && m.Strength == 0)
}

// At returns a shift between -Strength and Strength. Multiply it by an
// anatomy mask to limit its coverage. Each finer octave fades towards zero
// as the footprint approaches its wavelength, leaving the mean pigment.
func (m Mottle) At(rest maths.Vec, footprint float64) float64 {
	if m.Strength == 0 || !m.Valid() {
		return 0
	}
	point := maths.V(rest.X/m.Scale, rest.Y/m.Scale, rest.Z/m.Scale)
	width := max(0, footprint) / m.Scale
	var value float64
	for octave, strength := range [...]float64{.58, .28, .14} {
		// Above this lattice index the sample loses subcell precision.
		const gridLimit = 1 << 50
		if !(width < 1) || !finite(point) ||
			max(math.Abs(point.X), math.Abs(point.Y), math.Abs(point.Z)) > gridLimit {
			break
		}
		visible := 1 - maths.Smooth(.32, 1, width)
		seed := m.Seed + uint64(octave)*noise.Golden
		shift := noise.Gradient(seed, point.X, point.Y, point.Z)
		value += strength * visible * max(-1, min(1, shift))
		point = point.Scale(2.17)
		width *= 2.17
	}
	return m.Strength * value
}
