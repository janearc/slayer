package shape

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// HitCorners is Hit, and the three corners of the face the ray met, in
// the mesh's own space: what a picture of the mesh's edges needs, and
// nothing a shaded picture reads. The face is the one Hit meets, found
// by the same walk of the same tree.
func (m *Mesh) HitCorners(r maths.Ray, near, far float64) (float64,
	[3]maths.Vec, bool) {
	along, which, ok := m.first(r, near, far)
	if !ok {
		return along, [3]maths.Vec{}, false
	}
	face := &m.tris[which]
	return along, [3]maths.Vec{vec(face.p[0]), vec(face.p[1]),
		vec(face.p[2])}, true
}

// HitCorners is Hit, and the three corners of the face the ray met, in
// the group's own space: the member Hit keeps, asked again for its face
// as HitNormal asks it, and its corners put out of the member's place.
func (group *MeshGroup) HitCorners(ray maths.Ray, near,
	far float64) (float64, [3]maths.Vec, bool) {
	distance, member, hit := group.hit(ray, near, far)
	if !hit {
		return distance, [3]maths.Vec{}, false
	}
	scale := member.Space.Scale
	if scale == 0 {
		scale = 1
	}
	local := member.Space.RayIn(ray)
	corners, found := memberCorners(member.Mesh, local, near/scale,
		far/scale)
	for i, corner := range corners {
		corners[i] = member.Space.Out(corner)
	}
	return distance, corners, found
}

// memberCorners is the corners of the face a ray meets on one member of
// a group, in the member's own space, and whether it meets one. A group
// holds slayer's mesh and this package's, and nothing else.
func memberCorners(solid Shape, ray maths.Ray, near,
	far float64) ([3]maths.Vec, bool) {
	if skin, isMesh := solid.(*mesh.Mesh); isMesh {
		met, found := skin.Meet(ray, near, far)
		if !found {
			return [3]maths.Vec{}, false
		}
		one, two, three := skin.Corners(met.Face())
		return [3]maths.Vec{one, two, three}, true
	}
	_, corners, found := solid.(*Mesh).HitCorners(ray, near, far)
	return corners, found
}
