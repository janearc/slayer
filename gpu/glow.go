package gpu

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// glowOn is the word in z of a volume's air that says the world's glow
// lights it: the kernel adds its colour times the glow there to every
// step's light, as trace's scattered does for the air the glow is of.
const glowOn = 1

// glowWord is which word of a volume's air says glowOn: z, the third,
// since x says a gas and y a cloud's own phase.
const glowWord = 2

// lightNumbers is how many numbers a light takes in the boxes: red,
// green and blue.
const lightNumbers = 3

// glowLattice is a glow the kernel can read: values at the middles of a
// grid of cells over a box, read between the eight middles round a point
// and held at the edge beyond them. skyglow.Field is one.
type glowLattice interface {
	thing.Glow
	// Cells is the box the grid fills, in world metres, and its cells
	// across x, y and z.
	Cells() (low, high maths.Vec, count [3]int)
	// Cell is the value at the middle of one cell.
	Cell(x, y, z int) maths.Light
}

// glowGrid is the world's glow as the kernel reads it: the box, the
// cells across it, and where the first of its numbers is in the boxes.
type glowGrid struct {
	low, high maths.Vec
	cells     [3]int
	first     int
}

// glowRefused says why the world's glow cannot be drawn here, and
// whether it cannot: a glow that is no grid of cells, a grid of no cells,
// or a box a float cannot hold apart. A world with no glow is drawn.
func glowRefused(glow thing.Glow) (unsupported, bool) {
	if glow == nil {
		return unsupported{}, false
	}
	lattice, ok := glow.(glowLattice)
	if !ok {
		return unsupported{fmt.Sprintf("the world's glow field of kind %T",
			glow)}, true
	}
	low, high, count := lattice.Cells()
	if count[0] < 1 || count[1] < 1 || count[2] < 1 {
		return unsupported{"the world's glow field with no cells"}, true
	}
	if !boxHeld(low, high) {
		return unsupported{"the world's glow field in a box a float " +
			"cannot hold"}, true
	}
	return unsupported{}, false
}

// boxHeld is whether a box's corners are numbers a float holds, none
// of them not a number or infinite, with the high corner above the low
// on every axis once they are floats.
func boxHeld(low, high maths.Vec) bool {
	lows := []float64{low.X, low.Y, low.Z}
	highs := []float64{high.X, high.Y, high.Z}
	for axis, bottom := range lows {
		top := highs[axis]
		if !fits(bottom) || !fits(top) || !(float32(top) > float32(bottom)) {
			return false
		}
	}
	return true
}

// glowOf packs the world's glow after the boxed media's cells, where the
// world has one: three numbers a cell, x fastest, then y, then z, the
// order boxed_cell reads. refuse has passed over any glow it cannot.
func (encoded *scene) glowOf() error {
	lattice := encoded.glowing
	if lattice == nil {
		return nil
	}
	low, high, count := lattice.Cells()
	cells := count[0] * count[1] * count[2]
	if len(encoded.boxes.bytes) > math.MaxInt32-cells*lightNumbers*wordBytes {
		return fmt.Errorf("gpu: the world's glow field exceeds the volume " +
			"buffer's byte limit")
	}
	numbers := make([]float32, 0, cells*lightNumbers)
	for z := range count[2] {
		for y := range count[1] {
			for x := range count[0] {
				light := lattice.Cell(x, y, z)
				numbers = append(numbers, float32(light.R),
					float32(light.G), float32(light.B))
			}
		}
	}
	encoded.glow = &glowGrid{low: low, high: high, cells: count,
		first: encoded.boxes.floats(numbers)}
	return nil
}

// glowLast marks the volume appended last as the air the world's glow
// lights, where it is that air: by its own medium, before any tick, as
// trace asks the glow of a volume's source.
func (encoded *scene) glowLast(volume *thing.Volume) {
	if encoded.glowing == nil || !encoded.glowing.Lights(volume.Medium) {
		return
	}
	row := (encoded.counts.volumes - 1) * volumeStruct.size()
	start := row + fieldOffset(volumeStruct, "air") + glowWord*wordBytes
	binary.LittleEndian.PutUint32(encoded.volumes.bytes[start:], glowOn)
}

// glowHeader writes the world's glow's box and cells into the header, or
// a grid whose first number is minus one where the world has none.
func (encoded *scene) glowHeader(given values) {
	given["glow_low"], given["glow_high"] = nil, nil
	given["glow_grid"] = wholes(0, 0, 0, -1)
	if grid := encoded.glow; grid != nil {
		given["glow_low"], given["glow_high"] = point(grid.low),
			point(grid.high)
		given["glow_grid"] = wholes(grid.cells[0], grid.cells[1],
			grid.cells[2], grid.first)
	}
}
