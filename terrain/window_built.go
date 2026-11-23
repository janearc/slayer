package terrain

import (
	"fmt"
	"math"
)

// BuildWindow is a window whose heights the caller works out, for ground that
// did not come from one raster read on one grid. height is asked for each
// point, by column and row, across a row at a time from the first row down,
// and gives the point's height in metres.
//
// The window's first point is at west and north, and its points are step
// apart, as in Raster.Window. It is bound by the same payload budget as that
// window and is refused before any height is asked for when it is over. It is
// refused too when a height is not a finite number or the caller's function
// gives an error, and the error is the caller's own.
func BuildWindow(west, north float64, across, down int, step float64,
	height func(column, row int) (float64, error)) (Window, error) {
	limit, err := defaultWindowBytes()
	if err != nil {
		return Window{}, err
	}
	if across < 2 || down < 2 || !(step > 0) || height == nil {
		return Window{}, fmt.Errorf("terrain: a window of %d by %d, %g "+
			"apart, and a way to find its heights", across, down, step)
	}
	request := gridRequest{West: west, North: north, Across: across,
		Down: down, Step: step}
	count, err := request.capacity(limit, gridHeightBytes)
	if err != nil {
		return Window{}, err
	}
	window := Window{west: west, north: north, step: step, across: across,
		down: down, heights: make([]float64, 0, count)}
	for row := range down {
		for column := range across {
			value, err := height(column, row)
			if err != nil {
				return Window{}, err
			}
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return Window{}, fmt.Errorf("terrain: the height at column "+
					"%d, row %d is not a finite number", column, row)
			}
			window.heights = append(window.heights, value)
		}
	}
	return window, nil
}
