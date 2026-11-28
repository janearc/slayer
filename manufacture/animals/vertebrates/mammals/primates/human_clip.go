package primates

import (
	"time"

	"github.com/janearc/slayer/bioforms/skeleton"
)

// RestSkeleton gives an independent copy of the bound human's rest bones.
func (human *Human) RestSkeleton() *skeleton.Skeleton { return human.binding.RestSkeleton() }

// HumanClip holds a pose clip paired with the native human that it poses.
// Samples include the human's soft tissue and the frames used by its eyes.
type HumanClip struct {
	human *Human
	skin  *skeleton.SkinClip
}

// WithClip pairs the human with a clip authored on its rest skeleton.
func (human *Human) WithClip(clip *skeleton.PoseClip) (*HumanClip, error) {
	skin, err := clip.WithSkin(human.binding)
	if err != nil {
		return nil, err
	}
	return &HumanClip{human: human, skin: skin}, nil
}

// At deforms the native exterior at elapsed time and provides the same
// bone frames to attachments. The clip holds its end poses.
func (clip *HumanClip) At(elapsed time.Duration) (*HumanPose, error) {
	placed, err := clip.skin.At(elapsed)
	if err != nil {
		return nil, err
	}
	return clip.human.poseFromSkin(placed), nil
}
