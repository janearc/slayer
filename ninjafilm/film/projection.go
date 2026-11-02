package film

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/data"
)

// what a projection counts in, each defined
const (
	// pair is a count of two. It is defined as two.
	pair = 2
	// half is half of a share. It is defined as one over two.
	half = 0.5
	// ahead is the way a picture of no pixels looks, down its minus z.
	// It is defined as the way the sensor's space points.
	ahead = -1
	// whole is the whole of a share. It is defined as one.
	whole = 1
)

// pinholeWide is how wide a pinhole sees when none is given, in radians.
var pinholeWide = data.Picked("pinhole-wide-degrees") * maths.Degree

// Projection is the geometry of a picture. Each pixel of a picture has a
// ray it looks along. The projection gives it down its minus z, in its own
// space. The picture is so many pixels wide and high.
//
// The band is which part of the spectrum is asking. Glass in front of
// the projection can bend red and blue differently. Cameras are
// projections at a subspace.
//
// Projections record nothing. The darkroom makes what comes back along
// the ray into a pixel.
type Projection interface {
	Ray(x, y, width, height int, band maths.Band) maths.Ray
}

// Pinhole is the first projection. It is light through one point onto a
// flat picture. Straight lines stay straight and far things are smaller.
// It is perspective, as every camera before glass had it.
//
// Its wide is how much it sees from its left edge to its right, in
// radians. Nought means 60 degrees. Its pixel is one pixel's width over
// its height. Nought means square.
//
// Sextant pixels are half a cell wide and a third of one tall. On cells
// twice as tall as wide they are 0.75. Without that spheres come out as
// eggs.
//
//ninja:vow practice slayer's camera makes its pinhole here; tests so far
type Pinhole struct {
	wide, pixel float64
}

// NewPinhole is a pinhole that sees so many radians across, through
// pixels of a shape. Wides outside zero to a half turn are the default
// of 60 degrees. Pixels without a width greater than zero are square.
//
// Both are read so and not refused. The picture is then neither
// mirrored nor broken.
//
//ninja:vow practice slayer's camera makes its pinhole here; tests so far
func NewPinhole(wide, pixel float64) Pinhole {
	if !(wide >= none && wide < math.Pi) {
		wide = none
	}
	if !(pixel >= none) {
		pixel = none
	}
	return Pinhole{wide, pixel}
}

// seen is how wide the pinhole sees, with zero read as the default.
func (pinhole Pinhole) seen() float64 {
	if pinhole.wide == none {
		return pinholeWide
	}
	return pinhole.wide
}

// shape is the pixel's shape, with zero read as square.
func (pinhole Pinhole) shape() float64 {
	if pinhole.pixel == none {
		return whole
	}
	return pinhole.pixel
}

// Ray is the ray through the middle of pixel x, y of a picture so many pixels
// wide and high. Row zero is the top of the picture. Up in space is up the
// picture. Here rows counting downward meet y counting up. Pinholes have no
// glass, so they bend every band alike.
//
// Pictures of no pixels look straight ahead.
func (pinhole Pinhole) Ray(x, y, width, height int, _ maths.Band) maths.Ray {
	if width <= none || height <= none {
		return maths.Ray{Dir: maths.V(none, none, ahead)}
	}
	reach := math.Tan(pinhole.seen() / pair)
	across, down := float64(width), float64(height)
	// the picture measures the pixel's shape times the pixels across by
	// the pixels down, in pixel heights. Its aspect keeps circles round.
	aspect := across * pinhole.shape() / down
	right := (pair*(float64(x)+half)/across - whole) * reach
	up := (whole - pair*(float64(y)+half)/down) * reach / aspect
	return maths.Ray{Dir: maths.V(right, up, ahead).Unit()}
}

// Look is the ray a projection makes for a pixel, seen from an eye. It
// is taken out of the eye's space into the one the eye sits in. Nil
// projections are pinholes of 60 degrees.
//
//ninja:vow practice slayer's tracer looks through its camera here
func Look(eye maths.Subspace, projection Projection, x, y, width,
	height int, band maths.Band) maths.Ray {
	if projection == nil {
		projection = Pinhole{}
	}
	made := projection.Ray(x, y, width, height, band)
	return maths.Ray{At: eye.Out(made.At),
		Dir: eye.Orientation.Out(made.Dir).Unit()}
}
