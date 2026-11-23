package skeleton

import (
	"maps"

	"github.com/janearc/slayer/maths"
)

// Frame is a rigid motion: a turn, then a shift. The zero Frame moves
// nothing.
type Frame struct {
	turn  maths.Orientation
	shift maths.Vec
}

// NewFrame is the motion that turns by turn and then shifts by shift.
func NewFrame(turn maths.Orientation, shift maths.Vec) Frame {
	return Frame{turn: turn, shift: shift}
}

// Apply is point p carried by the motion.
func (frame Frame) Apply(p maths.Vec) maths.Vec {
	return frame.turn.Out(p).Add(frame.shift)
}

// Direction is direction v turned by the motion; a shift does not move
// a direction.
func (frame Frame) Direction(v maths.Vec) maths.Vec {
	return frame.turn.Out(v)
}

// then is inner followed by this frame: one motion that does both.
func (frame Frame) then(inner Frame) Frame {
	return Frame{turn: frame.turn.Then(inner.turn),
		shift: frame.Apply(inner.shift)}
}

// TurnAbout is the motion that turns space by angle about the line
// through point along axis, by the right-hand rule.
func TurnAbout(point, axis maths.Vec, angle float64) Frame {
	turn := maths.Rotation(axis, angle)
	return Frame{turn: turn, shift: point.Sub(turn.Out(point))}
}

// Turn is a joint turned by an angle, in radians, about an axis given in
// the skeleton's rest space, by the right-hand rule.
type Turn struct {
	axis  maths.Vec
	angle float64
}

// NewTurn is a turn by angle about axis.
func NewTurn(axis maths.Vec, angle float64) Turn {
	return Turn{axis: axis, angle: angle}
}

// Pose is the turn at each named joint and the motion that carries the
// whole skeleton, its root. The zero Pose is the rest pose.
type Pose struct {
	turns map[string]Turn
	root  Frame
}

// With is the pose with the joint at the head of the bone named name
// turned by turn, in place of any turn it had.
func (pose Pose) With(name string, turn Turn) Pose {
	turns := maps.Clone(pose.turns)
	if turns == nil {
		turns = map[string]Turn{}
	}
	turns[name] = turn
	return Pose{turns: turns, root: pose.root}
}

// Carried is the pose with the whole skeleton carried by root.
func (pose Pose) Carried(root Frame) Pose {
	return Pose{turns: pose.turns, root: root}
}

// Frames is the motion of each bone in pose, in the skeleton's order:
// frame i takes a rest point that moves with bone i to its posed place.
// Each bone moves as its parent does, then turns at its own head.
func (skeleton *Skeleton) Frames(pose Pose) []Frame {
	frames := make([]Frame, len(skeleton.bones))
	for i, bone := range skeleton.bones {
		parent := pose.root
		if bone.parent != noParent {
			parent = frames[bone.parent]
		}
		local := Frame{}
		if turn, ok := pose.turns[bone.name]; ok {
			local = TurnAbout(bone.head, turn.axis, turn.angle)
		}
		frames[i] = parent.then(local)
	}
	return frames
}

// Influence is how much one bone moves one point of skin.
type Influence struct {
	bone   int
	weight float64
}

// NewInfluence is bone moving a point by share weight.
func NewInfluence(bone int, weight float64) Influence {
	return Influence{bone: bone, weight: weight}
}

// Bone is the place of the bone that moves the point.
func (each Influence) Bone() int { return each.bone }

// Weight is the share by which it moves it.
func (each Influence) Weight() float64 { return each.weight }

// Carry is rest point p moved by the bones that influence it, each by
// its share of the whole weight: the blend of where each would put it.
// A point no bone influences stays where it is.
func Carry(p maths.Vec, frames []Frame, influences []Influence) maths.Vec {
	total := 0.
	sum := maths.Vec{}
	for _, each := range influences {
		if each.weight <= 0 {
			continue
		}
		total += each.weight
		sum = sum.Add(frames[each.bone].Apply(p).Scale(each.weight))
	}
	if total == 0 {
		return p
	}
	return sum.Scale(1 / total)
}
