package weather

import (
	"math"
	"testing"
)

// rough is a stream function zero on the walls and a field, both rough, for the
// jacobian's sums.
func rough(grid Grid) (psi, field []float64) {
	psi = make([]float64, grid.cells())
	field = make([]float64, grid.cells())
	for j := range grid.rows {
		for i := range grid.across {
			x, y := float64(i)*grid.run(), float64(j)
			psi[grid.at(i, j)] = math.Sin(3*x+y*0.4) * math.Sin(
				math.Pi*(y+0.5)/float64(grid.rows))
			field[grid.at(i, j)] = math.Cos(2*x-0.3*y) + 0.5*math.Sin(5*x+y)
		}
	}
	return psi, field
}

// summed over the sphere, each cell weighed by its area, arakawa's
// jacobian carries no field in or out, and none of its square: what the
// air carries round it keeps.
func TestTheJacobianKeepsTheFieldAndItsSquare(t *testing.T) {
	grid := mustGrid(t, 32, 24)
	psi, field := rough(grid)
	stream, carried := grid.walled(psi), grid.mirrored(field)
	total, squared, scale := 0.0, 0.0, 0.0
	for j := range grid.rows {
		area := math.Cos(grid.latitude(j))
		for i := range grid.across {
			change := grid.jacobian(stream, carried, i, j)
			total += area * change
			squared += area * change * field[grid.at(i, j)]
			scale += area * math.Abs(change)
		}
	}
	if math.Abs(total) > 1e-12*scale || math.Abs(squared) > 1e-12*scale {
		t.Errorf("the field changes by %g, its square by %g, of %g",
			total, squared, scale)
	}
}

// a stream function that runs round the circles alone, the same down
// each row, carries a field made the same way nowhere: both depend on
// latitude alone, and zonal flow leaves a zonal field as it is.
func TestAZonalFlowLeavesAZonalField(t *testing.T) {
	grid := mustGrid(t, 16, 12)
	psi := make([]float64, grid.cells())
	field := make([]float64, grid.cells())
	for j := range grid.rows {
		for i := range grid.across {
			psi[grid.at(i, j)] = math.Sin(float64(j))
			field[grid.at(i, j)] = math.Cos(float64(j) * 0.7)
		}
	}
	for j := range grid.rows {
		got := grid.jacobian(grid.walled(psi), grid.mirrored(field), 5, j)
		if math.Abs(got) > 1e-12 {
			t.Errorf("row %d: %g, want nought", j, got)
		}
	}
}
