package whales

import (
	"errors"
	"fmt"
	"github.com/janearc/slayer/manufacture/geometry"
	"slices"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

func (model *orcaModel) swimming(phase float64, at maths.Subspace, palette Palette, satin float64) ([]thing.Thing, error) {
	bones, err := model.orcaSkeleton()
	if err != nil {
		return nil, err
	}
	return model.orcaPosed(bones, orcaSwimPose(bones, phase), at, palette, satin)
}

func (model *orcaModel) spyHop(at maths.Subspace, palette Palette, satin float64) ([]thing.Thing, error) {
	bones, err := model.orcaSkeleton()
	if err != nil {
		return nil, err
	}
	return model.orcaPosed(bones, orcaSpyHopPose(bones), at, palette, satin)
}

// errOrcaPose is a pose that turns a joint the skeleton does not have.
var errOrcaPose = errors.New("orca: the pose turns joints it has not")

// orcaPosed is the skin carried by bones in pose: each point of it moved
// by the bones it is bound to. A pose that turns a joint bones lacks is
// refused, naming it.
func (model *orcaModel) orcaPosed(bones *skeleton.Skeleton, pose skeleton.Pose,
	at maths.Subspace, palette Palette, satin float64) ([]thing.Thing, error) {
	if unknown := bones.Unknown(pose); len(unknown) > 0 {
		return nil, fmt.Errorf("%w: %v", errOrcaPose, unknown)
	}
	binding, err := model.orcaBinding()
	if err != nil {
		return nil, err
	}
	rest := model.orcaRest()
	frames := bones.Frames(pose)
	posed := make([]maths.Vec, len(rest.relieved))
	for i, p := range rest.relieved {
		posed[i] = skeleton.Carry(p, frames, binding[i])
	}
	faces := slices.Clone(rest.faces)
	normals := geometry.UnitNormals(posed, faces)
	skin, err := mesh.NewWithRestPoints(posed, rest.points, faces, normals,
		true)
	if err != nil {
		return nil, err
	}
	return []thing.Thing{{Shape: skin, Material: model.orcaSkinMaterial(palette, satin),
		Motion: thing.Still(at)}}, nil
}

// orcaSkinMaterial is wet black and white skin: the colours come from
// the marking by place on the body, the sheen from a clear film of water
// over it. The film's index, the full Fresnel response it gives and the
// width of the highlight are chosen.
func (model *orcaModel) orcaSkinMaterial(palette Palette, satin float64) *thing.Material {
	const (
		water = 1.34 // refractive index of the film over the skin
	)
	return &thing.Material{
		Name:    "orca skin",
		Texture: orca05Paint{model: model, palette: palette},
		Colour:  palette.Black,
		Gloss:   1,
		Index:   water,
		Satin:   satin,
	}
}
