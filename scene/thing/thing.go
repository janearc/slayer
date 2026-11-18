// Package thing describes scene objects, materials, motion, volumes, and
// lights. A World assembles them for the renderer.
//
// Nested transforms place objects relative to their parents. Nested, Within,
// Spin, Growing, and Orbit compose those placements over time, using
// maths.Subspace for coordinate transformations.
package thing

import (
	"math"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/texture"
)

// Material describes surface response. Instances are shared: modifying one
// changes every object referencing it. Do not mutate it during rendering.
type Material struct {
	Name string
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
	// Index is refractive index. Zero selects 1.5, the glass default.
	// Reflection increases toward grazing incidence.
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
	// Splits, the unpolarised Fresnel share for its index and the side
	// the ray arrives from, and not by Mirrors' Schlick. A metal
	// reflects by Reflects whatever its Through.
	//
	// A partial reflection on a ray that has already bent is paid and
	// not followed. Existing trace entry points start in the air; the
	// explicit trace.WithInitialMedia entry points can seed a camera's
	// clear-medium state when the caller supplies it.
	Through float64
	// ClearMedium names this boundary's homogeneous interior for the CPU
	// tracer's unscattered Beer-Lambert transport. Its zero value preserves
	// the historical unattenuated clear-surface behavior.
	ClearMedium *ClearMedium
	// Metal, when it is given, makes the surface a conductor: it mirrors
	// by its complex index, a different share of each colour, scaled by
	// Gloss, and Index no longer says how much it mirrors. Film, when it
	// is given too, is a clear layer over the metal. See Reflects.
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

// Valid reports whether this authored response is finite and bounded.
func (foam WakeFoamResponse) Valid() bool {
	if foam.Coverage == 0 {
		return true
	}
	finite := func(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
	if foam.FrozenTime && (!finite(foam.Seconds) || foam.Seconds < 0) {
		return false
	}
	return finite(foam.OriginEast) && finite(foam.OriginNorth) &&
		finite(foam.Heading) && finite(foam.Speed) && foam.Speed >= 0 &&
		finite(foam.HalfWidth) && foam.HalfWidth > 0 &&
		finite(foam.Length) && foam.Length > 0 &&
		finite(foam.Wavelength) && foam.Wavelength > 0 &&
		foam.Wavelength <= foam.Length &&
		finite(foam.Coverage) && foam.Coverage > 0 && foam.Coverage <= 1 &&
		finite(foam.Band) && foam.Band > 0 && foam.Band <= 1 &&
		foam.Reflectance.R >= 0 && foam.Reflectance.R <= 1 &&
		foam.Reflectance.G >= 0 && foam.Reflectance.G <= 1 &&
		foam.Reflectance.B >= 0 && foam.Reflectance.B <= 1 &&
		finite(foam.Satin) && foam.Satin > 0 && foam.Satin <= 1
}

// At returns bounded foam coverage at a local east/up/south point and scene
// time. The breakup bands derive their period from the authored wake wavelength.
func (foam WakeFoamResponse) At(local maths.Vec, seconds float64) float64 {
	if foam.FrozenTime {
		seconds = foam.Seconds
	}
	if foam.Coverage <= 0 || !foam.Valid() || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 0 {
		return 0
	}
	forwardEast, forwardNorth := math.Sin(foam.Heading), math.Cos(foam.Heading)
	centerEast := foam.OriginEast + forwardEast*foam.Speed*seconds
	centerNorth := foam.OriginNorth + forwardNorth*foam.Speed*seconds
	deltaEast, deltaNorth := local.X-centerEast, -(local.Z)-centerNorth
	aft := -deltaEast*forwardEast - deltaNorth*forwardNorth
	cross := deltaEast*forwardNorth - deltaNorth*forwardEast
	if aft <= 0 || aft >= foam.Length || foam.Coverage <= 0 {
		return 0
	}
	longitudinal := math.Sin(math.Pi * aft / foam.Length)
	width := 2 * foam.HalfWidth
	if math.Abs(cross) >= width {
		return 0
	}
	// Smoothly fade the mask to zero at the authored support edge.
	edge := 1 - math.Abs(cross)/width
	edge *= edge
	lateral := math.Exp(-0.5*math.Pow(cross/foam.HalfWidth, 2)) * edge
	// One coherent-noise octave breaks the patch at the authored scale.
	// The affine map takes its signed base band into coverage coordinates.
	pattern := 0.5 + 0.5*noise.Fractal(uint64(foam.Seed),
		aft/foam.Wavelength, cross/foam.Wavelength, 0, 1)
	// The signed noise's zero contour separates patches; Band softens its edge.
	threshold := 0.5 - foam.Band/2
	breakup := math.Max(0, math.Min(1, (pattern-threshold)/foam.Band))
	return math.Max(0, math.Min(foam.Coverage,
		foam.Coverage*longitudinal*lateral*breakup))
}

// Dense is the refractive index a surface reflects and bends by:
// Index, or glass's 1.5 where none is given.
func (m *Material) Dense() float64 {
	if m.Index <= 0 {
		return 1.5
	}
	return m.Index
}

// Mirrors returns Schlick Fresnel reflectance at incidence cosine cos, scaled
// by Gloss. Nonpositive gloss returns zero; values above one are clamped so
// reflected light cannot exceed incident light.
func (m *Material) Mirrors(cos float64) float64 {
	if m.Gloss <= 0 {
		return 0
	}
	if m.Gloss > 1 {
		return (&Material{Gloss: 1, Index: m.Index}).Mirrors(cos)
	}
	n := m.Dense()
	f0 := (n - 1) / (n + 1)
	f0 *= f0
	away := 1 - math.Max(0, math.Min(1, cos))
	return m.Gloss * (f0 + (1-f0)*away*away*away*away*away)
}

// Splits returns the share a clear surface reflects, by Fresnel, by side:
// the light arriving at incidence cosine cos that a surface which
// transmits reflects, scaled by Gloss: the unpolarised
// Fresnel reflectance of the boundary between air and a thing of index
// Dense. Within says the light arrives from inside the thing; otherwise it
// arrives from the air.
//
// Light past the critical angle, where nothing can be transmitted,
// reflects entirely, at any Gloss. Otherwise nonpositive gloss returns
// zero, and gloss above one is clamped to one.
func (m *Material) Splits(cos float64, within bool) float64 {
	n := m.Dense()
	if within {
		n = 1 / n
	}
	return m.SplitsBetween(cos, n)
}

// SplitsBetween is Splits for a boundary that need not have air on
// either side: relative is the index of what the light goes into over
// the index of what it comes from. Water against glass is 1.333 over
// 1.5 going in and its inverse coming out. As Splits, light past the
// critical angle reflects entirely, at any Gloss.
func (m *Material) SplitsBetween(cos, relative float64) float64 {
	share := maths.Fresnel(cos, relative)
	if share >= 1 {
		return 1
	}
	return math.Min(math.Max(m.Gloss, 0), 1) * share
}

// Roughness returns local-space texture luminance, or one without a texture.
func (m *Material) Roughness(local maths.Vec) float64 {
	if m.Rough == nil {
		return 1
	}
	return m.Rough.At(local).Luminance()
}

// Attrs controls exceptional visibility and collision behaviour. The zero value
// is visible and solid.
type Attrs struct {
	Hidden bool // not drawn, and casts no shadow
	Ghost  bool // passes through other things
	Clear  bool // drawn, but casts no shadow: a star drawn as a ball
}

// Clear reports whether the object is excluded from shadow casting.
func (t *Thing) Clear() bool { return t.Attrs.Clear }

// Motion returns an object's subspace at elapsed time t.
type Motion interface {
	At(t time.Duration) maths.Subspace
}

// still is a time-independent transform.
type still maths.Subspace

// At returns the stationary transform for any time.
func (s still) At(time.Duration) maths.Subspace { return maths.Subspace(s) }

// Still constructs a stationary motion.
func Still(s maths.Subspace) Motion { return still(s) }

// Thing combines local geometry, a material, motion, and visibility flags.
type Thing struct {
	Shape    shape.Shape
	Material *Material
	Motion   Motion // nil is still, at the world's zero
	Attrs    Attrs
}

// Hidden reports whether the object is excluded from rendering and shadows.
func (t *Thing) Hidden() bool { return t.Attrs.Hidden }

// Ghost reports whether the object passes through other objects.
func (t *Thing) Ghost() bool { return t.Attrs.Ghost }

// At returns the object's transform at time at, or the identity placement when
// no motion is supplied.
func (t *Thing) At(at time.Duration) maths.Subspace {
	if t.Motion == nil {
		return maths.Subspace{}
	}
	return t.Motion.At(at)
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
func (b Beam) Flow(at maths.Vec) (dir maths.Vec, speed float64) {
	axis := b.Axis
	if axis == (maths.Vec{}) {
		axis = maths.Up
	}
	axis = axis.Unit()
	if !b.Round {
		if at.Dot(axis) < 0 {
			return axis.Neg(), b.Speed
		}
		return axis, b.Speed
	}
	out := at.Add(axis.Scale(-at.Dot(axis)))
	r := out.Len()
	if r == 0 {
		return maths.Vec{}, 0
	}
	speed = b.Speed
	if b.At > 0 {
		speed = math.Min(0.99, speed*math.Sqrt(b.At/r))
	}
	return axis.Cross(out.Scale(1 / r)), speed
}

// Boost returns the Doppler emission multiplier toward a direction at a
// local-space point. Stationary or invalid-speed flows return one.
func (b Beam) Boost(toward, at maths.Vec) float64 {
	if b.Speed <= 0 || b.Speed >= 1 {
		return 1
	}
	flow, speed := b.Flow(at)
	if speed == 0 {
		return 1
	}
	gamma := 1 / math.Sqrt(1-speed*speed)
	delta := 1 / (gamma * (1 - speed*flow.Dot(toward.Unit())))
	return delta * delta * delta
}

// Hidden reports whether the volume is excluded from the image.
func (v *Volume) Hidden() bool { return v.Attrs.Hidden }

// At returns the volume transform at time at, or identity without motion.
func (v *Volume) At(at time.Duration) maths.Subspace {
	if v.Motion == nil {
		return maths.Subspace{}
	}
	return v.Motion.At(at)
}

// Lamp aliases film.Emitter. The world holds lamps; film defines their
// light-emission contract.
type Lamp = film.Emitter

// OceanSurface identifies an authored finite ocean interface for CPU
// transport. Boundary is the exact staged sheet, Mean is its horizontal mean
// plane in world coordinates, and ExtentMetres/MetresPerUnit describe its
// finite square footprint.
//
// DirectSunMeanPlane and DirectMoonMeanPlane select mutually exclusive,
// explicitly authored directional source routes for the CPU-only source-path
// approximation. SunRoute records which validated hillside insertion contract
// supplied the light.
//
// SunLampIndex identifies the one tagged direct emitter; FillLampIndex
// identifies the ordinary-ground legacy fill, or is -1 for the groundless-air
// route. Negative indexes mean absent. SmoothNormalApproximation records the
// selected CPU-only bilinear vertex-slope shading normal.
// ShadingNormalApproximation names any explicit CPU-only shading-normal mode
// on the ocean.
type OceanSurface struct {
	Boundary             *Thing
	WaveMotion           *OceanWaveMotion
	Mean                 maths.Vec
	ExtentMetres         float64
	MetresPerUnit        float64
	DirectSunMeanPlane   bool
	DirectMoonMeanPlane  bool
	InterfaceEvents      int
	SpotInscatter        bool
	InfiniteMeanPlane    bool
	ClearCloudSkySamples int
	// ClearCloudWaterStepMode is per-gap (legacy) or explicit ray-spacing.
	ClearCloudWaterStepMode    string
	SmoothNormalApproximation  bool
	ShadingNormalApproximation string
	SunRoute                   OceanSunRoute
	SunLampIndex               int
	FillLampIndex              int
}

// MaxOceanInterfaceEvents is the picked upper bound on explicitly authored
// CPU ocean-interface traversal, limiting the branching transport work.
const MaxOceanInterfaceEvents = 16

// MeanPlaneSource reports an explicit directional Sun or Moon source route.
func (ocean *OceanSurface) MeanPlaneSource() bool {
	return ocean != nil && (ocean.DirectSunMeanPlane || ocean.DirectMoonMeanPlane)
}

// OceanWaveConfig is a copied wave recipe for one explicitly timed ocean.
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
// Its craft path is explicit and independent of Alfa's model or hydrodynamics.
type OceanSurfaceWakeConfig struct {
	OriginEastMetres, OriginNorthMetres                              float64
	HeadingRadians, SpeedMetresPerSecond                             float64
	AmplitudeMetres, HalfWidthMetres, LengthMetres, WavelengthMetres float64
	AddToSpectralShading                                             bool
}

// OceanWaveMotion keeps its recipe private so scene consumers cannot change
// the staged wave inputs between frames. It is nil for snapshot oceans.
type OceanWaveMotion struct{ config OceanWaveConfig }

// NewOceanWaveMotion copies a wave recipe into immutable scene metadata.
func NewOceanWaveMotion(config OceanWaveConfig) *OceanWaveMotion {
	return &OceanWaveMotion{config: config}
}

// Config returns a copy of the immutable wave recipe.
func (motion *OceanWaveMotion) Config() OceanWaveConfig {
	if motion == nil {
		return OceanWaveConfig{}
	}
	return motion.config
}

// OceanSunRoute names the scene path that staged the opt-in Sun or Moon
// approximation. Each route has a distinct, validated lamp contract.
type OceanSunRoute uint8

const (
	OceanSunRouteUnspecified OceanSunRoute = iota
	OceanSunRouteGround
	OceanSunRouteGroundlessAir
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
	// VolumeIllumination's zero value keeps legacy lamp-only painted media.
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
	// Pixel optionally fixes the angular texture footprint in radians. Zero
	// derives it from the render. A shot can set this to retain the same
	// texture detail across different output resolutions.
	Pixel float64
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
func (w *World) Add(t Thing) *Thing {
	p := &t
	w.Things = append(w.Things, p)
	return p
}

// AddVolume appends a volume and returns its stored pointer.
func (w *World) AddVolume(v Volume) *Volume {
	p := &v
	w.Volumes = append(w.Volumes, p)
	return p
}

// AddField appends a field and returns its stored pointer.
func (w *World) AddField(f Field) *Field {
	p := &f
	w.Fields = append(w.Fields, p)
	return p
}

// Lamp appends an emitter to the world.
func (w *World) Lamp(l Lamp) { w.Lamps = append(w.Lamps, l) }
