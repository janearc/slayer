package oakland

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// affine is a private column-vector transform at the glTF boundary.
type affine struct {
	linear [9]float64
	offset maths.Vec
}

type geometryKey struct {
	primitive int
	linear    [9]uint64
}

// affineIdentity leaves points and directions unchanged.
func affineIdentity() affine {
	return affine{linear: [9]float64{1, 0, 0, 0, 1, 0, 0, 0, 1}}
}

// nodeAffine composes a checked glTF node's scale, rotation and translation.
func nodeAffine(node nodeDef) (affine, error) {
	translation := maths.V(0, 0, 0)
	if len(node.Translation) != 0 {
		translation = maths.V(node.Translation[0], node.Translation[1], node.Translation[2])
	}
	if !validVec(translation) {
		return affine{}, fmt.Errorf("translation is not finite")
	}
	scale := maths.V(1, 1, 1)
	if len(node.Scale) != 0 {
		scale = maths.V(node.Scale[0], node.Scale[1], node.Scale[2])
	}
	if !validVec(scale) {
		return affine{}, fmt.Errorf("scale is not finite")
	}
	orientation := maths.Identity
	if len(node.Rotation) != 0 {
		q := node.Rotation
		length := math.Hypot(math.Hypot(q[0], q[1]), math.Hypot(q[2], q[3]))
		if !validNumber(length) || length == 0 {
			return affine{}, fmt.Errorf("rotation is not a finite nonzero quaternion")
		}
		x, y, z, w := q[0]/length, q[1]/length, q[2]/length, q[3]/length
		sine := math.Hypot(math.Hypot(x, y), z)
		if sine != 0 {
			orientation = maths.Rotation(maths.V(x/sine, y/sine, z/sine), 2*math.Atan2(sine, w))
		}
	}
	axes := [3]maths.Vec{orientation.Out(maths.XAxis), orientation.Out(maths.YAxis), orientation.Out(maths.ZAxis)}
	transform := affine{offset: translation}
	for row := range 3 {
		transform.linear[row*3] = component(axes[0], row) * scale.X
		transform.linear[row*3+1] = component(axes[1], row) * scale.Y
		transform.linear[row*3+2] = component(axes[2], row) * scale.Z
	}
	if !transform.finite() {
		return affine{}, fmt.Errorf("transform is not finite")
	}
	return transform, nil
}

// then places a child's coordinates in its parent's space.
func (parent affine) then(child affine) affine {
	var made affine
	for row := range 3 {
		for column := range 3 {
			for k := range 3 {
				made.linear[row*3+column] += parent.linear[row*3+k] * child.linear[k*3+column]
			}
		}
	}
	made.offset = parent.point(child.offset)
	return made
}

// point applies both the linear map and translation.
func (transform affine) point(point maths.Vec) maths.Vec {
	return transform.direction(point).Add(transform.offset)
}

// direction applies the linear map without translation.
func (transform affine) direction(direction maths.Vec) maths.Vec {
	return maths.V(
		transform.linear[0]*direction.X+transform.linear[1]*direction.Y+transform.linear[2]*direction.Z,
		transform.linear[3]*direction.X+transform.linear[4]*direction.Y+transform.linear[5]*direction.Z,
		transform.linear[6]*direction.X+transform.linear[7]*direction.Y+transform.linear[8]*direction.Z,
	)
}

// inversePoint recovers source coordinates, refusing singular transforms.
func (transform affine) inversePoint(point maths.Vec) (maths.Vec, error) {
	a, b, c := transform.linear[0], transform.linear[1], transform.linear[2]
	d, e, f := transform.linear[3], transform.linear[4], transform.linear[5]
	g, h, i := transform.linear[6], transform.linear[7], transform.linear[8]
	determinant := transform.determinant()
	if !validNumber(determinant) || determinant == 0 {
		return maths.Vec{}, fmt.Errorf("singular node transform")
	}
	shifted := point.Sub(transform.offset)
	cofactor := [9]float64{
		e*i - f*h, f*g - d*i, d*h - e*g,
		c*h - b*i, a*i - c*g, b*g - a*h,
		b*f - c*e, c*d - a*f, a*e - b*d,
	}
	result := maths.V(
		(cofactor[0]*shifted.X+cofactor[3]*shifted.Y+cofactor[6]*shifted.Z)/determinant,
		(cofactor[1]*shifted.X+cofactor[4]*shifted.Y+cofactor[7]*shifted.Z)/determinant,
		(cofactor[2]*shifted.X+cofactor[5]*shifted.Y+cofactor[8]*shifted.Z)/determinant,
	)
	if !validVec(result) {
		return maths.Vec{}, fmt.Errorf("inverse transform is not finite")
	}
	return result, nil
}

