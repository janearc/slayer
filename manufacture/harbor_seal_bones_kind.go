package manufacture

import (
	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// harborSealBonesKind is the harbor seal's skeleton alone, at rest, in
// the rig's space: the study each step of the animal is built over.
type harborSealBonesKind struct{}

// sealBone is the colour of dry bone, chosen.
var sealBone = maths.RGB(.62, .57, .48)

// made is the harbor seal's skeleton alone, bones without skin. It refuses
// a seed that was not made.
func (harborSealBonesKind) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	made, err := harborSealBones()
	if err != nil {
		return good{}, err
	}
	points, faces := made.bones.Draw(made.bones.Frames(skeleton.Pose{}),
		made.forms)
	solid, err := mesh.NewWithRestPoints(points, points, faces, nil, false)
	if err != nil {
		return good{}, err
	}
	look := &thing.Material{Name: "harbor seal bone", Colour: sealBone,
		Gloss: .08, Satin: .5}
	parts := []thing.Thing{{Shape: solid, Material: look,
		Motion: thing.Still(maths.Local())}}
	box := solid.Bounds()
	return good{native: parts, extent: &box}, nil
}
