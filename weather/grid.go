package weather

import (
	"fmt"
	"math"
)

// A Grid is where the weather is kept: so many cells round each circle
// of latitude, a power of two, and so many rows of latitude from a wall
// so many radians south of the equator to one as far north, on a planet
// of a radius in metres.
//
// the poles are walled off: a grid of latitude and longitude crowds its
// cells together there, and the air at a wall runs along it.
type Grid struct {
	across, rows  int
	limit, radius float64
}

// NewGrid is a grid, refused if its cells round a circle are not a power
// of two or it reaches a pole.
func NewGrid(across, rows int, limit, radius float64) (Grid, error) {
	if across < 4 || across&(across-1) != 0 || rows < 4 ||
		!(limit > 0 && limit < math.Pi/2) || !(radius > 0) {
		return Grid{}, fmt.Errorf("a grid %d across, %d rows, to %g "+
			"radians, radius %g m", across, rows, limit, radius)
	}
	return Grid{across: across, rows: rows, limit: limit, radius: radius},
		nil
}

// cells is how many cells the grid holds.
func (grid Grid) cells() int { return grid.across * grid.rows }

// rise is how far apart the rows are, in radians of latitude.
func (grid Grid) rise() float64 { return 2 * grid.limit / float64(grid.rows) }

// run is how far apart the cells of a row are, in radians of longitude.
func (grid Grid) run() float64 { return 2 * math.Pi / float64(grid.across) }

// latitude is a row's latitude in radians, at its middle; rows outside
// the grid lie past its walls, as far again.
func (grid Grid) latitude(j int) float64 {
	return -grid.limit + (float64(j)+0.5)*grid.rise()
}

// means is a field's mean round each circle, row by row.
func (grid Grid) means(field []float64) []float64 {
	made := make([]float64, grid.rows)
	for j := range grid.rows {
		for i := range grid.across {
			made[j] += field[grid.at(i, j)]
		}
		made[j] /= float64(grid.across)
	}
	return made
}

// at is where a cell's value is kept in a field: its row, then its
// place round the row, wrapped.
func (grid Grid) at(i, j int) int {
	i %= grid.across
	if i < 0 {
		i += grid.across
	}
	return j*grid.across + i
}
