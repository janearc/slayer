package flatland

import (
	"math"
	"sync"
	"time"
)

// interval is a radiance interval (paper equation 9): the intensity
// gathered along a stretch of ray, seen from its near end as if nothing
// lay beyond, and the optical depth of that stretch.
type interval struct {
	Intensity [Bands]float64
	Tau       [Bands]float64
}

// Stats counts the work a method did.
type Stats struct {
	Rays    int64
	Steps   int64
	Elapsed time.Duration
}

// march integrates the ray origin + distance*direction for distances in
// [from, to]. It also returns how many cells it read.
//
// The order matters. Equation 16 runs from the far end to the near end:
// I = I*exp(-chi*ds) + S*(1 - exp(-chi*ds)). Two cells, near N and far F,
// give I = I_F*att_N + emit_N. Marching near to far instead, the sum
// I = emit_N + att_N*emit_F is the same number, and it is equation 11
// with I(a,b) = emit_N, tau(a,b) = -ln(att_N) and I(b,c) = emit_F. So this
// marches near to far, adding each cell scaled by what survives before it.
func (grid *Grid) march(originX, originY, directionX, directionY,
	from, to float64) (interval, int64) {
	var result interval
	survive := [Bands]float64{1, 1, 1}
	to = math.Min(to, grid.diagonal)
	var steps int64
	entered := false
	for count := 0; ; count++ {
		start := from + float64(count)*stepLength
		if start >= to {
			break
		}
		length := math.Min(stepLength, to-start)
		middle := start + length/2
		cell, inside := grid.cellAt(originX+directionX*middle,
			originY+directionY*middle)
		if !inside {
			if entered {
				break
			}
			continue
		}
		entered = true
		grid.cross(cell, length, &result, &survive)
		steps++
	}
	return result, steps
}

// cellAt finds the cell holding a point, and whether the point is in the
// grid at all.
func (grid *Grid) cellAt(positionX, positionY float64) (int, bool) {
	column := int(math.Floor(positionX))
	row := int(math.Floor(positionY))
	if column < 0 || row < 0 || column >= grid.Side || row >= grid.Side {
		return 0, false
	}
	return (row*grid.Side + column) * Bands, true
}

// cross adds the light of one cell to the interval. Full steps use the
// tables made in Set; a shortened step at the end of an interval is worked
// out directly.
func (grid *Grid) cross(base int, length float64, result *interval,
	survive *[Bands]float64) {
	for band := 0; band < Bands; band++ {
		slot := base + band
		attenuation, emission := grid.attenuation[slot], grid.emission[slot]
		depth := grid.opticalStep[slot]
		if length != stepLength {
			attenuation, emission = stepTerms(grid.opacity[slot],
				grid.emissivity[slot], length)
			depth = grid.opacity[slot] * length
		}
		result.Intensity[band] += survive[band] * emission
		result.Tau[band] += depth
		survive[band] *= attenuation
	}
}

// forEachRow calls work for every row from 0 to count-1, spread over the
// given number of goroutines.
func forEachRow(count, workers int, work func(row int)) {
	rows := make(chan int)
	var group sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for row := range rows {
				work(row)
			}
		}()
	}
	for row := 0; row < count; row++ {
		rows <- row
	}
	close(rows)
	group.Wait()
}