// scaled scales the entire placed object about the scene origin.
func (transform affine) scaled(factor float64) affine {
	for i := range transform.linear {
		transform.linear[i] *= factor
	}
	transform.offset = transform.offset.Scale(factor)
	return transform
}

// determinant identifies singular maps and handedness changes.
func (transform affine) determinant() float64 {
	a, b, c := transform.linear[0], transform.linear[1], transform.linear[2]
	d, e, f := transform.linear[3], transform.linear[4], transform.linear[5]
	g, h, i := transform.linear[6], transform.linear[7], transform.linear[8]
	return a*(e*i-f*h) - b*(d*i-f*g) + c*(d*h-e*g)
}

// normal applies the inverse transpose and returns a unit surface normal.
func (transform affine) normal(normal maths.Vec) (maths.Vec, error) {
	a, b, c := transform.linear[0], transform.linear[1], transform.linear[2]
	d, e, f := transform.linear[3], transform.linear[4], transform.linear[5]
	g, h, i := transform.linear[6], transform.linear[7], transform.linear[8]
	determinant := transform.determinant()
	if !validNumber(determinant) || determinant == 0 {
		return maths.Vec{}, fmt.Errorf("singular node transform")
	}
	cofactor := [9]float64{
		e*i - f*h, f*g - d*i, d*h - e*g,
		c*h - b*i, a*i - c*g, b*g - a*h,
		b*f - c*e, c*d - a*f, a*e - b*d,
	}
	result := maths.V(
		(cofactor[0]*normal.X+cofactor[1]*normal.Y+cofactor[2]*normal.Z)/determinant,
		(cofactor[3]*normal.X+cofactor[4]*normal.Y+cofactor[5]*normal.Z)/determinant,
		(cofactor[6]*normal.X+cofactor[7]*normal.Y+cofactor[8]*normal.Z)/determinant,
	).Unit()
	if !validVec(result) || result.Len() == 0 {
		return maths.Vec{}, fmt.Errorf("transformed normal is not finite")
	}
	return result, nil
}

// finite checks every linear coefficient and translation coordinate.
func (transform affine) finite() bool {
	for _, part := range transform.linear {
		if !validNumber(part) {
			return false
		}
	}
	return validVec(transform.offset)
}

// validVec refuses any non-finite coordinate.
func validVec(vector maths.Vec) bool {
	return validNumber(vector.X) && validNumber(vector.Y) && validNumber(vector.Z)
}

// component selects an axis for the three-row matrix construction.
func component(vector maths.Vec, axis int) float64 {
	switch axis {
	case 0:
		return vector.X
	case 1:
		return vector.Y
	default:
		return vector.Z
	}
}

