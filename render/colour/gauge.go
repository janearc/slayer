package colour

import (
	"image"

	ninjacolour "github.com/janearc/slayer/ninjafilm/colour"
)

// Masked is a picture with the gauge's frame cut into it: black bars
// above and below, or either side, so that what is left is aspect wide
// against tall, measured in square pixels.
//
// pixel is the picture's own pixel's width over its height. A picture
// already that shape is left alone. The mask is ninjafilm's.
func Masked(img *image.RGBA, aspect, pixel float64) *image.RGBA {
	return ninjacolour.Masked(img, aspect, pixel)
}

// Grained is a picture with film grain on it: soft clumps about size
// pixels across, each lightening or darkening what is under it by up
// to amount, in linear light, the same clumps every time for a seed.
// Nought size or amount is the picture as it was.
//
// Clumps finer than a pixel average out inside it: the amount falls by
// the size, so a fine gauge at a coarse picture is nearly clean, which
// is what a fine gauge is for. The grain is ninjafilm's.
func Grained(img *image.RGBA, size, amount float64, seed uint64) *image.RGBA {
	return ninjacolour.Grained(img, size, amount, seed)
}
