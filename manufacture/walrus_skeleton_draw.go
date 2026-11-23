package manufacture

import (
	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// WalrusFresh5Skeleton is the walrus's skeleton alone, at rest lying on
// land, every bone a closed surface: the study of the bones the hide is
// built over.
func WalrusFresh5Skeleton() ([]thing.Thing, error) {
	body := walrusSkeleton()
	if body.fault != nil {
		return nil, body.fault
	}
	points, faces := body.tree.Draw(body.tree.Frames(skeleton.Pose{}), body.forms)
	bones, err := shape.NewMesh(points, faces, nil, false)
	if err != nil {
		return nil, err
	}
	material := &thing.Material{Name: "walrus bone",
		Colour: maths.RGB(.72, .67, .57), Gloss: .08, Satin: .5}
	return []thing.Thing{{Shape: bones, Material: material,
		Motion: thing.Still(maths.NewSubspace(maths.Vec{}, maths.Identity, 1))}}, nil
}
