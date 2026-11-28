package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// pterosaurPiece is one separately drawn piece of the animal: a skull, a
// bone, a panel of skin. It is points and triangles and the material it
// is drawn in. rest, when it is not nil, is the piece's undeformed shape.
type pterosaurPiece struct {
	name         string
	points, rest []maths.Vec
	faces        [][3]int
	look         *thing.Material
	// flat draws each face with its own normal and not a blend, for
	// pieces made of sharp folds, such as a ridge.
	flat bool
}

// solidPiece is a closed lofted surface as a piece, wound so its faces
// turn outward.
func solidPiece(name string, surface tubeSurface,
	look *thing.Material) pterosaurPiece {
	faces := surface.faces
	if solidVolume(surface.points, faces) < 0 {
		faces = flippedFaces(faces)
	}
	return pterosaurPiece{name: name, points: surface.points, faces: faces,
		look: look}
}

// thingOf is the piece as a placed thing, or an error naming the piece.
func (piece pterosaurPiece) thingOf() (thing.Thing, error) {
	made, err := meshOfTriangles(piece.points, piece.rest, piece.faces)
	if piece.flat {
		made, err = mesh.New(piece.points, piece.faces, nil, false)
	}
	if err != nil {
		return thing.Thing{}, fmt.Errorf("pterosaur %s: %w",
			piece.name, err)
	}
	return thing.Thing{Shape: made, Material: piece.look,
		Motion: thing.Still(maths.Local())}, nil
}

// thingsOf is each piece as a thing, or the first error.
func thingsOf(pieces []pterosaurPiece) ([]thing.Thing, error) {
	things := make([]thing.Thing, 0, len(pieces))
	for _, piece := range pieces {
		placed, err := piece.thingOf()
		if err != nil {
			return nil, err
		}
		things = append(things, placed)
	}
	return things, nil
}
