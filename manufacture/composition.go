package manufacture

import (
	"slices"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
)

type compositionRow struct{ spec cities.Composition }

// Composition makes a kind from a checked snapshot of a structure.
func Composition(spec cities.Composition) (Kind, error) {
	spec.Parts = slices.Clone(spec.Parts)
	for i := range spec.Parts {
		spec.Parts[i].RestsOn = slices.Clone(spec.Parts[i].RestsOn)
		spec.Parts[i].HangsFrom = slices.Clone(spec.Parts[i].HangsFrom)
		spec.Parts[i].SupportedBy = slices.Clone(spec.Parts[i].SupportedBy)
	}
	spec.Roads = slices.Clone(spec.Roads)
	if err := spec.Validate(); err != nil {
		return Kind{}, err
	}
	return Kind{name: spec.Name, how: compositionRow{spec}}, nil
}

func (row compositionRow) rungs() []string { return Complexities() }

func (row compositionRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

func (row compositionRow) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	design, err := row.spec.DesignSeed(cities.Seed(from.number))
	if err != nil {
		return good{}, err
	}
	drawn, err := design.Rendered(cities.Detail(detail))
	if err != nil {
		return good{}, err
	}
	return goodOfRendered(drawn)
}

func (row compositionRow) extent(from seed) (maths.Box, error) {
	design, err := row.spec.DesignSeed(cities.Seed(from.number))
	if err != nil {
		return maths.Box{}, err
	}
	return designExtent(design)
}

func (row compositionRow) seen(from seed, at maths.Subspace, by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}
