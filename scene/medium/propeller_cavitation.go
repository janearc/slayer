package medium

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// The limits and shapes of a propeller's cavity are chosen for the picture, in
// choices.yaml, and none is measured.
//
// The blades are bounded by
// propellerMaxBlades, a numerical work budget on the analytic filament cores.
// propellerCoreSupport is the standard deviations of the authored Gaussian
// core; propellerTransition softens the zero crossing of the density mask; the
// fragment scale runs from propellerFragmentLeast to propellerFragmentMost to
// keep the noise coordinates finite and local.
//
// The authored controls are bounded above by propellerTipMost, propellerRPMMost,
// propellerSlipMost, propellerLengthMost, propellerCoreMost,
// propellerDensityMost, propellerPressureMost and propellerRecoveryMost, an
// origin by propellerOriginMost, and an axis must be longer than
// propellerAxisLeast and no more parallel to up than propellerParallel.
//
// The dispersal controls are bounded by propellerWanderMost,
// propellerWanderScaleMost, propellerWanderScaleLeast, propellerSpreadStartMost
// and propellerSpreadMost.
//
// The meander reads noise of propellerWanderOctaves
// octaves at propellerWanderBlade, propellerWanderAcross and propellerWanderOut
// on the axes not along the cavity, and the fragments are propellerOctaves
// octaves.
var (
	propellerMaxBlades        = choiceCount("propeller-max-blades")
	propellerCoreSupport      = choice("propeller-core-support")
	propellerTransition       = choice("propeller-transition")
	propellerFragmentLeast    = choice("propeller-fragment-least")
	propellerFragmentMost     = choice("propeller-fragment-most")
	propellerTipMost          = choice("propeller-tip-most")
	propellerRPMMost          = choice("propeller-rpm-most")
	propellerSlipMost         = choice("propeller-slip-most")
	propellerLengthMost       = choice("propeller-length-most")
	propellerCoreMost         = choice("propeller-core-most")
	propellerDensityMost      = choice("propeller-density-most")
	propellerPressureMost     = choice("propeller-pressure-most")
	propellerRecoveryMost     = choice("propeller-recovery-most")
	propellerOriginMost       = choice("propeller-origin-most")
	propellerAxisLeast        = choice("propeller-axis-least")
	propellerParallel         = choice("propeller-parallel")
	propellerWanderMost       = choice("propeller-wander-most")
	propellerWanderScaleMost  = choice("propeller-wander-scale-most")
	propellerWanderScaleLeast = choice("propeller-wander-scale-least")
	propellerSpreadStartMost  = choice("propeller-spread-start-most")
	propellerSpreadMost       = choice("propeller-spread-most")
	propellerWanderOctaves    = choiceCount("propeller-wander-octaves")
	propellerWanderBlade      = choice("propeller-wander-blade")
	propellerWanderAcross     = choice("propeller-wander-across")
	propellerWanderOut        = choice("propeller-wander-out")
	propellerOctaves          = choiceCount("propeller-fragment-octaves")
)

// the numbers of a rotation, true by definition.
const (
	fullTurn         = 2 * math.Pi // defined: the angle of one turn
	secondsPerMinute = 60          // defined: seconds in a minute
	gaussianHalf     = 0.5         // defined: a gaussian is exp(-r^2 / 2 sigma^2)
	firstBlade       = 1           // defined: the fewest blades a propeller has
)

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

	// Authored centreline meander and opt-in downstream radial dilution.
	WanderM, WanderScaleM, SpreadStartM, SpreadPerM float64
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

	wander, wanderScale, spreadStart, spread float64
}

// finiteConfig is whether every number of the config is finite.
func finiteConfig(config PropellerCavitationConfig) bool {
	values := []float64{config.Origin.X, config.Origin.Y, config.Origin.Z,
		config.Axis.X, config.Axis.Y, config.Axis.Z, config.TipRadius,
		config.RPM, config.SlipstreamMPS, config.LengthM, config.CoreRadiusM,
		config.DensityPerM, config.PressureDeficitPa, config.RecoveryPaPerM,
		config.FragmentScaleM, config.FragmentContrast, config.WanderM,
		config.WanderScaleM, config.SpreadStartM, config.SpreadPerM}
	for _, value := range values {
		if !weatherFinite(value) {
			return false
		}
	}
	return true
}

