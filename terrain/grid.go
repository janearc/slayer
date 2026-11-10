package terrain

import "fmt"

// GridRequest describes a grid in the source's own projected coordinates.
// Rows proceed southward. Sampling selects the containing source cell.
type GridRequest struct {
	West, North, Step float64
	Across, Down      int
}

// Grid retains source absence separately from a zero-metre elevation.
// It is a sampled source grid, not a filtered overview or world-space mesh.
type Grid struct {
	request GridRequest
	heights []float64
	present []bool
}

// Layout returns the source-coordinate sampling request without exposing data.
func (grid Grid) Layout() GridRequest { return grid.request }

// At returns a sampled height and its availability. Outside the grid is absent.
func (grid Grid) At(column, row int) (float64, bool) {
	if column < 0 || row < 0 || column >= grid.request.Across || row >= grid.request.Down {
		return 0, false
	}
	index := row*grid.request.Across + column
	return grid.heights[index], grid.present[index]
}

// gridHeightBytes follows float64 storage; availability is one Go bool byte.
const gridHeightBytes = 64 / byteBits
const gridAvailabilityBytes = 1

// ReadGrid reads bounded samples and an availability mask for source fallback.
// limitBytes covers returned array payloads, separately from the tile cache.
// A missing height is masked; corrupt data and resource failures are errors.
func (raster *Raster) ReadGrid(request GridRequest, limitBytes int64) (Grid, error) {
	if err := raster.projectedCoordinates(); err != nil {
		return Grid{}, err
	}
	if !raster.placed {
		return Grid{}, fmt.Errorf("terrain: source has no projected placement")
	}
	count, err := request.capacity(limitBytes, gridHeightBytes+gridAvailabilityBytes)
	if err != nil {
		return Grid{}, err
	}
	grid := Grid{request: request, heights: make([]float64, count), present: make([]bool, count)}
	for row := range request.Down {
		for column := range request.Across {
			east, north := request.location(column, row)
			index := row*request.Across + column
			grid.heights[index], grid.present[index], err = raster.Elevation(east, north)
			if err != nil {
				return Grid{}, err
			}
		}
	}
	return grid, nil
}

// capacity checks dimensions, byte products and final coordinates before make.
func (request GridRequest) capacity(limitBytes, bytesPerSample int64) (int, error) {
	if request.Across < 1 || request.Down < 1 || !finiteCoordinate(request.West) ||
		!finiteCoordinate(request.North) || !finiteCoordinate(request.Step) || request.Step <= 0 {
		return 0, fmt.Errorf("terrain: invalid source grid layout")
	}
	if limitBytes < 0 || bytesPerSample < 1 {
		return 0, fmt.Errorf("terrain: invalid source grid byte budget")
	}
	limit := min(limitBytes, addressableBytes) / bytesPerSample
	if int64(request.Across) > limit/int64(request.Down) {
		return 0, fmt.Errorf("terrain: source grid exceeds its %d-byte payload budget", limitBytes)
	}
	east, south := request.location(request.Across-1, request.Down-1)
	if !finiteCoordinate(east) || !finiteCoordinate(south) {
		return 0, fmt.Errorf("terrain: source grid coordinates overflow")
	}
	return request.Across * request.Down, nil
}

// location retains float64 source coordinates until a world adapter transforms.
func (request GridRequest) location(column, row int) (float64, float64) {
	return request.West + float64(column)*request.Step,
		request.North - float64(row)*request.Step
}
