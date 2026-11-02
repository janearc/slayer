// Package gas supplies band-dependent absorption, scattering, and phase
// functions. Extinction combines absorption and scattering losses. Emission is
// not implemented: Gives returns no spectrum, and stored line data does not yet
// produce emitted light.
//
// The same coefficients serve view rays, lamp transmission, and cutoffs.
// Comments distinguish sourced values from unverified or artistically tuned
// ones. Existing tuned values remain bit-for-bit stable pending migration.
package gas

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Gas is a gas's optics, per metre at its own density: a table by
// libtheme's ranges, and for the visible a rule by wavelength, since
// the sky's blue is a slope and not a step.
type Gas struct {
	Name    string
	Sources []string
	// Ranges are the numbers by range: libtheme's radiation profiles
	// and thing.Visible for the eye's. A range not in it is opaque. A
	// band takes the narrowest range the table declares, as a
	// thing.Spectrum does.
	Ranges map[radiation.Range]Optics
	// Visible, when set, is the scattering and absorption by
	// wavelength across the visible and near it, per metre, in place of
	// the table's one number for the range.
	Visible func(nm float64) (absorb, scatter float64)
	// Phase is which way the gas scatters, as the asymmetry of a
	// henyey-greenstein lobe: zero for rayleigh's even lobe, toward one for
	// haze and cloud that throw light forward.
	Phase float64
	// Lines are the gas's emission lines, by wavelength in nanometres,
	// each with a strength per metre at unit excitation.
	Lines map[float64]float64
}

// Optics are a range's numbers: per metre, absorption and scattering.
type Optics struct {
	Absorb  float64
	Scatter float64
}

// Absorb is the absorption per metre in a band.
func (g Gas) Absorb(band maths.Band) float64 {
	a, _ := g.optics(band)
	return a
}

// Scatter is the scattering per metre in a band.
func (g Gas) Scatter(band maths.Band) float64 {
	_, s := g.optics(band)
	return s
}

// Extinction is absorption and scattering together, per metre.
func (g Gas) Extinction(band maths.Band) float64 {
	a, s := g.optics(band)
	return a + s
}

// Transmit is the share of a band's light that comes through d metres of the
// gas, by beer's law. through zero metres all of it comes, a band the gas is
// opaque to among them.
func (g Gas) Transmit(band maths.Band, d float64) float64 {
	if d == 0 {
		return 1
	}
	return math.Exp(-g.Extinction(band) * d)
}

// optics is the numbers for a band: the visible rule where there is
// one and the band is in its reach, else the range's table.
//
// The zero band is every band at once, as maths.Band says: the ray a
// sensor without glass sends. A gas has no one number for all of it, so
// it answers with what an eye would see, the visible: the rule at 550
// nm where there is one, as Spectrum does, else the table's visible.
func (g Gas) optics(band maths.Band) (absorb, scatter float64) {
	nm := band.Nanometres
	if nm == 0 {
		nm = 550
	}
	r := thing.Range(maths.Band{Nanometres: nm})
	near := (r == radiation.Ultraviolet() || r == radiation.Infrared()) &&
		nm > 300 && nm < 1200
	if g.Visible != nil && (r == thing.Visible || near) {
		return g.Visible(nm)
	}
	o, ok := thing.Lookup(g.Ranges, maths.Band{Nanometres: nm})
	if !ok {
		return math.Inf(1), 0
	}
	return o.Absorb, o.Scatter
}

// PhaseAt is how much of the scattered light goes at an angle whose cosine is
// cos, per steradian, by henyey and greenstein's lobe at the gas's asymmetry:
// rayleigh's even lobe at zero, which integrates to one over the sphere as
// every phase function must.
//
// so an even lobe of its own, scattering alike every way, cannot be said. an
// asymmetry of one or more either way, or of no number, is no lobe, and is
// refused.
func (g Gas) PhaseAt(cos float64) float64 {
	gg := g.Phase
	if gg == 0 {
		// rayleigh: three over sixteen pi times one plus cos squared
		return 3 / (16 * math.Pi) * (1 + cos*cos)
	}
	if !(math.Abs(gg) < 1) {
		panic(fmt.Sprintf("gas: %s's asymmetry of %v is no lobe: "+
			"it is between minus one and one", g.Name, gg))
	}
	d := 1 + gg*gg - 2*gg*cos
	return (1 - gg*gg) / (4 * math.Pi * math.Pow(d, 1.5))
}

// Inscatter is the light a step of a ray gathers from a lamp at a
// point: the lamp's light there, times the scattering per metre, times
// the phase at the angle between the lamp's way and the eye's, per
// metre of step.
//
// What reaches the point and what survives to the eye are the
// tracer's, through Transmit.
func (g Gas) Inscatter(band maths.Band, light maths.Light,
	cos float64) maths.Light {
	return light.Scale(g.Scatter(band) * g.PhaseAt(cos))
}

// Spectrum is the gas as a thing.Spectrum for a volume whose unit is metres
// times unit, at unit density: how much of each range passes one unit, by
// beer's law, and what each range gives off, zero here since lines are the
// aurora's and not this table's.
func (g Gas) Spectrum(unit float64) *thing.Spectrum {
	passes := map[radiation.Range]float64{}
	for r, o := range g.Ranges {
		passes[r] = math.Exp(-(o.Absorb + o.Scatter) * unit)
	}
	if g.Visible != nil {
		a, s := g.Visible(550)
		passes[thing.Visible] = math.Exp(-(a + s) * unit)
	}
	return &thing.Spectrum{Passes: passes,
		Gives: map[radiation.Range]float64{}}
}
