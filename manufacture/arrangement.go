package manufacture

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// size is how big a part is: how long, and its radius at its foot.
type size struct{ length, girth measure.Length }

// place is where a part goes on what carries it, and how big it is
// there. the space is the part's own in its carrier's: its foot at the
// space's start, its length up the space's y. an arrangement that does
// not size its parts leaves the size empty and scales the space.
type place struct {
	space maths.Subspace
	size  size
}

const (
	// fullTurn is once round, in radians.
	fullTurn = 2 * math.Pi
	// midway is the middle of a whorl's own share of its stem.
	midway = 0.5
)

// habit is how a kind of plant sets its parts up a stem: every number
// that was written inside ninja's BranchOf, named. a share here is of
// the stem's length unless it says otherwise.
type habit struct {
	// turn is how far each whorl is turned from the one below.
	turn measure.Angle
	// start and top are where the lowest and the highest whorl would
	// sit, as shares of the stem.
	start, top float64
	// thinning is the share of the lowest whorl's parts that is gone
	// by the top, and fewest the least a whorl ever has.
	thinning float64
	fewest   int
	// stray is how far round a part strays, as a share of the gap to
	// its neighbour.
	stray float64
	// low is how far from straight up the lowest parts point, lift how
	// much of that is gone by the top, and sway how far a part strays
	// from it.
	low, lift, sway measure.Angle
	// reach is the longest part's length, shortening the share of it
	// gone by the top, and uneven how far a part strays from it.
	reach, shortening, uneven float64
	// girth is a part's radius as a share of the stem's where it
	// joins, and least the smallest it can be.
	girth, least float64
}

// spruceHabit is how a spruce sets its branches.
//
// the numbers are ninja's, from BranchOf, set by eye from ninja-gaiden's lesson
// of 2026-10-06; none is measured. they move to the kinds table, where
// published ones can replace them. the turn is the golden angle, which is
// worked out: the turn that never lines two whorls up.
func spruceHabit() habit {
	return habit{
		turn:  measure.Radians(math.Pi * (3 - math.Sqrt(5))),
		start: 0.10, top: 0.97,
		thinning: 0.55, fewest: 3,
		stray: 0.35,
		low:   measure.Degrees(100), lift: measure.Degrees(38),
		sway:  measure.Degrees(6),
		reach: 0.30, shortening: 0.88, uneven: 0.15,
		girth: 0.35, least: 0.012,
	}
}

var errHabit = errors.New("manufacture: no plant has this habit")

// finite is a number that is finite.
func finite(number float64) bool {
	return !math.IsNaN(number) && !math.IsInf(number, 0)
}

// share is a number from zero to one.
func share(number float64) bool { return number >= 0 && number <= 1 }

// sound is what is wrong with a habit, by the name of the number, or
// nothing.
func (set habit) sound() error {
	checks := []struct {
		name string
		good bool
	}{
		{"turn", finite(set.turn.InRadians())},
		{"start", share(set.start)},
		{"top", set.top > set.start && set.top <= 1},
		{"thinning", share(set.thinning)},
		{"fewest", set.fewest >= 1},
		{"stray", share(set.stray)},
		{"low", finite(set.low.InRadians())},
		{"lift", finite(set.lift.InRadians())},
		{"sway", finite(set.sway.InRadians())},
		{"reach", positive(set.reach)},
		{"shortening", share(set.shortening)},
		{"uneven", set.uneven >= 0 && set.uneven < 1},
		{"girth", positive(set.girth)},
		{"least", set.least >= 0 && finite(set.least)},
	}
	for _, check := range checks {
		if !check.good {
			return fmt.Errorf("%w: its %s", errHabit, check.name)
		}
	}
	return nil
}

// whorls is parts set up a stem in rings, each ring turned from the
// last: a conifer's branches, a fern's leaflets. it is ninja's
// BranchOf and BranchCount with their numbers moved out into a habit.
//
// made only by newWhorls.
type whorls struct {
	count, lowest int
	set           habit
}

var errWhorls = errors.New("manufacture: whorls need one whorl or " +
	"more, and one part or more in the lowest")

// newWhorls is so many whorls, with so many parts in the lowest, set
// by a habit, or why there can be none.
func newWhorls(count, lowest int, set habit) (whorls, error) {
	if count < 1 || lowest < 1 {
		return whorls{}, errWhorls
	}
	if err := set.sound(); err != nil {
		return whorls{}, err
	}
	return whorls{count: count, lowest: lowest, set: set}, nil
}

// rows is how many whorls there are.
func (up whorls) rows() int { return up.count }

// climb is how far up the crown a whorl is, from zero to one.
func (up whorls) climb(row int) float64 {
	return (float64(row) + midway) / float64(up.count)
}

