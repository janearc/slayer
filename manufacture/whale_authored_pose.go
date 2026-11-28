package manufacture

import "github.com/janearc/slayer/bioforms/skeleton"

func (row whaleSwim) poseRig(seed) (poseRig, error) {
	bones, err := row.profile.RestSkeleton()
	if err != nil {
		return poseRig{}, err
	}
	return poseRig{
		bones: bones,
		pose: func(key skeleton.Pose) (good, skeleton.BoneFrames, error) {
			posed, err := row.profile.Pose(key)
			if err != nil {
				return good{}, nil, err
			}
			parts, err := posed.Surface()
			if err != nil {
				return good{}, nil, err
			}
			return good{native: parts}, posed, nil
		},
	}, nil
}
