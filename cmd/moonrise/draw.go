package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"

	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/trace"
)

// made draws the scene as asked and keeps it: the raw, and the plain
// preview when one is asked for.
func made(given asked) error {
	composed, err := compose(given)
	if err != nil {
		return err
	}
	picture, err := draw(composed, given)
	if err != nil {
		return err
	}
	if err := raw.Write(given.out, picture); err != nil {
		return err
	}
	if given.preview == "" {
		return nil
	}
	return plain(picture, given.stops, given.preview)
}

// draw is the scene through the lens.
func draw(composed scene, given asked) (*frame.Frame, error) {
	lens := film.Pinhole{Wide: given.wide * maths.Degree}
	return trace.Sampled(composed.world, composed.eye, lens, 0,
		given.width, given.height, given.samples)
}

// plain is a frame as a png with no film in it: its linear light by
// the stops, cut at white, through srgb's transfer as libtheme has it.
func plain(picture *frame.Frame, stops float64, out string) error {
	gain := math.Exp2(stops)
	shown := image.NewRGBA(image.Rect(0, 0, picture.W, picture.H))
	level := func(linear float32) uint8 {
		cut := math.Max(0, math.Min(1, float64(linear)*gain))
		return uint8(math.Round(srgb.FromLinear(cut) * 255))
	}
	for y := range picture.H {
		for x := range picture.W {
			pixel := picture.At(x, y)
			shown.Set(x, y, color.RGBA{R: level(pixel.R),
				G: level(pixel.G), B: level(pixel.B), A: 255})
		}
	}
	file, err := os.Create(out)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, shown)
}
