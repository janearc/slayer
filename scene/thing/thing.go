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
	n := m.Index
	if n <= 0 {
		n = 1.5
	}
	f0 := (n - 1) / (n + 1)
	f0 *= f0
	away := 1 - math.Max(0, math.Min(1, cos))
	return m.Gloss * (f0 + (1-f0)*away*away*away*away*away)
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

// Sun constructs a directional solar lamp with apparent diameter 0.53 degrees,
// matching the Sun viewed from Earth.
func Sun(toward maths.Vec, strength maths.Light) Lamp {
	return film.Sun{Toward: toward, Strength: strength,
		Size: 0.53 * maths.Degree}
}

// World assembles surfaces, volumes, fields, lights, sky, and global haze.
type World struct {
	// RayOriginReference's zero value keeps the geometry-derived heuristic.
	RayOriginReference RayOriginReference
	Things             []*Thing
	Volumes            []*Volume
	Fields             []*Field
	Lamps              []Lamp
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
