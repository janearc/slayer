package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
)

// A set's plan asks how big each kind is and whether a camera sees it,
// then builds only what it needs. A kind answers from its recipe's numbers;
// one that cannot yet is declared for greedyExtent, which makes the thing
// once to say how big it is.

// extentOf is the box of a kind from a seed, or why there is none. It is
// the one door, as makeOf is: a kind with no maker and a seed not sown are
// refused as they are there.
func extentOf(row kind, from seed) (maths.Box, error) {
	if row.how == nil {
		return maths.Box{}, errKind
	}
	if !from.made() {
		return maths.Box{}, errGood
	}
	return makerExtent(row.how, from)
}

// seenOf is how an eye sees a kind from a seed standing at a place, or why
// it cannot say.
func seenOf(row kind, from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	if row.how == nil {
		return strategies.Seen{}, errKind
	}
	if !from.made() {
		return strategies.Seen{}, errGood
	}
	if observer, ok := row.how.(interface {
		seen(seed, maths.Subspace, strategies.Eye) (strategies.Seen, error)
	}); ok {
		return observer.seen(from, at, by)
	}
	return seenFrom(row.how, from, at, by)
}

// seenFrom is the answer to seen for every maker that has none of its own:
// the box of its extent, placed where the thing stands and projected
// through the eye. It refuses an eye that sees nothing.
func seenFrom(row maker, from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	if err := by.Check(); err != nil {
		return strategies.Seen{}, err
	}
	box, err := makerExtent(row, from)
	if err != nil {
		return strategies.Seen{}, err
	}
	return by.Cover(transformed(box, at)), nil
}

// makerExtent retains an authored recipe bound when available and otherwise
// obtains the same greedy bound as the old per-maker method.
func makerExtent(row maker, from seed) (maths.Box, error) {
	if sized, ok := row.(interface{ extent(seed) (maths.Box, error) }); ok {
		return sized.extent(from)
	}
	if !declaredGreedy(row) {
		return maths.Box{}, fmt.Errorf("manufacture: maker %T has no declared extent", row)
	}
	return greedyExtent(row, from)
}

// declaredGreedy records every recipe for which asking extent deliberately
// makes the specimen. New recipes must provide a bound or be named here.
func declaredGreedy(row maker) bool {
	switch row.(type) {
	case
		voyagerMaker, *voyagerMaker, home, *home, civicObservatory, *civicObservatory, gargoyleRow, *gargoyleRow,
		signRow, *signRow, jimo, *jimo, summitInstrument, *summitInstrument, bubblesRow, *bubblesRow,
		groundObservatory, *groundObservatory, bivouacRow, *bivouacRow, gargoyleCorniceRow, *gargoyleCorniceRow, lunarModule, *lunarModule,
		j58Engine, *j58Engine, j58Stand, *j58Stand, cityMaker, *cityMaker, saucer, *saucer,
		yurtRow, *yurtRow, newHorizonsVTAD, *newHorizonsVTAD, truck, *truck, hatzGargoyleKind, *hatzGargoyleKind,
		coralMaker, *coralMaker, scavengedRow, *scavengedRow, pyre, *pyre, soho, *soho,
		yardEquipment, *yardEquipment, guideLauncher, *guideLauncher, stone, *stone, fern, *fern,
		can, *can, log, *log, cassiniHuygens, *cassiniHuygens, veneraLander, *veneraLander,
		buildingMaker, *buildingMaker, spacecraftSpecMaker, *spacecraftSpecMaker, shootKind, *shootKind, winnebagoRow, *winnebagoRow,
		personalTelescope, *personalTelescope, scaffolding, *scaffolding, physicalSpruce, *physicalSpruce, retailBlockRow, *retailBlockRow,
		cityFamily, *cityFamily, fire, *fire, stick, *stick, spot, *spot,
		maser, *maser, newHorizons, *newHorizons, birdKind, *birdKind, landformMaker, *landformMaker,
		kelpRow, *kelpRow, owlShelter, *owlShelter, alfaBoat, *alfaBoat, owlObservatory, *owlObservatory,
		beamhead, *beamhead, optical, *optical, mariner10, *mariner10, frontageRow, *frontageRow,
		columnarCactus, *columnarCactus, fishRow, *fishRow, clusteredCityMaker, *clusteredCityMaker, terraceRow, *terraceRow,
		conifer, *conifer, macrostructureMaker, *macrostructureMaker, whaleSwim, *whaleSwim, whaleStill, *whaleStill,
		theropodKind, *theropodKind, sealKind, *sealKind, squidKind, *squidKind, reptileStudy, *reptileStudy:
		return true
	default:
		return false
	}
}

// seenObject projects an object's checked physical extent through the eye.
func seenObject(row interface{ Extent() (maths.Box, error) }, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	if err := by.Check(); err != nil {
		return strategies.Seen{}, err
	}
	box, err := row.Extent()
	if err != nil {
		return strategies.Seen{}, err
	}
	return by.Cover(transformed(box, at)), nil
}

// greedyExtent makes a thing once and takes its bounds. It is the step a
// set's plan is meant to avoid; each declared maker still needs to be taught
// its own size from its recipe.
func greedyExtent(row maker, from seed) (maths.Box, error) {
	made, err := row.made(from)
	if err != nil {
		return maths.Box{}, err
	}
	return made.Bounds(), nil
}
