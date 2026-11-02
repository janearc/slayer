//go:build darwin

package gpu

import (
	"encoding/binary"
	"math"
	"time"

	"github.com/janearc/slayer/metal"
)

// bakeAll writes every wispy volume's density into its grid on the gpu,
// once a picture, so the trace reads a grid where it asked the noise.
// doc/fast.md says why; the grid is cubic, bakeSide cells a side. it
// says how long the gpu spent.
func (encoded *scene) bakeAll() (time.Duration, error) {
	if len(encoded.bakes) == 0 {
		return 0, nil
	}
	kernel, err := metal.Open(source(), "bake")
	if err != nil {
		return 0, err
	}
	defer kernel.Close()
	cells := make([]byte, 0, encoded.cells*4)
	for _, one := range encoded.bakes {
		which := binary.LittleEndian.AppendUint32(nil,
			uint32(one.index))
		which = binary.LittleEndian.AppendUint32(which,
			uint32(bakeSide))
		grid, err := kernel.Grid(bakeSide*bakeSide, bakeSide, rows, 1,
			[][]byte{encoded.volumes.bytes, which})
		if err != nil {
			return 0, err
		}
		for _, density := range grid {
			cells = binary.LittleEndian.AppendUint32(cells,
				math.Float32bits(density))
		}
	}
	encoded.baked.bytes = cells
	return kernel.GPUTime(), nil
}
