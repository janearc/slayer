package frame

import (
	"fmt"

	"github.com/janearc/slayer/maths"
)

// Relative fixes one render origin, normally the current world camera position.
// Every positional renderer input must use that same value before float32.
type Relative struct{ origin maths.Vec }

// NewRelative creates a translation without changing the scene's fixed anchor.
func NewRelative(origin maths.Vec) (Relative, error) {
	if !finitePoint(origin) {
		return Relative{}, fmt.Errorf("frame: non-finite render origin")
	}
	return Relative{origin: origin}, nil
}

// Origin is the world position subtracted from this render's positional inputs.
func (relative Relative) Origin() maths.Vec { return relative.origin }

// Point translates a world position while retaining float64 precision.
// Directions, dimensions and relative displacements must not pass through it.
func (relative Relative) Point(world maths.Vec) (maths.Vec, error) {
	if !finitePoint(world) {
		return maths.Vec{}, fmt.Errorf("frame: non-finite world position")
	}
	point := world.Sub(relative.origin)
	if !finitePoint(point) {
		return maths.Vec{}, fmt.Errorf("frame: render-relative position overflow")
	}
	return point, nil
}

// Restore puts a render-local point back in the same fixed world frame.
func (relative Relative) Restore(local maths.Vec) (maths.Vec, error) {
	if !finitePoint(local) {
		return maths.Vec{}, fmt.Errorf("frame: non-finite render-local position")
	}
	point := local.Add(relative.origin)
	if !finitePoint(point) {
		return maths.Vec{}, fmt.Errorf("frame: restored world position overflow")
	}
	return point, nil
}

// Placement translates the origin and preserves the orientation and scale.
// The caller still checks that the supplied orientation/scale is a valid pose.
func (relative Relative) Placement(world maths.Subspace) (maths.Subspace, error) {
	point, err := relative.Point(world.Origin)
	if err != nil {
		return maths.Subspace{}, err
	}
	world.Origin = point
	return world, nil
}
