package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
)

// the size and the sight of a kind, asked before anything is made: a plan
// of a set asks each kind how big it is and whether a camera sees it, and
// builds only what it needs. A kind answers extent from its recipe's
// numbers; one that cannot yet answers by greedyExtent, so the kinds that
// still make themselves to say how big they are are found by name.

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
	return row.how.extent(from)
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
	return row.how.seen(from, at, by)
}

// seenFrom is the answer to seen for every maker that has none of its own:
// the box of its extent, placed where the thing stands and projected
// through the eye. It refuses an eye that sees nothing.
func seenFrom(row maker, from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	if err := by.Check(); err != nil {
		return strategies.Seen{}, err
	}
	box, err := row.extent(from)
	if err != nil {
		return strategies.Seen{}, err
	}
	return by.Cover(transformed(box, at)), nil
}

// greedyExtent is the box of a maker that cannot yet say how big it is
// from its recipe: it makes the thing once, at the level its made gives,
// and takes the bounds of that. It is the step the plan of a set is meant
// to be rid of, and every maker that calls it is a kind still to be taught
// its own size. Find them by this name.
func greedyExtent(row maker, from seed) (maths.Box, error) {
	made, err := row.made(from)
	if err != nil {
		return maths.Box{}, err
	}
	return made.Bounds(), nil
}
