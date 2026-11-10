// Package mesh implements Slayer's triangle meshes. Construction validates the
// geometry, a bounding hierarchy accelerates intersection, and the ray
// intersection method avoids cracks along shared edges.
//
// The implementation follows the published methods cited beside each algorithm.
// It uses maths vectors, boxes, and rays and implements scene/shape.Shape
// through Bounds, Distance, Hit, and Normal.
package mesh

import (
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// ErrMesh is a mesh that cannot be made; the error says which face and
// why.
var ErrMesh = errors.New("mesh")

// Mesh is a surface of triangles in its own space. each face is three
// corners turning counterclockwise seen from outside, so the turn says
// which side is out. a ray meets a face from either side.
//
// nothing in it changes after New, so one mesh can stand in many
// places at once.
type Mesh struct {
	points  []maths.Vec
	faces   []face
	normals []maths.Vec
	smooth  bool
	closed  bool
	tree    tree
}

// face is three corners, by their places in the mesh's points.
type face [3]int

// New makes a mesh of corners and faces, each face three places in
// corners. normals may be nil, for flat faces; given, there is one a
// corner, and smooth blends them across each face.
//
// a face that names a corner there is not, has a corner or normal that
// is not a finite point, or has no area, is an error naming it; so is
// an edge two faces run the same way, and a closed piece inside out.
func New(corners []maths.Vec, faces [][3]int, normals []maths.Vec,
	smooth bool) (*Mesh, error) {
	if len(faces) == 0 {
		return nil, fmt.Errorf("%w: no faces", ErrMesh)
	}
	if normals != nil && len(normals) != len(corners) {
		return nil, fmt.Errorf("%w: %d normals for %d corners", ErrMesh,
			len(normals), len(corners))
	}
	made := &Mesh{points: slices.Clone(corners),
		normals: units(normals), smooth: smooth && normals != nil,
		faces: make([]face, len(faces))}
	for i, each := range faces {
		made.faces[i] = face(each)
		if err := made.check(i); err != nil {
			return nil, err
		}
	}
	closed, err := wound(made.points, made.faces)
	if err != nil {
		return nil, err
	}
	made.closed = closed
	made.tree = grown(made.points, made.faces)
	return made, nil
}

// units is normals each made one long, where it points somewhere.
func units(normals []maths.Vec) []maths.Vec {
	if normals == nil {
		return nil
	}
	made := slices.Clone(normals)
	for i, normal := range made {
		if somewhere(normal) {
			made[i] = normal.Unit()
		}
	}
	return made
}

// check is what is wrong with face i, if anything.
func (made *Mesh) check(i int) error {
	for _, corner := range made.faces[i] {
		if err := made.checkCorner(i, corner); err != nil {
			return err
		}
	}
	if !somewhere(made.across(i)) {
		return fmt.Errorf("%w: face %d has no area, its corners in a "+
			"line", ErrMesh, i)
	}
	return nil
}

// checkCorner is what is wrong with a corner of face i, if anything.
func (made *Mesh) checkCorner(i, corner int) error {
	switch {
	case corner < 0 || corner >= len(made.points):
		return fmt.Errorf("%w: face %d names corner %d of %d", ErrMesh,
			i, corner, len(made.points))
	case !finite(made.points[corner]):
		return fmt.Errorf("%w: face %d: corner %d is not a point",
			ErrMesh, i, corner)
	case made.normals != nil && !somewhere(made.normals[corner]):
		return fmt.Errorf("%w: face %d: the normal at corner %d "+
			"points nowhere", ErrMesh, i, corner)
	}
	return nil
}

// across is face i's two edges from its first corner crossed: its way
// out, twice its area long.
func (made *Mesh) across(i int) maths.Vec {
	one, two, three := made.Corners(i)
	return two.Sub(one).Cross(three.Sub(one))
}

// finite is whether a point is somewhere: no part of it nowhere or
// endlessly far.
func finite(point maths.Vec) bool {
	return number(point.X) && number(point.Y) && number(point.Z)
}

// number is whether x is a finite number.
func number(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

// somewhere is whether a way is finite and not zero.
func somewhere(way maths.Vec) bool {
	return finite(way) && (way.X != 0 || way.Y != 0 || way.Z != 0)
}

// FaceCount is how many faces the mesh has.
func (made *Mesh) FaceCount() int { return len(made.faces) }

// Corners is face i's three corners, in its turn.
func (made *Mesh) Corners(i int) (maths.Vec, maths.Vec, maths.Vec) {
	at := made.faces[i]
	return made.points[at[0]], made.points[at[1]], made.points[at[2]]
}

// CornerNormals is the normals at face i's three corners, in the order
// Corners gives them, for a mesh shaded smooth; shaded flat, it is
// false and three noughts.
func (made *Mesh) CornerNormals(i int) ([3]maths.Vec, bool) {
	if !made.smooth {
		return [3]maths.Vec{}, false
	}
	at := made.faces[i]
	return [3]maths.Vec{made.normals[at[0]], made.normals[at[1]],
		made.normals[at[2]]}, true
}

// Points is a copy of the mesh's corners, by their places.
func (made *Mesh) Points() []maths.Vec { return slices.Clone(made.points) }

// Faces is a copy of the mesh's faces, each three places in Points.
func (made *Mesh) Faces() [][3]int {
	copied := make([][3]int, len(made.faces))
	for i, at := range made.faces {
		copied[i] = [3]int(at)
	}
	return copied
}

// Smooth is whether the mesh blends its corners' normals across each
// face.
func (made *Mesh) Smooth() bool { return made.smooth }

// Closed is whether every edge is met by a face running it the other
// way, so the mesh has an inside.
func (made *Mesh) Closed() bool { return made.closed }

// Bounds is the box the mesh fills, in its own space.
func (made *Mesh) Bounds() maths.Box { return made.tree.nodes[0].box() }
