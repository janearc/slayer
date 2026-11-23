package set

// FlatAir is the atmosphere layers of a set that has no terrain and no
// globe: layers that say `extent: world`, which over no ground and no
// planet are layers of air over a flat world, each from its base, or from
// the ground at height nought, up by the depth of its weather. It is
// empty for a set with terrain or a globe, whose layers are those of
// the terrain or the globe. The slice belongs to the caller.
func (made Set) FlatAir() []AtmosphereLayer {
	if made.terrain != nil || made.globe != nil {
		return nil
	}
	flat := []AtmosphereLayer{}
	for _, layer := range made.AtmosphericLayers() {
		if layer.world {
			flat = append(flat, layer)
		}
	}
	return flat
}
