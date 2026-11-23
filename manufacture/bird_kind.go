package manufacture

import (
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// A bird kind is named bird, the species' name in birds.yaml, and the
// stance: bird-grey-heron-standing, bird-grey-heron-gliding,
// bird-grey-heron-bones. The seed picks the bird's size within the
// species' ranges and turns each feather a little at its root.

// birdKind is one species in one stance.
type birdKind struct {
	species, stance string
}

// birdKinds puts a kind for every species and stance in rows.
func birdKinds(rows map[string]maker) {
	for _, key := range birdRead.keys() {
		for _, stance := range birdStances {
			rows["bird-"+key+"-"+stance] = birdKind{species: key,
				stance: stance}
		}
	}
}

// made is the bird grown from from.
func (kind birdKind) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	parts, box, err := birdThings(kind.species, kind.stance, from,
		maths.Local())
	if err != nil {
		return good{}, err
	}
	return good{native: parts, extent: &box}, nil
}

// birdThings is the bird of species key in stance, grown from from and
// placed at at: a thing for each region of its plumage and bare parts,
// or one of bone, and the box round them all.
func birdThings(key, stance string, from seed,
	at maths.Subspace) ([]thing.Thing, maths.Box, error) {
	species, err := birdSpeciesNamed(key)
	if err != nil {
		return nil, maths.Box{}, err
	}
	measures, err := measured(species, from.draw(picking, 0, 0))
	if err != nil {
		return nil, maths.Box{}, err
	}
	body, err := newBirdBody(species, measures, from)
	if err != nil {
		return nil, maths.Box{}, err
	}
	frames := body.bones.Frames(body.pose(stance))
	switch stance {
	case birdBones:
		return body.things(body.bonePieces(), frames, at)
	case birdOutlined:
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
	return append([]thing.Thing{outline}, parts...), birdGrown(box, points),
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
			orcaVertexNormals(corners, faces), true)
		if err != nil {
			return nil, maths.Box{}, err
		}
		box = birdGrown(box, corners)
		parts = append(parts, thing.Thing{Shape: skin,
			Material: body.material(region), Motion: thing.Still(at)})
	}
	return parts, box, nil
}

// birdGrown is box grown to hold points.
func birdGrown(box maths.Box, points []maths.Vec) maths.Box {
	for _, p := range points {
		box.Min = maths.V(math.Min(box.Min.X, p.X), math.Min(box.Min.Y, p.Y),
			math.Min(box.Min.Z, p.Z))
		box.Max = maths.V(math.Max(box.Max.X, p.X), math.Max(box.Max.Y, p.Y),
			math.Max(box.Max.Z, p.Z))
	}
	return box
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
