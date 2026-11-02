package shape

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Tube is a hollow cylinder about its own y axis: a ring stood on edge,
// or the shell of a ring world. Radius is to the middle of the wall, Thick
// the wall's thickness, Height its length along y.
//
// It has no exact hit, so it is marched. Its surface faces outward on the
// outside and inward on the inside, which is where a ring world's ground
// is.
type Tube struct {
	Radius, Thick, Height float64
}

// Bounds is the box the tube just fits in.
func (t Tube) Bounds() maths.Box {
	r, h := t.Radius+t.Thick/2, t.Height/2
	return maths.Box{Min: maths.V(-r, -h, -r), Max: maths.V(r, h, r)}
}

// Distance is how far p is from the wall: the distance to a ring in the
// radial and axial plane, which is a box of half the thickness by half
// the height, centred on the radius.
func (t Tube) Distance(p maths.Vec) float64 {
	radial := math.Hypot(p.X, p.Z) - t.Radius
	dx, dy := math.Abs(radial)-t.Thick/2, math.Abs(p.Y)-t.Height/2
	out := math.Hypot(math.Max(dx, 0), math.Max(dy, 0))
	return out + math.Min(math.Max(dx, dy), 0)
}
