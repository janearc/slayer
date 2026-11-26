package birds

import (
	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// Build makes the checked bird in its stance, from its seed and place.
func (bird Specimen) Build(from branching.Draws, at maths.Subspace) ([]thing.Thing, maths.Box, error) {
	species, stance := bird.profile, bird.stance
	measures, err := measured(species, from.Draw(0, 0, 0))
	if err != nil {
		return nil, maths.Box{}, err
	}
	body, err := newBirdBody(species, measures, from)
	if err != nil {
		return nil, maths.Box{}, err
	}
	frames := body.bones.Frames(body.pose(stance))
	switch stance {
	case Bones:
		return body.things(body.bonePieces(), frames, at)
	case Outlined:
		return body.outlined(body.barePieces(), frames, at)
	}
	return body.outlined(body.skinPieces(), frames, at)
}

// outlined is the outline carried by frames and pieces round it.
func (body *birdBody) outlined(pieces []birdPiece, frames []skeleton.Frame,
	at maths.Subspace) ([]thing.Thing, maths.Box, error) {
	outline, points, err := body.outlineThing(frames, at)
	if err != nil {
		return nil, maths.Box{}, err
	}
	parts, box, err := body.things(pieces, frames, at)
	if err != nil {
		return nil, maths.Box{}, err
	}
	return append([]thing.Thing{outline}, parts...), geometry.GrowBox(box, points),
		nil
}

// birdDrawnRegions are the regions in the order their things are made.
var birdDrawnRegions = append(append([]string{}, birdRegions...),
	"plumes", "bone")

// things is pieces carried by frames, a closed mesh for each region
// that has any, and the box round them.
func (body *birdBody) things(pieces []birdPiece, frames []skeleton.Frame,
	at maths.Subspace) ([]thing.Thing, maths.Box, error) {
	parts := []thing.Thing{}
	box := maths.Box{Min: maths.V(math.Inf(1), math.Inf(1), math.Inf(1)),
		Max: maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))}
	for _, region := range birdDrawnRegions {
		corners, faces := []maths.Vec{}, [][3]int{}
		for _, piece := range pieces {
			if piece.region != region {
				continue
			}
			bone, _ := body.bones.Find(piece.bone)
			corners, faces = piece.form.Build(corners, faces,
				frames[bone].Apply)
		}
		if len(faces) == 0 {
			continue
		}
		skin, err := mesh.New(corners, faces,
			geometry.UnitNormals(corners, faces), true)
		if err != nil {
			return nil, maths.Box{}, err
		}
		box = geometry.GrowBox(box, corners)
		parts = append(parts, thing.Thing{Shape: skin,
			Material: body.material(region), Motion: thing.Still(at)})
	}
	return parts, box, nil
}

// birdBoneColour is a cleaned bone's reflectance; picked, as the orca's.
var birdBoneColour = maths.RGB(.62, .58, .50)

// material is region's material: its colour from the row, matte, or
// bone's for the skeleton.
func (body *birdBody) material(region string) *thing.Material {
	if region == "bone" {
		return &thing.Material{Name: "bird bone", Colour: birdBoneColour}
	}
	c := body.species.Plumage[region].Colour
	return &thing.Material{Name: body.species.Name + " " + region,
		Colour: maths.RGB(c[0], c[1], c[2])}
}
