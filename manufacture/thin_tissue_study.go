package manufacture

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/materials/tissue"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

type tissueStudy struct {
	depth     float64
	transmits bool
}

const (
	tissueStudyWidth  = .050
	tissueStudyHeight = .066
)

func tissueStudyRows() map[string]maker {
	return map[string]maker{
		"tissue-study-opaque": tissueStudy{depth: .002},
		"tissue-study-2mm":    tissueStudy{depth: .002, transmits: true},
		"tissue-study-4mm":    tissueStudy{depth: .004, transmits: true},
		"tissue-study-8mm":    tissueStudy{depth: .008, transmits: true},
	}
}

func (row tissueStudy) extent(seed) (maths.Box, error) {
	return maths.Box{
		Min: maths.V(-tissueStudyWidth/2, 0, -row.depth/2),
		Max: maths.V(tissueStudyWidth/2, tissueStudyHeight, row.depth/2),
	}, nil
}

func (row tissueStudy) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	block, err := newBox(measure.Metres(tissueStudyWidth),
		measure.Metres(tissueStudyHeight), measure.Metres(row.depth))
	if err != nil {
		return good{}, err
	}
	shape, err := block.skin(cut{}).meshed()
	if err != nil {
		return good{}, err
	}
	material := &thing.Material{
		Name: "tissue study slab", Colour: maths.Grey(.42),
		Index: 1.4,
	}
	if row.transmits {
		// Picked values for a visible thickness comparison.
		material.ThinTissue, err = thing.NewThinTissue([]tissue.Layer{
			{Thickness: .0015, Absorption: maths.RGB(156, 270, 420), Backscatter: maths.RGB(420, 520, 650)},
			{Thickness: .0105, Absorption: maths.RGB(115, 310, 515), Backscatter: maths.RGB(290, 440, 570)},
		}, .65)
		if err != nil {
			return good{}, err
		}
	}
	return good{native: []thing.Thing{{Shape: shape, Material: material}}}, nil
}
