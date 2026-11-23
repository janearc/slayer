package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
)

// the makers the tests stand in for kinds with. Each says its box by making
// itself, as a kind with no recipe does, and is seen from that box.

// extent is the box of the moving part, made once to say.
func (maker movingPartMaker) extent(from seed) (maths.Box, error) {
	return greedyExtent(maker, from)
}

// seen is how an eye sees the moving part, from its extent.
func (maker movingPartMaker) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(maker, from, at, by)
}

// extent is the box of the timed maker, made once to say.
func (maker timedTestMaker) extent(from seed) (maths.Box, error) {
	return greedyExtent(maker, from)
}

// seen is how an eye sees the timed maker, from its extent.
func (maker timedTestMaker) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(maker, from, at, by)
}

// extent is the box of the maker that always fails, which is its failure.
func (broken) extent(from seed) (maths.Box, error) {
	return greedyExtent(broken{}, from)
}

// seen is how an eye sees the maker that always fails, from its extent.
func (broken) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(broken{}, from, at, by)
}
