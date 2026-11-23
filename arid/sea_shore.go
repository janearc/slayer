// This file is a sea with no edge meeting the ground of a set: where its
// water lies, how its waves stand over shallow water, and a count of what
// that changed.

package arid

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/oceantex"
)

// Shoreline is what a coast did to a sea, counted at the ground's samples,
// which are the corners of its triangles.
type Shoreline struct {
	// Samples is how many heights the ground holds.
	Samples int
	// Land is how many are above the sea's level, and Lapped how many of
	// those the open sea's waves stood over.
	Land, Lapped int
	// Hollow is how many are below the sea's level and not joined to the
	// open sea, and Filled how many of those the open waves stood over.
	Hollow, Filled int
	// Wet is how many land or hollow samples the sea still stands over.
	// At a sample the share is all or nothing, so this is nought by the
	// way it is counted: it is kept for the tests and left out of String,
	// whose counts are the open sea's.
	Wet int
	// Whole is the depth, in metres, from which the waves stand whole.
	Whole float64
	// Open and Shored are the most the surface rises a metre, the bound
	// the search steps by, for the open sea and the sea over the ground.
	Open, Shored float64
}

// String is the shoreline in a sentence.
func (counted Shoreline) String() string {
	return fmt.Sprintf("sea over ground: of %d heights, %d are above the "+
		"sea's level and the open waves stood over %d; %d are below it "+
		"and not joined to the sea, and the open waves filled %d. The "+
		"waves stand whole from %.3g m of water; the surface rises at "+
		"most %.4g a metre, %.4g open",
		counted.Samples, counted.Land, counted.Lapped, counted.Hollow,
		counted.Filled, counted.Whole, counted.Shored, counted.Open)
}

// Ashore is the sea meeting a ground: east and south are where the
// ground's first sample is in the sea's frame, in metres, and level is
// the sea's mean level in the ground's own heights. It gives the sea over
// that ground and what it changed, counted.
//
// Water stands only where the open sea reaches (coast.flood): a place
// below the sea's level that no edge of the ground's triangles joins to
// the open sea through places below it is dry. Off the ground's grid
// there is water everywhere, and the depth is its nearest edge sample's.
//
// Over each place the waves are cut to the share the depth carries
// (coast.meet), so the water rises and falls about its mean by at most
// half the breaking ratio of the depth, and not at all over land. The
// waves keep their deep-water lengths and directions.
//
// What this does not do: shoaling, refraction, set-up, run-up, a swash
// line, breaking as foam, or the depth's change to the spectrum. A place
// is wet or dry at the grid's resolution, so a channel narrower than the
// ground's step joins nothing.
//
// The search's step stays sound; see shoredSteepest. A ray that must
// reach the surface through a dry triangle is not given the water beyond
// it: the direct sun's refracted ray, from a point under shore water,
// that crosses a dry column before it reaches the mean plane finds no
// boundary, and the draw is refused.
func (sea *Sea) Ashore(ground Ground, east, south, level float64) (*Sea,
	Shoreline, error) {
	if sea == nil {
		return nil, Shoreline{}, errors.New("arid: no sea to meet the ground")
	}
	shore, err := newCoast(ground, east, south, level, sea.reach)
	if err != nil {
		return nil, Shoreline{}, err
	}
	shored := sea.meeting(shore)
	shored.steepest = shoredSteepest(sea.steepest, shore.steepest())
	return shored, shored.shoreline(sea), nil
}

// meeting is a copy of the sea that meets a coast: every field the sea holds
// but its count of rays ended on the mean plane, which starts again at
// nought, as a field of a sea made by seaOf does. The sea's own steepest
// rise is carried, for the caller to widen, and kept as the open sea's.
// A field added to Sea is carried here.
func (sea *Sea) meeting(shore *coast) *Sea {
	return &Sea{bands: sea.bands, slopes: sea.slopes, reach: sea.reach,
		steepest: sea.steepest, open: sea.steepest, period: sea.period,
		finest: sea.finest, resolved: sea.resolved, unseen: sea.unseen,
		coast: shore}
}

// shoredSteepest is the most a sea's surface over a coast can rise a
// metre, from the open sea's own, open, and the ground's steepest rise in
// the triangles where the share of the waves changes, ground.
//
// The surface is the share times the open surface. Between two places a
// metre apart the open surface changes by at most open; the share by at
// most ground over full, and by nothing where it is held at none or all.
//
// The share is at most one and the open surface at most reach from the
// mean, so the surface changes by at most open plus reach times ground
// over full. full is twice reach over the breaking ratio, so that is open
// plus half the breaking ratio times ground, whatever the reach.
func shoredSteepest(open, ground float64) float64 {
	return open + oceantex.BreakingRatio/sides*ground
}

// shoreline counts what the sea's coast changed from the open sea.
func (sea *Sea) shoreline(open *Sea) Shoreline {
	shore := sea.coast
	counted := Shoreline{Samples: len(shore.height), Whole: shore.full,
		Open: open.steepest, Shored: sea.steepest}
	for at, height := range shore.height {
		east := shore.east + float64(at%shore.across)*shore.step
		south := shore.south + float64(at/shore.across)*shore.step
		counted.count(height, shore.cut[at], open.height(east, south),
			sea.standing(east, south))
	}
	return counted
}

// standing is the sea's height at a place, or minus infinity where the
// coast lets no water stand.
func (sea *Sea) standing(east, south float64) float64 {
	if !sea.coast.wet(east, south) {
		return math.Inf(-1)
	}
	return sea.height(east, south)
}

// count adds one sample to the shoreline: the ground there above the
// sea's mean, whether it is cut off from the sea, and how high the open
// sea and the sea over the ground stand over it.
func (counted *Shoreline) count(ground float64, cut bool, open,
	shored float64) {
	switch {
	case ground > 0:
		counted.Land++
		counted.Lapped += over(open, ground)
		counted.Wet += over(shored, ground)
	case cut:
		counted.Hollow++
		counted.Filled += over(open, ground)
		counted.Wet += over(shored, ground)
	}
}

// over is one if a surface stands above the ground, and nought if not.
func over(surface, ground float64) int {
	if surface > ground {
		return 1
	}
	return 0
}
