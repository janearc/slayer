package terrain

import (
	"fmt"
	"math"
)

// GeographicElevation interpolates the declared geographic sample lattice.
// Longitude and latitude are degrees in Reference().GeographicCRS, not an
// implicit WGS84 conversion. Returned heights retain the source's own datum.
// A point outside the sample lattice, or with any contributing no-data corner,
// is absent. Missing data never becomes a zero-height coastline.
func (raster *Raster) GeographicElevation(longitude, latitude float64) (float64, bool, error) {
	if raster == nil || raster.closed || !raster.placed {
		return 0, false, fmt.Errorf("terrain: unavailable geographic source")
	}
	ref := raster.reference
	// ModelTypeGeographic=2 and Angular_Degree=9102 are GeoTIFF registry codes.
	if ref.Model != 2 || ref.GeographicCRS == 0 || ref.AngularUnits != 9102 ||
		(ref.RasterType != 1 && ref.RasterType != 2) {
		return 0, false, fmt.Errorf("terrain: need a declared degree grid and pixel registration")
	}
	if !finiteCoordinate(longitude) || !finiteCoordinate(latitude) ||
		longitude < -180 || longitude > 180 || latitude < -90 || latitude > 90 {
		return 0, false, fmt.Errorf("terrain: invalid geographic query")
	}
	if raster.width < 2 || raster.height < 2 {
		return 0, false, fmt.Errorf("terrain: geographic interpolation needs two samples per axis")
	}
	offset := 0.0
	if ref.RasterType == 1 {
		offset = 0.5
	}
	x := (longitude-raster.west)/raster.spacing - offset
	y := (raster.north-latitude)/raster.rowSpacing() - offset
	if x < 0 || y < 0 || x > float64(raster.width-1) || y > float64(raster.height-1) {
		return 0, false, nil
	}
	column := min(int(math.Floor(x)), raster.width-2)
	row := min(int(math.Floor(y)), raster.height-2)
	u, v := x-float64(column), y-float64(row)
	weights := [4]float64{(1 - u) * (1 - v), u * (1 - v), (1 - u) * v, u * v}
	corners := [4][2]int{{column, row}, {column + 1, row}, {column, row + 1}, {column + 1, row + 1}}
	var height float64
	for index, corner := range corners {
		if weights[index] == 0 {
			continue
		}
		value, has, err := raster.sample(corner[0], corner[1])
		if err != nil || !has {
			return 0, false, err
		}
		height += weights[index] * value
	}
	if !finiteCoordinate(height) {
		return 0, false, fmt.Errorf("terrain: geographic interpolation overflow")
	}
	return height, true, nil
}
