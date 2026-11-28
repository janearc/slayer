package skeleton

import (
	"fmt"
	"slices"
	"time"

	"github.com/janearc/slayer/maths"
)

// SkinClip applies a pose clip to a skin bound to the same rest skeleton.
type SkinClip struct {
	clip *PoseClip
	skin *Skin
}

// WithSkin pairs a clip and a binding whose ordered bones match, including
// their rest geometry and parents. Weights stay with the original binding.
func (clip *PoseClip) WithSkin(skin *Skin) (*SkinClip, error) {
	if clip == nil || len(clip.keys) == 0 || skin == nil {
		return nil, fmt.Errorf("skeleton: need a pose clip and bound skin")
	}
	if !slices.Equal(clip.bones.bones, skin.bones.bones) {
		return nil, fmt.Errorf("skeleton: skin and clip use different rest skeletons")
	}
	return &SkinClip{clip: clip, skin: skin}, nil
}

// Duration is the elapsed time of the underlying clip's last key.
func (binding *SkinClip) Duration() time.Duration { return binding.clip.Duration() }

// RestPoints returns a copy of the bound points in the original vertex order.
func (binding *SkinClip) RestPoints() []maths.Vec {
	points := make([]maths.Vec, len(binding.skin.points))
	for i, point := range binding.skin.points {
		points[i] = point.rest
	}
	return points
}

// At samples once so the skin's vertices and named attachments use the
// same bone motions at this instant.
func (binding *SkinClip) At(elapsed time.Duration) (*SkinPose, error) {
	sample, err := binding.clip.Sample(elapsed)
	if err != nil {
		return nil, err
	}
	return &SkinPose{skin: binding.skin, frames: sample.frames}, nil
}
