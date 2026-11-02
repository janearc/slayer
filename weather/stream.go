package weather

import "math"

// a reading is a field's value at a cell, rows past the walls included,
// by whatever the field takes beyond them.
type reading func(i, j int) float64

// laplacian is the spherical laplacian of a field at a cell, in the
// grid's units: across the latitudes, the slope weighed by the cosine
// at each face between rows, and round the circle, the curvature over
// the cosine squared.
func (grid Grid) laplacian(field reading, i, j int) float64 {
	up, down, here := field(i, j+1), field(i, j-1), field(i, j)
	north := math.Cos(grid.latitude(j) + grid.rise()/2)
	south := math.Cos(grid.latitude(j) - grid.rise()/2)
	middle := math.Cos(grid.latitude(j))
	across := (north*(up-here) - south*(here-down)) /
		(middle * grid.rise() * grid.rise())
	round := (field(i+1, j) - 2*here + field(i-1, j)) /
		(middle * middle * grid.run() * grid.run())
	return (across + round) / (grid.radius * grid.radius)
}

// walled reads a field taken as the negative of the row inside past a wall, so
// it is zero on the wall, as the stream function is.
func (grid Grid) walled(field []float64) reading {
	return func(i, j int) float64 {
		switch {
		case j < 0:
			return -field[grid.at(i, -1-j)]
		case j >= grid.rows:
			return -field[grid.at(i, 2*grid.rows-1-j)]
		}
		return field[grid.at(i, j)]
	}
}

// mirrored reads a field taken as the row inside past a wall, so
// nothing crosses it, as vorticity and the bands' air do not.
func (grid Grid) mirrored(field []float64) reading {
	return func(i, j int) float64 {
		switch {
		case j < 0:
			return field[grid.at(i, -1-j)]
		case j >= grid.rows:
			return field[grid.at(i, 2*grid.rows-1-j)]
		}
		return field[grid.at(i, j)]
	}
}

// streamOf is the stream function whose laplacian is a vorticity, zero on the
// walls: each row turned into its wavenumbers round the circle, and for each
// wavenumber the rows solved together, a tridiagonal system across the
// latitudes, by thomas's sweep.
func (grid Grid) streamOf(vorticity []float64) []float64 {
	n, rows := grid.across, grid.rows
	modes := make([][]complex128, rows)
	for j := range rows {
		modes[j] = make([]complex128, n)
		for i := range n {
			modes[j][i] = complex(vorticity[grid.at(i, j)], 0)
		}
		fft(modes[j], -1)
	}
	column := make([]complex128, rows)
	for k := range n {
		for j := range rows {
			column[j] = modes[j][k]
		}
		grid.sweep(column, float64(wavenumber(k, n)))
		for j := range rows {
			modes[j][k] = column[j]
		}
	}
	psi := make([]float64, grid.cells())
	for j := range rows {
		fft(modes[j], 1)
		for i := range n {
			psi[grid.at(i, j)] = real(modes[j][i]) / float64(n)
		}
	}
	return psi
}

// sweep solves, in place, the rows of one wavenumber m: the laplacian's
// three terms across the latitudes, less the curvature round the
// circle, equal to the vorticity's.
//
// the curvature is the grid's own second difference of a wave of m turns,
// 2 (1 - cos m run) over the run squared, not m squared, so the solve
// undoes laplacian exactly.
func (grid Grid) sweep(column []complex128, m float64) {
	rows := grid.rows
	below := make([]float64, rows)
	middle := make([]float64, rows)
	above := make([]float64, rows)
	scale := grid.radius * grid.radius * grid.rise() * grid.rise()
	curve := 2 * (1 - math.Cos(m*grid.run())) /
		(grid.run() * grid.run() * grid.radius * grid.radius)
	for j := range rows {
		cosine := math.Cos(grid.latitude(j))
		south := math.Cos(grid.latitude(j)-grid.rise()/2) / (cosine * scale)
		north := math.Cos(grid.latitude(j)+grid.rise()/2) / (cosine * scale)
		round := curve / (cosine * cosine)
		below[j], above[j] = south, north
		middle[j] = -south - north - round
	}
	middle[0] -= below[0]
	middle[rows-1] -= above[rows-1]
	thomas(below, middle, above, column)
}

// thomas solves a tridiagonal system in place: below, middle and above
// each row's three terms, the right side given and the answer returned
// in it.
func thomas(below, middle, above []float64, side []complex128) {
	rows := len(side)
	ahead := make([]float64, rows)
	pivot := middle[0]
	ahead[0] = above[0] / pivot
	side[0] /= complex(pivot, 0)
	for j := 1; j < rows; j++ {
		pivot = middle[j] - below[j]*ahead[j-1]
		ahead[j] = above[j] / pivot
		side[j] = (side[j] - complex(below[j], 0)*side[j-1]) /
			complex(pivot, 0)
	}
	for j := rows - 2; j >= 0; j-- {
		side[j] -= complex(ahead[j], 0) * side[j+1]
	}
}
