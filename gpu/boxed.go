package gpu

import (
	"math"
	"runtime"
	"sync"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/texture"
)

// boxCells is how many cells a boxed medium's grid has along its box's
// longest side, the others as many as keep the cells near cubes: 96, so
// a flame's body ten metres tall has cells about ten centimetres across,
// half a step of the tracer's march across it. picked.
var boxCells = 96

// boxed is a medium the kernel cannot ask, with its paint and shine:
// asked on the cpu at every cell of a grid over its box, once a picture,
// and read by the kernel between the eight cells round a point.
type boxed struct {
	med          medium.Medium
	paint, shine texture.Texture
	low, high    maths.Vec
	cells        [3]int
}

// mostBoxCells is the most cells a grid has along its longest side,
// however fine its medium says it is: a grid this fine over a flame's box
// is about 40 MB and asks its medium seven million times. picked.
var mostBoxCells = 192

// fine is a medium that says the smallest thing it draws, in metres of
// its own space, as a flame's body does.
type fine interface {
	Finest() float64
}

// alongLongest is how many cells a grid has along its box's longest
// side: boxCells, or as many as put two cells across the smallest thing
// a fine medium draws, up to mostBoxCells.
func alongLongest(med medium.Medium, longest float64) int {
	cells := boxCells
	if detailed, ok := med.(fine); ok && detailed.Finest() > 0 {
		wanted := int(math.Ceil(2 * longest / detailed.Finest()))
		cells = max(cells, min(wanted, mostBoxCells))
	}
	return cells
}

// boxOf is a medium's grid over its box, with its paint and shine, the
// three as they are at the picture's time.
func boxOf(med medium.Medium, paint, shine texture.Texture) boxed {
	bounds := med.Bounds()
	size := bounds.Max.Sub(bounds.Min)
	longest := math.Max(size.X, math.Max(size.Y, size.Z))
	along := alongLongest(med, longest)
	var cells [3]int
	for axis, side := range []float64{size.X, size.Y, size.Z} {
		share := side / longest
		cells[axis] = max(2, int(math.Ceil(share*float64(along))))
	}
	return boxed{med: med, paint: paint, shine: shine, low: bounds.Min,
		high: bounds.Max, cells: cells}
}

// count is how many cells the grid has
func (grid boxed) count() int {
	return grid.cells[0] * grid.cells[1] * grid.cells[2]
}

// middle is the middle of cell x, y, z in the medium's own space
func (grid boxed) middle(x, y, z int) maths.Vec {
	size := grid.high.Sub(grid.low)
	part := func(index, cells int) float64 {
		return (float64(index) + 0.5) / float64(cells)
	}
	return grid.low.Add(maths.V(size.X*part(x, grid.cells[0]),
		size.Y*part(y, grid.cells[1]), size.Z*part(z, grid.cells[2])))
}

// asked is the medium's density at every cell, x fastest, then y, then
// z, and its shine's and paint's light at each, three numbers a cell,
// none for a texture it has not; slices of z shared among the cores.
func (grid boxed) asked() (density, shine, paint []float32) {
	cells := grid.count()
	density = make([]float32, cells)
	if grid.shine != nil {
		shine = make([]float32, 3*cells)
	}
	if grid.paint != nil {
		paint = make([]float32, 3*cells)
	}
	slices := make(chan int, grid.cells[2])
	for z := range grid.cells[2] {
		slices <- z
	}
	close(slices)
	var working sync.WaitGroup
	for range runtime.GOMAXPROCS(0) {
		working.Add(1)
		go func() {
			defer working.Done()
			for z := range slices {
				grid.slice(z, density, shine, paint)
			}
		}()
	}
	working.Wait()
	return density, shine, paint
}

// slice asks one slice of z of the grid
func (grid boxed) slice(z int, density, shine, paint []float32) {
	for y := range grid.cells[1] {
		for x := range grid.cells[0] {
			at := grid.middle(x, y, z)
			cell := (z*grid.cells[1]+y)*grid.cells[0] + x
			density[cell] = float32(grid.med.Density(at))
			if shine != nil {
				put(shine, cell, grid.shine.At(at))
			}
			if paint != nil {
				put(paint, cell, grid.paint.At(at))
			}
		}
	}
}

// put writes a light at a cell of a grid three numbers a cell
func put(grid []float32, cell int, light maths.Light) {
	grid[3*cell] = float32(light.R)
	grid[3*cell+1] = float32(light.G)
	grid[3*cell+2] = float32(light.B)
}
