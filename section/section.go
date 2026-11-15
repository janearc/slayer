// Package section divides a frame into sections and says which of them a
// body can show in, from the body's own bounds and with no ray traced: so
// a frame can be drawn a piece at a time, and a piece nothing has changed
// in can be kept.
package section

import (
	"image"
	"math"

	"github.com/janearc/slayer/maths"
)

// Split is a frame of width by height pixels as sections side pixels
// square, row by row from the top left, the last of a row and the last
// row as much smaller as the frame leaves them. A side under one is one.
func Split(width, height, side int) []image.Rectangle {
	side = max(side, one)
	var boxes []image.Rectangle
	for top := nothing; top < height; top += side {
		for left := nothing; left < width; left += side {
			right, bottom := min(left+side, width), min(top+side, height)
			box := image.Rect(left, top, right, bottom)
			boxes = append(boxes, box)
		}
	}
	return boxes
}

// What the package counts in, each defined.
const (
	nothing = 0 // defined: no amount
	one     = 1 // defined: a single unit
	two     = 2 // defined: one and one
)

// nudge is one pixel, defined as the step from a pixel to its neighbour.
const nudge = one

// A Sight is every way a section's rays can go: a cone about its middle
// ray, opened a little past its outermost, with the furthest any of its
// rays starts from the cone's point.
type Sight struct {
	from, along maths.Vec
	half, slack float64
}

// spare is how much wider than its outermost ray a sight is opened: a
// twentieth more, for a lens that bends the picture between the rays
// asked. picked.
const spare = 1.05

// SightOf is the sight of a section of a frame, given the ray through
// each pixel: found from the ray of every pixel of the section and of
// the pixels that border it, so that no ray of the section, nor of a
// sample within one of its pixels, is left outside.
//
// A section's corners alone are not enough. Through a lens with an
// aperture each pixel's ray starts somewhere else, and near a half turn
// the rays between the corners lean further than the corners' do.
func SightOf(look func(x, y int) maths.Ray, box image.Rectangle) Sight {
	middle := look(middleOf(box.Min.X, box.Max.X), middleOf(box.Min.Y, box.Max.Y))
	sight := Sight{from: middle.At, along: middle.Dir.Unit()}
	for y := box.Min.Y - nudge; y <= box.Max.Y; y++ {
		for x := box.Min.X - nudge; x <= box.Max.X; x++ {
			sight.take(look(x, y))
		}
	}
	sight.half *= spare
	return sight
}

// take opens the sight to hold a ray: as wide as the ray leans from the
// sight's middle, and as much slack as the ray starts from its point.
func (sight *Sight) take(ray maths.Ray) {
	sight.half = math.Max(sight.half, between(sight.along, ray.Dir))
	sight.slack = math.Max(sight.slack, ray.At.Sub(sight.from).Len())
}

// middleOf is the pixel in the middle of the pixels from first up to, and
// not including, past.
func middleOf(first, past int) int {
	return (first + past - nudge) / two
}

// between is the angle between two ways, in radians.
func between(first, second maths.Vec) float64 {
	return math.Atan2(first.Cross(second).Len(), first.Dot(second))
}

// meets is whether a ball about centre can be met by any ray of the
// sight. It may say yes for a ball just outside; it never says no for one
// a ray meets.
func (sight Sight) meets(centre maths.Vec, radius float64) bool {
	toward := centre.Sub(sight.from)
	far, reach := toward.Len(), radius+sight.slack
	if !(far > reach) {
		return true
	}
	return between(sight.along, toward) <= sight.half+math.Asin(reach/far)
}