// positiveUpTo is whether a value is above zero and no more than most.
func positiveUpTo(value, most float64) bool {
	return value > nought && value <= most
}

// zeroUpTo is whether a value is at least zero and no more than most.
func zeroUpTo(value, most float64) bool {
	return value >= nought && value <= most
}

// controlsWithin is whether the controls are within their authored bounds.
func controlsWithin(config PropellerCavitationConfig) bool {
	origin := config.Origin
	return config.Blades >= firstBlade && config.Blades <= propellerMaxBlades &&
		math.Abs(origin.X) <= propellerOriginMost &&
		math.Abs(origin.Y) <= propellerOriginMost &&
		math.Abs(origin.Z) <= propellerOriginMost &&
		positiveUpTo(config.TipRadius, propellerTipMost) &&
		positiveUpTo(config.RPM, propellerRPMMost) &&
		positiveUpTo(config.SlipstreamMPS, propellerSlipMost) &&
		positiveUpTo(config.LengthM, propellerLengthMost) &&
		positiveUpTo(config.CoreRadiusM, propellerCoreMost) &&
		zeroUpTo(config.DensityPerM, propellerDensityMost) &&
		positiveUpTo(config.PressureDeficitPa, propellerPressureMost) &&
		positiveUpTo(config.RecoveryPaPerM, propellerRecoveryMost)
}

// fragmentWithin is whether the fragmentation is in range: a contrast of
// nought to one, and a scale in range when the contrast is above nought.
func fragmentWithin(config PropellerCavitationConfig) bool {
	return zeroUpTo(config.FragmentContrast, unity) &&
		zeroUpTo(config.FragmentScaleM, propellerFragmentMost) &&
		!(config.FragmentContrast > nought &&
			config.FragmentScaleM < propellerFragmentLeast)
}

// dispersalWithin is whether the meander and the dilution are in range.
func dispersalWithin(config PropellerCavitationConfig) bool {
	return zeroUpTo(config.WanderM, propellerWanderMost) &&
		zeroUpTo(config.WanderScaleM, propellerWanderScaleMost) &&
		!(config.WanderM > nought &&
			config.WanderScaleM < propellerWanderScaleLeast) &&
		zeroUpTo(config.SpreadStartM, propellerSpreadStartMost) &&
		zeroUpTo(config.SpreadPerM, propellerSpreadMost)
}

