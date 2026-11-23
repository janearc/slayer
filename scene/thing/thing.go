// Package thing describes scene objects, materials, motion, volumes, and
// lights. A World assembles them for the renderer.
//
// Nested transforms place objects relative to their parents. Nested, Spin,
// Growing, Orbit and Keyed compose those placements over time, using
// maths.Subspace for coordinate transformations.
//
// Census and Storage count what a world holds. OceanSurface records an
// authored ocean for the renderers. Made joins a measured material to a
// finish from finishes.yaml.
package thing

import (
	"math"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/texture"
)

const (
	thingNought = 0   // defined: no amount
	thingUnity  = 1   // defined: one amount, the whole of a share
	thingPair   = 2   // defined: two, the parts of a pair
	thingHalf   = 0.5 // defined: one over two
)

// thingBeamSpeedCap is the fastest share of light speed a rotating beam is
// allowed to reach, read from picks.yaml.
var thingBeamSpeedCap = pick("beam-speed-cap")

// Material describes surface response. Instances are shared: modifying one
// changes every object referencing it. Do not mutate it during rendering.
type Material struct {
	Name string
	// Source records supplied optical provenance for water with observations.
	Source string
	// WaterComposition retains sourced observations independently of optics.
	WaterComposition *materials.WaterComposition
	// Colour is diffuse RGB reflectance, with each channel from zero to
	// one.
	Colour maths.Light
	// Texture overrides Colour with spatially varying local-space
	// reflectance.
	Texture texture.Texture
	// Glow is uniform emitted light, independent of illumination.
	Glow maths.Light
	// Shine overrides Glow with local-space emission, such as lava or a hot
	// disc.
	Shine texture.Texture
	// IndependentEmission adds a surface's emitted light without reducing it
	// by reflection or transmission. This is an explicit CPU-only approximation
	// for emitters carried at a transparent boundary. Without it, emission is
	// mixed with what the surface reflects and transmits. Metal encoding
	// refuses this option.
	IndependentEmission bool
	// Radiant describes band-dependent emission and transmission. Without
	// it, the material is opaque and dark outside visible rendering.
	Radiant Radiant
	// Rough supplies local roughness for radar scattering through texture
	// luminance. Without a texture, roughness is uniformly one.
	Rough texture.Texture
	// Gloss scales Fresnel reflection; zero disables it and one applies the
	// full glass or polished-surface response. The renderer follows one
	// reflected ray.
	//
	// Index is refractive index. An index at or below zero selects 1.5,
	// the glass default; see Dense. Reflection increases toward grazing
	// incidence.
	Gloss, Index float64
	// Rank says which of two clear things a point inside both is made
	// of: the one of the higher rank, and between equals the one the
	// light entered last.
	//
	// Water poured in a glass is modelled as a body
	// of water that runs a little way into the glass's wall; the glass
	// outranks it, so where they overlap there is glass, and the
	// surface where water meets glass has glass on one side and water on
	// the other, with no gap of air and no two faces in one place. The
	// zero rank is every clear thing's until one is given.
	Rank int
	// Through is the share of the light Fresnel does not reflect that
	// the surface lets through, bent by Index: zero is opaque, one a
	// clear dielectric like water or glass. The renderer follows the
	// refracted ray through as many surfaces as trace's refractions.
	//
	// A material with Through above zero that is no metal reflects by
	// SplitsBetween, the unpolarised Fresnel share for its index and the
	// side the ray arrives from, and not by mirrors' Schlick. A metal
	// reflects by Reflects whatever its Through.
	//
	// A partial reflection on a ray that has already bent is paid and
	// not followed. Existing trace entry points start in the air; the
	// explicit trace.WithInitialMedia entry points can seed a camera's
	// clear-medium state when the caller supplies it.
	Through float64
	// ClearMedium names this boundary's homogeneous interior for the CPU
	// tracer's unscattered Beer-Lambert transport. Nil leaves a clear
	// surface unattenuated.
	ClearMedium *ClearMedium
	// Metal, when it is given, makes the surface a conductor: it mirrors
	// by its complex index, a different share of each colour, scaled by
	// Gloss, and Index no longer says how much it mirrors. Film, when it
	// is given too, is a clear layer over the metal; without a Metal it
	// is ignored. See Reflects.
	Metal *Metal
	Film  *Film
	// Satin is how far a glossy surface is from polished, for the
	// highlights the lamps leave on it: zero is polished, whose
	// highlight is a point too small to draw, and one is a broad sheen.
	// Its square is the GGX width; see Microfacet. It does not blur
	// what the surface mirrors.
	Satin float64
	// SlopeVariance, when above nought, is the mean-square slope, both
	// ways across the surface together, of tiny faces too small for its
	// shape to carry. Its highlights then take Beckmann's lobe of that
	// slope in place of Satin's GGX lobe. See Beckmann.
	SlopeVariance float64
	// Crinkle, when it is given, wrinkles the surface: its normal
	// tilts by noise, for how it mirrors and how it is lit.
	Crinkle *Crinkle
	// WakeFoam is an optional, bounded opaque aerated-water mixture on a
	// surface wake. Nil or zero Coverage leaves the material unchanged.
	WakeFoam *WakeFoamResponse
}

