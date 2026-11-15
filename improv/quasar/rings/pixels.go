package rings

import (
	"fmt"
	"math"
)

// Camera is a pinhole camera's view: how wide it sees from its left
// edge to its right, in radians, and how many pixels that is.
type Camera struct {
	wide   float64
	pixels int
}

// NewCamera is a camera seeing wide radians across in so many pixels.
// It refuses a width that is not above nought and under half a turn,
// which no pinhole holds, and fewer than one pixel.
func NewCamera(wide float64, pixels int) (Camera, error) {
	if !(wide > none && wide < halfTurn) {
		return Camera{}, fmt.Errorf("rings: a camera %v radians wide "+
			"is no pinhole's: give one above nought and under %v",
			wide, halfTurn)
	}
	if pixels < one {
		return Camera{}, fmt.Errorf("rings: a picture %d pixels "+
			"across has none: give one or more", pixels)
	}
	return Camera{wide: wide, pixels: pixels}, nil
}

// Offset is where an angle off the picture's centre falls, in pixels
// from the centre: a pinhole puts it at the tangent, and the picture's
// half width is the tangent of half the camera's width.
func (camera Camera) Offset(angle float64) float64 {
	half := math.Tan(camera.wide / two)
	return math.Tan(angle) / half * float64(camera.pixels) / two
}

// Pixels is how many pixels across a band is in the camera's picture,
// for a camera looking at the hole's centre.
func (camera Camera) Pixels(band Band) float64 {
	return camera.Offset(band.far) - camera.Offset(band.near)
}

// Wide is how wide the camera sees, in radians.
func (camera Camera) Wide() float64 { return camera.wide }

// CloseIn is the camera, as many pixels across, narrowed until a band
// at the middle of its picture is so many pixels wide: what a close-in
// aimed at the band must see. At the middle a pinhole's pixel is the
// tangent of half its width over half its pixels. It refuses fewer
// than one pixel, and a band so wide the close-in would be no narrower.
func (camera Camera) CloseIn(band Band, across float64) (Camera, error) {
	if !(across >= one) {
		return Camera{}, fmt.Errorf("rings: a band %v pixels across "+
			"is under a pixel: ask for one or more", across)
	}
	pixel := band.Width() / across
	wide := two * math.Atan(pixel*float64(camera.pixels)/two)
	if !(wide < camera.wide) {
		return Camera{}, fmt.Errorf("rings: band %d at %v pixels "+
			"needs a camera %v radians wide, no narrower than this "+
			"one's %v", band.order, across, wide, camera.wide)
	}
	return Camera{wide: wide, pixels: camera.pixels}, nil
}
