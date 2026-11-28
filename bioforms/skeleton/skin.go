package skeleton

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// SkinBlend chooses how a point shared by several bones is moved.
type SkinBlend uint8

const (
	// LinearSkin averages where the bones would put the point, as Carry does.
	LinearSkin SkinBlend = iota
	// RigidSkin blends bone motions as dual quaternions.
	RigidSkin
)

// SkinInput supplies rest vertices and a rule assigning bone influences in
// the skeleton's order. Weigh is called once per point during binding.
type SkinInput struct {
	Points []maths.Vec
	Weigh  func(index int, rest maths.Vec) []Influence
	Blend  SkinBlend
}

type skinPoint struct {
	rest maths.Vec
	hold []Influence
}

// Skin keeps a rest surface's vertices bound to a snapshot of its skeleton.
// The maker retains the corresponding faces and paint coordinates.
type Skin struct {
	bones  Skeleton
	points []skinPoint
	blend  SkinBlend
}

// BindSkin checks the rest points and their weights, then owns a copy of
// the binding. A point must have at least one positive influence. Weights
// need not sum to one.
func (skeleton *Skeleton) BindSkin(input SkinInput) (*Skin, error) {
	if skeleton.Len() == 0 || len(input.Points) == 0 || input.Weigh == nil {
		return nil, fmt.Errorf("skeleton: binding needs bones, points and a weight rule")
	}
	if input.Blend != LinearSkin && input.Blend != RigidSkin {
		return nil, fmt.Errorf("skeleton: unknown skin blend %d", input.Blend)
	}
	bones := skeleton.snapshot()
	points := make([]skinPoint, len(input.Points))
	for i, rest := range input.Points {
		if !skinPointFinite(rest) {
			return nil, fmt.Errorf("skeleton: skin point %d is not finite", i)
		}
		hold := input.Weigh(i, rest)
		if err := checkSkinHold(hold, bones.Len()); err != nil {
			return nil, fmt.Errorf("skeleton: skin point %d: %w", i, err)
		}
		points[i] = skinPoint{rest: rest, hold: slices.Clone(hold)}
	}
	return &Skin{bones: bones, points: points, blend: input.Blend}, nil
}

func checkSkinHold(hold []Influence, count int) error {
	total := 0.0
	for _, each := range hold {
		if each.bone < 0 || each.bone >= count {
			return fmt.Errorf("bone index %d outside %d bones", each.bone, count)
		}
		if !skinNumber(each.weight) || each.weight < 0 {
			return fmt.Errorf("invalid weight %v for bone %d", each.weight, each.bone)
		}
		total += each.weight
	}
	if total == 0 || !skinNumber(total) {
		return fmt.Errorf("total influence must be positive and finite")
	}
	return nil
}

func skinNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func skinPointFinite(point maths.Vec) bool {
	return skinNumber(point.X) && skinNumber(point.Y) && skinNumber(point.Z)
}

// RestSkeleton returns an independent copy of the skeleton used to bind
// the skin. Use it to inspect bone names and rest frames or author a clip.
func (skin *Skin) RestSkeleton() *Skeleton {
	bones := skin.bones.snapshot()
	return &bones
}

// Influences returns a copy of the weights for one vertex in rest order.
func (skin *Skin) Influences(index int) []Influence {
	return slices.Clone(skin.points[index].hold)
}

// SkinPose shares one set of posed frames between the skin and any rigid
// attachments.
type SkinPose struct {
	skin   *Skin
	frames []Frame
}

// Pose checks the named turns and places the bones for this skin.
func (skin *Skin) Pose(pose Pose) (*SkinPose, error) {
	if err := skin.bones.Check(pose); err != nil {
		return nil, err
	}
	for _, bone := range skin.bones.bones {
		turn, turned := pose.turns[bone.name]
		if turned && (!skinNumber(turn.angle) || !skinNumber(turn.axis.Len())) {
			return nil, fmt.Errorf("skeleton: turn of %q is not finite", bone.name)
		}
	}
	if !rigidSkinFrame(pose.root) {
		return nil, fmt.Errorf("skeleton: skin needs a finite rigid root frame")
	}
	frames := skin.bones.Frames(pose)
	for i, frame := range frames {
		if !skinPointFinite(frame.shift) {
			return nil, fmt.Errorf("skeleton: non-finite position for %q", skin.bones.Bone(i).Name())
		}
	}
	return &SkinPose{skin: skin, frames: frames}, nil
}

// rigidSkinFrame checks that a caller-supplied root is a right-handed
// rotation with a finite translation. Zero Orientation denotes identity.
func rigidSkinFrame(frame Frame) bool {
	if !skinPointFinite(frame.shift) {
		return false
	}
	if frame.turn == (maths.Orientation{}) {
		return true
	}
	const tolerance = 1e-9
	x, y, z := frame.turn.X, frame.turn.Y, frame.turn.Z
	return math.Abs(x.Len()-1) <= tolerance &&
		math.Abs(y.Len()-1) <= tolerance &&
		z.Near(x.Cross(y), tolerance) &&
		math.Abs(x.Dot(y)) <= tolerance
}

// Frame is the rest-to-posed rigid motion of a named bone. Its motion
// can carry an attachment point or bring along a separate rigid surface.
func (posed *SkinPose) Frame(name string) (Frame, bool) {
	i, found := posed.skin.bones.Find(name)
	if !found {
		return Frame{}, false
	}
	return posed.frames[i], true
}

// Points returns posed vertices in their original order, so face indices,
// rest-space paint coordinates and barycentric attachments stay paired.
func (posed *SkinPose) Points() []maths.Vec {
	points := make([]maths.Vec, len(posed.skin.points))
	if posed.skin.blend == RigidSkin {
		frames := skinMotions(posed.frames)
		for i, point := range posed.skin.points {
			points[i] = rigidCarry(point, frames)
		}
		return points
	}
	for i, point := range posed.skin.points {
		points[i] = Carry(point.rest, posed.frames, point.hold)
	}
	return points
}
