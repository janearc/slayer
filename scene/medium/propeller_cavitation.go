package medium

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// This numerical work budget bounds the number of analytic filament cores.
const maxPropellerCavitationBlades = 16

// Three standard deviations bound the authored Gaussian core approximation.
const propellerCoreSupport = 3.0

// This chosen density-mask transition softens the zero crossing of noise.
const propellerFragmentTransition = .5

// These chosen numerical limits keep density-noise coordinates finite and local.
const minPropellerFragmentScale, maxPropellerFragmentScale = .001, 20.0

// PropellerCavitationConfig is an authored, bounded approximation of tip
// vortex vapor cavities. Pressure deficit and recovery are explicit visual
// controls; they are not inferred from an Alfa pressure solution.
type PropellerCavitationConfig struct {
	Blades            int
	Origin            maths.Vec
	Axis              maths.Vec
	TipRadius         float64
	RPM               float64
	SlipstreamMPS     float64
	LengthM           float64
	CoreRadiusM       float64
	DensityPerM       float64
	PressureDeficitPa float64
	RecoveryPaPerM    float64
	VisualOnset       bool
	FragmentScaleM    float64
	FragmentContrast  float64
	FragmentSeed      uint64
}

// PropellerCavitation is phase-locked, advected tip-vortex density
// filaments. It has no per-particle geometry and is finite in space.
type PropellerCavitation struct {
	origin, axis, radial, cross     maths.Vec
	blades                          int
	radius, rpm, slip, length       float64
	core, density                   float64
	pressureDeficit, recovery       float64
	activeLength                    float64
	visualOnset                     bool
	seconds                         float64
	fragmentScale, fragmentContrast float64
	fragmentSeed                    uint64
	bounds                          maths.Box
}

// NewPropellerCavitation validates the authored field and constructs its
// authored blade-count coordinate frame. The visual-onset flag must be explicit
// before the density can become nonzero.
func NewPropellerCavitation(config PropellerCavitationConfig) (PropellerCavitation, error) {
	values := []float64{config.Origin.X, config.Origin.Y, config.Origin.Z,
		config.Axis.X, config.Axis.Y, config.Axis.Z, config.TipRadius,
		config.RPM, config.SlipstreamMPS, config.LengthM, config.CoreRadiusM,
		config.DensityPerM, config.PressureDeficitPa, config.RecoveryPaPerM,
		config.FragmentScaleM, config.FragmentContrast}
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return PropellerCavitation{}, fmt.Errorf("medium: propeller cavitation values must be finite")
		}
	}
	axisLength := config.Axis.Len()
	if math.IsNaN(axisLength) || math.IsInf(axisLength, 0) || axisLength < 1e-9 {
		return PropellerCavitation{}, fmt.Errorf("medium: propeller cavitation needs a finite non-zero axis")
	}
	if config.Blades < 1 || config.Blades > maxPropellerCavitationBlades ||
		math.Abs(config.Origin.X) > 1e13 || math.Abs(config.Origin.Y) > 1e13 ||
		math.Abs(config.Origin.Z) > 1e13 || config.TipRadius <= 0 || config.TipRadius > 5 ||
		config.RPM <= 0 || config.RPM > 600 || config.SlipstreamMPS <= 0 ||
		config.SlipstreamMPS > 100 || config.LengthM <= 0 || config.LengthM > 200 ||
		config.CoreRadiusM <= 0 || config.CoreRadiusM > 2 ||
		config.DensityPerM < 0 || config.DensityPerM > 20 ||
		config.PressureDeficitPa <= 0 || config.PressureDeficitPa > 1e6 ||
		config.RecoveryPaPerM <= 0 || config.RecoveryPaPerM > 1e6 {
		return PropellerCavitation{}, fmt.Errorf("medium: propeller cavitation controls exceed their finite authored bounds")
	}
	if config.FragmentContrast < 0 || config.FragmentContrast > 1 ||
		config.FragmentScaleM < 0 || config.FragmentScaleM > maxPropellerFragmentScale ||
		config.FragmentContrast > 0 && config.FragmentScaleM < minPropellerFragmentScale {
		return PropellerCavitation{}, fmt.Errorf("medium: fragmentation needs contrast 0..1 and a scale from 0.001 to 20 m when enabled")
	}
	axis := config.Axis.Scale(1 / axisLength)
	up := maths.Up
	if math.Abs(axis.Dot(up)) > 0.99 {
		up = maths.XAxis
	}
	radial := axis.Cross(up).Unit()
	cross := axis.Cross(radial).Unit()
	if radial.Len() == 0 || cross.Len() == 0 {
		return PropellerCavitation{}, fmt.Errorf("medium: propeller cavitation axis cannot define a radial frame")
	}
	activeLength := math.Min(config.LengthM, config.PressureDeficitPa/config.RecoveryPaPerM)
	if math.IsNaN(activeLength) || math.IsInf(activeLength, 0) || activeLength <= 0 {
		return PropellerCavitation{}, fmt.Errorf("medium: pressure recovery produced an invalid cavity length")
	}
	field := PropellerCavitation{origin: config.Origin, axis: axis, blades: config.Blades,
		radial: radial, cross: cross, radius: config.TipRadius,
		rpm: config.RPM, slip: config.SlipstreamMPS,
		length: config.LengthM, core: config.CoreRadiusM,
		density: config.DensityPerM, pressureDeficit: config.PressureDeficitPa,
		recovery: config.RecoveryPaPerM, activeLength: activeLength,
		visualOnset: config.VisualOnset}
	field.fragmentScale, field.fragmentContrast = config.FragmentScaleM, config.FragmentContrast
	field.fragmentSeed = config.FragmentSeed
	field.bounds = field.bound()
	return field, nil
}