// WakeFoamResponse is a shot-authored diffuse surface response within the
// surface wake. Coordinates and dimensions use the ocean field's local metres;
// the stage converts authored map units once. It is an appearance mixture,
// not fluid, bubble, or cavitation simulation.
type WakeFoamResponse struct {
	OriginEast, OriginNorth float64
	Heading, Speed          float64
	HalfWidth, Length       float64
	Wavelength              float64
	Coverage, Band          float64
	Reflectance             maths.Light
	Satin                   float64
	// FrozenTime uses Seconds instead of the render clock for snapshot oceans.
	FrozenTime bool
	Seconds    float64
	Seed       uint32
}

// thingFinite reports whether a value is neither not-a-number nor infinite.
func thingFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, thingNought)
}

// thingShare reports whether a value lies from nought to unity, both
// included.
func thingShare(value float64) bool {
	return value >= thingNought && value <= thingUnity
}

// Valid reports whether this authored response is finite and bounded.
func (foam WakeFoamResponse) Valid() bool {
	if foam.Coverage == thingNought {
		return true
	}
	if foam.FrozenTime &&
		(!thingFinite(foam.Seconds) || foam.Seconds < thingNought) {
		return false
	}
	return thingFinite(foam.OriginEast) && thingFinite(foam.OriginNorth) &&
		thingFinite(foam.Heading) &&
		thingFinite(foam.Speed) && foam.Speed >= thingNought &&
		thingFinite(foam.HalfWidth) && foam.HalfWidth > thingNought &&
		thingFinite(foam.Length) && foam.Length > thingNought &&
		thingFinite(foam.Wavelength) && foam.Wavelength > thingNought &&
		foam.Wavelength <= foam.Length &&
		thingFinite(foam.Coverage) && foam.Coverage > thingNought &&
		foam.Coverage <= thingUnity &&
		thingFinite(foam.Band) && foam.Band > thingNought &&
		foam.Band <= thingUnity &&
		thingShare(foam.Reflectance.R) && thingShare(foam.Reflectance.G) &&
		thingShare(foam.Reflectance.B) &&
		thingFinite(foam.Satin) && foam.Satin > thingNought &&
		foam.Satin <= thingUnity
}