// NewPropellerCavitation validates the authored field and constructs its
// authored blade-count coordinate frame. The visual-onset flag must be explicit
// before the density can become nonzero.
func NewPropellerCavitation(config PropellerCavitationConfig) (
	PropellerCavitation, error) {
	if !finiteConfig(config) {
		return PropellerCavitation{}, fmt.Errorf("medium: propeller cavitation values must be finite")
	}
	axisLength := config.Axis.Len()
	if !weatherFinite(axisLength) || axisLength < propellerAxisLeast {
		return PropellerCavitation{}, fmt.Errorf("medium: propeller cavitation needs a finite non-zero axis")
	}
	if !controlsWithin(config) {
		return PropellerCavitation{}, fmt.Errorf("medium: propeller cavitation controls exceed their finite authored bounds")
	}
	if !fragmentWithin(config) {
		return PropellerCavitation{}, fmt.Errorf("medium: fragmentation needs contrast 0..1 and a scale from 0.001 to 20 m when enabled")
	}
	if !dispersalWithin(config) {
		return PropellerCavitation{}, fmt.Errorf("medium: dispersal controls exceed authored numerical bounds")
	}
	axis := config.Axis.Scale(unity / axisLength)
	up := maths.Up
	if math.Abs(axis.Dot(up)) > propellerParallel {
		up = maths.XAxis
	}
	radial := axis.Cross(up).Unit()
	cross := axis.Cross(radial).Unit()
	if radial.Len() == nought || cross.Len() == nought {
		return PropellerCavitation{}, fmt.Errorf("medium: propeller cavitation axis cannot define a radial frame")
	}
	activeLength := math.Min(config.LengthM,
		config.PressureDeficitPa/config.RecoveryPaPerM)
	if !weatherFinite(activeLength) || activeLength <= nought {
		return PropellerCavitation{}, fmt.Errorf("medium: pressure recovery produced an invalid cavity length")
	}
	field := PropellerCavitation{origin: config.Origin, axis: axis,
		blades: config.Blades, radial: radial, cross: cross,
		radius: config.TipRadius, rpm: config.RPM, slip: config.SlipstreamMPS,
		length: config.LengthM, core: config.CoreRadiusM,
		density: config.DensityPerM, pressureDeficit: config.PressureDeficitPa,
		recovery: config.RecoveryPaPerM, activeLength: activeLength,
		visualOnset: config.VisualOnset}
	field.fragmentScale = config.FragmentScaleM
	field.fragmentContrast = config.FragmentContrast
	field.fragmentSeed = config.FragmentSeed
	field.wander, field.wanderScale = config.WanderM, config.WanderScaleM
	field.spreadStart, field.spread = config.SpreadStartM, config.SpreadPerM
	field.bounds = field.bound()
	return field, nil
}

// reach is how far along one axis a circle of the radius reaches, given the
// components of its radial and cross directions on that axis.
func reach(radius, first, second float64) float64 {
	return radius * math.Sqrt(first*first+second*second)
}

// bound returns the axis-aligned box around the blades' swept envelope.
func (field PropellerCavitation) bound() maths.Box {
	end := field.origin.Add(field.axis.Scale(field.activeLength))
	widening := field.spread * math.Max(nought,
		field.activeLength-field.spreadStart)
	radius := field.radius + math.Sqrt2*field.wander +
		propellerCoreSupport*(field.core+widening)
	along := maths.V(reach(radius, field.radial.X, field.cross.X),
		reach(radius, field.radial.Y, field.cross.Y),
		reach(radius, field.radial.Z, field.cross.Z))
	return corners(
		maths.V(math.Min(field.origin.X, end.X)-along.X,
			math.Min(field.origin.Y, end.Y)-along.Y,
			math.Min(field.origin.Z, end.Z)-along.Z),
		maths.V(math.Max(field.origin.X, end.X)+along.X,
			math.Max(field.origin.Y, end.Y)+along.Y,
			math.Max(field.origin.Z, end.Z)+along.Z))
}

// Bounds is the finite axis-aligned envelope around the authored filaments.
func (field PropellerCavitation) Bounds() maths.Box { return field.bounds }

// Span clips a ray to the field's finite envelope.
func (field PropellerCavitation) Span(ray maths.Ray, near, far float64) (
	float64, float64, bool) {
	return field.bounds.Hit(ray, near, far)
}

// unitBound is a value kept from minus one to one.
func unitBound(value float64) float64 {
	return math.Max(-unity, math.Min(unity, value))
}

// filamentCentre returns the centre of one blade's filament at a downstream
// distance along the axis.
func (field PropellerCavitation) filamentCentre(downstream float64,
	blade int) maths.Vec {
	phase := fullTurn*field.rpm/secondsPerMinute*
		(field.seconds-downstream/field.slip) +
		fullTurn*float64(blade)/float64(field.blades)
	cx, cy := field.radius*math.Cos(phase), field.radius*math.Sin(phase)
	if field.wander > nought {
		packet := (downstream - field.slip*field.seconds) / field.wanderScale
		amount := field.wander * maths.Smooth(nought, field.wanderScale,
			downstream)
		first := noise.Fractal(field.fragmentSeed,
			float64(blade)+propellerWanderBlade, packet, propellerWanderAcross,
			propellerWanderOctaves)
		second := noise.Fractal(field.fragmentSeed,
			float64(blade)+propellerWanderBlade, packet, propellerWanderOut,
			propellerWanderOctaves)
		cx += amount * unitBound(first)
		cy += amount * unitBound(second)
	}
	return field.origin.Add(field.axis.Scale(downstream)).
		Add(field.radial.Scale(cx)).
		Add(field.cross.Scale(cy))
}

