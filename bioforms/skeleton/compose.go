package skeleton

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// composeTiny is a sine of half an angle below which a turn is no turn.
const composeTiny = 1e-12

// quaternion is a turn as a unit quaternion: w the cosine of half the
// angle, v the axis times its sine.
type quaternion struct {
	w float64
	v maths.Vec
}

// quaternionOf is the turn as a quaternion.
func quaternionOf(turn Turn) quaternion {
	if turn.axis.Len() < composeTiny {
		return quaternion{w: 1}
	}
	half := turn.angle / 2
	return quaternion{w: math.Cos(half), v: turn.axis.Unit().Scale(math.Sin(half))}
}

// after is the turn q made after first: one turn that does both.
func (q quaternion) after(first quaternion) quaternion {
	return quaternion{w: q.w*first.w - q.v.Dot(first.v),
		v: first.v.Scale(q.w).Add(q.v.Scale(first.w)).Add(q.v.Cross(first.v))}
}

// turn is the quaternion as an axis and an angle.
func (q quaternion) turn() Turn {
	sine := q.v.Len()
	if sine < composeTiny {
		return Turn{axis: maths.V(1, 0, 0)}
	}
	return Turn{axis: q.v.Scale(1 / sine), angle: 2 * math.Atan2(sine, q.w)}
}

// Compose is the pose with the joint at the head of the bone named name
// turned by turn after any turn it already had, where With would replace
// it.
func (pose Pose) Compose(name string, turn Turn) Pose {
	had, ok := pose.turns[name]
	if !ok {
		return pose.With(name, turn)
	}
	both := quaternionOf(turn).after(quaternionOf(had))
	return pose.With(name, both.turn())
}

// Aim is the pose with the bone named name turned at its head, after what
// pose already turns it by, so that in the pose it points from its head
// along way. A bone already pointing along way, or no bone of that name,
// leaves the pose as it is.
func (skeleton *Skeleton) Aim(pose Pose, name string, way maths.Vec) Pose {
	at, ok := skeleton.byName[name]
	if !ok {
		return pose
	}
	frames := skeleton.Frames(pose)
	bone := skeleton.bones[at]
	now := frames[at].Direction(bone.tail.Sub(bone.head)).Unit()
	want := way.Unit()
	axis := now.Cross(want)
	if axis.Len() < composeTiny {
		return pose
	}
	capped := math.Min(1, now.Dot(want))
	angle := math.Acos(math.Max(-1, capped))
	parent := pose.root
	if bone.parent != noParent {
		parent = frames[bone.parent]
	}
	inParent := parent.turn.In(axis)
	return pose.Compose(name, NewTurn(inParent, angle))
}
