package terrain

import (
	"fmt"
	"math"
)

// sample is the height at a column and row of the raster, and whether
// the raster has one there.
func (raster *Raster) sample(column, row int) (float64, bool, error) {
	if raster.closed {
		return 0, false, fmt.Errorf("terrain: source is closed")
	}
	if column < 0 || row < 0 || column >= raster.width ||
		row >= raster.height {
		return 0, false, nil
	}
	i := row/raster.tileH*raster.across() + column/raster.tileW
	samples, err := raster.tile(i)
	if err != nil {
		return 0, false, err
	}
	value := samples[row%raster.tileH*raster.tileW+column%raster.tileW]
	if value == raster.noData || math.IsNaN(float64(value)) {
		return 0, false, nil
	}
	// An infinite sample is a real reading the model published, however
	// absurd; it rides the window to mesh construction, which refuses a
	// non-finite point as a mesh refusal.
	return float64(value), true, nil
}

// A Window is heights on a square grid: so many points across, eastward,
// and down, southward, so many metres apart, its first point at a
// stated easting and northing.
type Window struct {
	west, north, step float64
	across, down      int
	heights           []float64
}

// At is the height at a point of the window, across and down.
func (window Window) At(across, down int) float64 {
	return window.heights[down*window.across+across]
}

// Size is how many points across and down the window has, and how far
// apart they are.
func (window Window) Size() (across, down int, step float64) {
	return window.across, window.down, window.step
}

// Window is the raster's heights on a grid whose first point is at an
// easting and a northing, so many points across and down, so many
// metres apart: each the sample whose cell holds it. it refuses a grid
// that leaves the raster or meets a place with no height.
func (raster *Raster) Window(west, north float64, across, down int,
	step float64) (Window, error) {
	limit, err := DefaultWindowBytes()
	if err != nil {
		return Window{}, err
	}
	return raster.WindowWithBudget(west, north, across, down, step, limit)
}

// WindowWithBudget retains strict coverage and bounds the returned float64
// grid. The byte limit is separate from decoded tile retention and temporary
// decoding.
func (raster *Raster) WindowWithBudget(west, north float64, across, down int,
	step float64, limitBytes int64,
) (Window, error) {
	if err := raster.projectedCoordinates(); err != nil {
		return Window{}, err
	}
	if !raster.placed {
		return Window{}, fmt.Errorf("terrain: the raster says nothing of " +
			"where it is, so it has no window")
	}
	if across < 2 || down < 2 || !(step > 0) {
		return Window{}, fmt.Errorf("terrain: a window of %d by %d, %g "+
			"apart", across, down, step)
	}
	request := GridRequest{West: west, North: north, Across: across, Down: down, Step: step}
	count, err := request.capacity(limitBytes, gridHeightBytes)
	if err != nil {
		return Window{}, err
	}
	window := Window{west: west, north: north, step: step, across: across,
		down: down, heights: make([]float64, 0, count)}
	for row := range down {
		for column := range across {
			east := west + float64(column)*step
			south := north - float64(row)*step
			height, err := raster.elevation(east, south)
			if err != nil {
				return Window{}, err
			}
			window.heights = append(window.heights, height)
		}
	}
	return window, nil
}

// elevation is the raster's height at an easting and a northing.
func (raster *Raster) elevation(east, north float64) (float64, error) {
	value, has, err := raster.Elevation(east, north)
	if err != nil {
		return 0, err
	}
	if !has {
		return 0, fmt.Errorf("terrain: no height at %.1f e, %.1f n", east,
			north)
	}
	return value, nil
}
