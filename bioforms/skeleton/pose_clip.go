package skeleton

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	"github.com/janearc/slayer/maths"
)

// PoseKey is a complete pose at an elapsed time. A joint omitted from a
// key is at rest relative to its parent at that key.
type PoseKey struct {
	At   time.Duration
	Pose Pose
}

// PoseClip holds authored keys and a snapshot of their skeleton. Sampling
// needs no playhead and can be done in any order or concurrently.
type PoseClip struct {
	bones  Skeleton
	keys   []PoseKey
	smooth bool
}

// NewPoseClip checks the skeleton and keys, then keeps its own copies.
// The first key must be at zero, and times must strictly increase.
func (skeleton *Skeleton) NewPoseClip(keys []PoseKey) (*PoseClip, error) {
	if skeleton == nil || skeleton.Len() == 0 {
		return nil, fmt.Errorf("skeleton: a pose clip needs bones")
	}
	if len(keys) == 0 || keys[0].At != 0 {
		return nil, fmt.Errorf("skeleton: a pose clip needs a key at zero")
	}
	for _, bone := range skeleton.bones {
		length := bone.Length()
		if !skinPointFinite(bone.head) || !skinPointFinite(bone.tail) ||
			!skinNumber(length) || length <= 0 {
			return nil, fmt.Errorf("skeleton: invalid rest bone %q", bone.name)
		}
	}
	for i, key := range keys {
		if i > 0 && key.At <= keys[i-1].At {
			return nil, fmt.Errorf("skeleton: key %d at %v must follow %v",
				i+1, key.At, keys[i-1].At)
		}
		if err := skeleton.checkClipKey(key.Pose); err != nil {
			return nil, fmt.Errorf("skeleton: key %d at %v: %w", i+1, key.At, err)
		}
	}
	bones := skeleton.snapshot()
	return &PoseClip{bones: bones, keys: slices.Clone(keys)}, nil
}

func (skeleton *Skeleton) checkClipKey(pose Pose) error {
	if err := skeleton.Check(pose); err != nil {
		return err
	}
	if !rigidSkinFrame(pose.root) {
		return fmt.Errorf("root must be a finite rigid frame")
	}
	for _, bone := range skeleton.bones {
		turn := pose.turns[bone.name]
		if !skinNumber(turn.angle) || !skinNumber(turn.axis.Len()) {
			return fmt.Errorf("turn of %q is not finite", bone.name)
		}
	}
	_, err := skeleton.clipFrames(pose)
	return err
}

// Duration is the elapsed time of the last key.
func (clip *PoseClip) Duration() time.Duration {
	if clip == nil || len(clip.keys) == 0 {
		return 0
	}
	return clip.keys[len(clip.keys)-1].At
}

// WithSmoothTiming returns a clip that eases in and out of each key. It
// shares the original clip's immutable skeleton and keys. Use it for a
// sequence whose root and joints should come to rest at every key.
func (clip PoseClip) WithSmoothTiming() *PoseClip {
	clip.smooth = true
	return &clip
}

// At gives the pose at elapsed time. Before the first key and after the
// last it holds that endpoint. Each rotation takes its shortest arc and
// root translation follows a straight path. A zero clip gives the rest pose.
func (clip *PoseClip) At(elapsed time.Duration) Pose {
	if clip == nil || len(clip.keys) == 0 {
		return Pose{}
	}
	i, exact := slices.BinarySearchFunc(clip.keys, elapsed,
		func(key PoseKey, target time.Duration) int {
			return cmp.Compare(key.At, target)
		})
	switch {
	case i == len(clip.keys):
		return clip.keys[i-1].Pose
	case exact || i == 0:
		return clip.keys[i].Pose
	}
	from, to := clip.keys[i-1], clip.keys[i]
	share := float64(elapsed-from.At) / float64(to.At-from.At)
	if clip.smooth {
		share = maths.Smooth(0, 1, share)
	}
	return betweenPoses(from.Pose, to.Pose, share)
}

// PoseSample is one clip sample's rest-to-posed bone motions.
type PoseSample struct {
	byName map[string]int
	frames []Frame
}

// Sample gives named bone frames for this instant, including when the
// maker has no bound skin. For bound skin, Skin.Pose shares its frames
// between the surface and its rigid attachments.
func (clip *PoseClip) Sample(elapsed time.Duration) (*PoseSample, error) {
	if clip == nil || len(clip.keys) == 0 {
		return nil, fmt.Errorf("skeleton: no pose clip to sample")
	}
	frames, err := clip.bones.clipFrames(clip.At(elapsed))
	if err != nil {
		return nil, fmt.Errorf("skeleton: sample at %v: %w", elapsed, err)
	}
	return &PoseSample{byName: clip.bones.byName, frames: frames}, nil
}

func (skeleton *Skeleton) clipFrames(pose Pose) ([]Frame, error) {
	frames := skeleton.Frames(pose)
	for i, frame := range frames {
		if !skinPointFinite(frame.shift) {
			return nil, fmt.Errorf("non-finite bone position for %q", skeleton.bones[i].name)
		}
	}
	return frames, nil
}

// Frame gives the motion from the skeleton's rest coordinates to its
// posed coordinates for the named bone.
func (sample *PoseSample) Frame(name string) (Frame, bool) {
	i, known := sample.byName[name]
	if !known {
		return Frame{}, false
	}
	return sample.frames[i], true
}
