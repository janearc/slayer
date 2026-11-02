package medium

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Plume is a volcanic plume: a column of gas and dust rising from the
// origin along y to Height, and an umbrella of the same arcing over
// from the top and falling back to the ground Width across, as io's
// plumes do.
//
// Thick is the density at the column's heart; the umbrella is a thin
// shell, half as dense at the top and thinning as it falls.
type Plume struct {
	Height, Width float64
	Thick         float64
}

// Bounds is the box the plume stands in.
func (p Plume) Bounds() maths.Box {
	w := p.Width / 2
	return maths.Box{Min: maths.V(-w, 0, -w), Max: maths.V(w, p.Height, w)}
}

// Density is the column plus the umbrella's shell.
func (p Plume) Density(at maths.Vec) float64 {
	if p.Height <= 0 || p.Width <= 0 {
		return 0
	}
	h := at.Y / p.Height
	if h <= 0 || h >= 1 {
		return 0
	}
	w := p.Width / 2
	rho := math.Hypot(at.X, at.Z)
	// the column, a tenth of the width, thinning as it rises
	core := 0.1 * w
	column := math.Exp(-rho*rho/(core*core)) * (1 - 0.5*h)
	// the umbrella, a dome: a shell closing from the width at the
	// ground to the column's top, a skin thick, thinner as it falls
	shell := w * math.Sqrt(1-h*h)
	skin := 0.15 * w
	d := (rho - shell) / skin
	umbrella := 0.5 * (0.4 + 0.6*h) * math.Exp(-d*d)
	return p.Thick * (column + umbrella)
}
