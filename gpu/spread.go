package gpu

// Where a pixel's samples start. film/place.go says the design: a
// pixel's samples take the first points of the (0,2)-sequence, each
// carried onto the filter's shape.
//
// The pixel is still the plain mean of its own samples, so the bands a
// picture is drawn in and the finishing of each band are as they were.
//
// The header's `spread` is whether the samples are spread, how many lie
// along a pixel's side, half a stratum's width, and the width of the
// slope of the filter's shape. With it nought the kernel starts every
// sample at its own middle, and the picture is the one it always was.
//
// The kernel's `placed` makes each place from the whole numbers film
// makes it from, in thirty-two bits. A pixel of up to four thousand
// samples has shares of thirteen bits at most, which a float holds.
//
// So the two tracers start from the same shares and differ only in the
// shape's square root, by a float's precision.
//
// Rays the cpu makes through a lens the kernel does not know are made
// by film.LookThrough at the same places.

import (
	"fmt"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/scene/thing"
)

// The whole numbers the kernel's places are made of.
const (
	// placeBits is defined as the bits a share of the square is kept
	// in, as film keeps it.
	placeBits = 32
	// placeHalf is half the square as such a share, two to the
	// thirty-first: the first column of Sobol's second sequence.
	placeHalf = 1 << (placeBits - 1)
	// placeDrop is how many of a share's bits are dropped to cut it to
	// the twenty-four a float holds.
	placeDrop = placeBits - hashBits
)

// spreadRow is the header's spread for a world drawn at side samples
// along a pixel's side: that they are spread, the side, half a
// stratum's width as film has it, and the width of the slope of the
// world's filter. A float holds each of the first three exactly.
//
// It is nothing where each sample starts at its own middle: where the
// world's filter is the grid, or a pixel has one sample.
func spreadRow(world *thing.World, side int) []float64 {
	if world == nil || side <= wholeOne {
		return nil
	}
	ramp, spread := world.Filter.Ramp()
	if !spread {
		return nil
	}
	return []float64{wholeOne, float64(side),
		float64(film.Stratum(side * side)), ramp}
}

// rampAt is where the header's spread holds the width of the filter's
// slope: its fourth number, counted from nought.
const rampAt = 3

// spreadOver tells an encoded scene where its samples start, for a
// world drawn at side samples along a pixel's side. The header is told;
// and rays the cpu made whole through the lens, which were made at the
// samples' middles, are made again at their places.
func (encoded *scene) spreadOver(world *thing.World, side int) {
	row := spreadRow(world, side)
	headerStruct.patch(encoded.header.bytes, values{"spread": row})
	lensed := encoded.through
	if lensed == nil || row == nil {
		return
	}
	lensed.side, lensed.ramp = side, row[rampAt]
	if lensed.whole() {
		encoded.rays.bytes = lensed.rays(firstRow, lensed.height)
	}
}

// placeNumbers is the kernel's declaration of the whole numbers its
// places are made of.
func placeNumbers() string {
	return fmt.Sprintf(`constexpr constant uint place_half = %dU;
constexpr constant int place_drop = %d;
`, uint32(placeHalf), placeDrop)
}