// At returns bounded foam coverage at a local east/up/south point and scene
// time. The breakup bands derive their period from the authored wake wavelength.
func (foam WakeFoamResponse) At(local maths.Vec, seconds float64) float64 {
	if foam.FrozenTime {
		seconds = foam.Seconds
	}
	if foam.Coverage <= thingNought || !foam.Valid() ||
		!thingFinite(seconds) || seconds < thingNought {
		return thingNought
	}
	forwardEast, forwardNorth := math.Sin(foam.Heading), math.Cos(foam.Heading)
	centerEast := foam.OriginEast + forwardEast*foam.Speed*seconds
	centerNorth := foam.OriginNorth + forwardNorth*foam.Speed*seconds
	deltaEast, deltaNorth := local.X-centerEast, -(local.Z)-centerNorth
	aft := -deltaEast*forwardEast - deltaNorth*forwardNorth
	cross := deltaEast*forwardNorth - deltaNorth*forwardEast
	if aft <= thingNought || aft >= foam.Length {
		return thingNought
	}
	longitudinal := math.Sin(math.Pi * aft / foam.Length)
	width := thingPair * foam.HalfWidth
	if math.Abs(cross) >= width {
		return thingNought
	}
	// Smoothly fade the mask to zero at the authored support edge.
	edge := thingUnity - math.Abs(cross)/width
	edge *= edge
	lateral := math.Exp(-thingHalf*math.Pow(cross/foam.HalfWidth, thingPair)) *
		edge
	// One coherent-noise octave breaks the patch at the authored scale.
	// The affine map takes its signed base band into coverage coordinates.
	pattern := thingHalf + thingHalf*noise.Fractal(uint64(foam.Seed),
		aft/foam.Wavelength, cross/foam.Wavelength, thingNought, thingUnity)
	// The signed noise's zero contour separates patches; Band softens its edge.
	threshold := thingHalf - foam.Band/thingPair
	breakup := math.Max(thingNought,
		math.Min(thingUnity, (pattern-threshold)/foam.Band))
	return math.Max(thingNought, math.Min(foam.Coverage,
		foam.Coverage*longitudinal*lateral*breakup))
}

// glassIndex is the refractive index of glass, which a surface or a film
// that gives none, or one at or below zero, is read as having.
var glassIndex = pick("glass-index")

// Dense is the refractive index a surface reflects and bends by:
// Index, or glass's 1.5 where none is given or it is at or below zero.
func (material *Material) Dense() float64 {
	if material.Index <= thingNought {
		return glassIndex
	}
	return material.Index
}

// Mirrors returns Schlick Fresnel reflectance at incidence cosine cos, scaled
// by Gloss. Nonpositive gloss returns zero; values above one are clamped so
// reflected light cannot exceed incident light.
func (material *Material) Mirrors(cos float64) float64 {
	if material.Gloss <= thingNought {
		return thingNought
	}
	if material.Gloss > thingUnity {
		clamped := &Material{Gloss: thingUnity, Index: material.Index}
		return clamped.Mirrors(cos)
	}
	index := material.Dense()
	normal := (index - thingUnity) / (index + thingUnity)
	normal *= normal
	inside := math.Min(thingUnity, cos)
	away := thingUnity - math.Max(thingNought, inside)
	return material.Gloss *
		(normal + (thingUnity-normal)*away*away*away*away*away)
}

// SplitsBetween returns the share a clear surface reflects, by Fresnel:
// the light arriving at incidence cosine cos that a surface which
// transmits reflects, scaled by Gloss: the unpolarised Fresnel
// reflectance of a boundary where relative is the index of what the light
// goes into over the index of what it comes from.
//
// Water against glass is
// 1.333 over 1.5 going in and its inverse coming out. Light past the
// critical angle, where nothing can be transmitted, reflects entirely, at
// any Gloss. Otherwise nonpositive gloss returns zero, and gloss above one
// is clamped to one.
func (material *Material) SplitsBetween(cos, relative float64) float64 {
	share := maths.Fresnel(cos, relative)
	if share >= thingUnity {
		return thingUnity
	}
	gloss := math.Max(material.Gloss, thingNought)
	return math.Min(gloss, thingUnity) * share
}

// Roughness returns local-space texture luminance, or one without a texture.
func (material *Material) Roughness(local maths.Vec) float64 {
	if material.Rough == nil {
		return thingUnity
	}
	return material.Rough.At(local).Luminance()
}

// Attrs controls exceptional visibility and collision behaviour. The zero value
// is visible and solid.
type Attrs struct {
	Hidden bool // not drawn, and casts no shadow
	Ghost  bool // passes through other things
	Clear  bool // drawn, but casts no shadow: a star drawn as a ball
}

// Clear reports whether the object is excluded from shadow casting.
func (thing *Thing) Clear() bool { return thing.Attrs.Clear }

