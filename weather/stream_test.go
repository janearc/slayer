package weather

import (
	"math"
	"testing"
)

// mustGrid is a grid, or the test stops.
func mustGrid(t *testing.T, across, rows int) Grid {
	t.Helper()
	grid, err := NewGrid(across, rows, 70*math.Pi/180, 6e7)
	if err != nil {
		t.Fatal(err)
	}
	return grid
}

// the stream function streamOf finds has the vorticity it was given for
// its laplacian, cell by cell: a field of waves round and across, its
// laplacian taken, solved back to the field.
func TestTheStreamFunctionUndoesTheLaplacian(t *testing.T) {
	grid := mustGrid(t, 32, 24)
	psi := make([]float64, grid.cells())
	for j := range grid.rows {
		for i := range grid.across {
			psi[grid.at(i, j)] = 1e12 * (math.Sin(float64(3*i)*grid.run()+
				0.4)*math.Sin(float64(j+1)*0.3) + math.Cos(float64(j)*0.7))
		}
	}
	vorticity := make([]float64, grid.cells())
	for j := range grid.rows {
		for i := range grid.across {
			vorticity[grid.at(i, j)] = grid.laplacian(grid.walled(psi), i, j)
		}
	}
	back := grid.streamOf(vorticity)
	for k := range psi {
		if math.Abs(back[k]-psi[k]) > 1e-6*1e12 {
			t.Fatalf("cell %d: %g, want %g", k, back[k], psi[k])
		}
	}
}

// a wall lies half a row past the last: the field's mirror there is its
// negative, so it is zero on the wall.
func TestTheStreamFunctionIsNoughtOnTheWalls(t *testing.T) {
	grid := mustGrid(t, 8, 6)
	field := make([]float64, grid.cells())
	for k := range field {
		field[k] = float64(k + 1)
	}
	walled := grid.walled(field)
	for _, j := range []int{0, grid.rows - 1} {
		past := j - 1
		if j > 0 {
			past = j + 1
		}
		if sum := walled(3, j) + walled(3, past); sum != 0 {
			t.Errorf("row %d and its mirror sum to %g", j, sum)
		}
	}
}
