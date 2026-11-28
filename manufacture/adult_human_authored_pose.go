package manufacture

import (
	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals/primates"
)

func (row adultHumanStudy) poseRig(from seed) (poseRig, error) {
	body, err := row.body(from)
	if err != nil {
		return poseRig{}, err
	}
	groom, err := row.groom(body, from)
	if err != nil {
		return poseRig{}, err
	}
	return poseRig{
		bones: body.RestSkeleton(),
		pose: func(key skeleton.Pose) (good, skeleton.BoneFrames, error) {
			pose, err := body.Pose(key)
			if err != nil {
				return good{}, nil, err
			}
			made, err := adultHumanFrame(pose, primates.Gaze{}, groom)
			return made, pose, err
		},
	}, nil
}
