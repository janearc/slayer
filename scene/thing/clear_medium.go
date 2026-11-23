package thing

import (
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
)

// The counts a clear medium is checked against.
const (
	// clearNone is defined: no amount.
	clearNone = 0
	// clearWhole is defined: one amount, the whole of a share, and the
	// fewest steps or samples there can be.
	clearWhole = 1
	// clearEitherSign is defined by math.IsInf: a sign of nought tests for
	// infinity of both signs.
	clearEitherSign = 0
)

// maxClearInscatterSteps bounds the number of camera-path source evaluations
// per ray. Lamp work is separately bounded by maxClearInscatterLamps in trace.
// It is read from picks.yaml.
var maxClearInscatterSteps = int(pick("clear-inscatter-steps-most"))

// maxClearSkyInscatterSamples bounds sky-source rays per camera-path sample.
// It is read from picks.yaml.
var maxClearSkyInscatterSamples = int(pick("clear-sky-inscatter-samples-most"))

// ClearMedium is the immutable optical identity of a homogeneous clear
// interior. Participating values retain separate RGB absorption and scattering
// at 610, 550 and 465 nm, in inverse metres. NewClearMedium retains only
// its supplied total extinction. MetresPerUnit converts scene
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
	water                      *materials.WaterComposition
	opticalSource              string
}

// WithWaterComposition retains observations and the optical source alongside
// an independently authored optical medium. It does not change transport.
func (medium *ClearMedium) WithWaterComposition(water *materials.WaterComposition,
	opticalSource string,
) (*ClearMedium, error) {
	if medium == nil || !medium.Defined() || water == nil {
		return nil, fmt.Errorf("thing: water composition needs a defined medium and observations")
	}
	configured := *medium
	configured.water = water
	configured.opticalSource = opticalSource
	return &configured, nil
}

// WaterComposition reports independent observations and optical provenance.
func (medium *ClearMedium) WaterComposition() (*materials.WaterComposition, string, bool) {
	if medium == nil || medium.water == nil {
		return nil, "", false
	}
	return medium.water, medium.opticalSource, true
}

// SameOpticalIdentity compares the transport values, including named identity
// and work budgets. Observation provenance does not alter optical identity.
func (medium *ClearMedium) SameOpticalIdentity(other *ClearMedium) bool {
	if medium == nil || other == nil {
		return medium == nil && other == nil
	}
	ours, theirs := *medium, *other
	ours.water, theirs.water = nil, nil
	ours.opticalSource, theirs.opticalSource = "", ""
	return ours == theirs
}

// NewClearMedium checks and copies a named homogeneous clear medium whose
// supplied coefficient is total extinction. It has no scattering source term;
// its Absorption accessor reports the supplied value as effective absorption
// because a total extinction cannot be split into sigma_a and sigma_s.
// Reusing a name in one render requires identical coefficients and units.
func NewClearMedium(name string, extinction maths.Light,
	metresPerUnit float64) (*ClearMedium, error) {
	if name == "" || strings.TrimSpace(name) != name {
		return nil, fmt.Errorf("thing: clear medium needs a nonempty trimmed name")
	}
	if !finiteClear(extinction.R, extinction.G, extinction.B) ||
		extinction.R < clearNone || extinction.G < clearNone ||
		extinction.B < clearNone {
		return nil, fmt.Errorf("thing: clear medium %q extinction must be finite and nonnegative per metre", name)
	}
	if !finiteClear(metresPerUnit) || metresPerUnit <= clearNone {
		return nil, fmt.Errorf("thing: clear medium %q metres per world unit must be finite and positive", name)
	}
	return &ClearMedium{name: name, absorption: extinction,
		extinction: extinction, metresPerUnit: metresPerUnit}, nil
}

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
// route omits diffuse-sky and ordinary-ground fill scattering.
// WithDiffuseSkyInscatter separately enables an air-hemisphere sky source on
// an actual infinite mean-plane continuation.
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
		absorption.R < clearNone || absorption.G < clearNone ||
		absorption.B < clearNone {
		return nil, fmt.Errorf("thing: clear medium %q absorption must be finite and nonnegative per metre", name)
	}
	if !finiteClear(scattering.R, scattering.G, scattering.B) ||
		scattering.R < clearNone || scattering.G < clearNone ||
		scattering.B < clearNone {
		return nil, fmt.Errorf("thing: clear medium %q scattering must be finite and nonnegative per metre", name)
	}
	extinction := maths.RGB(absorption.R+scattering.R,
		absorption.G+scattering.G, absorption.B+scattering.B)
	if !finiteClear(extinction.R, extinction.G, extinction.B) {
		return nil, fmt.Errorf("thing: clear medium %q total extinction must be finite per metre", name)
	}
	if !finiteClear(phase) || math.Abs(phase) >= clearWhole {
		return nil, fmt.Errorf("thing: clear medium %q phase must be finite and strictly between -1 and 1", name)
	}
	if !finiteClear(metresPerUnit) || metresPerUnit <= clearNone {
		return nil, fmt.Errorf("thing: clear medium %q metres per world unit must be finite and positive", name)
	}
	if steps < clearWhole || steps > maxClearInscatterSteps {
		return nil, fmt.Errorf("thing: clear medium %q in-scatter steps must be between 1 and %d", name, maxClearInscatterSteps)
	}
	if !finiteClear(maxDistanceMetres) || maxDistanceMetres <= clearNone {
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
		return maths.Grey(clearNone)
	}
	return medium.extinction
}

