package terrain

import (
	"fmt"
	"math"
)

// Coverage is a source cell extent in its declared coordinate units.
// Reference identifies those units; this is not a datum or projection adapter.
type Coverage struct {
	West, North, East, South float64
}

// Coverage distinguishes a finite source extent from a world's visible extent.
// It includes the last cell, whose centre is inside the east/south boundary.
func (raster *Raster) Coverage() (Coverage, bool) {
	if !raster.placed {
		return Coverage{}, false
	}
	return Coverage{West: raster.west, North: raster.north,
		East:  raster.west + float64(raster.width)*raster.spacing,
		South: raster.north - float64(raster.height)*raster.rowSpacing()}, true
}

// Elevation returns whether this source has a sample at a projected location.
// Out-of-coverage and no-data cells are absent; malformed input is an error.
// A world can choose another source or an explicit fallback when absent.
func (raster *Raster) Elevation(east, north float64) (float64, bool, error) {
	if err := raster.projectedCoordinates(); err != nil {
		return 0, false, err
	}
	if !raster.placed {
		return 0, false, fmt.Errorf("terrain: source has no projected placement")
	}
	if !finiteCoordinate(east) || !finiteCoordinate(north) {
		return 0, false, fmt.Errorf("terrain: non-finite projected location")
	}
	column := math.Floor((east - raster.west) / raster.spacing)
	row := math.Floor((raster.north - north) / raster.rowSpacing())
	if column < 0 || row < 0 || column >= float64(raster.width) ||
		row >= float64(raster.height) {
		return 0, false, nil
	}
	return raster.sample(int(column), int(row))
}

// finiteCoordinate keeps invalid data separate from absent source coverage.
func finiteCoordinate(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
