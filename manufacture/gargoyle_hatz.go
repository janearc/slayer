package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// the numbers a gargoyle's Hatz has its wings half folded by. The animal
// is the study of hatz_gargoyle.go, whose wings lie flat and out to
// either side as on a roof berth; a gargoyle crouches with them raised
// and swept back. All are picked by eye, against a render of it.
const (
	// wingHinge is how far, in metres of the study's own frame, from the
	// body's line a wing is flat before it begins to rise.
	wingHinge = 0.9
	// wingRadius is how tightly a wing curls up past its hinge, as the
	// radius of the arc its leading line follows, in metres.
	wingRadius = 2.4
	// wingTurn is how far, in radians, a wing has curled up where its arc
	// ends; past that it runs straight on, at that angle.
	wingTurn = 1.0
	// wingSweep is how far back along the body, in metres for each metre
	// out along the span, a wing is swept.
	wingSweep = 0.5
)

// foldedWing is a point of the study with its wings raised: a point near
// the body is where it was, and one out along a wing is bent up about the
// line along the wing's hinge, so that the wing curls up and then runs
// straight, and then swept back along the body. The study faces along x
// with its wings along z and y up.
func foldedWing(point maths.Vec) maths.Vec {
	side := 1.0
	if point.Z < 0 {
		side = -1
	}
	reach := math.Abs(point.Z) - wingHinge
	if reach <= 0 {
		return point
	}
	around := math.Min(reach/wingRadius, wingTurn)
	arm := wingRadius - point.Y
	across := arm * math.Sin(around)
	rise := wingRadius - arm*math.Cos(around)
	if straight := reach - wingRadius*wingTurn; straight > 0 {
		across += straight * math.Cos(wingTurn)
		rise += straight * math.Sin(wingTurn)
	}
	return maths.V(point.X-wingSweep*across, rise, side*(wingHinge+across))
}

// smoothed is the corner normals of a surface of corners and faces: each
// the sum of the normals of the faces that meet at the corner, longer
// for a larger face, made one unit long.
func smoothed(corners []maths.Vec, faces [][3]int) []maths.Vec {
	normals := geometry.AreaNormals(corners, faces)
	for index := range normals {
		normals[index] = normals[index].Unit()
	}
	return normals
}

// hatzGargoyleStudy is the Hatz of hatz_gargoyle.go, which is another
// worker's and is used as it is, with its wings raised and swept back:
// the animal as a gargoyle crouches, in its own frame. The study's one
// surface is taken apart into its corners and faces and put together
// again with the wings bent; the study itself is not changed.
func hatzGargoyleStudy() ([]thing.Thing, error) {
	rest, err := hatzGargoyleRest()
	if err != nil {
		return nil, err
	}
	folded := make([]thing.Thing, len(rest))
	for index, piece := range rest {
		whole, is := piece.Shape.(*mesh.Mesh)
		if !is {
			return nil, fmt.Errorf("hatz gargoyle: the study's surface is a %T"+
				", not a mesh", piece.Shape)
		}
		bare := soupOfMesh(whole)
		corners := make([]maths.Vec, len(bare.corners))
		for at, corner := range bare.corners {
			corners[at] = foldedWing(corner)
		}
		piece.Shape, err = mesh.New(corners, bare.faces,
			smoothed(corners, bare.faces), true)
		if err != nil {
			return nil, err
		}
		folded[index] = piece
	}
	return folded, nil
}