// bound returns the axis-aligned box around the blades' swept envelope.
func (field PropellerCavitation) bound() maths.Box {
	end := field.origin.Add(field.axis.Scale(field.activeLength))
	radius := field.radius + propellerCoreSupport*field.core
	extent := func(a, b float64) float64 { return radius * math.Sqrt(a*a+b*b) }
	return maths.Box{
		Min: maths.V(math.Min(field.origin.X, end.X)-extent(field.radial.X, field.cross.X),
			math.Min(field.origin.Y, end.Y)-extent(field.radial.Y, field.cross.Y),
			math.Min(field.origin.Z, end.Z)-extent(field.radial.Z, field.cross.Z)),
		Max: maths.V(math.Max(field.origin.X, end.X)+extent(field.radial.X, field.cross.X),
			math.Max(field.origin.Y, end.Y)+extent(field.radial.Y, field.cross.Y),
			math.Max(field.origin.Z, end.Z)+extent(field.radial.Z, field.cross.Z)),
	}
}

// Bounds is the finite axis-aligned envelope around the authored filaments.
func (field PropellerCavitation) Bounds() maths.Box { return field.bounds }

// Span clips a ray to the field's finite envelope.
func (field PropellerCavitation) Span(ray maths.Ray, near, far float64) (float64, float64, bool) {
	return field.bounds.Hit(ray, near, far)
}

// filamentCentre returns the centre of one blade's filament at a downstream
// distance along the axis.
func (field PropellerCavitation) filamentCentre(downstream float64, blade int) maths.Vec {
	phase := 2*math.Pi*field.rpm/60*(field.seconds-downstream/field.slip) +
		2*math.Pi*float64(blade)/float64(field.blades)
	return field.origin.Add(field.axis.Scale(downstream)).
		Add(field.radial.Scale(field.radius * math.Cos(phase))).
		Add(field.cross.Scale(field.radius * math.Sin(phase)))
}

// Density is the sum of compact Gaussian cores. Their phase advances at
// authored shaft RPM and their downstream position is convected at the
// authored slipstream speed. Linear pressure recovery and the explicit
// deficit/recovery inputs bound cavity persistence; these values are shot
// controls, not measured Alfa hydrodynamics.
func (field PropellerCavitation) Density(point maths.Vec) float64 {
	if !field.visualOnset || field.density == 0 {
		return 0
	}
	delta := point.Sub(field.origin)
	downstream := delta.Dot(field.axis)
	if downstream < 0 || downstream > field.activeLength {
		return 0
	}
	x, y := delta.Dot(field.radial), delta.Dot(field.cross)
	var density float64
	for blade := range field.blades {
		centre := field.filamentCentre(downstream, blade).Sub(field.origin).
			Sub(field.axis.Scale(downstream))
		cx, cy := centre.Dot(field.radial), centre.Dot(field.cross)
		r2 := (x-cx)*(x-cx) + (y-cy)*(y-cy)
		if r2 > propellerCoreSupport*propellerCoreSupport*field.core*field.core {
			continue
		}
		density += math.Exp(-0.5 * r2 / (field.core * field.core))
	}
	startFade := maths.Smooth(0, field.core*2, downstream)
	endFade := 1 - maths.Smooth(field.activeLength-field.core*2,
		field.activeLength, downstream)
	pressureFade := math.Max(0, 1-field.recovery*downstream/field.pressureDeficit)
	fragment := 1.0
	if density > 0 && field.fragmentContrast > 0 {
		packet := downstream - field.slip*field.seconds
		pattern := noise.Fractal(field.fragmentSeed, x/field.fragmentScale,
			y/field.fragmentScale, packet/field.fragmentScale, 3)
		mask := maths.Smooth(0, propellerFragmentTransition, pattern)
		amount := field.fragmentContrast * maths.Smooth(field.core*2,
			field.activeLength/2, downstream)
		fragment -= amount * (1 - mask)
	}
	return field.density * density * startFade * endFade * pressureFade * fragment
}

// Tick returns the same authored field with a deterministic shaft phase.
func (field PropellerCavitation) Tick(at time.Duration) Medium {
	field.seconds = at.Seconds()
	return field
}

// Parameters returns the validated authored controls with a normalized axis.
// The copy permits alternate backends to evaluate the same generic density
// field without baking it into a sampled volume or individual particle meshes.
func (field PropellerCavitation) Parameters() PropellerCavitationConfig {
	return PropellerCavitationConfig{Blades: field.blades, Origin: field.origin, Axis: field.axis,
		TipRadius: field.radius, RPM: field.rpm, SlipstreamMPS: field.slip,
		LengthM: field.length, CoreRadiusM: field.core, DensityPerM: field.density,
		PressureDeficitPa: field.pressureDeficit, RecoveryPaPerM: field.recovery,
		VisualOnset: field.visualOnset, FragmentScaleM: field.fragmentScale,
		FragmentContrast: field.fragmentContrast, FragmentSeed: field.fragmentSeed}
}

// Seconds is the evaluated deterministic phase clock, after Tick.
func (field PropellerCavitation) Seconds() float64 { return field.seconds }
