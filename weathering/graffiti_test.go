package weathering_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/weathering"
)

// the numbers the graffiti tests lay a wall out by, each a choice of the
// tests
const (
	// roofHeight is how high the test roof is, in metres: far over the
	// street's reach.
	roofHeight = 100.0
	// roofDepth is how far the test roof runs back from its wall.
	roofDepth = 5.0
	// parapetFoot and parapetTop are the heights, over the roof, between
	// which the test sites on a parapet's face lie.
	parapetFoot = 0.4
	parapetTop  = 2.2
	// wallStep is the step of the grid of sites on a test wall, in metres:
	// fine enough to see inside a mark.
	wallStep = 0.05
	// wallLong is how far the grid of sites runs along a test wall.
	wallLong = 12.0
	// shunned is the share a writer keeps on what they shun in the tests,
	// and sought how much more they put on what they seek.
	shunned = 0.05
	sought  = 2.0
	// cellSquash is how many times wider than tall a cell of marks is.
	cellSquash = 2.2
)

// writer is a graffiti agent in three paints over a reach a little over
// a man's, every cell marked unless share says less.
func writer(share float64) weathering.Agent {
	return weathering.Agent{Kind: weathering.Graffiti, Name: "writer",
		Colour: maths.RGB(0.6, 0.05, 0.04), Other: maths.RGB(0.05, 0.25, 0.6),
		Line: maths.RGB(0.01, 0.01, 0.01), Strength: 1, Level: 0.2,
		Grain: 1.2, Low: 0.3, High: 2.4, Share: share, Shelter: 3}
}

// wallSites is a fine grid of sites on a wall at z of nought facing
// south, of a material, from one height to another.
func wallSites(name string, from, to float64) []*weathering.Site {
	var all []*weathering.Site
	for x := -wallLong / 2; x < wallLong/2; x += wallStep {
		for y := from; y < to; y += wallStep {
			all = append(all, &weathering.Site{Point: maths.V(x, y, 0),
				Normal: faceSouth, Name: name})
		}
	}
	return all
}

// roofBehind is a shelter holding a roof that runs back from the wall at
// z of nought, at the roof's height.
func roofBehind() *weathering.Shelter {
	shelter := weathering.NewShelter(testBlock)
	plate(shelter, maths.V(-ledgeSpan, roofHeight, -roofDepth),
		maths.V(ledgeSpan, roofHeight, 0), "roof")
	return shelter
}

// TestAMarkIsShapesInThreePaints says that a wall where every cell is
// marked is not painted over: some of it is bare between and inside the
// shapes, and of what is painted some is the line's ink and some a fill.
func TestAMarkIsShapesInThreePaints(t *testing.T) {
	agent := writer(1)
	layer := layerOf(nil, agent)
	var bare, inked, filled int
	for _, site := range wallSites("wall", agent.Low, agent.High) {
		if !(layer.Amount(agent, site) > 0) {
			bare++
			continue
		}
		if layer.Lay(maths.Light{}, site) == agent.Line {
			inked++
			continue
		}
		filled++
	}
	if bare == 0 || inked == 0 || filled == 0 {
		t.Errorf("marks were bare %d, inked %d, filled %d: each must be "+
			"some", bare, inked, filled)
	}
}

// paintsOn is the distinct paints an agent lays over the sites of a wall
// where it lays any, with a count of the sites of each.
func paintsOn(agent weathering.Agent, sites []*weathering.Site) map[maths.Light]int {
	layer := layerOf(nil, agent)
	found := map[maths.Light]int{}
	for _, site := range sites {
		if layer.Amount(agent, site) >= 1 {
			found[layer.Lay(maths.Light{}, site)]++
		}
	}
	return found
}

// TestAThrowUpHasAHighlightAndATagHasNone says that a writer of throw-ups
// lays a third paint, lighter than the fill, on the top of each bubble,
// and a writer of tags lays only the fill and the line.
func TestAThrowUpHasAHighlightAndATagHasNone(t *testing.T) {
	agent := writer(1)
	agent.Other, agent.Grain = agent.Colour, 3
	agent.Mark = weathering.MarkThrowUp
	sites := wallSites("wall", agent.Low, agent.High)
	if got := len(paintsOn(agent, sites)); got != 3 {
		t.Errorf("throw-ups were laid in %d paints, not fill, line and "+
			"highlight", got)
	}
	agent.Mark = weathering.MarkTag
	if got := len(paintsOn(agent, sites)); got > 2 {
		t.Errorf("tags were laid in %d paints, and may be fill and line", got)
	}
}

