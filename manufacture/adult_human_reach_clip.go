package manufacture

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals/primates"
	"github.com/janearc/slayer/maths"
)

// adultHumanReachClip performs the existing studio reach, holds, and
// returns to neutral. The pelvis and leg joints hold their rest turns.
type adultHumanReachClip struct{}

func (adultHumanReachClip) made(from seed) (good, error) {
	return (adultHumanStudy{}).made(from)
}

func (adultHumanReachClip) extent(seed) (maths.Box, error) {
	// Allow for the maximum arm swing and the studio's short groom.
	return maths.Box{Min: maths.V(-.66, -.07, -.40), Max: maths.V(.66, 1.80, .40)}, nil
}

func (adultHumanReachClip) prepareDeformation(from seed, detail string) (func(time.Duration) (good, error), error) {
	if detail != "quick" {
		return nil, fmt.Errorf("adult human reach: unsupported complexity %q", detail)
	}
	study := adultHumanStudy{}
	body, err := study.body(from)
	if err != nil {
		return nil, err
	}
	reach := adultReachPose()
	clip, err := body.RestSkeleton().NewPoseClip([]skeleton.PoseKey{
		{At: 0},
		{At: 1400 * time.Millisecond, Pose: reach},
		{At: 2 * time.Second, Pose: reach},
		{At: 3200 * time.Millisecond},
	})
	if err != nil {
		return nil, err
	}
	animation, err := body.WithClip(clip.WithSmoothTiming())
	if err != nil {
		return nil, err
	}
	groom, err := study.groom(body, from)
	if err != nil {
		return nil, err
	}
	return func(at time.Duration) (good, error) {
		pose, err := animation.At(at)
		if err != nil {
			return good{}, err
		}
		return adultHumanFrame(pose, primates.Gaze{}, groom)
	}, nil
}
