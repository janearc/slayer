package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// roundProfile is a turned profile whose sides stand for a round wall,
// as a telescope's tube is round. Its faces are the turned profile's;
// each corner carries the normal of the round wall there. Shaded by
// those, a polished wall mirrors as one round thing, not as many flat
// strips each mirroring its own piece of the world at its own angle.
type roundProfile struct{ turnedProfile }

// rounded is a form whose flat faces stand for a surface curved round
// the y axis, and which says that surface's normal at each corner of a
// skin it made, face by face, in the form's own space.
type rounded interface {
	form
	normals(skin surface) [][3]maths.Vec
}

// roundDish is an unsegmented mirror dish whose rings and strips stand
// for its paraboloid: its faces are the dish's, and the corners of its
// top carry the paraboloid's normal, so it mirrors as one curved mirror
// and not as rings of flat strips. Its back and rim turn as a round
// profile's do.
type roundDish struct{ mirrorDish }

// normals is the paraboloid's normal at each corner of the dish's top,
// a face that looks more up than aside, and the round wall's at every
// other corner: the back looks down and the rim aside, give or take
// rounding, and a dish of a spacecraft's depth tilts its top by degrees.
func (shape roundDish) normals(skin surface) [][3]maths.Vec {
	normals := aroundAxis(skin)
	slope := twice * shape.depth / (shape.radius * shape.radius)
	for index, one := range skin {
		flat := one.normal()
		if flat.Y <= math.Hypot(flat.X, flat.Z) {
			continue
		}
		for corner, point := range one {
			normals[index][corner] = maths.V(-slope*point.X, spaceOne,
				-slope*point.Z).Unit()
		}
	}
	return normals
}

// twice is the power of a paraboloid's rise in its radius, which its
// slope takes as a factor: depth times radius squared, over its own.
const twice = 2 // defined: d/dr of r squared

// onAxis is the distance from the y axis of a corner standing on it.
const onAxis = 0 // defined: no distance at all

// normals is the round wall's normal at each corner of the skin.
func (shape roundProfile) normals(skin surface) [][3]maths.Vec {
	return aroundAxis(skin)
}

// aroundAxis is, for each face of a skin turned about the y axis, the
// normal at each corner of the round surface the faces stand for: the
// face's own normal turned about the axis to the corner. It leans from
// the axis as the face's own does and keeps the face's rise along it.
func aroundAxis(skin surface) [][3]maths.Vec {
	normals := make([][3]maths.Vec, len(skin))
	for index, one := range skin {
		flat := one.normal()
		middle := one[quadFirstCorner].Add(one[quadSecondCorner]).
			Add(one[quadThirdCorner])
		out := math.Hypot(flat.X, flat.Z)
		if flat.X*middle.X+flat.Z*middle.Z < onAxis {
			out = -out
		}
		for corner, point := range one {
			normals[index][corner] = turnedTo(flat, out, point)
		}
	}
	return normals
}

// turnedTo is a face's normal at one of its corners: out from the axis
// by out and along it by the face's own rise, turned round the axis to
// the corner. A corner on the axis keeps the face's own normal, since
// the faces that meet at a point there say no one way round.
func turnedTo(flat maths.Vec, out float64, point maths.Vec) maths.Vec {
	across := math.Hypot(point.X, point.Z)
	if across == onAxis {
		return flat
	}
	return maths.V(out*point.X/across, flat.Y, out*point.Z/across)
}

// normal is the face's own way out, one long, by its winding.
func (one face) normal() maths.Vec {
	first := one[quadFirstCorner]
	return one[quadSecondCorner].Sub(first).
		Cross(one[quadThirdCorner].Sub(first)).Unit()
}

// flatly is each face's own normal at all three of its corners, which
// shades the face flat inside a mesh that is shaded smooth.
func flatly(skin surface) [][3]maths.Vec {
	normals := make([][3]maths.Vec, len(skin))
	for index, one := range skin {
		flat := one.normal()
		normals[index] = [3]maths.Vec{flat, flat, flat}
	}
	return normals
}

// skinned is the part's surface in the thing's space and, for a rounded
// form, the normal at each corner of each of its faces, turned with the
// part. A form that is not rounded has none, and is shaded flat.
func (one part) skinned() (surface, [][3]maths.Vec) {
	own := one.shape.skin(one.fine)
	skin := own.placed(one.space)
	round, isRound := one.shape.(rounded)
	if !isRound {
		return skin, nil
	}
	normals := round.normals(own)
	for index := range normals {
		for corner, normal := range normals[index] {
			normals[index][corner] = one.space.Orientation.Out(normal)
		}
	}
	return skin, normals
}

// roundSlots is the matters with a rounded part in them, which are shaded
// smooth whole.
func (made good) roundSlots() map[slot]bool {
	round := map[slot]bool{}
	for _, one := range made.parts {
		_, isRound := one.shape.(rounded)
		round[one.of] = round[one.of] || isRound
	}
	return round
}

// shadings is, for each matter with a rounded part in it, the normal at
// each corner of each face of the matter's skin, in the order skins
// gives the faces. A matter with no rounded part is not in it, and its
// mesh is shaded flat as it always was.
func (made good) shadings() map[slot][][3]maths.Vec {
	round := made.roundSlots()
	shadings := map[slot][][3]maths.Vec{}
	for _, one := range made.parts {
		if !round[one.of] {
			continue
		}
		skin, normals := one.skinned()
		if normals == nil {
			normals = flatly(skin)
		}
		shadings[one.of] = append(shadings[one.of], normals...)
	}
	return shadings
}
