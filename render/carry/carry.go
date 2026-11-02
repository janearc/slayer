// Package carry converts between Slayer’s render/frame and ninjafilm’s frame
// types. They have the same structure but separate type identities: the tracer
// returns the former, while ninjafilm’s roll and lens accept the latter. The
// adapter can be removed when those callers share a frame API.
package carry

import (
	ninjaframe "github.com/janearc/slayer/ninjafilm/frame"
	"github.com/janearc/slayer/render/frame"
)

// Ninja is slayer's frame as ninjafilm's. The pixels are copied, so the
// two frames share nothing. A frame that is nil stays nil.
func Ninja(drawn *frame.Frame) *ninjaframe.Frame {
	if drawn == nil {
		return nil
	}
	carried := ninjaframe.New(drawn.W, drawn.H)
	for index, pixel := range drawn.Pix {
		carried.Pix[index] = NinjaPixel(pixel)
	}
	return carried
}

// Slayer is ninjafilm's frame as slayer's. The pixels are copied, so the
// two frames share nothing. A frame that is nil stays nil.
func Slayer(exposed *ninjaframe.Frame) *frame.Frame {
	if exposed == nil {
		return nil
	}
	carried := frame.New(exposed.W, exposed.H)
	for index, pixel := range exposed.Pix {
		carried.Pix[index] = SlayerPixel(pixel)
	}
	return carried
}

// NinjaPixel is slayer's pixel as ninjafilm's, bit for bit.
func NinjaPixel(pixel frame.Pixel) ninjaframe.Pixel {
	return ninjaframe.Pixel(pixel)
}

// SlayerPixel is ninjafilm's pixel as slayer's, bit for bit.
func SlayerPixel(pixel ninjaframe.Pixel) frame.Pixel {
	return frame.Pixel(pixel)
}

// NinjaPixels is slayer's pixels as ninjafilm's, in order.
func NinjaPixels(pixels []frame.Pixel) []ninjaframe.Pixel {
	carried := make([]ninjaframe.Pixel, len(pixels))
	for index, pixel := range pixels {
		carried[index] = NinjaPixel(pixel)
	}
	return carried
}

// NinjaFrames is slayer's frames as ninjafilm's, in order.
func NinjaFrames(drawn []*frame.Frame) []*ninjaframe.Frame {
	carried := make([]*ninjaframe.Frame, len(drawn))
	for index, each := range drawn {
		carried[index] = Ninja(each)
	}
	return carried
}
