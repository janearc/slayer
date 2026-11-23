package thing

import (
	"time"

	"github.com/janearc/slayer/maths"
)

// The places and the count of a hole's three axes, true by definition.
const (
	holeAxes   = 3 // defined: the axes of space
	holeFirst  = 0 // defined: the first place in a list
	holeSecond = 1 // defined: the second place in a list
	holeThird  = 2 // defined: the third place in a list
)

// Field is a black hole's gravity: within Reach of its centre a ray is bent,
// marched as a geodesic of a kerr hole by celestial's hole, and outside it is
// straight, since the bending there is small.
//
// Mass is the hole's mass as a length in the field's own units, half the
// schwarzschild radius; Spin its angular momentum per unit mass, from zero up
// to just under Mass; the spin axis is the field's up. Light from a volume
// inside the reach arrives shifted.
type Field struct {
	Mass, Spin float64
	Reach      float64
	// LetGo is the radius, in masses, past which a ray heading outward is
	// let go straight; nought is the tracer's own, twelve. A shot that
	// wants the bending to the reach sets it at the reach: past twelve
	// the straight run is up to eight degrees short, which steps a disc's
	// far rim and creases its face.
	LetGo  float64
	Motion Motion // nil is still, at the world's zero
	// EyeTime and Eye are the eye's four-velocity when the eye is
	// inside the reach, as the hole's march hands it back: the time
	// part, and the space part in the field's own axes. Nought is an
	// eye held still, which a camera that falls sets each frame.
	EyeTime float64
	Eye     maths.Vec
}

// At is where the field's subspace is at time at.
func (field *Field) At(at time.Duration) maths.Subspace {
	if field.Motion == nil {
		return maths.Local()
	}
	return field.Motion.At(at)
}

// InHole takes a point or direction in the field's own space into the
// hole's maths, whose spin axis is z where the field's is up: a turn,
// so a spinning hole keeps its handedness.
func (field *Field) InHole(vector maths.Vec) [holeAxes]float64 {
	return [holeAxes]float64{vector.X, -vector.Z, vector.Y}
}

// FromHole is InHole undone: a camera the hole's maths carries comes
// back through it.
func (field *Field) FromHole(vector [holeAxes]float64) maths.Vec {
	return maths.V(vector[holeFirst], vector[holeThird], -vector[holeSecond])
}
