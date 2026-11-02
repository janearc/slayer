package thing

import (
	"time"

	"github.com/janearc/slayer/maths"
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
	Motion     Motion // nil is still, at the world's zero
	// EyeTime and Eye are the eye's four-velocity when the eye is
	// inside the reach, as the hole's march hands it back: the time
	// part, and the space part in the field's own axes. Nought is an
	// eye held still, which a camera that falls sets each frame.
	EyeTime float64
	Eye     maths.Vec
}

// At is where the field's subspace is at time t.
func (f *Field) At(at time.Duration) maths.Subspace {
	if f.Motion == nil {
		return maths.Subspace{}
	}
	return f.Motion.At(at)
}

// InHole takes a point or direction in the field's own space into the
// hole's maths, whose spin axis is z where the field's is up: a turn,
// so a spinning hole keeps its handedness.
func (f *Field) InHole(v maths.Vec) [3]float64 {
	return [3]float64{v.X, -v.Z, v.Y}
}

// FromHole is InHole undone: a camera the hole's maths carries comes
// back through it.
func (f *Field) FromHole(v [3]float64) maths.Vec {
	return maths.V(v[0], v[2], -v[1])
}
