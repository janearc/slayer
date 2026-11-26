package cities

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// stretch is a length of deck that is one level: where it starts and
// ends along the span, the height of its top above the water, and whether
// it is an approach, and if so how far down the approach it is, from
// nought at the span to one at the ground.
type stretch struct {
	from, to, height float64
	approach         bool
	down             float64
}

// deckLooks are the materials of a bridge.
type deckLooks struct{ steel, deck, street *thing.Material }

// looks draws the bridge's materials.
func (bridge *Bridge) looks() deckLooks {
	drawing, at := bridge.drawing, place{}
	pick := func(name string) option {
		return drawing.pick(name, drawing.options(name), at)
	}
	return deckLooks{steel: paint("steel", pick("steel-colour")),
		deck:   paint("deck", pick("deck-colour")),
		street: paint("street", pick("street-colour"))}
}

// level is the height of the deck over the water in the span between
// two supports, counted from the first shore: the deck's own height,
// stepped up and down at each station if the bridge steps.
func (bridge *Bridge) level(span int) float64 {
	drawing := bridge.drawing
	height := drawing.number("deck-height", place{})
	if bridge.Step && span%2 == 1 {
		height += drawing.number("deck-step-height", place{Bay: span})
	}
	return height
}

// sagged is a level lowered, for a deck that sags, by the dip of its
// span at a share of the way along it: nothing at either support, the
// deepest at the middle.
func (bridge *Bridge) sagged(span int, height, share float64) float64 {
	if !bridge.Sag {
		return height
	}
	dip := math.Sin(math.Pi * share)
	return height - bridge.drawing.number("deck-sag-depth",
		place{Bay: span})*dip*dip
}

// stretches are the levels of the deck from one end to the other: the
// approach stepping up from the ground, the main deck in blocks, level or
// sagged or stepped, and the approach stepping down.
func (bridge *Bridge) stretches() []stretch {
	drawing := bridge.drawing
	block := drawing.fixed("block-length")
	supports := append([]float64{-bridge.Span / 2}, bridge.Stations...)
	supports = append(supports, bridge.Span/2)
	var made []stretch
	for span := range len(supports) - 1 {
		length := supports[span+1] - supports[span]
		blocks := int(math.Ceil(length / block))
		for index := range blocks {
			share := (float64(index) + 0.5) / float64(blocks)
			from := supports[span] + length*float64(index)/float64(blocks)
			to := supports[span] + length*float64(index+1)/float64(blocks)
			made = append(made, stretch{from: from, to: to,
				height: bridge.sagged(span, bridge.level(span), share)})
		}
	}
	return bridge.approached(made)
}

// approached is the stretches with an approach added at each end: blocks
// that step down to the ground, each lower than the last.
func (bridge *Bridge) approached(made []stretch) []stretch {
	drawing := bridge.drawing
	block := drawing.fixed("block-length")
	steps := int(math.Ceil(drawing.fixed("approach-share") * bridge.Span /
		block))
	first, last := made[0], made[len(made)-1]
	var before, after []stretch
	for step := range steps {
		down := float64(step+1) / float64(steps+1)
		before = append([]stretch{{from: first.from - float64(step+1)*block,
			to: first.from - float64(step)*block, approach: true, down: down,
			height: first.height * (1 - down)}}, before...)
		after = append(after, stretch{to: last.to + float64(step+1)*block,
			from: last.to + float64(step)*block, approach: true, down: down,
			height: last.height * (1 - down)})
	}
	return append(append(before, made...), after...)
}

// heightAt is the height of the deck at a place along the span, or the
// ground if the place is beyond the approaches.
func heightAt(stretches []stretch, along float64) float64 {
	for _, each := range stretches {
		if along >= each.from && along <= each.to {
			return each.height
		}
	}
	return 0
}

