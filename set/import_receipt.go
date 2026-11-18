package set

// RequestedHeightMetres is the importer normalization request, in metres.
func (made Import) RequestedHeightMetres() float64 { return made.height }

// AssetRoot is the resolved directory against which the source closure is
// recorded and checked.
func (made Import) AssetRoot() string { return made.root }

// SelectedScene is the explicit source scene, when the author selected one.
func (made Import) SelectedScene() (int, bool) {
	if made.scene == nil {
		return 0, false
	}
	return *made.scene, true
}

// Source is the authored provenance retained for an import-bearing set's
// native material receipt. Older sets keep their unchanged representation.
func (made Material) Source() string { return made.source }
