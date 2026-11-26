package cities

import (
	"fmt"
	"math"
)

// bridgeCulture is the data every bridge is drawn from. Its scopes: a row
// is the whole bridge, a bay is one station of pylons, a storey is one
// stretch of deck, and an opening is one lot on the deck.
const bridgeCulture = "bridge"

// lotSeedBits is how many of a bridge seed's low bits number its lots: a
// lot's seed is the bridge's, moved up by that many, and the lot's index.
const lotSeedBits = 16

// pylonSeedBits is the same for the towers of the pylons.
const pylonSeedBits = 8

// Bridge is a span whose structure is buildings. Old London Bridge carried
// houses on its deck; this carries a district, and is held up by towers.
//
// There is no ceiling on its height and no limit on its span or its
// materials: a bridge of two kilometres has pylons that are inhabited
// towers, drawn by NewTower, and a deck that is a level to build on, with
// a street down the middle and terraces, condominiums and towers along its
// length on both sides, each lot a design of its own kind stood on the
// deck.
//
// A main cable hangs in a catenary from the top of each pylon to the next
// and to the shore, and hangers drop from it to the deck. At each end an
// approach steps down to the ground, its buildings getting lower.
//
// The irregularity, from nought to one, gates what strays from that: a
// deck that sags between stations, that steps at each station, or that
// carries a second deck, and cables of several strands.
//
// Any kind is a part of a bigger kind. A lot of the deck may itself be a
// bridge, and its lots bridges, down to the depth the set gives, so that
// the seed decides how deep it goes.
type Bridge struct {
	Irregularity float64
	// Span is the distance between the two shores, in metres.
	Span float64
	// Depth is how many more bridges may be lots of this bridge's deck,
	// counted down: a bridge of depth nought has no bridge for a lot.
	Depth int
	// Stations are the places along the span, in metres from its middle,
	// where a pair of pylons stands, one each side of the deck.
	Stations []float64
	// Sag, Step and Second say that the deck sags, steps, or has a second
	// deck over it. Strands is how many strands each cable has.
	Sag, Step, Second bool
	Strands           int
	// Bare says the bridge has no lots on its deck and a deck as wide as
	// a road, as a bridge that carries traffic and not a district does.
	Bare bool
	// Hub says the bridge is a tower in the middle with an arm out of each
	// face and both decks' roads running through it (see hubDesign).
	Hub     bool
	seed    Seed
	drawing *drawer
	pylons  []*Tower
}

// NewBridge draws a bridge of a span in metres from a seed at an
// irregularity. A span of nought or less is the culture's default, and a
// depth under nought is its default depth. It refuses an irregularity
// outside nought to one.
func NewBridge(seed Seed, irregularity Irregularity, span float64,
	depth int) (*Bridge, error) {
	return newBridge(seed, irregularity, span, depth, false)
}

// NewBareBridge draws a bridge as NewBridge does, but a plain one: a deck
// as wide as a road, with the pylons and the cables and nothing built on
// it, and no lots to nest, so it has no depth.
func NewBareBridge(seed Seed, irregularity Irregularity,
	span float64) (*Bridge, error) {
	return newBridge(seed, irregularity, span, 0, true)
}

// NewHubBridge draws a bridge as NewBareBridge does, and makes it a hub:
// a tower in the middle that the decks run through, with an arm out of
// each face.
func NewHubBridge(seed Seed, irregularity Irregularity,
	span float64) (*Bridge, error) {
	bridge, err := newBridge(seed, irregularity, span, 0, true)
	if bridge != nil {
		bridge.Hub = true
	}
	return bridge, err
}

// newBridge is the bridge of either kind, bare or built on.
func newBridge(seed Seed, irregularity Irregularity, span float64,
	depth int, bare bool) (*Bridge, error) {
	drawing, err := newDrawer(bridgeCulture, seed)
	if err != nil {
		return nil, err
	}
	reach, err := drawing.resolved(irregularity)
	if err != nil {
		return nil, err
	}
	if !(span > 0) {
		span = drawing.fixed("default-span")
	}
	if depth < 0 {
		depth = int(drawing.fixed("default-depth"))
	}
	bridge := &Bridge{Irregularity: reach, Span: span, Depth: depth,
		Bare: bare, seed: seed, drawing: drawing}
	if err := bridge.draw(); err != nil {
		return nil, err
	}
	return bridge, drawing.trouble
}

// draw is everything about the bridge that is not a lot: the stations,
// the shape of the deck, the cables, and the pylon towers.
func (bridge *Bridge) draw() error {
	drawing, reach := bridge.drawing, bridge.Irregularity
	count := max(2, int(bridge.Span/drawing.number("pylon-spacing",
		place{})))
	for index := range count {
		bridge.Stations = append(bridge.Stations, -bridge.Span/2+
			float64(index+1)*bridge.Span/float64(count+1))
	}
	bridge.Sag = drawing.gate("deck-sag", place{}, reach, nil)
	bridge.Step = drawing.gate("deck-step", place{}, reach, nil)
	bridge.Second = drawing.gate("second-deck", place{}, reach, nil)
	bridge.Strands = drawing.reach("cable-strands", place{}, reach)
	for index := range bridge.Stations {
		tower, err := NewTower(Seed(uint64(bridge.seed)<<pylonSeedBits|
			uint64(index)), Irregularity{Value: math.Min(1,
			reach*drawing.fixed("pylon-reach"))}, Pins{Shafts: 1,
			Span: SpanNone})
		if err != nil {
			return fmt.Errorf("cities: pylon %d: %w", index, err)
		}
		bridge.pylons = append(bridge.pylons, tower)
	}
	if drawing.trouble != nil {
		return drawing.trouble
	}
	return bridge.check()
}

// catenaryDrop is how far a hanging cable is below the line between its
// supports at a distance along them, for the sag it has at its lowest. A
// cable hanging under its own weight is a catenary, a cosh, whose
// parameter is found from the sag by bisection.
func catenaryDrop(length, sag, along float64) float64 {
	const (
		steps   = 80
		tightly = 1e-6
		loosely = 1e3
	)
	low, high := tightly*length, loosely*length
	for range steps {
		middle := (low + high) / 2
		if middle*(math.Cosh(length/(2*middle))-1) > sag {
			low = middle
		} else {
			high = middle
		}
	}
	a := (low + high) / 2
	return a * (math.Cosh(length/(2*a)) - math.Cosh((along-length/2)/a))
}