// width is the cavity's core radius at a downstream distance, widened where
// the dilution is on.
func (field PropellerCavitation) width(downstream float64) float64 {
	width := field.core
	if field.spread > nought {
		width += field.spread * math.Max(nought, downstream-field.spreadStart)
	}
	return width
}

// fragmentShare is how much of the cavity the fragmentation leaves at a point:
// one with none, down to one less the contrast where the noise mask is low.
func (field PropellerCavitation) fragmentShare(x, y, downstream float64) float64 {
	fragment := float64(unity)
	if field.fragmentContrast > nought {
		packet := downstream - field.slip*field.seconds
		pattern := noise.Fractal(field.fragmentSeed, x/field.fragmentScale,
			y/field.fragmentScale, packet/field.fragmentScale,
			propellerOctaves)
		mask := maths.Smooth(nought, propellerTransition, pattern)
		amount := field.fragmentContrast * maths.Smooth(field.core*double,
			field.activeLength/double, downstream)
		fragment -= amount * (unity - mask)
	}
	return fragment
}

// Density is the sum of compact Gaussian cores. Their phase advances at
// authored shaft RPM and their downstream position is convected at the
// authored slipstream speed. Linear pressure recovery and the explicit
// deficit/recovery inputs bound cavity persistence; these values are shot
// controls, not measured Alfa hydrodynamics.
func (field PropellerCavitation) Density(point maths.Vec) float64 {
	if !field.visualOnset || field.density == nought {
		return nought
	}
	delta := point.Sub(field.origin)
	downstream := delta.Dot(field.axis)
	if downstream < nought || downstream > field.activeLength {
		return nought
	}
	x, y := delta.Dot(field.radial), delta.Dot(field.cross)
	var density float64
	width := field.width(downstream)
	for blade := range field.blades {
		centre := field.filamentCentre(downstream, blade).Sub(field.origin).
			Sub(field.axis.Scale(downstream))
		cx, cy := centre.Dot(field.radial), centre.Dot(field.cross)
		squared := (x-cx)*(x-cx) + (y-cy)*(y-cy)
		if squared > propellerCoreSupport*propellerCoreSupport*width*width {
			continue
		}
		density += math.Exp(-gaussianHalf * squared / (width * width))
	}
	if field.spread > nought {
		density *= (field.core / width) * (field.core / width)
	}
	startFade := maths.Smooth(nought, field.core*double, downstream)
	endFade := unity - maths.Smooth(field.activeLength-field.core*double,
		field.activeLength, downstream)
	pressureFade := math.Max(nought,
		unity-field.recovery*downstream/field.pressureDeficit)
	fragment := float64(unity)
	if density > nought {
		fragment = field.fragmentShare(x, y, downstream)
	}
	return field.density * density * startFade * endFade * pressureFade *
		fragment
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
	return PropellerCavitationConfig{Blades: field.blades, Origin: field.origin,
		Axis: field.axis, TipRadius: field.radius, RPM: field.rpm,
		SlipstreamMPS: field.slip, LengthM: field.length,
		CoreRadiusM: field.core, DensityPerM: field.density,
		PressureDeficitPa: field.pressureDeficit,
		RecoveryPaPerM:    field.recovery, VisualOnset: field.visualOnset,
		FragmentScaleM:   field.fragmentScale,
		FragmentContrast: field.fragmentContrast,
		FragmentSeed:     field.fragmentSeed, WanderM: field.wander,
		WanderScaleM: field.wanderScale, SpreadStartM: field.spreadStart,
		SpreadPerM: field.spread}
}

// Seconds is the evaluated deterministic phase clock, after Tick.
func (field PropellerCavitation) Seconds() float64 { return field.seconds }
