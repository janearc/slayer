package stage

import "github.com/janearc/slayer/set"

// WorldOnly selects the producer route without terrain, planets or placements.
func WorldOnly(read set.Set) bool {
	if _, groundlessAir := read.GroundlessAir(); groundlessAir {
		return false
	}
	_, hasWorld := read.World()
	_, grounded := read.Ground()
	_, planeted := read.Planet()
	return hasWorld && !grounded && !planeted && len(read.Things()) == 0 && len(read.Stands()) == 0
}
