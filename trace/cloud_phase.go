package trace

import (
	"math"

	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
)

type phasedClearCloud interface {
	PhaseAsymmetry() (float64, bool)
}

// A value selection avoids allocating a capturing callback at each cloud cell.
type cloudPhaseSelection struct {
	g        float64
	explicit bool
}

// clearCloudPhaseOf returns the phase the clear cloud sets for field. It
// returns an empty selection when field is not a clear cloud.
func clearCloudPhaseOf(field medium.Medium) cloudPhaseSelection {
	cloud, ok := field.(phasedClearCloud)
	if !ok {
		return cloudPhaseSelection{}
	}
	g, explicit := cloud.PhaseAsymmetry()
	return cloudPhaseSelection{g: g, explicit: explicit}
}

// at returns the phase at cosine. An explicit phase uses Henyey-Greenstein;
// otherwise the inherited gas phase is used.
func (phase cloudPhaseSelection) at(cosine, inherited float64) float64 {
	if phase.explicit {
		return clearCloudHenyeyGreenstein(phase.g, cosine)
	}
	return (gas.Gas{Phase: inherited}).PhaseAt(cosine)
}

// The cosine uses the existing toward-source dot camera-ray convention.
// This stable HG form includes isotropic g=0 explicitly and does not divide
// a source that was already weighted by the water's unrelated phase law.
func clearCloudHenyeyGreenstein(g, cosine float64) float64 {
	abs := math.Abs(g)
	near := 1 - abs
	aligned := cosine
	if g < 0 {
		aligned = -cosine
	}
	d := near*near + 2*abs*(1-aligned)
	return ((1 - g) * (1 + g)) / (4 * math.Pi * d * math.Sqrt(d))
}
