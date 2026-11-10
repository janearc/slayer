package terrain

import (
	"fmt"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/footprint"
)

// PatchGeometry describes full geometry bounds and a displacement bound.
// Bounds include unsampled peaks. Error is the maximum change introduced by
// this patch's representation, supplied by the source or its overview builder.
// Sparse height samples alone do not establish either bound.
type PatchGeometry struct {
	Bounds maths.Box
	Error  measure.Length
}

// SelectProjectedPatches uses the actual lens and raster to choose source
// patches. Target is a pixel displacement bound. A more distant or wider view
// can retain coarser patches without changing source coverage or patch IDs.
// The caller owns source projection, height datum and representation bounds.
func SelectProjectedPatches(root Patch, budget SelectionBudget,
	view footprint.Perspective,
	geometry func(Patch) (PatchGeometry, error),
) (Selection, error) {
	if geometry == nil {
		return Selection{}, fmt.Errorf("terrain: missing patch geometry bounds")
	}
	limited := make(map[Patch]bool)
	selected, err := SelectPatches(root, budget,
		func(patch Patch) (PatchView, error) {
			bounds, err := geometry(patch)
			if err != nil {
				return PatchView{}, err
			}
			assessment, err := view.Assess(bounds.Bounds, bounds.Error)
			if err != nil {
				return PatchView{}, err
			}
			limited[patch] = assessment.Limited
			return PatchView{Visible: assessment.Visible,
				Score: assessment.Pixels}, nil
		})
	if err != nil {
		return Selection{}, err
	}
	for _, patch := range selected.Patches {
		selected.Limited = selected.Limited || limited[patch]
	}
	return selected, nil
}
