package trace

// The normal a clear surface of water is shaded, mirrored and bent by:
// the waves' own continuous slope, as wide a pixel as sees it can show,
// made to face the ray from the side it met the surface on.

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// seenWater is a surface whose normal depends on how wide a pixel sees
// it, arid.Sea. Seen is its normal at a point of its own space for a ray
// from from that met it there, turned to the side the ray came from, and
// the slope variance of the waves a pixel width wide cannot show.
type seenWater interface {
	Seen(point, from maths.Vec, width float64) (maths.Vec, float64)
}

// roughWater is a sea that says the slope variance of the waves shorter
// than it holds, both ways together: what its mirror is roughened by
// beside the waves a pixel is too wide to show.
type roughWater interface {
	Unseen() float64
}

// clearedWater is a sea whose search takes a ray within a touch of its
// surface as met: Clearance, in its own units, is how far a look leaving
// it starts from the surface so as not to meet it again at once.
type clearedWater interface {
	Clearance() float64
}

// frontLean is how far a fronted normal faces the ray, as the cosine
// between them: picked, as small as keeps a mirror about the normal off
// the surface, so the tilt it takes does not show.
const frontLean = 1e-3

// shadingNormal is the normal a hit is shaded, mirrored and bent by, the
// side of the surface the ray met it on, and the slope variance a pixel
// width wide there, in the thing's own units, cannot show.
//
// Water (seenWater) gives its own. A clear surface whose geometry has a
// normal apart from its shading one, a height field, takes the shading
// normal fronted to the geometric side. Anything else is as it was, with
// no side apart from its normal: side is then nought, and the caller
// takes its final normal for it.
func (met hit) shadingNormal(ray maths.Ray, point maths.Vec,
	width float64) (normal, side maths.Vec, lost float64) {
	placed, dir := met.thing, ray.Dir
	if water, is := placed.thing.Shape.(seenWater); is {
		faces, lost := water.Seen(placed.at.In(point),
			placed.at.In(ray.At), width)
		side = placed.at.Orientation.Out(faces).Unit()
		return fronted(side, side, dir), side, lost
	}
	normal = met.normal(point, dir)
	material := placed.thing.Material
	if material == nil || !(material.Through > 0) {
		return normal, maths.Vec{}, 0
	}
	if _, has := placed.thing.Shape.(shape.GeometricNormaler); !has {
		return normal, maths.Vec{}, 0
	}
	side = met.boundaryNormal(point, dir)
	return fronted(normal, side, dir), side, 0
}

// fronted is a shading normal made to face a ray along dir from side, the
// way out of the surface on the ray's side: turned to side's half where
// it points to the other, then, where it still leans away from the ray,
// tilted toward it until the ray meets it at frontLean. A mirror about it
// then leaves the surface toward the ray's side, and a bend crosses it.
//
// The normal is a unit and so is the way, a the cosine between them. Take
// t of the way off the normal: what is left is 1 - 2ta + t^2 long and
// meets the way at a - t, so the unit normal's cosine is that over the
// root of that length. Set it to minus frontLean and let s be t - a:
// s^2 = frontLean^2 (1 - a^2 + s^2), so
// s = frontLean * root((1 - a^2) / (1 - frontLean^2)), and t = a + s.
//
// Two unit vectors can have a dot a hair over one by rounding, and 1 - a^2
// is then below nothing, where the root would be of less than nothing. A
// ray along the normal, to the float, leaves no plane of the two to take
// the way off in: see acrossTheWay.
func fronted(normal, side, dir maths.Vec) maths.Vec {
	if normal.Dot(side) < 0 {
		normal = normal.Neg()
	}
	way := dir.Unit()
	along := normal.Dot(way)
	if along <= -frontLean {
		return normal
	}
	spare := 1 - along*along
	if !(spare > 0) {
		return acrossTheWay(normal, way)
	}
	past := frontLean * math.Sqrt(spare/(1-frontLean*frontLean))
	return normal.Sub(way.Scale(along + past)).Unit()
}

// acrossTheWay is the unit normal fronted gives where the ray runs along
// the normal, and there is no plane of the two to tilt in: one across the
// way, tilted so that the way meets it at minus frontLean.
func acrossTheWay(normal, way maths.Vec) maths.Vec {
	first, _ := across(normal)
	return first.Scale(math.Sqrt(1 - frontLean*frontLean)).
		Sub(way.Scale(frontLean))
}

// lastMirror is the light a water surface mirrors toward the eye when the
// look has used its bounces: the sky along the mirrored way, by fresnel's
// share, met by nothing on the way. A look mirrored off one wave into the
// next would otherwise see the second as clear water's own colour, none,
// and draw it black. It is nought for anything but water (watery).
//
// Approximation: the second mirror is not followed through the scene, so
// a wave, a hull or a headland it would meet does not hide that sky.
func (scene *staged) lastMirror(met hit, dir, normal maths.Vec,
	reflects maths.Light) maths.Light {
	if !met.watery() || reflects == (maths.Light{}) {
		return maths.Light{}
	}
	mirrored := dir.Sub(normal.Scale(2 * dir.Dot(normal)))
	return scene.skyAt(mirrored).Times(reflects)
}

// watery is whether a hit is on water: the sea (seenWater), or a clear
// surface whose geometry has a normal apart from its shading one.
func (met hit) watery() bool {
	shaped := met.thing.thing
	if _, water := shaped.Shape.(seenWater); water {
		return true
	}
	_, field := shaped.Shape.(shape.GeometricNormaler)
	return field && shaped.Material != nil && shaped.Material.Through > 0
}

// roughened is material with lost added to its slope variance, for its
// highlight only, as a copy; or material itself where nothing is lost. The
// copy is never the material a look is inside.
func roughened(material *thing.Material, lost float64) *thing.Material {
	if !(lost > 0) {
		return material
	}
	own := *material
	own.SlopeVariance += lost
	return &own
}
