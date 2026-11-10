package gpu

import (
	"encoding/binary"
	"math"
)

// smokeGrid retains a canonical smoke box for optional lighting preparation.
// Density keeps its original grid. Lighting is a separate, coarser source term.
type smokeGrid struct {
	index int
	grid  boxed
	first int
}

// lightCells is an authored speed/accuracy choice, not a physical constant.
// lighting_grid_test measures 32, 64 and 128 against direct lighting.
const lightCells = 64

// fieldOffset locates a named field in a packed structure.
func fieldOffset(made structure, name string) int {
	for i, field := range made.fields {
		if field.name == name {
			return i * 16
		}
	}
	panic("gpu: missing buffer field " + name)
}

// reserveLighting keeps source-grid ranges fixed before GPU preparation.
func (encoded *scene) reserveLighting() {
	encoded.reserveLightingAt(lightCells)
}

// reserveLightingAt prepares a positive longest-side resolution. Production
// uses lightCells; the independent comparison varies it without global state.
func (encoded *scene) reserveLightingAt(cells int) {
	for i := range encoded.smokes {
		job := &encoded.smokes[i]
		size := job.grid.high.Sub(job.grid.low)
		longest := math.Max(size.X, math.Max(size.Y, size.Z))
		for axis, side := range []float64{size.X, size.Y, size.Z} {
			job.grid.cells[axis] = max(2, int(math.Ceil(side/longest*float64(cells))))
		}
		job.first = len(encoded.boxes.bytes) / 4
		encoded.boxes.bytes = append(encoded.boxes.bytes, make([]byte, job.grid.count()*3*4)...)
		offset := job.index*volumeStruct.size() + fieldOffset(volumeStruct, "lighting")
		for j, value := range append(job.grid.cells[:], job.first) {
			binary.LittleEndian.PutUint32(encoded.volumes.bytes[offset+j*4:], uint32(value))
		}
	}
}
