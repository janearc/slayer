package lens

import (
	"errors"
	"time"

	"github.com/janearc/slayer/ninjafilm/frame"
)

// what an exposure counts in, each defined
const (
	firstPlace = 0 // defined: the place of the first item of a list
	onePixel   = 1 // defined: a frame of one pixel across and one down
)

var errExpose = errors.New("ninjafilm: an exposure is a shutter open for " +
	"more than no time, and one glimpse or more, every one the same size")

// Expose is the frame of film that a shutter open for so long made
// through the lens. Each glimpse holds the radiance of each pixel,
// rendered at instants spread evenly through the opening. The lens
// gathers it onto the film as irradiance.
//
// It is held for the share of the open time that the glimpse stands
// for, and the glimpses are added.
//
// What comes out is in joules a square metre of film for each watt a
// square metre a steradian of the light the glimpses were rendered in.
// Nothing is clipped. It is a raw.
//
//ninja:vow practice slayer's film step makes the raw here; tests only so far
func (glass Thin) Expose(open time.Duration, glimpses []*frame.Frame) (
	exposed *frame.Frame, err error) {
	if open <= none || len(glimpses) == none ||
		glimpses[firstPlace] == nil {
		return exposed, errExpose
	}
	width, height := glimpses[firstPlace].W, glimpses[firstPlace].H
	for _, each := range glimpses {
		if each == nil || each.W != width || each.H != height {
			return exposed, errExpose
		}
	}
	exposed = frame.New(width, height)
	for y := range height {
		for x := range width {
			made := glass.pixelOf(x, y, open, glimpses)
			exposed.Set(x, y, made)
		}
	}
	return exposed, nil
}

// pixelOf is the exposure of pixel (x, y) from what each glimpse saw
// there.
func (glass Thin) pixelOf(x, y int, open time.Duration,
	glimpses []*frame.Frame) frame.Pixel {
	seen := make([]frame.Pixel, len(glimpses))
	for index, each := range glimpses {
		seen[index] = each.At(x, y)
	}
	first := glimpses[firstPlace]
	return glass.Exposing(x, y, first.W, first.H, open, seen)
}

// Exposing is one pixel's exposure as Expose makes it. The pixel is
// (x, y) of a picture width by height. The seen pixels say what each
// glimpse saw there.
//
// Each channel's light is added over the glimpses in their order. It is
// then taken once by the gathering and the share of time. So the same
// glimpses make the same pixel bit for bit. Its cover is the mean of
// the glimpses. Its depth is the nearest any of them met.
//
//ninja:vow practice slayer's replay proves one pixel here; tests only so far
func (glass Thin) Exposing(x, y, width, height int, open time.Duration,
	seen []frame.Pixel) frame.Pixel {
	if len(seen) == none {
		return frame.New(onePixel, onePixel).Pix[firstPlace]
	}
	var red, green, blue, cover float64
	depth := seen[firstPlace].Depth
	for _, each := range seen {
		red, green, blue = red+float64(each.R), green+float64(each.G),
			blue+float64(each.B)
		cover += float64(each.Cover)
		depth = min(depth, each.Depth)
	}
	count := float64(len(seen))
	held := glass.Gather(x, y, width, height) * open.Seconds() / count
	mean := cover / count
	return frame.NewPixel(float32(red*held), float32(green*held),
		float32(blue*held), float32(mean), depth)
}