// Design is the bridge drawn at a level and a time: the deck, its lots,
// the pylons, the cables and the hangers. Along the span is x, the deck
// is as wide as the culture says along z, and up is y.
func (bridge *Bridge) Design(level Detail, seconds float64) (Design, error) {
	if bridge.Hub {
		return bridge.hubDesign(), bridge.drawing.trouble
	}
	looks := bridge.looks()
	stretches := bridge.stretches()
	parts := bridge.deck(stretches, looks)
	if !bridge.Bare {
		lots, err := bridge.lots(stretches, level)
		if err != nil {
			return Design{}, err
		}
		parts = append(parts, lots...)
	}
	towers, err := bridge.pylonParts(level, seconds)
	if err != nil {
		return Design{}, err
	}
	parts = append(parts, towers...)
	parts = append(parts, bridge.cables(stretches, looks)...)
	return Design{Name: "bridge", Elements: parts}, bridge.drawing.trouble
}

// deck is a slab and a strip of street for each stretch, the slab under
// the deck's level and the street just over it.
func (bridge *Bridge) deck(stretches []stretch, looks deckLooks) []placedElement {
	drawing := bridge.drawing
	width, thick := bridge.deckWidth(), drawing.fixed("deck-thickness")
	street := drawing.fixed("street-width")
	var parts []placedElement
	for _, each := range stretches {
		long := each.to - each.from
		parts = append(parts,
			put("deck", Box{long, thick, width}, looks.deck,
				maths.V(each.from, each.height-thick, -width/2)),
			put("street", Box{long, thick / 10, street}, looks.street,
				maths.V(each.from, each.height, -street/2)))
		if bridge.Second && !each.approach {
			parts = append(parts, put("second deck", Box{long, thick, width},
				looks.deck, maths.V(each.from, each.height+
					drawing.number("second-deck-gap", place{})-thick,
					-width/2)))
		}
	}
	return parts
}

// pylonParts are the towers of the pylons: a pair at each station, one
// each side of the deck, the same tower, standing on the water.
func (bridge *Bridge) pylonParts(level Detail, seconds float64) ([]placedElement,
	error) {
	var parts []placedElement
	for index, at := range bridge.Stations {
		tower := bridge.pylons[index]
		design, err := tower.Design(level, seconds)
		if err != nil {
			return nil, err
		}
		for _, side := range []float64{-1, 1} {
			place := maths.NewSubspace(maths.V(at, 0,
				side*bridge.pylonOffset(tower)), maths.Identity, 0)
			parts = append(parts, design.at(place).Elements...)
		}
	}
	return parts, nil
}

// pylonOffset is how far from the deck's middle a pylon's middle stands:
// half the deck, the culture's gap, and the pylon's own radius.
func (bridge *Bridge) pylonOffset(tower *Tower) float64 {
	drawing := bridge.drawing
	return bridge.deckWidth()/2 + drawing.fixed("pylon-gap") +
		tower.Shafts[0].radius()
}

// deckWidth is how wide the deck is: the culture's, wide enough for a
// street and a block of buildings each side, or for a bare bridge the
// culture's width of a road.
func (bridge *Bridge) deckWidth() float64 {
	if bridge.Bare {
		return bridge.drawing.fixed("bare-deck-width")
	}
	return bridge.drawing.fixed("deck-width")
}

// chord is a bar between two points, a box of the thickness given laid
// along the line, centred on it.
func chord(name string, from, to maths.Vec, thick float64,
	look *thing.Material) placedElement {
	along := to.Sub(from)
	axis := along.Unit()
	up := maths.YAxis
	if math.Abs(axis.Dot(up)) > 1-1e-6 {
		up = maths.XAxis
	}
	side := axis.Cross(up).Unit()
	lift := side.Cross(axis)
	origin := from.Sub(lift.Scale(thick / 2)).Sub(side.Scale(thick / 2))
	return placedElement{Element: solid{Name: name, Material: look,
		Form: Box{along.Len(), thick, thick}},
		Place: maths.NewSubspace(origin, maths.Orientation{X: axis, Y: lift,
			Z: side}, 0)}
}

