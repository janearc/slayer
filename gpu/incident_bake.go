//go:build darwin

package gpu

import (
	"time"

	"github.com/janearc/slayer/metal"
)

// bakeIncident evaluates the same direct sky integral at every grid node.
// It always uses current geometry, density, light and clock. The pass reads no
// interpolated source values, so it cannot recursively gather its own cache.
func (encoded *scene) bakeIncident(kernel *metal.Kernel, inputs [][]byte) (time.Duration, error) {
	return encoded.bakeIncidentOn(kernel, copied(kernel, inputs))
}

// bakeIncidentOn is bakeIncident with its grid run by run, which a
// retained drawer gives over the inputs it holds.
func (encoded *scene) bakeIncidentOn(kernel *metal.Kernel,
	run grids) (time.Duration, error) {
	grid := encoded.incident
	if grid == nil {
		return 0, nil
	}
	before := kernel.GPUTime()
	if err := kernel.Function("environment_light"); err != nil {
		return 0, err
	}
	writer, err := newPackedBandWriter(encoded.boxes.bytes, grid.first, grid.nodes, 3)
	if err != nil {
		return kernel.GPUTime() - before, err
	}
	if err = run(grid.nodes[0]*grid.nodes[1], grid.nodes[2], 3, writer.consume); err != nil {
		return kernel.GPUTime() - before, err
	}
	return kernel.GPUTime() - before, kernel.Function("trace")
}
