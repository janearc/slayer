package film

// Where a pixel's rays start.
//
// A ray costs what it costs wherever in its pixel it starts, so where
// it starts is free, and it decides how an edge is drawn.
//
// Rays on a regular grid of side by side meet an edge that runs along
// the grid a whole row at a time. A pixel on that edge can be only side
// plus one shades, however many rays it took.
//
// Here a pixel's rays take the first points of the (0,2)-sequence in
// base two, the first two dimensions of Sobol's sequence: van der
// Corput's radical inverse along one side of the square and Sobol's
// second along the other.
//
// At any count the points cover the square evenly. At a power of two,
// no two share a row or a column of that many, so sixteen rays draw an
// edge along the grid in seventeen even shades (Pharr, Jakob and
// Humphreys, Physically Based Rendering, third edition, section 7.5).
//
// The point of the square is then carried onto the filter's shape, so
// rays start as thick as the shape is high. A pixel is then the plain
// mean of its own rays, and that mean is the picture weighed by the
// shape.
//
// So the filter is taken by where the rays start, and no ray is weighed
// or shared with a neighbour: filter importance sampling (the same
// book, fourth edition, section 5.4.3).
//
// The shape is the pixel's own square with a slope at each side. With
// no slope it is the box, one pixel wide; sloped all the way from the
// middle it is the tent, two pixels wide; between, a trapezoid.
//
// None is ever under nought, so none can ring. Each is the square
// smoothed by a narrower square, so its copies, one at every pixel, add
// to the same everywhere: a flat field stays flat, and a thin line is
// as strong wherever it falls.
//
// A wider slope draws a smoother edge and softer detail. The share of
// its contrast a stripe pattern keeps is the product of two sincs, of
// the stripes' cycles a pixel and of that times the slope's width.
//
// Every pixel has the same pattern. A pattern of each pixel's own
// leaves a grain along every edge, since neighbours guess the edge a
// little differently, and this tracer has no other grain.
//
// One pattern draws a straight edge the same all along it, and a still
// scene filmed is still.
//
// The gpu's kernel makes the same places from the same whole numbers
// (trace.metal's placed), to a float's precision.

import (
	"math"
	"math/bits"

	"github.com/janearc/slayer/maths"
)

// The numbers a place is made of.
const (
	// shareBits is defined as the bits a share of the square is kept
	// in, as a whole number: the sequence is made of whole numbers that
	// wide, and the gpu's kernel has the same.
	shareBits = 32
	// halfSquare is half the square as such a share, two to the
	// thirty-first: the sequence's first column, and half of the one
	// stratum a single ray has.
	halfSquare = 1 << (shareBits - 1)
	// one, oneHalf and twice are defined as one, a half and two: the
	// square's side, its middle, and the two a slope's running sum is
	// halved by.
	one     = 1
	oneHalf = 0.5
	twice   = 2
	// pixelMiddle is a pixel's middle from its edge, in pixels. Defined
	// as a half.
	pixelMiddle = 0.5
)

// Placed is a sensor that can look through any place of a pixel and not
// only its middle: across and down are how far from the middle, in
// pixels of the picture w by h. At nought and nought it is Ray.
type Placed interface {
	Sensor
	RayThrough(x, y int, across, down float64, w, h int,
		band maths.Band) maths.Ray
}

// Stratum is half the width of the strata a pixel of count rays is cut
// into along each side, as a share of the square in thirty-two bits.
// Count is at least one.
//
// The strata are the least power of two that is count or more. Each
// point of the sequence starts at a stratum's edge and is moved by
// this to its middle.
func Stratum(count int) uint32 {
	return uint32(halfSquare) >> bits.Len(uint(count-1))
}

// crossed is Sobol's second sequence at an index, as a share of the
// square in thirty-two bits: each bit of the index, from its last,
// turns on a column of Pascal's triangle, odd and even, written from
// the share's first bit down.
func crossed(index uint32) uint32 {
	var share uint32
	for column := uint32(halfSquare); index != 0; index >>= 1 {
		if index&1 != 0 {
			share ^= column
		}
		column ^= column >> 1
	}
	return share
}

// shared is a share of the square in thirty-two bits as a number from
// nought up to one.
func shared(share uint32) float64 {
	return float64(share) / (1 << shareBits)
}

// carried takes a share of the square, nought up to one, onto a shape
// that is flat over the pixel's middle and slopes to nought over ramp
// pixels at each side: how far from the pixel's middle, in pixels.
//
// It is the inverse of the shape's running sum, so shares spread
// evenly land as thick as the shape is high.
//
// The shape is the pixel's own square smoothed by a square ramp wide, so
// its copies at every pixel add to the same everywhere, whatever the
// ramp. A ramp of one is the tent; of nought, the square itself.
func carried(share, ramp float64) float64 {
	foot, reach := ramp*oneHalf, (one+ramp)*oneHalf
	switch {
	case share < foot:
		return math.Sqrt(twice*ramp*share) - reach
	case share > one-foot:
		return reach - math.Sqrt(twice*ramp*(one-share))
	}
	return share - oneHalf
}

// Place is how far from a pixel's middle its ray numbered index, of
// count, starts, in pixels across and down the picture, under a shape
// whose slopes are ramp pixels wide, nought up to one.
//
// Each is from minus half of one and the ramp up to it. The same three
// numbers give the same place forever, in every pixel.
func Place(index, count int, ramp float64) (across, down float64) {
	number, centring := uint32(index), Stratum(count)
	return carried(shared(bits.Reverse32(number)+centring), ramp),
		carried(shared(crossed(number)+centring), ramp)
}

// Within is Place for a picture drawn as a finer one, side rays along
// each side of a pixel, counted along the pixel's rows, under a shape
// whose slopes are ramp pixels wide. Side is at least one.
//
// It is how far from the middle of fine pixel (x, y) its ray starts, in
// fine pixels across and down.
func Within(x, y, side int, ramp float64) (across, down float64) {
	over, under := x%side, y%side
	right, below := Place(under*side+over, side*side, ramp)
	span := float64(side)
	return (right+pixelMiddle)*span - (float64(over) + pixelMiddle),
		(below+pixelMiddle)*span - (float64(under) + pixelMiddle)
}

// LookThrough is Look through a place of the pixel, across and down from
// its middle in pixels. A sensor that has only its pixels' middles looks
// through the middle, as Look does.
func LookThrough(eye maths.Subspace, s Sensor, x, y int, across,
	down float64, w, h int, band maths.Band) maths.Ray {
	if s == nil {
		s = Pinhole{}
	}
	placed, can := s.(Placed)
	if !can {
		return Look(eye, s, x, y, w, h, band)
	}
	r := placed.RayThrough(x, y, across, down, w, h, band)
	return maths.Ray{At: eye.Out(r.At),
		Dir: eye.Orientation.Out(r.Dir).Unit()}
}
