package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
)

// the makers that cannot yet say how big they are from their recipes. Each
// makes its thing once, to say, by greedyExtent: that is the one step the
// plan of a set is meant to be rid of, and each of these is a kind still to
// be taught its own size. They are found by the name greedyExtent, and the
// ones a set asks for come first.

// extent is the box of a bay seal, made once to say.
func (row baySealKind) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a bay seal, from its extent.
func (row baySealKind) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a voyager's part, made once to say.
func (row voyagerMaker) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a voyager's part, from its extent.
func (row voyagerMaker) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a home, made once to say.
func (row home) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a home, from its extent.
func (row home) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// seen is how an eye sees a spacecraft, from its extent.
func (row spacecraft) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a civic observatory, made once to say.
func (row civicObservatory) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a civic observatory, from its extent.
func (row civicObservatory) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a gargoyle, made once to say.
func (row gargoyleRow) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a gargoyle, from its extent.
func (row gargoyleRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a sign, made once to say.
func (row signRow) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a sign, from its extent.
func (row signRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of the jimo, made once to say.
func (row jimo) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees the jimo, from its extent.
func (row jimo) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a summit instrument, made once to say.
func (row summitInstrument) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a summit instrument, from its extent.
func (row summitInstrument) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a bubble column, made once to say.
func (row bubblesRow) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a bubble column, from its extent.
func (row bubblesRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a ground observatory, made once to say.
func (row groundObservatory) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a ground observatory, from its extent.
func (row groundObservatory) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a bivouac, made once to say.
func (row bivouacRow) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a bivouac, from its extent.
func (row bivouacRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a gargoyle cornice, made once to say.
func (row gargoyleCorniceRow) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a gargoyle cornice, from its extent.
func (row gargoyleCorniceRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a lunar module, made once to say.
func (row lunarModule) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a lunar module, from its extent.
func (row lunarModule) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a jet engine, made once to say.
func (row j58Engine) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a jet engine, from its extent.
func (row j58Engine) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a jet engine's stand, made once to say.
func (row j58Stand) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a jet engine's stand, from its extent.
func (row j58Stand) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a city, made once to say.
func (row cityMaker) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a city, from its extent.
func (row cityMaker) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a walrus, made once to say.
func (row walrusFresh5Kind) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a walrus, from its extent.
func (row walrusFresh5Kind) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a saucer, made once to say.
func (row saucer) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a saucer, from its extent.
func (row saucer) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a yurt, made once to say.
func (row yurtRow) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a yurt, from its extent.
func (row yurtRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a deep-space probe, made once to say.
func (row newHorizonsVTAD) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a deep-space probe, from its extent.
func (row newHorizonsVTAD) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a truck, made once to say.
func (row truck) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a truck, from its extent.
func (row truck) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a lofted seal, made once to say.
func (row sealLoftedKind) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a lofted seal, from its extent.
func (row sealLoftedKind) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a gargoyle of the hatz kind, made once to say.
func (row hatzGargoyleKind) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a gargoyle of the hatz kind, from its extent.
func (row hatzGargoyleKind) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of an orca, made once to say.
func (row orcaClean05Kind) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees an orca, from its extent.
func (row orcaClean05Kind) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a coral, made once to say.
func (row coralMaker) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a coral, from its extent.
func (row coralMaker) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a scavenged cluster, made once to say.
func (row scavengedRow) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a scavenged cluster, from its extent.
func (row scavengedRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a school of fish, made once to say.
func (row fishSchoolRow) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a school of fish, from its extent.
func (row fishSchoolRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a pyre, made once to say.
func (row pyre) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a pyre, from its extent.
func (row pyre) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a solar observatory craft, made once to say.
func (row soho) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a solar observatory craft, from its extent.
func (row soho) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a piece of yard equipment, made once to say.
func (row yardEquipment) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a piece of yard equipment, from its extent.
func (row yardEquipment) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a guide star launcher, made once to say.
func (row guideLauncher) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a guide star launcher, from its extent.
func (row guideLauncher) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a stone, made once to say.
func (row stone) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a stone, from its extent.
func (row stone) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a fern, made once to say.
func (row fern) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a fern, from its extent.
func (row fern) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a can, made once to say.
func (row can) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a can, from its extent.
func (row can) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a log, made once to say.
func (row log) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a log, from its extent.
func (row log) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a ringed-planet probe, made once to say.
func (row cassiniHuygens) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a ringed-planet probe, from its extent.
func (row cassiniHuygens) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a venus lander, made once to say.
func (row veneraLander) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a venus lander, from its extent.
func (row veneraLander) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a building, made once to say.
func (row buildingMaker) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a building, from its extent.
func (row buildingMaker) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a spacecraft from a spec, made once to say.
func (row spacecraftSpecMaker) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a spacecraft from a spec, from its extent.
func (row spacecraftSpecMaker) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a needled shoot, made once to say.
func (row shootKind) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a needled shoot, from its extent.
func (row shootKind) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a camper van, made once to say.
func (row winnebagoRow) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a camper van, from its extent.
func (row winnebagoRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a personal telescope, made once to say.
func (row personalTelescope) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a personal telescope, from its extent.
func (row personalTelescope) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of scaffolding, made once to say.
func (row scaffolding) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees scaffolding, from its extent.
func (row scaffolding) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a spruce, made once to say.
func (row physicalSpruce) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a spruce, from its extent.
func (row physicalSpruce) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a block of shops, made once to say.
func (row retailBlockRow) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a block of shops, from its extent.
func (row retailBlockRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a building of a city's family, made once to say.
func (row cityFamily) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a building of a city's family, from its extent.
func (row cityFamily) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a seal's bones, made once to say.
func (row harborSealBonesKind) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a seal's bones, from its extent.
func (row harborSealBonesKind) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of an orca in a still pose, made once to say.
func (row orcaClean05Still) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees an orca in a still pose, from its extent.
func (row orcaClean05Still) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a fire, made once to say.
func (row fire) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a fire, from its extent.
func (row fire) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a burning stick, made once to say.
func (row stick) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a burning stick, from its extent.
func (row stick) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a spotlight, made once to say.
func (row spot) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a spotlight, from its extent.
func (row spot) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a maser, made once to say.
func (row maser) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a maser, from its extent.
func (row maser) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a horizons probe, made once to say.
func (row newHorizons) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a horizons probe, from its extent.
func (row newHorizons) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a bird, made once to say.
func (row birdKind) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a bird, from its extent.
func (row birdKind) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// seen is how an eye sees a rocket stack, from its extent.
func (row stack) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a landform, made once to say.
func (row landformMaker) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a landform, from its extent.
func (row landformMaker) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a kelp, made once to say.
func (row kelpRow) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a kelp, from its extent.
func (row kelpRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of an owl shelter, made once to say.
func (row owlShelter) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees an owl shelter, from its extent.
func (row owlShelter) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a racing boat, made once to say.
func (row alfaBoat) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a racing boat, from its extent.
func (row alfaBoat) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of an owl observatory, made once to say.
func (row owlObservatory) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees an owl observatory, from its extent.
func (row owlObservatory) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a beam head, made once to say.
func (row beamhead) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a beam head, from its extent.
func (row beamhead) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of an optical observatory, made once to say.
func (row optical) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees an optical observatory, from its extent.
func (row optical) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a planetary probe, made once to say.
func (row mariner10) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a planetary probe, from its extent.
func (row mariner10) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a shop front, made once to say.
func (row frontageRow) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a shop front, from its extent.
func (row frontageRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a harbor seal, made once to say.
func (row harborSealKind) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a harbor seal, from its extent.
func (row harborSealKind) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a saguaro, made once to say.
func (row saguaro) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a saguaro, from its extent.
func (row saguaro) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a fish, made once to say.
func (row fishRow) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a fish, from its extent.
func (row fishRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a clustered city, made once to say.
func (row clusteredCityMaker) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a clustered city, from its extent.
func (row clusteredCityMaker) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a terrace, made once to say.
func (row terraceRow) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a terrace, from its extent.
func (row terraceRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a conifer, made once to say.
func (row conifer) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a conifer, from its extent.
func (row conifer) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a macrostructure, made once to say.
func (row macrostructureMaker) extent(from seed) (maths.Box, error) {
	return greedyExtent(row, from)
}

// seen is how an eye sees a macrostructure, from its extent.
func (row macrostructureMaker) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}
