//go:build darwin

package gpu

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/metal"
)

// bakeWisps writes every wispy volume's density into its grid on the gpu,
// once a picture, so the trace reads a grid where it asked the noise.
// doc/fast.md says why; the grid is cubic, bakeSide cells a side. it
// says how long the gpu spent.
func (encoded *scene) bakeWisps() (time.Duration, error) {
	if len(encoded.bakes) == 0 {
		return 0, nil
	}
	if err := encoded.validWispyReservation(); err != nil {
		return 0, err
	}
	kernel, err := metal.Open(source(), "bake")
	if err != nil {
		return 0, err
	}
	defer kernel.Close()
	cells := make([]byte, encoded.cells*4)
	for _, one := range encoded.bakes {
		writer, err := newPackedBandWriter(cells, one.first, [3]int{bakeSide, bakeSide, bakeSide}, 1)
		if err != nil {
			return kernel.GPUTime(), err
		}
		which := binary.LittleEndian.AppendUint32(nil,
			uint32(one.index))
		which = binary.LittleEndian.AppendUint32(which,
			uint32(bakeSide))
		err = kernel.GridBands(bakeSide*bakeSide, bakeSide, rows, 1,
			[][]byte{encoded.volumes.bytes, which}, writer.consume)
		if err != nil {
			return kernel.GPUTime(), err
		}
	}
	encoded.baked.bytes = cells
	return kernel.GPUTime(), nil
}

// validWispyReservation checks actual jobs before allocating the packed field.
// A changed global side cannot reinterpret offsets reserved for another grid.
func (encoded *scene) validWispyReservation() error {
	if bakeSide < 1 || bakeSide > (math.MaxInt32/4)/bakeSide/bakeSide {
		return fmt.Errorf("gpu: invalid wispy bake dimensions")
	}
	perGrid := bakeSide * bakeSide * bakeSide
	if len(encoded.bakes) > (math.MaxInt32/4)/perGrid || encoded.cells != len(encoded.bakes)*perGrid {
		return fmt.Errorf("gpu: invalid wispy bake capacity")
	}
	for index, job := range encoded.bakes {
		if job.first != index*perGrid {
			return fmt.Errorf("gpu: invalid wispy bake offsets")
		}
	}
	return nil
}

// bakeAll completes every deferred field before the trace reads its buffers.
func (encoded *scene) bakeAll() (time.Duration, error) {
	wisps, err := encoded.bakeWisps()
	if err != nil {
		return wisps, err
	}
	flames, err := encoded.bakeFlames()
	return wisps + flames, err
}
