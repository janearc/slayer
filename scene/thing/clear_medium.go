package thing

import (
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/maths"
)

// ClearMedium is the immutable optical identity of a homogeneous clear
// interior. Participating values retain separate RGB absorption and scattering
// at 610, 550 and 465 nm, in inverse metres. The legacy constructor retains
// only its supplied total extinction. MetresPerUnit converts scene
// world-distance units to metres.
//
// NewClearMedium describes extinction only. NewClearMediumWithScattering opts
// into bounded CPU first-order in-scattering; its source-path approximation is
// documented with that constructor and in trace's clear-medium notes.
type ClearMedium struct {
	name                       string
	absorption                 maths.Light
	scattering                 maths.Light
	extinction                 maths.Light
	phase                      float64
	metresPerUnit              float64
	inscatterSteps             int
	maxInscatterDistanceMetres float64
	diffuseSkySamples          int
}

// NewClearMedium checks and copies a named homogeneous clear medium whose
// supplied coefficient is total extinction. It has no scattering source term;
// its Absorption accessor reports the supplied value as effective absorption
// because the legacy constructor cannot recover a sigma_a/sigma_s split.
// Reusing a name in one render requires identical coefficients and units.
func NewClearMedium(name string, extinction maths.Light,
	metresPerUnit float64) (*ClearMedium, error) {
	if name == "" || strings.TrimSpace(name) != name {
		return nil, fmt.Errorf("thing: clear medium needs a nonempty trimmed name")
	}
	if !finiteClear(extinction.R, extinction.G, extinction.B) ||
		extinction.R < 0 || extinction.G < 0 || extinction.B < 0 {
		return nil, fmt.Errorf("thing: clear medium %q extinction must be finite and nonnegative per metre", name)
	}
	if !finiteClear(metresPerUnit) || metresPerUnit <= 0 {
		return nil, fmt.Errorf("thing: clear medium %q metres per world unit must be finite and positive", name)
	}
	return &ClearMedium{name: name, absorption: extinction,
		extinction: extinction, metresPerUnit: metresPerUnit}, nil
}

// maxClearInscatterSteps bounds the number of camera-path source evaluations
// per ray. Lamp work is separately bounded by maxClearInscatterLamps in trace.
const maxClearInscatterSteps = 64

// NewClearMediumWithScattering creates an explicitly participating homogeneous
// medium. Absorption and scattering are separate RGB coefficients per metre;
// extinction is their sum. Phase follows scene/gas.Gas: phase zero is
// Rayleigh, and nonzero values are Henyey-Greenstein asymmetry g strictly
// between -1 and 1. This API does not encode isotropic HG at g=0.
//
// The CPU tracer evaluates incident light at midpoint samples and integrates
// extinction exactly over each outgoing cell.
//
// The general clear-medium source path treats light as unattenuated and
// unrefracted, and refuses transparent boundaries, skies, haze, volume
// transport, fields and directionless ambient lamps.
//
// One separate opt-in ocean route may use the explicitly tagged,
// refracted mean-plane direct sun for its named ocean medium; by default that
// route omits diffuse-sky and legacy-fill scattering. WithDiffuseSkyInscatter
// separately enables an air-hemisphere sky source on an actual infinite
// mean-plane continuation.
//
// Steps and maximum camera-path distance are explicit work bounds, not
// quality presets. Coefficients are caller supplied; this constructor invents
// none.
func NewClearMediumWithScattering(name string, absorption, scattering maths.Light,
	phase, metresPerUnit float64, steps int,
	maxDistanceMetres float64) (*ClearMedium, error) {
	if name == "" || strings.TrimSpace(name) != name {
		return nil, fmt.Errorf("thing: clear medium needs a nonempty trimmed name")
	}
	if !finiteClear(absorption.R, absorption.G, absorption.B) ||
		absorption.R < 0 || absorption.G < 0 || absorption.B < 0 {
		return nil, fmt.Errorf("thing: clear medium %q absorption must be finite and nonnegative per metre", name)
	}
	if !finiteClear(scattering.R, scattering.G, scattering.B) ||
		scattering.R < 0 || scattering.G < 0 || scattering.B < 0 {
		return nil, fmt.Errorf("thing: clear medium %q scattering must be finite and nonnegative per metre", name)
	}
	extinction := maths.RGB(absorption.R+scattering.R,
		absorption.G+scattering.G, absorption.B+scattering.B)
	if !finiteClear(extinction.R, extinction.G, extinction.B) {
		return nil, fmt.Errorf("thing: clear medium %q total extinction must be finite per metre", name)
	}
	if !finiteClear(phase) || math.Abs(phase) >= 1 {
		return nil, fmt.Errorf("thing: clear medium %q phase must be finite and strictly between -1 and 1", name)
	}
	if !finiteClear(metresPerUnit) || metresPerUnit <= 0 {
		return nil, fmt.Errorf("thing: clear medium %q metres per world unit must be finite and positive", name)
	}
	if steps < 1 || steps > maxClearInscatterSteps {
		return nil, fmt.Errorf("thing: clear medium %q in-scatter steps must be between 1 and %d", name, maxClearInscatterSteps)
	}
	if !finiteClear(maxDistanceMetres) || maxDistanceMetres <= 0 {
		return nil, fmt.Errorf("thing: clear medium %q maximum in-scatter distance must be finite and positive metres", name)
	}
	return &ClearMedium{name: name, absorption: absorption,
		scattering: scattering, extinction: extinction, phase: phase,
		metresPerUnit: metresPerUnit, inscatterSteps: steps,
		maxInscatterDistanceMetres: maxDistanceMetres}, nil
}

