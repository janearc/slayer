//go:build darwin

package gpu

import (
	"time"

	"github.com/janearc/slayer/metal"
)

// the kernel's entry points that fill the lamp store, and the trace's
const (
	reachGridKernel = "reach_grid"
	reachBeamKernel = "reach_beam"
	traceKernel     = "trace"
)

// bakeReachOn fills the lamp store on the trace kernel through run: a
// grid for each medium and wide lamp, then a table for each narrow
// beam. Each job changes the header and the boxes on the host, so run
// must give the device what the host holds at each call.
//
// It leaves the header saying the store is to be read, which a frame
// after this one that only moves the camera draws from as it is.
func (encoded *scene) bakeReachOn(kernel *metal.Kernel,
	run grids) (time.Duration, error) {
	before := kernel.GPUTime()
	err := encoded.bakeReachGrids(kernel, run)
	if err == nil {
		err = encoded.bakeReachTables(kernel, run)
	}
	encoded.writeReachHeader(0, 0, err == nil)
	if err != nil {
		return kernel.GPUTime() - before, err
	}
	return kernel.GPUTime() - before, kernel.Function(traceKernel)
}

// bakeReachGrids fills every medium's grid of every wide lamp.
func (encoded *scene) bakeReachGrids(kernel *metal.Kernel,
	run grids) error {
	store := encoded.reach
	if len(store.grids) == 0 || len(store.wide) == 0 {
		return nil
	}
	if err := kernel.Function(reachGridKernel); err != nil {
		return err
	}
	for _, job := range store.grids {
		cells := job.grid.cells
		for slot, lamp := range store.wide {
			first := job.first + slot*job.grid.count()*reachChannels
			writer, err := newPackedBandWriter(encoded.boxes.bytes, first,
				cells, reachChannels)
			if err != nil {
				return err
			}
			encoded.writeReachHeader(job.index, lamp, false)
			err = run(cells[0]*cells[1], cells[2], reachChannels,
				writer.consume)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// bakeReachTables fills every narrow beam's table, one row of
// reachBeamCells places along its axis.
func (encoded *scene) bakeReachTables(kernel *metal.Kernel,
	run grids) error {
	store := encoded.reach
	if len(store.narrow) == 0 {
		return nil
	}
	if err := kernel.Function(reachBeamKernel); err != nil {
		return err
	}
	row := [3]int{reachBeamCells, 1, 1}
	for _, lamp := range store.narrow {
		first := store.tables + lamp*reachBeamCells*reachChannels
		writer, err := newPackedBandWriter(encoded.boxes.bytes, first, row,
			reachChannels)
		if err != nil {
			return err
		}
		encoded.writeReachHeader(0, lamp, false)
		err = run(reachBeamCells, 1, reachChannels, writer.consume)
		if err != nil {
			return err
		}
	}
	return nil
}
