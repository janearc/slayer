// Package flatland is a two-dimensional test bed for radiance cascades
// (Osborne and Sannikov 2024, arXiv 2408.14425): a grid of absorbing and
// emitting cells, one ray marcher, a brute-force reference and the cascades.
package flatland

import "math"

// Bands is the number of independent wavelength bands, drawn as red,
// green and blue.
const Bands = 3

// stepLength is the march step in cells. Picked: half a cell, small enough
// to read every cell a ray crosses.
const stepLength = 0.5

// tinyOpacity is the opacity per cell below which a cell counts as empty
// when dividing emissivity by opacity. Picked: far below anything the scene
// uses, above the point where the division loses precision.
const tinyOpacity = 1e-12

// Grid holds, per cell and per band, the opacity chi and emissivity eta per
// cell length, plus what one full march step through the cell does.
type Grid struct {
	Side        int
	diagonal    float64
	opacity     []float64
	emissivity  []float64
	attenuation []float64
	emission    []float64
	opticalStep []float64
}

// NewGrid makes an empty square grid: no absorption and no emission.
func NewGrid(side int) *Grid {
	size := side * side * Bands
	grid := &Grid{
		Side:        side,
		diagonal:    float64(side) * math.Sqrt2,
		opacity:     make([]float64, size),
		emissivity:  make([]float64, size),
		attenuation: make([]float64, size),
		emission:    make([]float64, size),
		opticalStep: make([]float64, size),
	}
	for slot := range grid.attenuation {
		grid.attenuation[slot] = 1
	}
	return grid
}

// Set gives one cell its opacity and emissivity in every band.
func (grid *Grid) Set(column, row int, opacity, emissivity [Bands]float64) {
	base := (row*grid.Side + column) * Bands
	for band := 0; band < Bands; band++ {
		slot := base + band
		grid.opacity[slot] = opacity[band]
		grid.emissivity[slot] = emissivity[band]
		grid.attenuation[slot], grid.emission[slot] =
			stepTerms(opacity[band], emissivity[band], stepLength)
		grid.opticalStep[slot] = opacity[band] * stepLength
	}
}

// Opacity returns the opacity per cell length of one cell in one band.
func (grid *Grid) Opacity(column, row, band int) float64 {
	return grid.opacity[(row*grid.Side+column)*Bands+band]
}

// stepTerms is the piecewise-constant solution (paper equation 16) for a
// path of the given length through one cell: the fraction of light that
// survives it, and the light the cell adds, before any attenuation beyond.
func stepTerms(opacity, emissivity, length float64) (float64, float64) {
	depth := opacity * length
	if opacity < tinyOpacity {
		return math.Exp(-depth), emissivity * length
	}
	return math.Exp(-depth), emissivity / opacity * -math.Expm1(-depth)
}

// Field is an angle-averaged intensity J per cell and band.
type Field struct {
	Side   int
	Values []float64
}

// At returns the intensity of one cell in every band.
func (field Field) At(column, row int) [Bands]float64 {
	var result [Bands]float64
	copy(result[:], field.Values[(row*field.Side+column)*Bands:])
	return result
}
