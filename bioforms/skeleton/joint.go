package skeleton

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// jointAxes is how many ways a joint turns: flex about the bone's own
// across axis, bend about its own up axis, and twist about its length.
const jointAxes = 3

// The three ways a joint turns, by their place in a joint's range.
const (
	Flex  = 0 // about the bone's across axis: up and down for a column
	Bend  = 1 // about the bone's up axis: side to side for a column
	Twist = 2 // about the bone's own length
)

// alongUp is how near a bone may lie to straight up or down, as a
// cosine, before its axes are measured from the world's x instead.
const alongUp = .95

var (
	errUnjointed = errors.New("skeleton: a bone has no joint range")
	errPastJoint = errors.New("skeleton: a turn is past its joint")
	errNoBone    = errors.New("skeleton: no such bone")
)

// Joint is how far a bone may turn at its head from rest, in radians,
// low and high on each of flex, bend and twist.
type Joint struct{ low, high [jointAxes]float64 }

// Fixed is a joint that does not turn, as a fused bone or a rib.
func Fixed() Joint { return Joint{} }

// Hinge is a joint that flexes from low to high and does not bend or
// twist, as an elbow or a jaw.
func Hinge(low, high float64) Joint {
	return Joint{low: [jointAxes]float64{low},
		high: [jointAxes]float64{high}}
}

// Limits is a joint with a range of its own on each axis.
func Limits(flexLow, flexHigh, bendLow, bendHigh, twistLow,
	twistHigh float64,
) Joint {
	return Joint{low: [jointAxes]float64{flexLow, bendLow, twistLow},
		high: [jointAxes]float64{flexHigh, bendHigh, twistHigh}}
}

// allows is nil when every turn is a number within the joint's range.
func (joint Joint) allows(turn [jointAxes]float64) error {
	for axis, angle := range turn {
		low, high := joint.low[axis], joint.high[axis]
		if math.IsNaN(angle) || angle < low || angle > high {
			return fmt.Errorf("%w: axis %d turn %.3g outside [%.3g, %.3g]",
				errPastJoint, axis, angle, low, high)
		}
	}
	return nil
}

// Joints is each bone's joint range, by the bone's name. A bone not
// named has none and is not turned through Joints.
type Joints struct{ ranges map[string]Joint }

// With is the joints with the named bone's range set; the joints it is
// made from are unchanged.
func (joints Joints) With(name string, joint Joint) Joints {
	ranges := make(map[string]Joint, len(joints.ranges)+1)
	for each, held := range joints.ranges {
		ranges[each] = held
	}
	ranges[name] = joint
	return Joints{ranges: ranges}
}

// Limit is how far the named bone turns on one axis, low and high in
// radians, and whether it has a range and the axis is one of the three.
func (joints Joints) Limit(name string, axis int) (float64, float64, bool) {
	joint, found := joints.ranges[name]
	if !found || axis < 0 || axis >= jointAxes {
		return 0, 0, false
	}
	return joint.low[axis], joint.high[axis], true
}

// Turned is the pose with the named bone turned at its head by twist,
// then bend, then flex, on the bone's own rest axes, in place of any turn
// it had, made with Compose. A bone that is not here, has no range, or a
// turn past its range is refused.
func (joints Joints) Turned(skeleton *Skeleton, pose Pose, name string,
	flex, bend, twist float64,
) (Pose, error) {
	at, found := skeleton.Find(name)
	if !found {
		return pose, fmt.Errorf("%w: %s", errNoBone, name)
	}
	joint, jointed := joints.ranges[name]
	if !jointed {
		return pose, fmt.Errorf("%w: %s", errUnjointed, name)
	}
	if err := joint.allows([jointAxes]float64{flex, bend,
		twist}); err != nil {
		return pose, fmt.Errorf("%s: %w", name, err)
	}
	across, up, length := BoneAxes(skeleton.Bone(at))
	return pose.With(name, NewTurn(length, twist)).
		Compose(name, NewTurn(up, bend)).
		Compose(name, NewTurn(across, flex)), nil
}

// BoneAxes is a bone's own three axes at rest: across it (flex turns
// about this), up from it (bend turns about this) and along it from tail
// to head (twist turns about this). Up is as near the world's up as the
// bone allows; a bone near upright takes the world's x instead.
func BoneAxes(bone Bone) (across, up, length maths.Vec) {
	length = bone.head.Sub(bone.tail).Unit()
	reference := maths.Up
	if math.Abs(length.Dot(reference)) > alongUp {
		reference = maths.XAxis
	}
	across = reference.Cross(length).Unit()
	return across, length.Cross(across), length
}