// across is how many parts a whorl has: fewer the higher it is.
func (up whorls) across(row int) int {
	kept := 1 - up.set.thinning*up.climb(row)
	return max(up.set.fewest, int(math.Round(float64(up.lowest)*kept)))
}

var errStem = errors.New("manufacture: whorls stand on a stem with a " +
	"length and a girth above nought, each with an end")

// at is where one part of one whorl goes on a stem of a size, and how
// big it is, by the seed: its foot on the stem's axis, pointing out
// and down or up by how high it is. a stem of no size is refused, or
// every place on it would be no number and nothing would say so.
func (up whorls) at(stem size, row, index int,
	from seed) (place, error) {
	set, climb := up.set, up.climb(row)
	length, girth := inScene(stem.length), inScene(stem.girth)
	if !positive(length) || !positive(girth) {
		return place{}, errStem
	}
	height := length * (set.start + (set.top-set.start)*climb)
	gap := fullTurn / float64(up.across(row))
	turn := float64(row)*set.turn.InRadians() + gap*float64(index) +
		set.stray*from.swing(row, index, 0)*gap
	down := set.low.InRadians() - set.lift.InRadians()*climb +
		set.sway.InRadians()*from.swing(row, index, 1)
	reach := set.reach * length * (1 - set.shortening*climb) *
		(1 + set.uneven*from.swing(row, index, 2))
	stemHere := girth * (1 - height/length)
	way := maths.V(math.Sin(down)*math.Cos(turn), math.Cos(down),
		-math.Sin(down)*math.Sin(turn))
	// the way is a unit long whatever the angles, so it is a way to
	// go: the error is spaceAlong's, and is never seen from here
	space, err := spaceAlong(maths.V(0, height, 0), way)
	if err != nil {
		return place{}, err
	}
	thick := math.Max(set.girth*stemHere, set.least*length)
	return place{space: space, size: size{
		length: measure.Metres(reach),
		girth:  measure.Metres(thick)}}, nil
}

// ring is parts set round a middle, on the ground: the stones of a
// fire, the logs leaned over it, the fronds of a fern.
//
// made only by newRing.
type ring struct {
	count                int
	radius               measure.Length
	lean                 measure.Angle
	stray, uneven, twist float64
}

var errRing = errors.New("manufacture: a ring needs one part or more, " +
	"a radius of nought or above with an end, a lean that is a number, " +
	"a stray and an unevenness from nought to under one, and a twist " +
	"from nought to one")

// newRing is so many parts round a middle at a radius, each leaning in toward
// it by an angle (out, less than zero), straying round by a share of the gap
// between them, uneven in size by a share, and twisted about its own length by
// up to a share of a full turn, or why there can be none.
func newRing(count int, radius measure.Length, lean measure.Angle,
	stray, uneven, twist float64) (ring, error) {
	out := inScene(radius)
	sound := count >= 1 && out >= 0 && finite(out) &&
		finite(lean.InRadians()) && stray >= 0 && stray < 1 &&
		uneven >= 0 && uneven < 1 && share(twist)
	if !sound {
		return ring{}, errRing
	}
	return ring{count: count, radius: radius, lean: lean, stray: stray,
		uneven: uneven, twist: twist}, nil
}

// rows is one: a ring is a single row.
func (round ring) rows() int { return 1 }

// across is how many parts the ring has.
func (round ring) across(int) int { return round.count }

// at is where one part of the ring goes, whatever carries it: out from the
// middle on the ground, leaning in, twisted about its own length and scaled,
// each by the seed.
//
// before its twist a part's x runs round the ring, so every part faces the
// middle the same way: a fern's fronds all lie face up. it sizes nothing; the
// part comes at its carrier's size and the space scales it.
func (round ring) at(_ size, row, index int, from seed) (place, error) {
	gap := fullTurn / float64(round.count)
	angle := gap * (float64(index) + round.stray*from.swing(row, index, 0))
	out := maths.V(math.Cos(angle), 0, -math.Sin(angle))
	tangent := maths.V(-math.Sin(angle), 0, -math.Cos(angle))
	lean := round.lean.InRadians()
	way := maths.Up.Scale(math.Cos(lean)).Sub(out.Scale(math.Sin(lean)))
	facing := maths.Orientation{X: tangent, Y: way, Z: tangent.Cross(way)}
	twist := fullTurn * round.twist * from.draw(row, index, 1)
	return place{space: maths.Subspace{
		Origin:      out.Scale(inScene(round.radius)),
		Orientation: facing.Then(maths.About(maths.YAxis, twist)),
		Scale:       1 + round.uneven*from.swing(row, index, 2),
	}}, nil
}
