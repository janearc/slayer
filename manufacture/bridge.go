package manufacture

import "github.com/janearc/slayer/manufacture/cities"

// BridgeSpec is what a set says of a bridge recipe: how irregular its
// bridges are, or that each draws its own, the span in metres, and how
// many bridges deep a deck's lots may nest. An unsaid span or depth is the
// culture's default.
type BridgeSpec struct {
	// Irregularity is from nought to one; Drawn takes it from the
	// distribution of the bridge's data instead.
	Irregularity float64
	Drawn        bool
	// Span is the distance between the shores, in metres, or nought for
	// the default.
	Span float64
	// Depth is how deep bridges may nest as lots of a deck, if DepthSaid.
	Depth     int
	DepthSaid bool
	// Bare is a bridge with nothing built on its deck: the deck as wide
	// as a road, the pylons and the cables. It has no lots, so it says
	// no depth.
	Bare bool
}

// bridgeRow is the kind bridge: a span whose structure is buildings,
// drawn by package cities from the seed and made a good here. Without a
// recipe it is a plain bridge of the default span and depth.
type bridgeRow struct{ spec BridgeSpec }

// Bridge is the kind bridge with a recipe's irregularity, span and depth.
// It draws a bridge once to check them, and refuses what it cannot keep.
func Bridge(spec BridgeSpec) (Kind, error) {
	row := bridgeRow{spec: spec}
	if _, err := row.drawn(sow(0)); err != nil {
		return Kind{}, err
	}
	return Kind{name: "bridge", how: row}, nil
}

// drawn is the bridge of a seed, before it is made a good.
func (row bridgeRow) drawn(from seed) (*cities.Bridge, error) {
	depth := -1
	if row.spec.DepthSaid {
		depth = row.spec.Depth
	}
	if row.spec.Bare {
		return cities.NewBareBridge(cities.Seed(from.number),
			cities.Irregularity{Value: row.spec.Irregularity,
				Drawn: row.spec.Drawn}, row.spec.Span)
	}
	return cities.NewBridge(cities.Seed(from.number),
		cities.Irregularity{Value: row.spec.Irregularity,
			Drawn: row.spec.Drawn}, row.spec.Span, depth)
}

// made is the bridge at the quick detail level.
func (row bridgeRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the bridge at the named detail level, at time nought. It
// returns errGood when the seed is not made.
func (row bridgeRow) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	bridge, err := row.drawn(from)
	if err != nil {
		return good{}, err
	}
	design, err := bridge.Design(cities.Detail(detail), 0)
	if err != nil {
		return good{}, err
	}
	drawn, err := design.Rendered(cities.Detail(detail))
	if err != nil {
		return good{}, err
	}
	return goodOfRendered(drawn)
}
