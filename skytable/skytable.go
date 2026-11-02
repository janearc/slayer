// Package skytable samples sky illumination into a table shared by CPU and GPU
// surface shading.
//
// Rows span nadir to zenith; Columns span azimuth from negative z toward x.
// Entries average light over the surrounding angular cell, and lookup
// interpolates between four samples. Averaging lets point-like stars supply
// stable illumination instead of depending on a ray hitting one directly.
//
// Primary rays that miss geometry still sample the original sky.
package skytable

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Rows and Columns are the table's size, picked: a surface gathers the
// sky over half of it, so a table this coarse, read between its
// points, is close.
const Rows, Columns = 33, 64

// within is how many looks a side the mean of a cell is taken over.
const within = 4

// Table is a sky's light, a cell's mean at each of its points.
type Table struct {
	cells [Rows * Columns]maths.Light
}

// Of is a sky's table; with no sky, a table of none.
func Of(sky thing.Sky) *Table {
	made := &Table{}
	if sky == nil {
		return made
	}
	high, round := math.Pi/(Rows-1), 2*math.Pi/Columns
	for row := range Rows {
		for column := range Columns {
			var sum maths.Light
			for i := range within * within {
				up := -math.Pi/2 + high*(float64(row)+
					(float64(i/within)+0.5)/within-0.5)
				about := round * (float64(column) +
					(float64(i%within)+0.5)/within - 0.5)
				sum = sum.Add(sky.Light(wayOf(up, about)))
			}
			mean := sum.Scale(1.0 / (within * within))
			made.cells[row*Columns+column] = mean
		}
	}
	return made
}

// wayOf is the way so high and so far round from minus z toward x.
func wayOf(up, about float64) maths.Vec {
	return maths.V(math.Sin(about)*math.Cos(up), math.Sin(up),
		-math.Cos(about)*math.Cos(up))
}

// Cells is the table's entries, row by row, for the gpu.
func (table *Table) Cells() []maths.Light {
	return table.cells[:]
}

// Light is the table's light along a way, read between its four
// nearest points, round the horizon wrapping. The kernel's sky_at reads
// it the same way.
func (table *Table) Light(way maths.Vec) maths.Light {
	way = way.Unit()
	up := math.Asin(math.Max(-1, math.Min(1, way.Y)))
	about := math.Atan2(way.X, -way.Z)
	if about < 0 {
		about += 2 * math.Pi
	}
	row := math.Max(0, math.Min((up+math.Pi/2)/math.Pi*(Rows-1),
		Rows-1))
	column := about / (2 * math.Pi) * Columns
	low := min(int(row), Rows-2)
	left := int(column) % Columns
	right := (left + 1) % Columns
	down, across := row-float64(low), column-math.Floor(column)
	mix := func(a, b maths.Light, share float64) maths.Light {
		return a.Scale(1 - share).Add(b.Scale(share))
	}
	near := mix(table.cells[low*Columns+left],
		table.cells[low*Columns+right], across)
	far := mix(table.cells[(low+1)*Columns+left],
		table.cells[(low+1)*Columns+right], across)
	return mix(near, far, down)
}
