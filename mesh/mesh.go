// Package mesh implements Slayer's triangle meshes. Construction validates the
// geometry. A bounding hierarchy accelerates intersection and nearest-point
// queries; it is grown once, at the first ray or the first nearest point, and
// not by construction. The ray intersection method avoids cracks along shared
// edges.
//
// The implementation follows the published methods cited beside each algorithm.
// It uses maths vectors, boxes, and rays and implements scene/shape.Shape
// through Bounds, Distance, Hit, and Normal.
package mesh

import (
	"fmt"
	"math"
	"slices"
	"sync"

	"github.com/janearc/slayer/maths"
)

// meshRefusal is a refusal of this package, held as a constant so that no
// one can assign it.
type meshRefusal string

// Error is the refusal's words.
func (r meshRefusal) Error() string { return string(r) }

// errMesh is a mesh that cannot be made. Every reason below wraps it.
const errMesh meshRefusal = "mesh"

// ErrMesh is the error every refusal to make a mesh is, by errors.Is: a
// mesh that cannot be made. Each refusal says which face and why, where
// a face is the reason.
const ErrMesh = errMesh

// The reasons a mesh is refused. Each wraps errMesh, and each is told
// from the others by errors.Is.
var (
	// errNoFaces is a mesh with no faces.
	errNoFaces = fmt.Errorf("%w: no faces", errMesh)
	// errNormalCount is a count of normals that is not the corners'.
	errNormalCount = fmt.Errorf("%w: the normals are not one a corner",
		errMesh)
	// errNoCorner is a face that names a corner there is not.
	errNoCorner = fmt.Errorf("%w: a face names a corner there is not",
		errMesh)
	// errNotPoint is a face that has a corner that is not a finite point.
	errNotPoint = fmt.Errorf("%w: a corner is not a point", errMesh)
	// errNormalNowhere is a face with a normal that points nowhere.
	errNormalNowhere = fmt.Errorf("%w: a normal points nowhere", errMesh)
	// errNoArea is a face whose corners are in a line.
	errNoArea = fmt.Errorf("%w: a face has no area", errMesh)
	// errSameWay is two faces that run one edge the same way.
	errSameWay = fmt.Errorf("%w: two faces run an edge the same way",
		errMesh)
	// errCrowded is more than two faces on one edge, whichever ways they
	// run it.
	errCrowded = fmt.Errorf("%w: an edge is shared by more than two faces",
		errMesh)
	// errInsideOut is a closed piece whose faces turn inward.
	errInsideOut = fmt.Errorf("%w: a closed piece is inside out", errMesh)
	// errTooMany is more corners than a face can name.
	errTooMany = fmt.Errorf("%w: more corners than a face can name", errMesh)
	// errRestCount is a count of rest points that is not the corners'.
	errRestCount = fmt.Errorf("%w: the rest points are not one a corner",
		errMesh)
	// errRestPoint is a rest point that is not a finite point.
	errRestPoint = fmt.Errorf("%w: a rest point is not a point", errMesh)
)

// Mesh is a surface of triangles in its own space. each face is three
// corners turning counterclockwise seen from outside, so the turn says
// which side is out. a ray meets a face from either side.
//
// nothing a caller can see changes after New. The tree that finds faces
// is grown once, at the first ray or nearest point, under a lock, and
// every answer is the same before it and after. So one mesh can stand
// in many places at once, and be asked from many goroutines.
type Mesh struct {
	fibres   *FibreField
	rest     []maths.Vec
	points   []maths.Vec
	faces    []face
	normals  []maths.Vec
	smooth   bool
	closed   bool
	tree     tree
	box      maths.Box
	treeOnce sync.Once
	treeMu   sync.RWMutex
}

// NewWithRestPoints makes a mesh whose faces also name the corresponding
// vertices in its undeformed shape, for surface paint that follows motion.
// There is one rest point a corner, and each is a finite point; else the
// mesh is refused, as New refuses.
func NewWithRestPoints(corners, rest []maths.Vec,
	faces [][triangleCorners]int, normals []maths.Vec, smooth bool,
) (*Mesh, error) {
	if len(rest) != len(corners) {
		return nil, fmt.Errorf("%w: %d rest points for %d corners",
			errRestCount, len(rest), len(corners))
	}
	for index, point := range rest {
		if !finite(point) {
			return nil, fmt.Errorf("%w: rest point %d", errRestPoint, index)
		}
	}
	made, err := New(corners, faces, normals, smooth)
	if err != nil {
		return nil, err
	}
	made.rest = slices.Clone(rest)
	return made, nil
}

// face is three corners, by their places in the mesh's points. A place
// is an int32, half an int: a mesh past two thousand million corners
// would hold 48 GB of points, and New refuses more corners than a place
// can name before any place is narrowed.
type face [triangleCorners]int32

// mostCorners is the most corners a mesh can have: a face names one by
// an int32, whose largest place is one less.
const mostCorners = math.MaxInt32 // defined by int32: its largest value

