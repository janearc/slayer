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
	corners := make([]maths.Vec, 0, len(skin)*len(face{}))
	faces := make([][3]int, 0, len(skin))
	for _, one := range skin {
		first := len(corners)
		corners = append(corners, one[0], one[1], one[2])
		faces = append(faces, [3]int{first, first + 1, first + 2})
	}
	return mesh.New(corners, faces, nil, false)
}
