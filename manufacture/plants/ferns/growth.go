package ferns

import (
	"cmp"
	"fmt"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture/branching"
)

// Growth makes a fern crown carrying fronds, each bearing paired leaflets.
// body and leaf name the maker's material roles for stalk and blades.
func Growth(fronds, leaflets, body, leaf int, anatomy Anatomy) (branching.Growth, error) {
	if !anatomy.valid() {
		return branching.Growth{}, fmt.Errorf("invalid fern anatomy")
	}
	const crownSides, stalkSides = 6, 4
	const lean, stray, uneven = -35, 0.3, 0.2
	blade, bladeError := branching.NewLeaf([]branching.Joint{{OfGirth: 1}, {OfGirth: anatomy.LeafletTip}}, leaf)
	stalk, stalkError := branching.NewSleeve([]branching.Joint{{OfGirth: 1}, {OfGirth: anatomy.StalkTip}}, stalkSides, body)
	pairs, pairsError := branching.NewWhorls(leaflets, 1, anatomy.habit())
	frond, frondError := branching.StemCarrying(1, 0, []branching.Sleeve{stalk}, pairs, 1, 0,
		[]branching.Sleeve{blade})
	stub, stubError := branching.NewSleeve([]branching.Joint{{OfGirth: anatomy.CrownRadius}, {OfGirth: anatomy.CrownRadius}}, crownSides, body)
	crown, crownError := branching.NewLimb(1, anatomy.CrownSpan, 0, stub)
	round, roundError := branching.NewRing(fronds, 0, measure.Degrees(lean).InRadians(), stray, uneven, 0)
	if err := cmp.Or(bladeError, stalkError, pairsError, frondError,
		stubError, crownError, roundError); err != nil {
		return branching.Growth{}, err
	}
	return branching.NewGrowth(crown, branching.Carried{Where: round, What: frond})
}
