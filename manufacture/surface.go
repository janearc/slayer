package manufacture

import (
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// face is one triangle: three corners turning counterclockwise seen
// from outside, which is slayer's mesh's rule and so the rule here.
type face [3]maths.Vec

// surface is the faces of a skin, in some space. a form's skin is made
// in the form's own space, where up is y as in the world, and placed after.
type surface []face

// hasArea rejects repeated or nearly collinear triangle corners.
func (one face) hasArea() bool { return geometry.HasArea(one) }

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
	return geometry.Placed(skin, space)
}

// meshed is the surface as slayer's mesh, with flat faces, three
// corners of its own to each. the mesh finds the edges faces share by
// where their corners sit.
//
// it refuses a face with no area, neighbours wound against each other,
// and a closed piece wound inside out, so a mesh that comes back is a
// surface a scene can hold.
func (skin surface) meshed() (*mesh.Mesh, error) { return geometry.FaceMesh(skin, nil) }

// shaded converts the surface with optional corner normals.
func (skin surface) shaded(normals [][3]maths.Vec) (*mesh.Mesh, error) {
	return geometry.FaceMesh(skin, normals)
}