// cables are the main cables, each of its strands a chain of links hung
// in a catenary between the shore and the pylon tops and on to the other
// shore, with the hangers that drop from the middle strand to the deck.
func (bridge *Bridge) cables(stretches []stretch, looks deckLooks) []placedElement {
	drawing := bridge.drawing
	var parts []placedElement
	for _, side := range []float64{-1, 1} {
		z := side * bridge.pylonOffset(bridge.pylons[0])
		supports := bridge.supports(stretches, z)
		for strand := range bridge.Strands {
			shift := (float64(strand) - float64(bridge.Strands-1)/2) *
				drawing.fixed("strand-gap")
			parts = append(parts, bridge.strand(supports, shift, looks)...)
		}
		parts = append(parts, bridge.hangers(stretches, supports, side,
			looks)...)
	}
	return parts
}

// supports are the points a cable hangs from: the deck's level at one
// shore, the top of each pylon, and the deck's level at the other.
func (bridge *Bridge) supports(stretches []stretch, z float64) []maths.Vec {
	points := []maths.Vec{maths.V(-bridge.Span/2, heightAt(stretches,
		-bridge.Span/2), z)}
	for index, at := range bridge.Stations {
		points = append(points, maths.V(at, bridge.pylons[index].height(0), z))
	}
	return append(points, maths.V(bridge.Span/2, heightAt(stretches,
		bridge.Span/2), z))
}

// cableAt is the height of a cable between two supports at a distance
// along the way, its straight line less the catenary's drop.
func cableAt(from, to maths.Vec, sag, along float64) maths.Vec {
	run := math.Hypot(to.X-from.X, to.Z-from.Z)
	share := along / run
	line := from.Add(to.Sub(from).Scale(share))
	return maths.V(line.X, line.Y-catenaryDrop(run, sag, along), line.Z)
}

// strand is one strand of a cable: a chain of links through the supports,
// each stretch hung with its own sag, shifted along z by the strand's.
func (bridge *Bridge) strand(supports []maths.Vec, shift float64,
	looks deckLooks) []placedElement {
	drawing := bridge.drawing
	link := drawing.fixed("cable-segment")
	thick := drawing.fixed("cable-thickness")
	var parts []placedElement
	for index := 0; index+1 < len(supports); index++ {
		from, to := supports[index], supports[index+1]
		sag := drawing.number("cable-sag-share", place{Bay: index}) *
			math.Hypot(to.X-from.X, to.Z-from.Z)
		links := max(1, int(math.Ceil(math.Hypot(to.X-from.X,
			to.Z-from.Z)/link)))
		run := math.Hypot(to.X-from.X, to.Z-from.Z)
		for step := range links {
			start := cableAt(from, to, sag, run*float64(step)/float64(links))
			end := cableAt(from, to, sag, run*float64(step+1)/float64(links))
			start.Z += shift
			end.Z += shift
			parts = append(parts, chord("cable", start, end, thick,
				looks.steel))
		}
	}
	return parts
}

// hangers are the bars that drop from a cable to the edge of the deck
// beneath it, at the culture's spacing, along the stretch between shores.
func (bridge *Bridge) hangers(stretches []stretch, supports []maths.Vec,
	side float64, looks deckLooks) []placedElement {
	drawing := bridge.drawing
	spacing := drawing.fixed("hanger-spacing")
	thick := drawing.fixed("hanger-thickness")
	edge := side * bridge.deckWidth() / 2
	var parts []placedElement
	for index := 0; index+1 < len(supports); index++ {
		from, to := supports[index], supports[index+1]
		run := math.Hypot(to.X-from.X, to.Z-from.Z)
		sag := drawing.number("cable-sag-share", place{Bay: index}) * run
		for along := spacing; along < run-spacing/2; along += spacing {
			top := cableAt(from, to, sag, along)
			foot := maths.V(top.X, heightAt(stretches, top.X), edge)
			parts = append(parts, chord("hanger", top, foot, thick,
				looks.steel))
		}
	}
	return parts
}

// check is a refusal of a bridge that cannot be drawn: here, one whose
// pylons are not as tall as its deck.
func (bridge *Bridge) check() error {
	deck := bridge.drawing.number("deck-height", place{})
	for index, tower := range bridge.pylons {
		if tower.height(0) < deck {
			return fmt.Errorf("cities: pylon %d is %v m, under the deck at "+
				"%v m", index, tower.height(0), deck)
		}
	}
	return nil
}