// Absorption returns the copied RGB absorption coefficients per metre for a
// participating medium. For an extinction-only value it returns the
// supplied total extinction as effective absorption; its original split is
// unavailable.
func (medium *ClearMedium) Absorption() maths.Light {
	if medium == nil {
		return maths.Grey(clearNone)
	}
	return medium.absorption
}

// Scattering returns the copied RGB scattering coefficients per metre.
func (medium *ClearMedium) Scattering() maths.Light {
	if medium == nil {
		return maths.Grey(clearNone)
	}
	return medium.scattering
}

// Phase returns the gas phase convention: zero is Rayleigh; nonzero is
// Henyey-Greenstein asymmetry g. It is unused when Scattering is zero.
func (medium *ClearMedium) Phase() float64 {
	if medium == nil {
		return clearNone
	}
	return medium.phase
}

// HasScattering reports whether any RGB scattering coefficient is positive.
func (medium *ClearMedium) HasScattering() bool {
	return medium != nil && (medium.scattering.R > clearNone ||
		medium.scattering.G > clearNone || medium.scattering.B > clearNone)
}

// InscatterSteps returns the fixed number of camera-path source samples.
// Zero means no source integration was configured by the extinction-only API.
func (medium *ClearMedium) InscatterSteps() int {
	if medium == nil {
		return clearNone
	}
	return medium.inscatterSteps
}

// MaxInscatterDistanceMetres returns the explicit maximum camera-path distance
// integrated for in-scattering. Zero means it was not configured.
func (medium *ClearMedium) MaxInscatterDistanceMetres() float64 {
	if medium == nil {
		return clearNone
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
	if samples < clearWhole || samples > maxClearSkyInscatterSamples {
		return nil, fmt.Errorf("thing: diffuse-sky in-scattering samples must be between 1 and %d", maxClearSkyInscatterSamples)
	}
	configured := *medium
	configured.diffuseSkySamples = samples
	return &configured, nil
}

// DiffuseSkyInscatterSamples reports the explicit upper-hemisphere quadrature
// count, or zero when diffuse-sky source integration is disabled.
func (medium *ClearMedium) DiffuseSkyInscatterSamples() int {
	if medium == nil {
		return clearNone
	}
	return medium.diffuseSkySamples
}

// MetresPerUnit returns the explicit conversion from scene distance to metres.
func (medium *ClearMedium) MetresPerUnit() float64 {
	if medium == nil {
		return clearNone
	}
	return medium.metresPerUnit
}

// Defined reports whether this value was made by NewClearMedium. The zero
// value means an optically unattenuated clear medium.
func (medium *ClearMedium) Defined() bool {
	return medium != nil && medium.name != ""
}

// finiteClear reports whether every number is a real number.
func finiteClear(numbers ...float64) bool {
	for _, number := range numbers {
		if math.IsNaN(number) || math.IsInf(number, clearEitherSign) {
			return false
		}
	}
	return true
}
