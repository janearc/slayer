package main

import (
	"errors"
	"image"
	"image/png"
	"io"
	"math"
	"os"

	"github.com/janearc/slayer/interfaces/terminal"
	"golang.org/x/image/draw"
)

// terminalFilePreview decodes the saved PNG before showing its cells.
func terminalFilePreview(out io.Writer, path string, given preview) error {
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	picture, decodeErr := png.Decode(input)
	if err := errors.Join(decodeErr, input.Close()); err != nil {
		return err
	}
	return terminalPreview(out, picture, given)
}

type still struct {
	picture     image.Image
	pixelAspect float64
}

// Frame fits the PNG into terminal subpixels. A sextant pixel is assumed to
// be 3/4 as wide as it is tall, matching terminal.Measure's fallback.
func (source still) Frame(_ int, wide, high int) (image.Image, error) {
	bounds := source.picture.Bounds()
	pixelAspect := source.pixelAspect
	originalWidth := float64(bounds.Dx())
	originalHeight := float64(bounds.Dy())
	across := float64(wide) * pixelAspect / originalWidth
	down := float64(high) / originalHeight
	scale := math.Min(across, down)
	wantedWidth := originalWidth * scale / pixelAspect
	wantedHeight := originalHeight * scale
	width := int(math.Round(wantedWidth))
	height := int(math.Round(wantedHeight))
	width, height = max(one, width), max(one, height)
	left, top := (wide-width)/halves, (high-height)/halves
	target := image.NewRGBA(image.Rect(nothing, nothing, wide, high))
	fitted := image.Rect(left, top, left+width, top+height)
	draw.ApproxBiLinear.Scale(target, fitted, source.picture,
		bounds, draw.Src, nil)
	return target, nil
}

// terminalPreview streams one fitted view without taking over the keyboard.
func terminalPreview(out io.Writer, picture image.Image, given preview) error {
	source := still{picture: picture, pixelAspect: given.pixelAspect}
	stream, err := terminal.NewStream(given.cells, nothing, one, source)
	if err != nil {
		return err
	}
	if err := stream.Run(out); err != nil {
		return err
	}
	_, err = io.WriteString(out, "\n")
	return err
}