// TestAThrowUpDripsBelowItsBubbles says that, of the cells of a wall marked
// with throw-ups, some have as their lowest paint the fill and not the
// outline: a drip runs below the bubbles.
func TestAThrowUpDripsBelowItsBubbles(t *testing.T) {
	agent := writer(1)
	agent.Grain, agent.Mark = 3, weathering.MarkThrowUp
	layer := layerOf(nil, agent)
	lowest := map[[2]int]*weathering.Site{}
	for _, site := range wallSites("wall", 0, agent.High) {
		if !(layer.Amount(agent, site) >= 1) {
			continue
		}
		cell := [2]int{int(math.Floor(site.Point.X / agent.Grain)),
			int(math.Floor(site.Point.Y * cellSquash / agent.Grain))}
		if held, ok := lowest[cell]; !ok || site.Point.Y < held.Point.Y {
			lowest[cell] = site
		}
	}
	var dripped int
	for _, site := range lowest {
		if layer.Lay(maths.Light{}, site) != agent.Line {
			dripped++
		}
	}
	if dripped == 0 {
		t.Error("no cell of throw-ups had a drip under its bubbles")
	}
}

// TestNoMarkOutOfReach says that a wall high over the street with nothing
// to stand on under it holds no mark, however far a writer will climb.
func TestNoMarkOutOfReach(t *testing.T) {
	agent := writer(1)
	agent.Heaven, agent.Behind = 1, 0.6
	layer := layerOf(weathering.NewShelter(testBlock), agent)
	if got := total(layer, agent, wallSites("wall", 20, 24)); got != 0 {
		t.Errorf("a wall out of reach held %v of marks", got)
	}
}

// TestAWriterReachesOverAParapet says that the face of a parapet over a
// roof holds marks where a writer climbs, and none where they do not.
func TestAWriterReachesOverAParapet(t *testing.T) {
	agent := writer(1)
	agent.Heaven, agent.Behind = 1, 0.6
	parapet := wallSites("parapet", roofHeight+parapetFoot,
		roofHeight+parapetTop)
	climbing := layerOf(roofBehind(), agent)
	if got := total(climbing, agent, parapet); !(got > 0) {
		t.Error("a parapet over a roof held no marks")
	}
	agent.Heaven = 0
	grounded := layerOf(roofBehind(), agent)
	if got := total(grounded, agent, parapet); got != 0 {
		t.Errorf("a writer who does not climb left %v on a parapet", got)
	}
}

// TestNoMarkUnderCover says that a writer standing behind a wall with a
// roof over them, as on any storey under the top, leaves nothing on it.
func TestNoMarkUnderCover(t *testing.T) {
	agent := writer(1)
	agent.Heaven, agent.Behind = 1, 0.6
	shelter := roofBehind()
	plate(shelter, maths.V(-ledgeSpan, roofHeight+parapetTop+1, -roofDepth),
		maths.V(ledgeSpan, roofHeight+parapetTop+1, 0), "ceiling")
	parapet := wallSites("wall", roofHeight+parapetFoot,
		roofHeight+parapetTop)
	if got := total(layerOf(shelter, agent), agent, parapet); got != 0 {
		t.Errorf("a wall under a ceiling held %v of marks", got)
	}
}

// TestWritersSeekAndShun says that a writer marks what they seek more than
// anything else, and what they shun less.
func TestWritersSeekAndShun(t *testing.T) {
	agent := writer(0.2)
	agent.Seek, agent.Shun = []string{"shutter"}, []string{"polished"}
	agent.Sought, agent.Shunned = sought, shunned
	layer := layerOf(nil, agent)
	on := func(name string) float64 {
		return total(layer, agent, wallSites(name, agent.Low, agent.High))
	}
	shutter, plain, polished := on("cities shutter"), on("cities podium"),
		on("cities granite lobby polished")
	if !(shutter > plain && plain > polished) {
		t.Errorf("a shutter held %v, a plain wall %v and polished stone %v",
			shutter, plain, polished)
	}
}

// TestAShelterFindsWhatIsUnder says that a look down finds a floor within
// reach and nothing past it, and nothing over a place.
func TestAShelterFindsWhatIsUnder(t *testing.T) {
	shelter := shelterWithLedge("ledge")
	over := maths.V(0, ledgeHeight+1, ledgeDepth/2)
	if drop, found := shelter.Below(over, 2); !found || drop != 1 {
		t.Errorf("a ledge a metre down was found %v at %v", found, drop)
	}
	if _, found := shelter.Below(over, 0.5); found {
		t.Error("a ledge past reach was found")
	}
	under := maths.V(0, ledgeHeight-1, ledgeDepth/2)
	if _, found := shelter.Below(under, 2); found {
		t.Error("a ledge over a place was found under it")
	}
}