// Motion returns an object's subspace at elapsed time.
type Motion interface {
	At(elapsed time.Duration) maths.Subspace
}

// still is a time-independent transform.
type still maths.Subspace

// At returns the stationary transform for any time.
func (placement still) At(time.Duration) maths.Subspace {
	return maths.Subspace(placement)
}

// Still constructs a stationary motion.
func Still(placement maths.Subspace) Motion { return still(placement) }

// Thing combines local geometry, a material, motion, and visibility flags.
type Thing struct {
	Shape    shape.Shape
	Material *Material
	Motion   Motion // nil is still, at the world's zero
	Attrs    Attrs
}

// Hidden reports whether the object is excluded from rendering and shadows.
func (thing *Thing) Hidden() bool { return thing.Attrs.Hidden }

// At returns the object's transform at time at, or the identity placement when
// no motion is supplied.
func (thing *Thing) At(at time.Duration) maths.Subspace {
	if thing.Motion == nil {
		return maths.Local()
	}
	return thing.Motion.At(at)
}

// Volume combines a density field, material, and motion. Rays integrate through
// it rather than stopping at a surface. Colour or Texture controls scattered
// light; Glow controls emission.
type Volume struct {
	Medium   medium.Medium
	Material *Material
	Motion   Motion // nil is still, at the world's zero
	Attrs    Attrs
	// Beam describes relativistic bulk flow and its directional emission
	// boost. Nil denotes a stationary volume.
	Beam *Beam
}

// Beam describes axial or rotational flow as a fraction of light speed. Axial
// flow travels away from the origin in both directions.
//
// Emission is boosted by the Doppler factor cubed, modelling a moving blob with
// a flat spectrum. A steady jet would use a square instead.
type Beam struct {
	Axis  maths.Vec
	Speed float64
	// Round selects counterclockwise rotation viewed from the positive
	// axis. Speed applies at radius At and falls as the inverse square root
	// of radius farther out; the approaching side appears brighter.
	Round bool
	At    float64
}

// Flow returns local flow direction and speed. The default axis is up. Axial
// flow points away from the origin on the sample's side.
//
// Rotational flow follows the tangent, scales as sqrt(At/radius) when At is
// set, and is capped at 0.99c. Flow on the axis is zero.
func (beam Beam) Flow(at maths.Vec) (direction maths.Vec, speed float64) {
	axis := beam.Axis
	if axis == thingNoDirection {
		axis = maths.Up
	}
	axis = axis.Unit()
	if !beam.Round {
		if at.Dot(axis) < thingNought {
			return axis.Neg(), beam.Speed
		}
		return axis, beam.Speed
	}
	along := at.Dot(axis)
	out := at.Add(axis.Scale(-along))
	radius := out.Len()
	if radius == thingNought {
		return thingNoDirection, thingNought
	}
	speed = beam.Speed
	if beam.At > thingNought {
		ratio := math.Sqrt(beam.At / radius)
		speed = math.Min(thingBeamSpeedCap, speed*ratio)
	}
	return axis.Cross(out.Scale(thingUnity / radius)), speed
}

// thingNoDirection is the vector with no length: the direction of nothing.
var thingNoDirection = maths.V(thingNought, thingNought, thingNought)

// Boost returns the Doppler emission multiplier toward a direction at a
// local-space point. Stationary or invalid-speed flows return one.
func (beam Beam) Boost(toward, at maths.Vec) float64 {
	if beam.Speed <= thingNought || beam.Speed >= thingUnity {
		return thingUnity
	}
	flow, speed := beam.Flow(at)
	if speed == thingNought {
		return thingUnity
	}
	gamma := thingUnity / math.Sqrt(thingUnity-speed*speed)
	aligned := flow.Dot(toward.Unit())
	delta := thingUnity / (gamma * (thingUnity - speed*aligned))
	return delta * delta * delta
}

// Hidden reports whether the volume is excluded from the image.
func (volume *Volume) Hidden() bool { return volume.Attrs.Hidden }

// At returns the volume transform at time at, or identity without motion.
func (volume *Volume) At(at time.Duration) maths.Subspace {
	if volume.Motion == nil {
		return maths.Local()
	}
	return volume.Motion.At(at)
}

