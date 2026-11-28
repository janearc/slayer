package skeleton

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// skinMotion pairs the unit rotation with twice its dual part. A factor
// of two is removed and restored together in forming and using the pair.
type skinMotion struct {
	rotation quaternion
	offset   quaternion
}

func skinMotions(frames []Frame) []skinMotion {
	motions := make([]skinMotion, len(frames))
	for i, frame := range frames {
		rotation := skinRotation(frame.turn)
		shift := quaternion{v: frame.shift}
		motions[i] = skinMotion{rotation: rotation, offset: shift.after(rotation)}
	}
	return motions
}

// skinRotation converts a right-handed orientation to a unit quaternion.
// Choosing the largest diagonal term near a half-turn avoids cancellation.
func skinRotation(orientation maths.Orientation) quaternion {
	x, y, z := orientation.X, orientation.Y, orientation.Z
	trace := x.X + y.Y + z.Z
	var q quaternion
	switch {
	case trace > 0:
		size := math.Sqrt(1 + trace)
		q = quaternion{w: size / 2, v: maths.V(y.Z-z.Y, z.X-x.Z, x.Y-y.X).Scale(1 / (2 * size))}
	case x.X >= y.Y && x.X >= z.Z:
		size := math.Sqrt(1 + x.X - y.Y - z.Z)
		q = quaternion{w: (y.Z - z.Y) / (2 * size), v: maths.V(size*size, x.Y+y.X, x.Z+z.X).Scale(1 / (2 * size))}
	case y.Y >= z.Z:
		size := math.Sqrt(1 + y.Y - x.X - z.Z)
		q = quaternion{w: (z.X - x.Z) / (2 * size), v: maths.V(x.Y+y.X, size*size, y.Z+z.Y).Scale(1 / (2 * size))}
	default:
		size := math.Sqrt(1 + z.Z - x.X - y.Y)
		q = quaternion{w: (x.Y - y.X) / (2 * size), v: maths.V(x.Z+z.X, y.Z+z.Y, size*size).Scale(1 / (2 * size))}
	}
	return q.scaled(1 / math.Sqrt(q.dot(q)))
}

func (q quaternion) dot(other quaternion) float64 {
	return q.w*other.w + q.v.Dot(other.v)
}

func (q quaternion) scaled(by float64) quaternion {
	return quaternion{w: q.w * by, v: q.v.Scale(by)}
}

func (q quaternion) plus(other quaternion) quaternion {
	return quaternion{w: q.w + other.w, v: q.v.Add(other.v)}
}

// rigidCarry blends rotations in the hemisphere of the dominant bone, then
// restores a unit rotation and extracts its paired translation.
func rigidCarry(point skinPoint, motions []skinMotion) maths.Vec {
	dominant := point.hold[0]
	for _, each := range point.hold[1:] {
		if each.weight > dominant.weight {
			dominant = each
		}
	}
	reference := motions[dominant.bone].rotation
	var rotation, offset quaternion
	for _, each := range point.hold {
		motion := motions[each.bone]
		// Divide before blending to keep the largest contribution at unit weight.
		weight := each.weight / dominant.weight
		if motion.rotation.dot(reference) < 0 {
			weight = -weight
		}
		rotation = rotation.plus(motion.rotation.scaled(weight))
		offset = offset.plus(motion.offset.scaled(weight))
	}
	inverseLength := 1 / math.Sqrt(rotation.dot(rotation))
	rotation = rotation.scaled(inverseLength)
	conjugate := quaternion{w: rotation.w, v: rotation.v.Scale(-1)}
	shift := offset.scaled(inverseLength).after(conjugate).v
	rest := quaternion{v: point.rest}
	turned := rotation.after(rest).after(conjugate).v
	return turned.Add(shift)
}
