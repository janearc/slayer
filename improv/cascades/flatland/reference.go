package flatland

import (
	"math"
	"sync/atomic"
	"time"
)

// Reference is the brute-force answer (long characteristics): from every
// cell centre, march the given number of evenly spaced directions out to
// the edge of the grid and average what arrives.
func Reference(grid *Grid, directions, workers int) (Field, Stats) {
	begin := time.Now()
	field := Field{Side: grid.Side, Values: make([]float64, grid.Side*grid.Side*Bands)}
	cosines := make([]float64, directions)
	sines := make([]float64, directions)
	for direction := range cosines {
		angle := (float64(direction) + 0.5) * 2 * math.Pi / float64(directions)
		sines[direction], cosines[direction] = math.Sincos(angle)
	}
	var steps atomic.Int64
	forEachRow(grid.Side, workers, func(row int) {
		var rowSteps int64
		for column := 0; column < grid.Side; column++ {
			mean, used := cellMean(grid, column, row, cosines, sines)
			copy(field.Values[(row*grid.Side+column)*Bands:], mean[:])
			rowSteps += used
		}
		steps.Add(rowSteps)
	})
	rays := int64(grid.Side) * int64(grid.Side) * int64(directions)
	return field, Stats{Rays: rays, Steps: steps.Load(), Elapsed: time.Since(begin)}
}

// cellMean averages the intensity arriving at one cell centre over all
// the directions given.
func cellMean(grid *Grid, column, row int, cosines, sines []float64) ([Bands]float64, int64) {
	var sum [Bands]float64
	var steps int64
	centreX, centreY := float64(column)+0.5, float64(row)+0.5
	for direction := range cosines {
		interval, used := grid.march(centreX, centreY,
			cosines[direction], sines[direction], 0, math.Inf(1))
		for band := 0; band < Bands; band++ {
			sum[band] += interval.Intensity[band]
		}
		steps += used
	}
	for band := range sum {
		sum[band] /= float64(len(cosines))
	}
	return sum, steps
}
