package footprint

import (
	"fmt"

	"github.com/janearc/slayer/maths"
)

// bounds encloses a world box in camera axes with outward-rounded arithmetic.
// It may be larger than the rotated box; that only retains extra geometry.
func (view Perspective) bounds(box maths.Box) (maths.Box, error) {
	x := (interval{box.Min.X, box.Max.X}).sub(exact(view.eye.Origin.X))
	y := (interval{box.Min.Y, box.Max.Y}).sub(exact(view.eye.Origin.Y))
	z := (interval{box.Min.Z, box.Max.Z}).sub(exact(view.eye.Origin.Z))
	orientation := view.eye.Orientation
	if orientation == (maths.Orientation{}) {
		orientation = maths.Identity
	}
	var axes [3]interval
	for index, axis := range [3]maths.Vec{orientation.X, orientation.Y, orientation.Z} {
		axes[index] = x.mul(exact(axis.X)).add(y.mul(exact(axis.Y))).add(z.mul(exact(axis.Z)))
	}
	local := maths.Box{Min: maths.V(axes[0].low, axes[1].low, axes[2].low),
		Max: maths.V(axes[0].high, axes[1].high, axes[2].high)}
	if !validBox(local) {
		return maths.Box{}, fmt.Errorf("footprint: camera-relative bounds exceed the numeric range")
	}
	return local, nil
}

// outside rejects only a whole box outside one expanded frustum half-space.
// No far plane or planetary horizon is invented. Unbounded interval arithmetic
// cannot establish exclusion and therefore retains the geometry.
func (view Perspective) outside(box maths.Box, displacement float64) bool {
	x, y := interval{box.Min.X, box.Max.X}, interval{box.Min.Y, box.Max.Y}
	depth := interval{-box.Max.Z, -box.Min.Z}
	margin := exact(displacement)
	across := x.div(exact(view.horizontal))
	down := y.div(exact(view.vertical))
	marginAcross := margin.add(margin.div(exact(view.horizontal)))
	marginDown := margin.add(margin.div(exact(view.vertical)))
	planes := [5]interval{depth.add(margin).sub(exact(view.near)),
		depth.add(across).add(marginAcross), depth.sub(across).add(marginAcross),
		depth.add(down).add(marginDown), depth.sub(down).add(marginDown)}
	for _, plane := range planes {
		if plane.high < 0 {
			return true
		}
	}
	return false
}
