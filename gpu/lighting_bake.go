//go:build darwin

package gpu

import (
	"encoding/binary"
	"time"

	"github.com/janearc/slayer/metal"
)

// bakeLighting shares the trace kernel's native hierarchy and scene buffers.
// Every frame recomputes lighting; this cache does not lag moving lamps.
func (encoded *scene) bakeLighting(kernel *metal.Kernel, inputs [][]byte) (time.Duration, error) {
	before := kernel.GPUTime()
	if err := kernel.Function("smoke_light"); err != nil {
		return 0, err
	}
	offset := fieldOffset(headerStruct, "lighting")
	for _, job := range encoded.smokes {
		writer, err := newPackedBandWriter(encoded.boxes.bytes, job.first, job.grid.cells, 3)
		if err != nil {
			return kernel.GPUTime() - before, err
		}
		binary.LittleEndian.PutUint32(encoded.header.bytes[offset:], uint32(job.index))
		err = kernel.GridBands(job.grid.cells[0]*job.grid.cells[1], job.grid.cells[2],
			rows, 3, inputs, writer.consume)
		if err != nil {
			return kernel.GPUTime() - before, err
		}
	}
	binary.LittleEndian.PutUint32(encoded.header.bytes[offset:], 0)
	return kernel.GPUTime() - before, kernel.Function("trace")
}
