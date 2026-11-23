package manufacture

import (
	"errors"
	"fmt"
	"slices"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// OrcaClean05 constructs one closed skin of an adult cow orca and
// carries it through a swim at phase, made by turning the joints of its
// skeleton. The trunk, the falcate dorsal fin, the paddle flippers, the
// notched flukes, the melon, the mouth and teeth, the blowhole, the
// eyes and the belly slits are all one surface, with filleted roots,
// cut from a field and bound to the bones.
//
// The skin carries ridges,
// rake marks and scars as geometry, and a wet sheen from its material.
func OrcaClean05(phase float64, at maths.Subspace) ([]thing.Thing, error) {
	bones, err := orcaSkeleton()
	if err != nil {
		return nil, err
	}
	return orcaPosed(bones, orcaSwimPose(bones, phase), at)
}

// OrcaClean05SpyHop is the same skin spy-hopping: upright, its head
// raised, its flippers out.
func OrcaClean05SpyHop(at maths.Subspace) ([]thing.Thing, error) {
	bones, err := orcaSkeleton()
	if err != nil {
		return nil, err
	}
	return orcaPosed(bones, orcaSpyHopPose(bones), at)
}

// errOrcaPose is a pose that turns a joint the skeleton does not have.
var errOrcaPose = errors.New("orca: the pose turns joints it has not")

// orcaPosed is the skin carried by bones in pose: each point of it moved
// by the bones it is bound to. A pose that turns a joint bones lacks is
// refused, naming it.
func orcaPosed(bones *skeleton.Skeleton, pose skeleton.Pose,
	at maths.Subspace) ([]thing.Thing, error) {
	if unknown := bones.Unknown(pose); len(unknown) > 0 {
		return nil, fmt.Errorf("%w: %v", errOrcaPose, unknown)
	}
	binding, err := orcaBinding()
	if err != nil {
		return nil, err
	}
	rest := orcaRest()
	frames := bones.Frames(pose)
	posed := make([]maths.Vec, len(rest.relieved))
	for i, p := range rest.relieved {
		posed[i] = skeleton.Carry(p, frames, binding[i])
	}
	faces := slices.Clone(rest.faces)
	normals := orcaVertexNormals(posed, faces)
	skin, err := mesh.NewWithRestPoints(posed, rest.points, faces, normals,
		true)
	if err != nil {
		return nil, err
	}
	return []thing.Thing{{Shape: skin, Material: orcaSkinMaterial(),
		Motion: thing.Still(at)}}, nil
}

// orcaSkinMaterial is wet black and white skin: the colours come from
// the marking by place on the body, the sheen from a clear film of water
// over it. The film's index, the full Fresnel response it gives and the
// width of the highlight are chosen.
func orcaSkinMaterial() *thing.Material {
	const (
		water = 1.34 // refractive index of the film over the skin
		sheen = .12  // roughness of the film; its square is the highlight's width
	)
	return &thing.Material{
		Name:    "orca skin",
		Texture: orca05Paint{},
		Colour:  orcaBlack,
		Gloss:   1,
		Index:   water,
		Satin:   sheen,
	}
}