// geometryFor shares similarity transforms and bakes other linear maps into
// a checked native mesh, preserving normals and handedness.
func geometryFor(source *mesh.Mesh, transform affine, primitive int) (*mesh.Mesh,
	maths.Subspace, geometryKey, error,
) {
	key, placement, shareSource, err := geometryPlacement(transform, primitive)
	if err != nil {
		return nil, maths.Subspace{}, geometryKey{}, err
	}
	if shareSource {
		return source, placement, key, nil
	}
	determinant := transform.determinant()
	points := source.Points()
	for i := range points {
		points[i] = transform.point(points[i]).Sub(transform.offset)
		if !validVec(points[i]) {
			return nil, maths.Subspace{}, geometryKey{}, fmt.Errorf("oakland: transformed vertex is not finite")
		}
	}
	sourceFaces := source.Faces()
	faces := append([][3]int(nil), sourceFaces...)
	if determinant < 0 {
		for i := range faces {
			faces[i][1], faces[i][2] = faces[i][2], faces[i][1]
		}
	}
	var normals []maths.Vec
	if source.Smooth() {
		normals = make([]maths.Vec, len(points))
		assigned := make([]bool, len(points))
		for faceIndex, face := range sourceFaces {
			faceNormals, _ := source.CornerNormals(faceIndex)
			for corner, pointIndex := range face {
				normal, err := transform.normal(faceNormals[corner])
				if err != nil {
					return nil, maths.Subspace{}, geometryKey{}, fmt.Errorf("oakland: transform normal: %w", err)
				}
				normals[pointIndex] = normal
				assigned[pointIndex] = true
			}
		}
		for index, ok := range assigned {
			if !ok {
				return nil, maths.Subspace{}, geometryKey{}, fmt.Errorf("oakland: smooth vertex %d has no face normal", index)
			}
		}
	}
	geometry, err := mesh.New(points, faces, normals, normals != nil)
	if err != nil {
		return nil, maths.Subspace{}, geometryKey{}, fmt.Errorf("oakland: transformed geometry: %w", err)
	}
	placement = maths.NewSubspace(transform.offset, maths.Identity, 1)
	return geometry, placement, key, nil
}

// geometryPlacement separates reusable geometry from native instance placement.
func geometryPlacement(transform affine, primitive int) (geometryKey, maths.Subspace, bool, error) {
	if !transform.finite() {
		return geometryKey{}, maths.Subspace{}, false, fmt.Errorf("oakland: transformed instance is not finite")
	}
	determinant := transform.determinant()
	if !validNumber(determinant) || determinant == 0 {
		return geometryKey{}, maths.Subspace{}, false, fmt.Errorf("oakland: singular node transform")
	}
	if orientation, scale, ok := similarity(transform); ok && determinant > 0 {
		return geometryKey{primitive: primitive},
			maths.NewSubspace(transform.offset, orientation, scale), true, nil
	}
	key := transformedGeometryKey(transform, primitive)
	return key, maths.NewSubspace(transform.offset, maths.Identity, 1), false, nil
}

// transformedGeometryKey excludes translation so repeated placements can share.
func transformedGeometryKey(transform affine, primitive int) geometryKey {
	key := geometryKey{primitive: primitive}
	for i, part := range transform.linear {
		key.linear[i] = math.Float64bits(part)
	}
	return key
}

// similarity recognizes orthogonal axes with a common positive length.
func similarity(transform affine) (maths.Orientation, float64, bool) {
	columns := [3]maths.Vec{
		maths.V(transform.linear[0], transform.linear[3], transform.linear[6]),
		maths.V(transform.linear[1], transform.linear[4], transform.linear[7]),
		maths.V(transform.linear[2], transform.linear[5], transform.linear[8]),
	}
	length := columns[0].Len()
	if !validNumber(length) || length == 0 {
		return maths.Orientation{}, 0, false
	}
	tolerance := math.Max(length*1e-12, 1e-14)
	for _, axis := range columns {
		if math.Abs(axis.Len()-length) > tolerance {
			return maths.Orientation{}, 0, false
		}
	}
	if math.Abs(columns[0].Dot(columns[1])) > length*length*1e-12 ||
		math.Abs(columns[0].Dot(columns[2])) > length*length*1e-12 ||
		math.Abs(columns[1].Dot(columns[2])) > length*length*1e-12 {
		return maths.Orientation{}, 0, false
	}
	orientation := maths.Orientation{X: columns[0].Scale(1 / length),
		Y: columns[1].Scale(1 / length), Z: columns[2].Scale(1 / length)}
	if !validVec(orientation.X) || !validVec(orientation.Y) || !validVec(orientation.Z) {
		return maths.Orientation{}, 0, false
	}
	return orientation, length, true
}