// Lamp aliases film.Emitter. The world holds lamps; film defines their
// light-emission contract.
type Lamp = film.Emitter

// OceanSurface identifies an authored finite ocean interface for the
// renderers. It is a record: thing refuses no combination of its fields.
// trace refuses, when a render starts, the combinations it cannot draw, and
// gpu refuses the CPU-only ones. Each field says when it is read.
//
// A zero OceanSurface with only a Boundary is a plain sea. The optional
// source routes and transport approximations below are all off in it, but
// its lamp indexes are zero, which names lamp 0, and not absent.
type OceanSurface struct {
	// Boundary is the exact staged sheet. It is one of the world's Things.
	Boundary *Thing
	// WaveMotion is the staged wave recipe for a sea that moves with the
	// scene clock, and nil for a snapshot sea.
	WaveMotion *OceanWaveMotion
	// Mean is the boundary's origin in world coordinates, the point its
	// horizontal mean plane passes through. The Moon route refuses a
	// boundary whose origin is elsewhere.
	Mean maths.Vec
	// ExtentMetres is the side of the finite square footprint, in metres.
	// A source route and a clear-cloud sky source need it finite and above
	// zero.
	ExtentMetres float64
	// MetresPerUnit converts the world's distance units to metres. A source
	// route needs it finite, above zero and equal to the boundary's
	// ClearMedium.MetresPerUnit.
	MetresPerUnit float64
	// DirectSunMeanPlane selects the direct Sun route, the refracted
	// mean-plane Sun that is a CPU-only approximation. It needs SunRoute
	// Ground or GroundlessAir, and is refused with DirectMoonMeanPlane.
	DirectSunMeanPlane bool
	// DirectMoonMeanPlane selects the direct Moon route, in place of the
	// Sun route. It needs SunRoute Moon and FillLampIndex -1. A flat finite
	// height field or an exact calm infinite plane is drawn by both the CPU
	// and the GPU.
	DirectMoonMeanPlane bool
	// InterfaceEvents is how many interface crossings one ray may follow.
	// Zero is off. Above zero it needs a visible clear boundary that is no
	// metal and has the exact infinite mean-plane continuation as its
	// shape, and at most MaxOceanInterfaceEvents. The GPU refuses it.
	InterfaceEvents int
	// SpotInscatter opts the ocean's medium into in-scattering of spot
	// lamps. It is read where the boundary's ClearMedium scatters, and
	// is refused with wake foam and with the Moon route.
	SpotInscatter bool
	// InfiniteMeanPlane says the boundary is continued past its finite
	// sheet by the exact infinite mean plane. Diffuse-sky in-scattering
	// and a clear-cloud sky source need it, and the boundary's shape must
	// then be that continuation. The GPU refuses it except with the Moon
	// route. OceanWaveConfig carries the same choice for the recipe.
	InfiniteMeanPlane bool
	// ClearCloudSkySamples is how many sky directions a clear-cloud
	// volume's source reads. Zero is off. Above zero it needs a Sky and
	// the infinite continuation, and the Moon route refuses it.
	ClearCloudSkySamples int
	// ClearCloudWaterStepMode is "ray-spacing" to step the water at the
	// ray's own spacing. Empty, "per-gap" or any other word gives the
	// whole water budget to every gap between cloud volumes.
	ClearCloudWaterStepMode string
	// SmoothNormalApproximation is true when ShadingNormalApproximation
	// is the bilinear vertex-slope mode. It is a record of what the stage
	// chose; the renderers shade from the boundary's own height field and
	// read neither.
	SmoothNormalApproximation bool
	// ShadingNormalApproximation names the shading-normal mode the stage
	// chose for the boundary, or is empty for the exact cell normal.
	ShadingNormalApproximation string
	// SunRoute says which staged path supplied the source lamp. It is read
	// only when DirectSunMeanPlane or DirectMoonMeanPlane is set, and
	// OceanSunRouteUnspecified is then refused.
	SunRoute OceanSunRoute
	// SunLampIndex is the place in the world's lamps of the one tagged
	// direct emitter, an emit.Distant that points above the horizon. It is
	// read only with a source route, and a negative index, absent, is then
	// refused. Its zero value is lamp 0.
	SunLampIndex int
	// FillLampIndex is the place of the ordinary-ground fill lamp, which
	// must differ from SunLampIndex on the Ground route and be -1, absent,
	// on the GroundlessAir and Moon routes. Its zero value is lamp 0.
	FillLampIndex int
}

