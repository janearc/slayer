package film

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Sensor is the geometry of a picture: for each pixel of a picture w by
// h, the ray it looks along, in the sensor's own space, down its minus z.
// The band is which part of the spectrum is asking, so glass in front of
// it can bend red and blue differently. A camera is a sensor at a subspace.
//
// A Sensor records nothing itself. What comes back along the ray is made
// into a pixel by render/colour, which develops it, and that stays there.
type Sensor interface {
	Ray(x, y, w, h int, band maths.Band) maths.Ray
}

// Pinhole is the first sensor: light through one point onto a flat
// picture, so straight lines stay straight and far things are smaller.
// It is perspective, as every camera before glass had it.
type Pinhole struct {
	// Wide is how much it sees from its left edge to its right, in
	// radians. Zero means 60 degrees.
	Wide float64
	// Pixel is one pixel's width over its height; zero means square.
	//
	// A sextant pixel is half a cell wide and a third of one tall, so on
	// a cell twice as tall as it is wide it is 0.75. Without it, spheres
	// come out as eggs.
	Pixel float64
}

// wide is how wide the pinhole sees, with zero read as 60 degrees.
func (p Pinhole) wide() float64 {
	if p.Wide == 0 {
		return 60 * maths.Degree
	}
	return p.Wide
}

// pixel is the pixel's shape, with zero read as square.
func (p Pinhole) pixel() float64 {
	if p.Pixel == 0 {
		return 1
	}
	return p.Pixel
}

// Ray is the ray through the middle of pixel (x, y) of a picture w by h pixels.
// Row 0 is the top of the picture, and up in space is up the picture: this is
// the one place where rows counting downward meet y counting up. A pinhole has
// no glass, so it bends every band alike.
//
// a picture of no pixels looks straight ahead; a Wide that is not a field from
// zero to a half turn is the 60 degree default, and a Pixel that is not a width
// greater than zero is square, rather than a mirrored or a broken picture.
//
// the guards are here and not in wide and pixel, which the compiler writes into
// their callers.
func (p Pinhole) Ray(x, y, w, h int, _ maths.Band) maths.Ray {
	if w <= 0 || h <= 0 {
		return maths.Ray{Dir: maths.V(0, 0, -1)}
	}
	if !(p.Wide >= 0 && p.Wide < math.Pi) {
		p.Wide = 0
	}
	if !(p.Pixel >= 0) {
		p.Pixel = 0
	}
	half := math.Tan(p.wide() / 2)
	// the picture, measured in pixel heights, is w*pixel wide and h tall;
	// keeping that ratio is what keeps circles round
	aspect := float64(w) * p.pixel() / float64(h)
	u := (2*(float64(x)+0.5)/float64(w) - 1) * half
	v := (1 - 2*(float64(y)+0.5)/float64(h)) * half / aspect
	return maths.Ray{Dir: maths.V(u, v, -1).Unit()}
}

// Look is the ray a sensor makes for a pixel, seen from eye: taken out of
// the eye's space into the one the eye sits in. A nil sensor is a pinhole
// at 60 degrees.
func Look(eye maths.Subspace, s Sensor, x, y, w, h int,
	band maths.Band) maths.Ray {
	if s == nil {
		s = Pinhole{}
	}
	r := s.Ray(x, y, w, h, band)
	return maths.Ray{At: eye.Out(r.At),
		Dir: eye.Orientation.Out(r.Dir).Unit()}
}
