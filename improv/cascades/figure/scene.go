package main

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/improv/cascades/flatland"
)

// opaqueOpacity is the opacity per cell of an opaque circle. Picked: a
// ray loses all but a few parts in a million of its light in one cell.
const opaqueOpacity = 1000.0

// squareLow and squareHigh bound the absorbing square, as fractions of
// the frame side. Picked to match the paper's figure 1.
const (
	squareLow  = 0.28
	squareHigh = 0.72
)

// Optical depth of the absorbing square per frame width, so the picture
// looks alike at any grid size. Picked: red is weak, green and blue are
// one hundred times stronger, as in the paper.
const (
	redDepthPerFrame  = 1.5
	blueDepthPerFrame = 150.0
)

// minimumRadius is the smallest circle radius in cells. Picked so a circle
// on a small grid still covers a few cells.
const minimumRadius = 1.2

// circle is a round object, in fractions of the frame side.
type circle struct {
	centreX, centreY, radius float64
	brightness               [flatland.Bands]float64
}

// emitters are the opaque glowing circles. Picked by eye to echo the
// spread of the paper's figure 1: brightness is the colour each one
// radiates in red, green and blue.
var emitters = []circle{
	{0.15, 0.22, 0.030, [flatland.Bands]float64{2, 0, 6}},
	{0.45, 0.06, 0.035, [flatland.Bands]float64{0, 0, 6}},
	{0.93, 0.40, 0.030, [flatland.Bands]float64{8, 0, 0}},
	{0.82, 0.72, 0.045, [flatland.Bands]float64{6, 0, 6}},
	{0.10, 0.88, 0.050, [flatland.Bands]float64{3, 6, 0}},
	{0.55, 0.93, 0.045, [flatland.Bands]float64{8, 3, 0}},
	{0.06, 0.55, 0.030, [flatland.Bands]float64{1, 1, 5}},
}

// blockers are the two small black circles, one near the top and one near
// the bottom.
var blockers = []circle{
	{0.50, 0.17, 0.012, [flatland.Bands]float64{}},
	{0.50, 0.82, 0.012, [flatland.Bands]float64{}},
}

// lone is the single source of the "single" scene, a check that can be
// worked out by hand. Picked: one mid-sized disc in the middle of the
// frame, equally bright in every band.
var lone = circle{0.5, 0.5, 0.04, [flatland.Bands]float64{4, 4, 4}}

// buildScene makes the grid and a mask of the cells inside opaque circles.
// The scene is "figure" (the paper's figure 1) or "single" (one disc in
// empty space).
func buildScene(side int, name string) (*flatland.Grid, []bool, error) {
	grid := flatland.NewGrid(side)
	opaque := make([]bool, side*side)
	if name == "single" {
		paintCircle(grid, opaque, lone)
		return grid, opaque, nil
	}
	if name != "figure" {
		return nil, nil, fmt.Errorf("unknown scene %q", name)
	}
	paintSquare(grid)
	for _, item := range emitters {
		paintCircle(grid, opaque, item)
	}
	for _, item := range blockers {
		paintCircle(grid, opaque, item)
	}
	return grid, opaque, nil
}

// paintSquare sets the absorbing square of the figure, from squareLow to
// squareHigh of the side, to absorb and emit nothing else. Red gets the red
// depth and green and blue the blue depth, each divided by the side. The
// square is not marked in the opaque mask.
func paintSquare(grid *flatland.Grid) {
	low := int(squareLow * float64(grid.Side))
	high := int(squareHigh * float64(grid.Side))
	side := float64(grid.Side)
	absorption := [flatland.Bands]float64{
		redDepthPerFrame / side, blueDepthPerFrame / side, blueDepthPerFrame / side}
	for row := low; row < high; row++ {
		for column := low; column < high; column++ {
			grid.Set(column, row, absorption, [flatland.Bands]float64{})
		}
	}
}

// paintCircle makes every cell whose centre is inside the circle opaque,
// with emissivity opacity times brightness, and marks it in the mask.
func paintCircle(grid *flatland.Grid, opaque []bool, item circle) {
	side := float64(grid.Side)
	radius := math.Max(item.radius*side, minimumRadius)
	var opacity, emissivity [flatland.Bands]float64
	for band := range opacity {
		opacity[band] = opaqueOpacity
		emissivity[band] = opaqueOpacity * item.brightness[band]
	}
	for row := 0; row < grid.Side; row++ {
		for column := 0; column < grid.Side; column++ {
			offsetX := float64(column) + 0.5 - item.centreX*side
			offsetY := float64(row) + 0.5 - item.centreY*side
			if math.Hypot(offsetX, offsetY) <= radius {
				grid.Set(column, row, opacity, emissivity)
				opaque[row*grid.Side+column] = true
			}
		}
	}
}
