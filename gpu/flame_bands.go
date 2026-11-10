//go:build darwin

package gpu

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// flameBandWriter deinterleaves one raw band into already-reserved scene bytes.
// It borrows that storage, retains one raw band and creates no split arrays.
type flameBandWriter struct {
	encoded *scene
	job     flameBake
	next    int
}

// bakeFlameInto streams the unchanged field kernel into its reserved grids.
// Failure may leave private bytes partly filled, but no completed scene
// escapes.
func (encoded *scene) bakeFlameInto(job flameBake) (time.Duration, error) {
	writer := flameBandWriter{encoded: encoded, job: job}
	if err := writer.validReservation(); err != nil {
		return 0, err
	}
	kernel, inputs, err := prepareFlameBake(job.grid)
	if err != nil {
		return 0, err
	}
	defer kernel.Close()
	err = kernel.GridBands(job.grid.cells[0]*job.grid.cells[1], job.grid.cells[2],
		rows, 4, inputs, writer.consume)
	return kernel.GPUTime(), err
}

// validReservation guards every product and offset before a Metal allocation.
func (writer *flameBandWriter) validReservation() error {
	grid := writer.job.grid
	remaining := math.MaxInt32 / 16
	for _, cells := range grid.cells {
		if cells < 1 || cells > mostBoxCells || cells > remaining {
			return fmt.Errorf("gpu: invalid reserved flame dimensions")
		}
		remaining /= cells
	}
	count := grid.count()
	first, shine := writer.job.density, writer.job.shine
	available := len(writer.encoded.boxes.bytes) / 4
	if first < 0 || first > available || count > available-first ||
		shine != first+count || shine > available || count > (available-shine)/3 {
		return fmt.Errorf("gpu: invalid reserved flame ranges")
	}
	return nil
}

// consume preserves x-fastest, then y, then z addressing of the field kernel.
func (writer *flameBandWriter) consume(first, down int, samples []float32) error {
	grid := writer.job.grid
	plane := grid.cells[0] * grid.cells[1]
	if first != writer.next || down <= 0 || first < 0 || down > grid.cells[2]-first ||
		len(samples) != plane*down*4 {
		return fmt.Errorf("gpu: invalid or out-of-order flame band")
	}
	for cell := range plane * down {
		at := first*plane + cell
		writer.put(writer.job.density+at, samples[4*cell])
		for channel := range 3 {
			writer.put(writer.job.shine+3*at+channel, samples[4*cell+1+channel])
		}
	}
	writer.next += down
	return nil
}

// put writes the same scalar bits as the old split-array packing path.
func (writer *flameBandWriter) put(index int, value float32) {
	binary.LittleEndian.PutUint32(writer.encoded.boxes.bytes[index*4:], math.Float32bits(value))
}
