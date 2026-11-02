package emit

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// ErrDistant is a distant light with no way to come from, or a light
// that is less than none or has no end.
var ErrDistant = errors.New("emit: no such distant light")

// Distant is light from so far away that it comes one way, as strong,
// to every point: a sun seen from a planet. its light is what it lands
// on a surface square to it.
type Distant struct {
	toward maths.Vec
	light  maths.Light
}

// NewDistant is a distant light that lies a way from everything and
// lands so much light, or why there is none.
func NewDistant(toward maths.Vec, light maths.Light) (Distant, error) {
	way := toward.Len()
	parts := []float64{toward.X, toward.Y, toward.Z, light.R, light.G,
		light.B}
	for _, part := range parts {
		if math.IsNaN(part) || math.IsInf(part, 0) {
			return Distant{}, ErrDistant
		}
	}
	if way == 0 || light.R < 0 || light.G < 0 || light.B < 0 {
		return Distant{}, ErrDistant
	}
	return Distant{toward: toward.Scale(1 / way), light: light}, nil
}

// Light is the way to the light, an endless distance, and its light,
// the same at every point.
func (sun Distant) Light(_, _ maths.Vec) (maths.Vec, float64,
	maths.Light) {
	return sun.toward, math.Inf(1), sun.light
}

// a Distant is a lamp a tracer can use as it is
var _ thing.Lamp = Distant{}