// MaxOceanInterfaceEvents is the upper bound on explicitly authored
// CPU ocean-interface traversal, limiting the branching transport work.
const MaxOceanInterfaceEvents = 16

// MeanPlaneSource reports an explicit directional Sun or Moon source route.
func (ocean *OceanSurface) MeanPlaneSource() bool {
	return ocean != nil &&
		(ocean.DirectSunMeanPlane || ocean.DirectMoonMeanPlane)
}

// OceanWaveConfig is a copied wave recipe for one explicitly timed ocean.
// InfiniteMeanPlane is the recipe's copy of OceanSurface.InfiniteMeanPlane.
type OceanWaveConfig struct {
	WindMetresPerSecond float64
	FetchMetres         float64
	DepthMetres         float64
	TowardRadians       float64
	ExtentMetres        float64
	Samples             int
	Seed                uint64
	SceneClock          bool
	ShadingNormal       string
	FlatMean            bool
	InfiniteMeanPlane   bool
	MeanLevelMetres     float64
	CollarWidthMetres   float64
	SurfaceWake         *OceanSurfaceWakeConfig
}

// OceanSurfaceWakeConfig is a copied authored surface-height wake envelope.
// Its craft path is explicit and independent of any craft's model or
// hydrodynamics.
type OceanSurfaceWakeConfig struct {
	OriginEastMetres, OriginNorthMetres                              float64
	HeadingRadians, SpeedMetresPerSecond                             float64
	AmplitudeMetres, HalfWidthMetres, LengthMetres, WavelengthMetres float64
	AddToSpectralShading                                             bool
}

// OceanWaveMotion keeps its recipe private so scene consumers cannot change
// the staged wave inputs between frames. It is nil for snapshot oceans.
// The recipe, wake included, is copied in and copied out.
type OceanWaveMotion struct{ config OceanWaveConfig }

// NewOceanWaveMotion copies a wave recipe, and the surface wake it points
// to, into immutable scene metadata. A later change to the caller's wake
// does not reach the motion.
func NewOceanWaveMotion(config OceanWaveConfig) *OceanWaveMotion {
	return &OceanWaveMotion{config: config.detached()}
}

// Config returns a copy of the immutable wave recipe, with a copy of its
// surface wake. A change to what it returns does not reach the motion.
func (motion *OceanWaveMotion) Config() OceanWaveConfig {
	if motion == nil {
		return OceanWaveConfig{}
	}
	return motion.config.detached()
}

// detached is the recipe with a wake of its own, a copy of the one it
// points to, so that two holders of recipes never share a wake.
func (config OceanWaveConfig) detached() OceanWaveConfig {
	if config.SurfaceWake != nil {
		wake := *config.SurfaceWake
		config.SurfaceWake = &wake
	}
	return config
}

// OceanSunRoute names the scene path that staged the opt-in Sun or Moon
// approximation. Each route has a distinct, validated lamp contract.
type OceanSunRoute uint8

const (
	// OceanSunRouteUnspecified is no route: a source flag with it is
	// refused.
	OceanSunRouteUnspecified OceanSunRoute = iota
	// OceanSunRouteGround is the Sun over ordinary ground, with a Sun lamp
	// and a distinct fill lamp.
	OceanSunRouteGround
	// OceanSunRouteGroundlessAir is the Sun with no ground, under a staged
	// sky and with no fill lamp.
	OceanSunRouteGroundlessAir
	// OceanSunRouteMoon is the Moon as the source, with no fill lamp.
	OceanSunRouteMoon
)

