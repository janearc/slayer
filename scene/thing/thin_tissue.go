package thing

import (
	"fmt"
	"slices"

	"github.com/janearc/slayer/materials/tissue"
	"github.com/janearc/slayer/maths"
)

// ThinTissue describes optional visible diffuse transmission. The zero
// value leaves the material's ordinary diffuse reflection in place.
type ThinTissue struct {
	slab *tissueSlab
}

type tissueSlab struct {
	layers    []tissue.Layer
	thickness float64
	share     float64
}

// NewThinTissue copies a slab template. The sum of the layer thicknesses
// is the cutoff in scene metres. Their proportions give the fractional
// thicknesses at shorter chords. Share reserves part of the material's
// diffuse budget for transmission. Zero disables the response.
func NewThinTissue(layers []tissue.Layer, share float64) (ThinTissue, error) {
	if !thingShare(share) {
		return ThinTissue{}, fmt.Errorf("thin tissue: share must be between zero and one")
	}
	if share == 0 {
		return ThinTissue{}, nil
	}
	if _, err := tissue.SolveSlab(layers); err != nil {
		return ThinTissue{}, err
	}
	var thickness float64
	for _, layer := range layers {
		thickness += layer.Thickness
	}
	if !(thickness > 0) || !thingFinite(thickness) {
		return ThinTissue{}, fmt.Errorf("thin tissue: the total layer thickness must be positive and finite")
	}
	return ThinTissue{slab: &tissueSlab{
		layers: slices.Clone(layers), thickness: thickness, share: share,
	}}, nil
}

// Enabled reports whether the material reserves a positive diffuse share.
func (made ThinTissue) Enabled() bool { return made.slab != nil }

// Share returns the fraction reserved from diffuse reflection.
func (made ThinTissue) Share() float64 {
	if made.slab == nil {
		return 0
	}
	return made.slab.share
}

// MaxThickness gives the supported slab depth in scene metres.
func (made ThinTissue) MaxThickness() float64 {
	if made.slab == nil {
		return 0
	}
	return made.slab.thickness
}

// Transmittance gives the three flux fractions through a supported slab
// depth. The renderer supplies the separate surface-interface losses.
func (made ThinTissue) Transmittance(depth float64) maths.Light {
	if !(depth > 0 && depth <= made.MaxThickness()) {
		return maths.Light{}
	}
	// The constructor solved the copied stack at its maximum thickness.
	response, _ := tissue.SolveScaledSlab(made.slab.layers, depth/made.slab.thickness)
	return response.Transmittance
}
