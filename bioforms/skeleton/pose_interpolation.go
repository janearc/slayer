package skeleton

import (
	"math"

	"github.com/janearc/slayer/maths"
)

func betweenPoses(from, to Pose, share float64) Pose {
	turns := make(map[string]Turn, len(from.turns)+len(to.turns))
	for name, start := range from.turns {
		turns[name] = betweenTurns(start, to.turns[name], share)
	}
	for name, end := range to.turns {
		if _, present := from.turns[name]; !present {
			turns[name] = betweenTurns(Turn{}, end, share)
		}
	}
	root := from.root
	if from.root.turn != to.root.turn {
		start := clipRotation(from.root.turn)
		end := clipRotation(to.root.turn)
		turn := start.towards(end, share).turn()
		root.turn = maths.Rotation(turn.axis, turn.angle)
	}
	if from.root.shift != to.root.shift {
		root.shift = from.root.shift.Scale(1 - share).Add(to.root.shift.Scale(share))
	}
	return Pose{turns: turns, root: root}
}

func betweenTurns(from, to Turn, share float64) Turn {
	if from == to {
		return from
	}
	start := skinRotation(maths.Rotation(from.axis, from.angle))
	end := skinRotation(maths.Rotation(to.axis, to.angle))
	return start.towards(end, share).turn()
}

func clipRotation(turn maths.Orientation) quaternion {
	if turn == (maths.Orientation{}) {
		return quaternion{w: 1}
	}
	return skinRotation(turn)
}

// towards interpolates rotations on the shorter arc. For nearby rotations
// it normalizes a linear blend to avoid division by a tiny sine.
func (q quaternion) towards(other quaternion, share float64) quaternion {
	dot := q.dot(other)
	if dot < 0 {
		other = other.scaled(-1)
		dot = -dot
	}
	const nearby = 0.9995
	if dot > nearby {
		blend := q.scaled(1 - share).plus(other.scaled(share))
		return blend.scaled(1 / math.Sqrt(blend.dot(blend)))
	}
	angle := math.Acos(dot)
	sine := math.Sin(angle)
	return q.scaled(math.Sin((1-share)*angle) / sine).
		plus(other.scaled(math.Sin(share*angle) / sine))
}

// Place carries an attachment from its rest placement to its posed
// placement, keeping its own scale. Use a placement measured in the
// skeleton's rest coordinates, including its orientation.
func (frame Frame) Place(rest maths.Subspace) maths.Subspace {
	rest.Origin = frame.Apply(rest.Origin)
	rest.Orientation = frame.turn.Then(rest.Orientation)
	return rest
}
