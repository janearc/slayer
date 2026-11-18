package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// face is one triangle: three corners turning counterclockwise seen
// from outside, which is slayer's mesh's rule and so the rule here.
type face [3]maths.Vec

// surface is the faces of a skin, in some space. a form's skin is made
// in the form's own space, where up is y as in the world, and placed after.
type surface []face

// sineLeast is the least sine of the angle between a face's two edges
// at which it has an area to face a way by. below it slayer's mesh takes
// the corners to be in a line and refuses the face. read: scene/shape/mesh.go,
// check, where the cutoff is 1e-12.
const sineLeast = 1e-12

// hasArea is whether the face's corners are not in a line, nor within
// sineLeast of one: the cross of its edges, squared, is above the sine
// squared times the edges' squares. a repeated corner has none.
func (one face) hasArea() bool {
	first, second := one[1].Sub(one[0]), one[2].Sub(one[0])
	across := first.Cross(second)
	return across.Dot(across) >
		sineLeast*sineLeast*first.Dot(first)*second.Dot(second)
}

// withArea is the surface without its faces that have no area: they cover
// nothing, so leaving them out moves no other face.
func (skin surface) withArea() surface {
	kept := make(surface, 0, len(skin))
	for _, one := range skin {
		if one.hasArea() {
			kept = append(kept, one)
		}
	}
	return kept
}

// placed is the surface put in a space of its parent's.
func (skin surface) placed(space maths.Subspace) surface {
	out := make(surface, len(skin))
	for index, one := range skin {
		for corner, point := range one {
			out[index][corner] = space.Out(point)
		}
	}
	return out
}

// meshed is the surface as slayer's mesh, with flat faces, three
// corners of its own to each. the mesh finds the edges faces share by
// where their corners sit.
//
// it refuses a face with no area, neighbours wound against each other,
// and a closed piece wound inside out, so a mesh that comes back is a
// surface a scene can hold.
func (skin surface) meshed() (*mesh.Mesh, error) {
	corners, faces := skin.cornered()
	return mesh.New(corners, faces, nil, false)
}

// shaded is the surface as slayer's mesh shaded smooth by the given
// normals, three to a face in the order of its corners; given none, it
// is meshed, with flat faces. The mesh refuses what meshed refuses, and
// a normal that points nowhere.
func (skin surface) shaded(normals [][3]maths.Vec) (*mesh.Mesh, error) {
	if normals == nil {
		return skin.meshed()
	}
	corners, faces := skin.cornered()
	each := make([]maths.Vec, 0, len(corners))
	for _, three := range normals {
		each = append(each, three[:]...)
	}
	return mesh.New(corners, faces, each, true)
}

// cornered is the surface's corners, three of its own to each face, and
// its faces as three places among them.
func (skin surface) cornered() ([]maths.Vec, [][3]int) {
	corners := make([]maths.Vec, 0, len(skin)*len(face{}))
	faces := make([][3]int, 0, len(skin))
	for _, one := range skin {
		first := len(corners)
		corners = append(corners, one[0], one[1], one[2])
		faces = append(faces, [3]int{first, first + 1, first + 2})
	}
	return corners, faces
}