// Sun constructs a directional solar lamp with apparent diameter 0.53 degrees,
// matching the Sun viewed from Earth.
func Sun(toward maths.Vec, strength maths.Light) Lamp {
	return film.Sun{Toward: toward, Strength: strength,
		Size: 0.53 * maths.Degree}
}

// World assembles surfaces, volumes, fields, lights, sky, and global haze.
type World struct {
	// VolumeIllumination's zero value lights painted media by lamps alone.
	VolumeIllumination VolumeIllumination
	// RayOriginReference's zero value keeps the geometry-derived heuristic.
	RayOriginReference RayOriginReference
	Things             []*Thing
	// OceanSurface is nil unless the set stages an authored ocean sheet.
	OceanSurface *OceanSurface
	Volumes      []*Volume
	Fields       []*Field
	Lamps        []Lamp
	// Sky supplies distant light by direction. Nil returns black for rays
	// that miss all geometry.
	Sky Sky
	// Haze applies global atmospheric attenuation. Its zero value is clear.
	Haze Haze
	// Glow is nil unless a flat layer of air holds the city's own light.
	Glow Glow
	// Pixel optionally fixes the angular texture footprint in radians. Zero
	// derives it from the render. A shot can set this to retain the same
	// texture detail across different output resolutions.
	Pixel float64
	// Filter is where in a pixel its rays start, and so how they are
	// joined, where a picture takes more than one ray a pixel. The zero
	// value, GridFilter, starts each ray at the middle of its own square of
	// the pixel.
	Filter Filter
}

// Filter is where in a pixel its rays start, by film.Place. A pixel is
// the plain mean of its own rays under every one; the filter is taken by
// where the rays start, and is named for the shape they are spread
// under, across the pixel.
type Filter uint8

// The filters. A grid is a regular grid, each ray at the middle of its
// own square of the pixel. The other three spread the rays evenly by a
// sequence.
//
// A box spreads them over the pixel's own square; a trapezoid, flat
// over half the pixel and sloped half a pixel each side, a pixel and a
// half wide; a tent, sloped all the way from its middle, two wide.
const (
	GridFilter Filter = iota
	BoxFilter
	TrapezoidFilter
	TentFilter
)

// The widths of the slope at each side of a filter's shape, in pixels:
// a box has none, a tent is all slope, and a trapezoid is defined here
// as half of each.
const (
	boxRamp       = 0
	trapezoidRamp = 0.5
	tentRamp      = 1
)

// Ramp is how wide the slope at each side of the filter's shape is, in
// pixels, and whether the rays are spread at all: a grid's are not.
func (filter Filter) Ramp() (ramp float64, spread bool) {
	switch filter {
	case BoxFilter:
		return boxRamp, true
	case TrapezoidFilter:
		return trapezoidRamp, true
	case TentFilter:
		return tentRamp, true
	}
	return boxRamp, false
}

// Sky aliases sky.Sky so scene declarations can use one package.
type Sky = sky.Sky

// Haze blends a surface toward a haze colour over distance d using
// exp(-Density*d). It affects camera rays but does not attenuate lamp
// illumination, a deliberate simplification.
type Haze struct {
	// Density is extinction per scene distance unit. Zero is clear.
	Density float64
	// Colour supplies haze colour when FromSky is false.
	Colour maths.Light
	// FromSky uses sky light in the viewing direction as haze colour,
	// blending distant surfaces into the horizon. Without a sky it uses
	// black.
	FromSky bool
}

// NewWorld returns an empty world.
func NewWorld() *World { return &World{} }

// Add appends a surface and returns its stored pointer.
func (world *World) Add(added Thing) *Thing {
	stored := &added
	world.Things = append(world.Things, stored)
	return stored
}

// AddVolume appends a volume and returns its stored pointer.
func (world *World) AddVolume(added Volume) *Volume {
	stored := &added
	world.Volumes = append(world.Volumes, stored)
	return stored
}

// AddField appends a field and returns its stored pointer.
func (world *World) AddField(added Field) *Field {
	stored := &added
	world.Fields = append(world.Fields, stored)
	return stored
}

// Lamp appends an emitter to the world.
func (world *World) Lamp(added Lamp) {
	world.Lamps = append(world.Lamps, added)
}