// New makes a mesh of corners and faces, each face three places in
// corners. normals may be nil, for flat faces; given, there is one a
// corner, and smooth blends them across each face.
//
// corners are one place only if their parts are equal to the bit: two a
// rounding apart are two places, and faces that meet at them leave an
// edge open, so the mesh is not closed.
//
// a face that names a corner there is not, has a corner or normal that
// is not a finite point, or has no area, is an error naming it; so is
// an edge two faces run the same way, an edge more than two faces share,
// and a closed piece inside out; so is a count of corners past what a
// face can name. Each is an error wrapping ErrMesh and a reason of its
// own.
//
// Solids that touch along an edge are refused as an edge shared by more
// than two faces, so each is a mesh of its own. A cavity, a shell turned
// in toward its hollow, is a closed piece inside out.
func New(corners []maths.Vec, faces [][triangleCorners]int,
	normals []maths.Vec, smooth bool,
) (*Mesh, error) {
	if err := fits(len(corners)); err != nil {
		return nil, err
	}
	if len(faces) == nought {
		return nil, errNoFaces
	}
	if normals != nil && len(normals) != len(corners) {
		return nil, fmt.Errorf("%w: %d normals for %d corners",
			errNormalCount, len(normals), len(corners))
	}
	made := &Mesh{
		points:  slices.Clone(corners),
		normals: units(normals), smooth: smooth && normals != nil,
		faces: make([]face, len(faces)),
	}
	for i, each := range faces {
		if err := made.check(i, each); err != nil {
			return nil, err
		}
	}
	closed, err := wound(made.points, made.faces)
	if err != nil {
		return nil, err
	}
	made.closed = closed
	made.box = boxOf(made.points, made.faces)
	return made, nil
}

// fits is whether a count of corners is one every face can name, or the
// error it is not.
func fits(corners int) error {
	if int64(corners) > mostCorners {
		return fmt.Errorf("%w: %d corners, at most %d", errTooMany,
			corners, int64(mostCorners))
	}
	return nil
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

// check is what is wrong with face i, given as each, if anything. Its
// corners are checked as given, and only then kept as the mesh's face i,
// so a place too large for a face is refused and never narrowed.
func (made *Mesh) check(i int, each [triangleCorners]int) error {
	for _, corner := range each {
		if err := made.checkCorner(i, corner); err != nil {
			return err
		}
	}
	made.faces[i] = face{int32(each[first]), int32(each[second]),
		int32(each[third])}
	if !somewhere(made.across(i)) {
		return fmt.Errorf("%w: face %d, its corners in a line", errNoArea,
			i)
	}
	return nil
}

// checkCorner is what is wrong with a corner of face i, if anything.
func (made *Mesh) checkCorner(i, corner int) error {
	switch {
	case corner < nought || corner >= len(made.points):
		return fmt.Errorf("%w: face %d names corner %d of %d", errNoCorner,
			i, corner, len(made.points))
	case !finite(made.points[corner]):
		return fmt.Errorf("%w: face %d: corner %d", errNotPoint, i, corner)
	case made.normals != nil && !somewhere(made.normals[corner]):
		return fmt.Errorf("%w: face %d: the normal at corner %d",
			errNormalNowhere, i, corner)
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
func number(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, nought) }

// somewhere is whether a way is finite and not zero.
func somewhere(way maths.Vec) bool {
	return finite(way) && (way.X != nought || way.Y != nought ||
		way.Z != nought)
}

// FaceCount is how many faces the mesh has.
func (made *Mesh) FaceCount() int { return len(made.faces) }

// Corners is face i's three corners, in its turn.
func (made *Mesh) Corners(i int) (maths.Vec, maths.Vec, maths.Vec) {
	at := made.faces[i]
	return made.points[at[first]], made.points[at[second]],
		made.points[at[third]]
}

// CornerNormals is the normals at face i's three corners, in the order
// Corners gives them, for a mesh shaded smooth; shaded flat, it is
// false and three noughts.
func (made *Mesh) CornerNormals(i int) ([triangleCorners]maths.Vec, bool) {
	if !made.smooth {
		return [triangleCorners]maths.Vec{}, false
	}
	at := made.faces[i]
	return [triangleCorners]maths.Vec{
		made.normals[at[first]], made.normals[at[second]],
		made.normals[at[third]],
	}, true
}

// Points is a copy of the mesh's corners, by their places.
func (made *Mesh) Points() []maths.Vec { return slices.Clone(made.points) }

// VertexNormals copies the mesh's supplied normals in vertex order when
// the mesh is shaded smooth.
func (made *Mesh) VertexNormals() []maths.Vec {
	if !made.smooth {
		return nil
	}
	return slices.Clone(made.normals)
}

// Faces is a copy of the mesh's faces, each three places in Points.
func (made *Mesh) Faces() [][triangleCorners]int {
	copied := make([][triangleCorners]int, len(made.faces))
	for i, at := range made.faces {
		copied[i] = [triangleCorners]int{int(at[first]), int(at[second]),
			int(at[third])}
	}
	return copied
}

// Smooth is whether the mesh blends its corners' normals across each
// face.
func (made *Mesh) Smooth() bool { return made.smooth }

// Closed is whether every piece of the mesh is closed: every edge of
// every piece is met by a face running it the other way. A mesh of a
// closed solid and an open sheet is not closed, and its solid has no
// inside by this account: Distance to it is not below zero anywhere.
func (made *Mesh) Closed() bool { return made.closed }

// Bounds is the box the mesh fills, in its own space.
func (made *Mesh) Bounds() maths.Box { return made.box }

// boxOf is the box round every face, joined face by face in their order:
// the same steps, on the same numbers, as the tree's root box takes, so
// it is that box to the bit without the tree being grown.
func boxOf(points []maths.Vec, faces []face) maths.Box {
	around := emptySpan()
	for _, each := range faces {
		around = around.joined(spanOf(points, each))
	}
	return maths.NewBox(vectorOf(around.low), vectorOf(around.high))
}

// ensureTree builds the immutable CPU intersection index at first use:
// a ray or a nearest point asks for it, and so does TreeSize, which the
// GPU encoder asks to pack this tree's boxes as they are.
func (made *Mesh) ensureTree() { made.treeOnce.Do(made.growTree) }

// growTree grows the tree under the lock Storage reads it by.
func (made *Mesh) growTree() {
	made.treeMu.Lock()
	defer made.treeMu.Unlock()
	made.tree = grown(made.points, made.faces)
}
