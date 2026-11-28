package mesh

import (
	"fmt"
	"slices"
	"sort"

	"github.com/janearc/slayer/maths"
)

// FibreField holds a centreline point and root-to-tip tangent at each
// vertex. Faces of a shaft occupy a contiguous range, recorded by its
// exclusive end. Adjacent shafts may have different tessellations.
type FibreField struct {
	Centres, Tangents []maths.Vec
	FaceEnds          []int
}

// NewWithFibres makes a closed mesh whose separate shafts retain frames
// for near-field optics. The vertices, frames and face ranges are copied.
func NewWithFibres(corners, rest []maths.Vec, faces [][3]int, normals []maths.Vec,
	field FibreField,
) (*Mesh, error) {
	if len(field.Centres) != len(corners) || len(field.Tangents) != len(corners) {
		return nil, fmt.Errorf("%w: fibre frame count differs from corner count", ErrMesh)
	}
	for i, centre := range field.Centres {
		length := field.Tangents[i].Len()
		if !finite(centre) || !(length > .99 && length < 1.01) {
			return nil, fmt.Errorf("%w: invalid fibre frame at corner %d", ErrMesh, i)
		}
	}
	start := 0
	for _, end := range field.FaceEnds {
		if end <= start || end > len(faces) {
			return nil, fmt.Errorf("%w: invalid fibre face range", ErrMesh)
		}
		start = end
	}
	if start != len(faces) {
		return nil, fmt.Errorf("%w: fibre ranges must cover all faces", ErrMesh)
	}
	made, err := NewWithRestPoints(corners, rest, faces, normals, true)
	if err != nil {
		return nil, err
	}
	if !made.closed {
		return nil, fmt.Errorf("%w: fibre shafts require closed boundaries", ErrMesh)
	}
	made.fibres = &FibreField{
		Centres: slices.Clone(field.Centres), Tangents: slices.Clone(field.Tangents),
		FaceEnds: slices.Clone(field.FaceEnds),
	}
	return made, nil
}

// HasFibres reports whether the mesh has shaft-local optical frames.
func (made *Mesh) HasFibres() bool { return made.fibres != nil }

// FibreHit gives the local frame and geometric boundary at a strand hit.
type FibreHit struct {
	Point, Tangent, Radial, Normal maths.Vec
	Shaft                          int
}

// FibreAt recovers the cylindrical frame where a ray met a shaft.
func (made *Mesh) FibreAt(met Met) (FibreHit, bool) {
	if !made.HasFibres() || met.face < 0 || met.face >= len(made.faces) {
		return FibreHit{}, false
	}
	point, _ := made.VertexPointAt(met, made.points)
	centre, _ := made.VertexPointAt(met, made.fibres.Centres)
	along, _ := made.VertexPointAt(met, made.fibres.Tangents)
	tangent := along.Unit()
	reach := point.Sub(centre)
	radial := reach.Sub(tangent.Scale(tangent.Dot(reach))).Unit()
	normal := made.fibreFaceNormal(met.face)
	if radial == (maths.Vec{}) {
		radial = normal.Sub(tangent.Scale(tangent.Dot(normal))).Unit()
	}
	if radial == (maths.Vec{}) || tangent == (maths.Vec{}) {
		return FibreHit{}, false
	}
	return FibreHit{point, tangent, radial, normal, made.shaft(met.face)}, true
}

func (made *Mesh) shaft(face int) int {
	return sort.Search(len(made.fibres.FaceEnds), func(shaft int) bool {
		return face < made.fibres.FaceEnds[shaft]
	})
}

func (made *Mesh) fibreFaceNormal(face int) maths.Vec {
	a, b, c := made.Corners(face)
	return b.Sub(a).Cross(c.Sub(a)).Unit()
}

// FibreBlocks reports occlusion by this mesh along a source ray. A ray
// starting on the inside of its originating shaft may consume one exit
// through that shaft, since the scattering event accounts for that
// transmission. Earlier neighbouring hits and any later re-entry block.
func (made *Mesh) FibreBlocks(ray maths.Ray, far float64, shaft int, inside bool) bool {
	hit, found := made.Meet(ray, 0, far)
	if !found || !made.HasFibres() || !inside {
		return found
	}
	if made.shaft(hit.face) != shaft || made.fibreFaceNormal(hit.face).Dot(ray.Dir) <= 0 {
		return true
	}
	_, again := made.Meet(ray, hit.along, far)
	return again
}
