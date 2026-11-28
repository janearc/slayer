package manufacture

import (
	"fmt"
	"slices"
	"time"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// BioformClipSpec describes complete pose keys and optional rigid props.
// Keys use the selected maker's rest axes and are sampled at scene time.
type BioformClipSpec struct {
	Keys   []skeleton.PoseKey
	Smooth bool
	Props  []BoneProp
}

// BoneProp carries a static manufacture kind on a named bone. Placement
// gives its offset from the bone's head in the figure's rest axes.
type BoneProp struct {
	Name, Bone, Kind string
	Placement        maths.Subspace
}

type poseRigMaker interface {
	poseRig(seed) (poseRig, error)
}

type poseRig struct {
	bones *skeleton.Skeleton
	pose  func(skeleton.Pose) (good, skeleton.BoneFrames, error)
}

// WithBioformClip adds authored posing to a kind that supplies a native rig.
// The rig keeps its own appearance, deformation and attachment geometry.
func WithBioformClip(kind Kind, spec BioformClipSpec) (Kind, error) {
	rig, ok := kind.how.(poseRigMaker)
	if !ok {
		return Kind{}, fmt.Errorf("manufacture: %s does not take a pose clip", kind.name)
	}
	if len(spec.Keys) == 0 || spec.Keys[0].At != 0 {
		return Kind{}, fmt.Errorf("manufacture: a pose clip needs a key at zero")
	}
	for i := 1; i < len(spec.Keys); i++ {
		if spec.Keys[i].At <= spec.Keys[i-1].At {
			return Kind{}, fmt.Errorf("manufacture: pose key %d must follow the preceding key", i+1)
		}
	}
	for _, prop := range spec.Props {
		kind, err := KindNamed(prop.Kind)
		if err != nil {
			return Kind{}, fmt.Errorf("attachment %q: %w", prop.Name, err)
		}
		if Deforms(kind) {
			return Kind{}, fmt.Errorf("attachment %q needs a static kind", prop.Name)
		}
	}
	spec.Keys, spec.Props = slices.Clone(spec.Keys), slices.Clone(spec.Props)
	kind.how = authoredBioformClip{rig: rig, spec: spec}
	return kind, nil
}

type authoredBioformClip struct {
	rig  poseRigMaker
	spec BioformClipSpec
}

func (row authoredBioformClip) made(from seed) (good, error) {
	sample, err := row.prepareDeformation(from, "quick")
	if err != nil {
		return good{}, err
	}
	return sample(0)
}

func (row authoredBioformClip) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

func (row authoredBioformClip) prepareDeformation(from seed, detail string) (func(time.Duration) (good, error), error) {
	if detail != "quick" {
		return nil, fmt.Errorf("manufacture: pose clips give the quick representation")
	}
	rig, err := row.rig.poseRig(from)
	if err != nil {
		return nil, err
	}
	clip, err := rig.bones.NewPoseClip(row.spec.Keys)
	if err != nil {
		return nil, err
	}
	if row.spec.Smooth {
		clip = clip.WithSmoothTiming()
	}
	props, err := prepareBoneProps(rig.bones, from, row.spec.Props)
	if err != nil {
		return nil, err
	}
	return func(at time.Duration) (good, error) {
		made, frames, err := rig.pose(clip.At(at))
		if err != nil {
			return good{}, err
		}
		for _, prop := range props {
			placement, err := prop.socket.Place(frames)
			if err != nil {
				return good{}, fmt.Errorf("attachment %q: %w", prop.name, err)
			}
			for _, part := range prop.parts {
				part.Motion = thing.Still(part.At(0).Within(placement))
				made.native = append(made.native, part)
			}
		}
		return made, nil
	}, nil
}

type boundBoneProp struct {
	name   string
	socket skeleton.Socket
	parts  []thing.Thing
}

func prepareBoneProps(bones *skeleton.Skeleton, from seed, specs []BoneProp) ([]boundBoneProp, error) {
	props := make([]boundBoneProp, 0, len(specs))
	for _, spec := range specs {
		socket, err := bones.NewSocket(spec.Bone, spec.Placement)
		if err != nil {
			return nil, fmt.Errorf("attachment %q: %w", spec.Name, err)
		}
		kind, err := KindNamed(spec.Kind)
		if err != nil {
			return nil, fmt.Errorf("attachment %q: %w", spec.Name, err)
		}
		made, err := makeOf(kind, from)
		if err != nil {
			return nil, fmt.Errorf("attachment %q: %w", spec.Name, err)
		}
		if len(made.lights) != 0 {
			return nil, fmt.Errorf("attachment %q cannot carry lights", spec.Name)
		}
		parts, err := made.things(maths.Local())
		if err != nil {
			return nil, fmt.Errorf("attachment %q: %w", spec.Name, err)
		}
		props = append(props, boundBoneProp{name: spec.Name, socket: socket, parts: parts})
	}
	return props, nil
}
