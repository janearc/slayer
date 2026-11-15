package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"

	"github.com/janearc/slayer/improv/cascades/flatland"
)

// gapPixels is the width of the dark gap between panels. Picked by eye.
const gapPixels = 8

// gapLevel is the grey of the gap, out of 255. Picked: dark, but not the
// black of a fully shadowed cell.
const gapLevel = 24

// encode turns a linear intensity into an 8-bit value: exposure, the
// simple curve x/(1+x) to bring high dynamic range into view, then sRGB.
func encode(value, exposure float64) uint8 {
	scaled := math.Max(value, 0) * exposure
	mapped := scaled / (1 + scaled)
	if mapped <= 0.0031308 {
		mapped *= 12.92
	} else {
		mapped = 1.055*math.Pow(mapped, 1/2.4) - 0.055
	}
	return uint8(math.Round(math.Min(mapped, 1) * 255))
}

// absoluteDifference is the per-band gap between two fields, times gain.
func absoluteDifference(first, second flatland.Field, gain float64) flatland.Field {
	result := flatland.Field{Side: first.Side, Values: make([]float64, len(first.Values))}
	for slot := range first.Values {
		result.Values[slot] = math.Abs(first.Values[slot]-second.Values[slot]) * gain
	}
	return result
}

// writePicture puts the panels side by side and writes a PNG.
func writePicture(path string, panels []flatland.Field, exposure float64, zoom int) error {
	side := panels[0].Side * zoom
	width := len(panels)*side + (len(panels)-1)*gapPixels
	picture := image.NewRGBA(image.Rect(0, 0, width, side))
	fillGap(picture)
	for index, panel := range panels {
		drawPanel(picture, panel, index*(side+gapPixels), exposure, zoom)
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, picture)
}

// fillGap paints the whole picture with the grey gapLevel, so that what the
// panels do not cover shows as the gap between them.
func fillGap(picture *image.RGBA) {
	bounds := picture.Bounds()
	for row := bounds.Min.Y; row < bounds.Max.Y; row++ {
		for column := bounds.Min.X; column < bounds.Max.X; column++ {
			picture.SetRGBA(column, row, color.RGBA{gapLevel, gapLevel, gapLevel, 255})
		}
	}
}

// drawPanel draws panel with its left edge at pixel column left, each cell
// zoom pixels square, with each band encoded at exposure.
func drawPanel(picture *image.RGBA, panel flatland.Field, left int, exposure float64, zoom int) {
	for row := 0; row < panel.Side*zoom; row++ {
		for column := 0; column < panel.Side*zoom; column++ {
			value := panel.At(column/zoom, row/zoom)
			picture.SetRGBA(left+column, row, color.RGBA{
				encode(value[0], exposure), encode(value[1], exposure),
				encode(value[2], exposure), 255})
		}
	}
}
