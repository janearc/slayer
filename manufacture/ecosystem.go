package manufacture

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	"github.com/janearc/slayer/scene/biome"
)

// errEcosystem is the refusal of an ecosystem whose footprint, as far as
// the ground goes, or whose height band holds none of the ground, so there
// is nowhere for a member to be rooted.
var errEcosystem = errors.New("manufacture: an ecosystem's footprint and " +
	"height band hold none of the ground")

// Ecosystem is a mixed population of a place: a region of the map in which
// a seed roots members of a mixture of recipes at a density, over the ground
// there, between two heights. It is an object of the manufactory that makes
// no member: it says where roots may stand and which are chosen, and the
// stand it feeds makes each member the kind says.
//
// Live state does not live here. The work a run allows for choosing roots is
// shared by every ecosystem of a set, so the budget is the caller's, given to
// Roots; the stock of members made, and the ground a root is set on, stay
// with the staging that places them.
type Ecosystem struct {
	// Name is the ecosystem's.
	Name string
	// Population is the bounded generator of roots, which enumerates none
	// until it is asked.
	Population *biome.Region
	// Window is the ground's window in the map's east and south metres. A
	// scene place is the map's less the window's corner: x is the east less
	// its east, z the south less its south.
	Window biome.Rect
	// Within is the box of the ground in the scene's frame, its lowest and
	// highest heights the window's.
	Within maths.Box
	// Lowest is the height of the ground's lowest point over the sea, which
	// the scene's heights are taken from. Above and Below are the band of
	// heights over the sea in which a member may stand, open at both ends
	// where the set says none.
	Lowest, Above, Below float64
	// Canopy is how far over its root the tallest member of the mixture
	// stands, which the kinds of the mixture answer. It is nought until a
	// caller sets it, and then the box is of where roots may be.
	Canopy float64
}

// Ecosystem is an object.
var _ object = Ecosystem{}

// Roots is the roots the population chooses over the window, within what a
// budget allows. Each stands at a map's east and south, and is of a recipe
// and a seed. It makes no member and does not read the ground.
func (ecosystem Ecosystem) Roots(budget biome.Budget) (biome.Selection,
	error) {
	return ecosystem.Population.SelectWithUsage(ecosystem.Window, budget)
}

// extent is the box of the ecosystem, from its numbers alone.
func (ecosystem Ecosystem) extent(from seed) (maths.Box, error) {
	return ecosystem.Extent()
}

// Extent is the box that holds every root the population can choose over the
// window, in the scene's frame: the population's footprint where it meets
// the ground, as high and as low as the ground there and the height band let
// it be, and over it the canopy.
func (ecosystem Ecosystem) Extent() (maths.Box, error) {
	foot := ecosystem.Population.Bounds()
	window := ecosystem.Window
	east := [2]float64{math.Max(foot.MinEast, window.MinEast),
		math.Min(foot.MaxEast, window.MaxEast)}
	south := [2]float64{math.Max(foot.MinSouth, window.MinSouth),
		math.Min(foot.MaxSouth, window.MaxSouth)}
	low := math.Max(ecosystem.Within.Min.Y, ecosystem.Above-ecosystem.Lowest)
	high := math.Min(ecosystem.Within.Max.Y, ecosystem.Below-ecosystem.Lowest)
	if east[1] < east[0] || south[1] < south[0] || high < low {
		return maths.Box{}, errEcosystem
	}
	return maths.Box{Min: maths.V(east[0]-window.MinEast, low,
		south[0]-window.MinSouth), Max: maths.V(east[1]-window.MinEast,
		high+ecosystem.Canopy, south[1]-window.MinSouth)}, nil
}

// seen is how an eye sees the ecosystem's box.
func (ecosystem Ecosystem) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenObject(ecosystem, at, by)
}
