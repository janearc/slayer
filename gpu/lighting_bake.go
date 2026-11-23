//go:build darwin

package gpu

import (
	"encoding/binary"
	"time"

	"github.com/janearc/slayer/metal"
)

// bakeLightingOn is bakeLighting with its grids run by run. Each job
// changes the header and the boxes on the host between grids, so run
// must give the device what the host holds at each call.
func (encoded *scene) bakeLightingOn(kernel *metal.Kernel,
	run grids) (time.Duration, error) {
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
		err = run(job.grid.cells[0]*job.grid.cells[1], job.grid.cells[2],
			3, writer.consume)
		if err != nil {
			return kernel.GPUTime() - before, err
		}
	}
	binary.LittleEndian.PutUint32(encoded.header.bytes[offset:], 0)
	return kernel.GPUTime() - before, kernel.Function("trace")
}
