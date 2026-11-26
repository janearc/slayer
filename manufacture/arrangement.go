package manufacture

import (
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture/branching"
)

// Growth geometry is shared with the branching engine.
type size = branching.Size
type place = branching.Place

const (
	// fullTurn is once round, in radians.
	fullTurn = 2 * math.Pi
	// midway is the middle of a whorl's own share of its stem.
	midway = 0.5
)

// habit is the shared branching arrangement data used by this maker.
type habit = branching.Habit

var errHabit = branching.ErrHabit

// finite is a number that is finite.
func finite(number float64) bool {
	return !math.IsNaN(number) && !math.IsInf(number, 0)
}

// share is a number from zero to one.
func share(number float64) bool { return number >= 0 && number <= 1 }

// Ring and whorl arrangements share the branching engine's representation.
type whorls = branching.Whorls
type ring = branching.Ring

var errWhorls = branching.ErrWhorls
var errStem = branching.ErrStem
var errRing = branching.ErrRing
var newWhorls = branching.NewWhorls

// newRing checks a ring while keeping the manufactory units at its boundary.
func newRing(count int, radius measure.Length, lean measure.Angle, stray, uneven, twist float64) (ring, error) {
	return branching.NewRing(count, inScene(radius), lean.InRadians(), stray, uneven, twist)
}
