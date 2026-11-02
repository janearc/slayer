// Package file writes rendered images as PNG files.
package file

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// Block is how many png pixels wide and tall one render pixel is drawn, so a
// pixel that is not square keeps its true shape: a sextant pixel, three wide to
// four tall, is drawn 3 by 4. (a sextant is a sixth of a terminal cell, two
// across and three down, and a cell is twice as tall as it is wide.)
//
// It is the nearest small whole numbers to the shape, never more than 8. a
// shape no block of up to 8 by 8 can draw, and a width that is no number
// greater than zero, are drawn 1 by 1.
func Block(pixel float64) (w, h int) {
	if pixel <= 0 {
		return 1, 1
	}
	w, h, best := 1, 1, math.Inf(1)
	for bh := 1; bh <= 8; bh++ {
		bw := int(math.Round(pixel * float64(bh)))
		if bw < 1 || bw > 8 {
			continue
		}
		e := math.Abs(float64(bw)/float64(bh) - pixel)
		if e < best-1e-9 {
			w, h, best = bw, bh, e
		}
	}
	return w, h
}

// Grow is img with every pixel drawn as a block bw by bh, nearest neighbour:
// crisp blocks, never blurred. a block of less than one pixel either way is
// drawn one pixel that way. it holds bw times bh times as many pixels as img,
// so a large img grown far is as large again in memory.
func Grow(img image.Image, bw, bh int) *image.RGBA {
	bw, bh = max(bw, 1), max(bh, 1)
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx()*bw, b.Dy()*bh))
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			out.Set(x, y, img.At(b.Min.X+x/bw, b.Min.Y+y/bh))
		}
	}
	return out
}

var errNoPicture = errors.New("file: no picture to write")

// PNG writes img to path, each pixel grown to keep its true shape, readable
// by all and writable by its owner. the picture is made whole first and
// written beside path, and takes path's place only once it is all there, so
// a picture that cannot be made or written leaves what was at path as it was.
func PNG(path string, img image.Image, pixel float64) error {
	if img == nil {
		return errNoPicture
	}
	bw, bh := Block(pixel)
	var made bytes.Buffer
	if err := png.Encode(&made, Grow(img, bw, bh)); err != nil {
		return err
	}
	beside, err := os.CreateTemp(filepath.Dir(path),
		"."+filepath.Base(path)+".")
	if err != nil {
		return err
	}
	_, err = beside.Write(made.Bytes())
	err = errors.Join(err, beside.Chmod(0o644), beside.Close())
	if err == nil {
		err = os.Rename(beside.Name(), path)
	}
	if err != nil {
		os.Remove(beside.Name())
	}
	return err
}
