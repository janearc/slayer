package titan

// reachedDepth is how many times the mesh was split to reach the triangle the
// place is in.
func (sample Sample) reachedDepth() int { return sample.depth }

// edgeLength is about how long that triangle's edges are, in metres.
func (sample Sample) edgeLength() float64 { return sample.edge }

// mapSource is what the map under the sample says it is.
func (sample Sample) mapSource() string { return sample.source }

// cornersHeld is how many of the map's corners the ground is holding.
func (ground *Ground) cornersHeld() int {
	ground.guard.Lock()
	defer ground.guard.Unlock()
	return len(ground.kept)
}