// Name is the stable identity used to associate the same medium across its
// boundaries and an explicitly seeded camera state.
func (medium *ClearMedium) Name() string {
	if medium == nil {
		return ""
	}
	return medium.name
}

// Extinction returns the copied RGB extinction coefficients per metre.
func (medium *ClearMedium) Extinction() maths.Light {
	if medium == nil {
		return maths.Light{}
	}
	return medium.extinction
}

// Absorption returns the copied RGB absorption coefficients per metre for a
// participating medium. For a legacy extinction-only value it returns the
// supplied total extinction as effective absorption; its original split is
// unavailable.
func (medium *ClearMedium) Absorption() maths.Light {
	if medium == nil {
		return maths.Light{}
	}
	return medium.absorption
}

// Scattering returns the copied RGB scattering coefficients per metre.
func (medium *ClearMedium) Scattering() maths.Light {
	if medium == nil {
		return maths.Light{}
	}
	return medium.scattering
}

// Phase returns the gas phase convention: zero is Rayleigh; nonzero is
// Henyey-Greenstein asymmetry g. It is unused when Scattering is zero.
func (medium *ClearMedium) Phase() float64 {
	if medium == nil {
		return 0
	}
	return medium.phase
}

// HasScattering reports whether any RGB scattering coefficient is positive.
func (medium *ClearMedium) HasScattering() bool {
	return medium != nil && (medium.scattering.R > 0 ||
		medium.scattering.G > 0 || medium.scattering.B > 0)
}

// InscatterSteps returns the fixed number of camera-path source samples.
// Zero means no source integration was configured by the extinction-only API.
func (medium *ClearMedium) InscatterSteps() int {
	if medium == nil {
		return 0
	}
	return medium.inscatterSteps
}

// MaxInscatterDistanceMetres returns the explicit maximum camera-path distance
// integrated for in-scattering. Zero means it was not configured.
func (medium *ClearMedium) MaxInscatterDistanceMetres() float64 {
	if medium == nil {
		return 0
	}
	return medium.maxInscatterDistanceMetres
}

// WithDiffuseSkyInscatter opts a participating ocean medium into a bounded
// upper-hemisphere sky source integral. The returned medium is a copy; the
// receiver and its optical identity are unchanged. Samples are equal-solid-
// angle directions over the air hemisphere. This requires the explicitly
// tagged ocean mean-plane route and an infinite mean-plane continuation.
func (medium *ClearMedium) WithDiffuseSkyInscatter(samples int) (*ClearMedium, error) {
	if medium == nil || !medium.Defined() || !medium.HasScattering() {
		return nil, fmt.Errorf("thing: diffuse-sky in-scattering requires a defined participating clear medium")
	}
	if samples < 1 || samples > maxClearSkyInscatterSamples {
		return nil, fmt.Errorf("thing: diffuse-sky in-scattering samples must be between 1 and %d", maxClearSkyInscatterSamples)
	}
	configured := *medium
	configured.diffuseSkySamples = samples
	return &configured, nil
}

// maxClearSkyInscatterSamples bounds sky-source rays per camera-path sample.
const maxClearSkyInscatterSamples = 256

// DiffuseSkyInscatterSamples reports the explicit upper-hemisphere quadrature
// count, or zero when diffuse-sky source integration is disabled.
func (medium *ClearMedium) DiffuseSkyInscatterSamples() int {
	if medium == nil {
		return 0
	}
	return medium.diffuseSkySamples
}

// MetresPerUnit returns the explicit conversion from scene distance to metres.
func (medium *ClearMedium) MetresPerUnit() float64 {
	if medium == nil {
		return 0
	}
	return medium.metresPerUnit
}

// Defined reports whether this value was made by NewClearMedium. The zero
// value means a legacy, optically unattenuated clear medium.
func (medium *ClearMedium) Defined() bool {
	return medium != nil && medium.name != ""
}

// finiteClear reports whether every number is a real number.
func finiteClear(numbers ...float64) bool {
	for _, number := range numbers {
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return false
		}
	}
	return true
}
