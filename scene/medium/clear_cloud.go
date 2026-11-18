package medium

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
)

// ClearCloud is an explicitly opted-in bounded local participating medium
// that may be composed with named clear-medium extinction by the CPU tracer.
// It wraps an existing density field; its scattering albedo remains the
// volume material's Colour. This is a single-scattering cloud approximation,
// not resolved bubbles or fluid simulation.
type ClearCloud struct {
	of            Medium
	phase         float64
	explicitPhase bool
}

// PhaseAsymmetry reports an explicit Henyey-Greenstein law. False retains
// the legacy inherited water law; explicit zero means isotropic HG.
func (cloud ClearCloud) PhaseAsymmetry() (float64, bool) {
	return cloud.phase, cloud.explicitPhase
}

// NewClearCloudWithHenyeyGreenstein selects an authored bulk cloud phase,
// independent of the surrounding water. It is not a bubble-size/Mie model.
// Positive g scatters forward; zero is isotropic, not legacy water Rayleigh.
func NewClearCloudWithHenyeyGreenstein(of Medium,
	g float64,
) (ClearCloud, error) {
	if math.IsNaN(g) || math.IsInf(g, 0) || !(g > -1 && g < 1) {
		return ClearCloud{}, fmt.Errorf("medium: cloud HG asymmetry must be " +
			"finite and strictly between -1 and 1")
	}
	cloud, err := NewClearCloud(of)
	if err != nil {
		return ClearCloud{}, err
	}
	cloud.phase, cloud.explicitPhase = g, true
	return cloud, nil
}

// DensityField is the immutable wrapped field. Native backends use its exact
// analytic recipe after Tick rather than sampling the wrapper into a grid.
func (cloud ClearCloud) DensityField() Medium { return cloud.of }

// NewClearCloud admits only a finite density field no wider than 500 scene
// units on any axis. The finite bound keeps the mixed transport work local.
func NewClearCloud(of Medium) (ClearCloud, error) {
	if of == nil {
		return ClearCloud{}, fmt.Errorf("medium: clear cloud needs a density field")
	}
	bounds := of.Bounds()
	for _, value := range []float64{bounds.Min.X, bounds.Min.Y, bounds.Min.Z,
		bounds.Max.X, bounds.Max.Y, bounds.Max.Z} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return ClearCloud{}, fmt.Errorf("medium: clear cloud bounds must be finite")
		}
	}
	for _, width := range []float64{bounds.Max.X - bounds.Min.X,
		bounds.Max.Y - bounds.Min.Y, bounds.Max.Z - bounds.Min.Z} {
		if width < 0 || width > maxClearCloudWidth {
			return ClearCloud{}, fmt.Errorf("medium: clear cloud bounds must be ordered and at most %g units wide", maxClearCloudWidth)
		}
	}
	return ClearCloud{of: of}, nil
}

const maxClearCloudWidth = 500.0

// Bounds is the wrapped field's finite integration domain.
func (cloud ClearCloud) Bounds() maths.Box { return cloud.of.Bounds() }

// Density is the wrapped field's local coefficient.
func (cloud ClearCloud) Density(point maths.Vec) float64 {
	return cloud.of.Density(point)
}

// Span keeps exact entry intervals when the wrapped field has them.
func (cloud ClearCloud) Span(ray maths.Ray, near, far float64) (float64, float64, bool) {
	if spanner, ok := cloud.of.(Spanner); ok {
		return spanner.Span(ray, near, far)
	}
	return cloud.Bounds().Hit(ray, near, far)
}

// ClearMediumOverlay marks the field for the CPU's explicit mixed transport.
func (ClearCloud) ClearMediumOverlay() {}

// Tick preserves animation of a wrapped medium.
func (cloud ClearCloud) Tick(at time.Duration) Medium {
	if timed, ok := cloud.of.(Timed); ok {
		cloud.of = timed.Tick(at)
	}
	return cloud
}
