package terrain

import (
	"fmt"
	"math"
)

// Smoothed is the window's heights blurred by a gaussian so many metres wide,
// its standard deviation, each height read from its neighbours to three widths
// either side, across and then down, the window's edge standing in for what
// lies past it. zero is the window as it is.
//
// a lidar's cells, drawn as flat faces, show as steps a few metres across;
// blurred over a cell or two, the ground keeps its rims, walls and floors and
// loses the steps. the heights are blurred, not measured.
func (window Window) Smoothed(width float64) (Window, error) {
	if !(width >= 0) || math.IsInf(width, 0) {
		return Window{}, fmt.Errorf("terrain: a blur %g m wide", width)
	}
	if width == 0 {
		return window, nil
	}
	kernel := gaussian(width / window.step)
	along := blurred(window.heights, window.across, window.down, kernel,
		func(column, row, k int) (int, int) { return column + k, row })
	window.heights = blurred(along, window.across, window.down, kernel,
		func(column, row, k int) (int, int) { return column, row + k })
	return window, nil
}

// gaussian is a gaussian's weights so many cells wide, its standard
// deviation, to three widths either side of its middle, summing to one.
func gaussian(cells float64) []float64 {
	reach := int(math.Ceil(3 * cells))
	weights := make([]float64, 2*reach+1)
	sum := 0.0
	for k := range weights {
		off := float64(k - reach)
		weights[k] = math.Exp(-off * off / (2 * cells * cells))
		sum += weights[k]
	}
	for k := range weights {
		weights[k] /= sum
	}
	return weights
}

// blurred is heights on a grid so many across and down, each the sum
// of its neighbours along one way, weighed by a kernel whose middle is
// the height itself, the grid's edge held where the kernel reaches past
// it.
func blurred(heights []float64, across, down int, kernel []float64,
	step func(column, row, k int) (int, int)) []float64 {
	reach := len(kernel) / 2
	made := make([]float64, len(heights))
	for row := range down {
		for column := range across {
			sum := 0.0
			for k, weight := range kernel {
				c, r := step(column, row, k-reach)
				c = max(0, min(across-1, c))
				r = max(0, min(down-1, r))
				sum += weight * heights[r*across+c]
			}
			made[row*across+column] = sum
		}
	}
	return made
}
