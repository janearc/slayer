package cities

import (
	"fmt"
	"strings"

	"github.com/janearc/slayer/maths"
)

// terracePrefix begins the name of a lot that is a terrace: the rest of
// the name is its culture.
const terracePrefix = "terrace-"

// lotKinds are the lots that are not terraces.
const (
	lotCondominium = "condominium"
	lotTower       = "tower"
	lotBridge      = "bridge"
)

// lotKind is what stands on the lot at an index of a stretch: on the
// deck, anything the culture draws, though a bridge only if the depth
// allows one more; on an approach, taller buildings near the span and
// lower ones near the ground.
func (bridge *Bridge) lotKind(each stretch, index int) string {
	drawing, at := bridge.drawing, place{Opening: index}
	name := "lot-kind"
	var skip []string
	if each.approach {
		name = "approach-far"
		if each.down < 0.5 {
			name = "approach-near"
		}
	} else if bridge.Depth < 1 {
		skip = []string{lotBridge}
	}
	return drawing.pick(name, drawing.options(name), at, skip...).Name
}

// lot is the design of a lot: a terrace of the culture the kind names, a
// condominium, a tower, or a bridge of its own, in its own space. A bridge
// that is a lot has one less depth than the bridge it is on.
func (bridge *Bridge) lot(kind string, index int, level Detail) (Design,
	error) {
	drawing, at := bridge.drawing, place{Opening: index}
	seed := Seed(uint64(bridge.seed)<<lotSeedBits | uint64(index))
	switch {
	case strings.HasPrefix(kind, terracePrefix):
		return terraceAtDetail(strings.TrimPrefix(kind, terracePrefix), seed, nil,
			level)
	case kind == lotCondominium:
		return CondominiumAt(seed, nil, level)
	case kind == lotTower:
		tower, err := NewTower(seed, Irregularity{
			Value: drawing.number("lot-tower-irregularity", at)},
			Pins{Shafts: 1, Span: SpanNone})
		if err != nil {
			return Design{}, err
		}
		return tower.Design(level, 0)
	case kind == lotBridge:
		nested, err := NewBridge(seed, Irregularity{Value: bridge.Irregularity *
			drawing.fixed("lot-bridge-reach")},
			drawing.number("lot-bridge-span", at), bridge.Depth-1)
		if err != nil {
			return Design{}, err
		}
		return nested.Design(level, 0)
	}
	if drawing.trouble != nil {
		return Design{}, drawing.trouble
	}
	return Design{}, fmt.Errorf("cities: no lot of kind %q", kind)
}

// lots are the lots along both sides of the deck, packed end to end in
// each stretch, a lot that does not fit left out unless it is the first of
// its stretch. If the bridge has a second deck its lots stand on that too,
// at the share of them the culture says.
func (bridge *Bridge) lots(stretches []stretch, level Detail) ([]placedElement,
	error) {
	var parts []placedElement
	index := 0
	for _, side := range []float64{-1, 1} {
		for _, each := range stretches {
			made, err := bridge.fill(each, side, 0, &index, level)
			if err != nil {
				return nil, err
			}
			parts = append(parts, made...)
			if bridge.Second && !each.approach {
				made, err := bridge.fill(each, side,
					bridge.drawing.number("second-deck-gap", place{}), &index,
					level)
				if err != nil {
					return nil, err
				}
				parts = append(parts, made...)
			}
		}
	}
	return parts, nil
}

// fill packs a stretch with lots along one side of the deck, standing on
// the deck's level raised by lift.
func (bridge *Bridge) fill(each stretch, side, lift float64, index *int,
	level Detail) ([]placedElement, error) {
	drawing := bridge.drawing
	cursor := each.from + drawing.fixed("deck-end-margin")
	var parts []placedElement
	for first := true; cursor < each.to; first = false {
		at := place{Opening: *index}
		*index++
		if lift > 0 && !drawing.chance("second-deck-share", at) {
			cursor += drawing.fixed("block-length") / 4
			continue
		}
		design, err := bridge.lot(bridge.lotKind(each, *index), *index, level)
		if err != nil {
			return nil, err
		}
		low, high, err := design.extent(hullLevel)
		if err != nil {
			return nil, err
		}
		length := high.X - low.X
		if !first && cursor+length > each.to {
			break
		}
		parts = append(parts, bridge.standLot(design, low, high, side,
			cursor, each.height+lift).Elements...)
		cursor += length + drawing.fixed("lot-gap")
	}
	return parts, nil
}

// hullLevel is the detail a lot is measured at, the cheapest.
const hullLevel Detail = "hull"

// standLot is a lot stood on a side of the deck: its nearest edge on the
// street's walk, its start at the cursor, on the level given. A lot on the
// far side is turned half a turn so that its front faces the street.
func (bridge *Bridge) standLot(design Design, low, high maths.Vec, side,
	cursor, height float64) Design {
	drawing := bridge.drawing
	near := drawing.fixed("street-width")/2 + drawing.fixed("sidewalk")
	if side < 0 {
		at := maths.V(cursor-low.X, height, -near-high.Z)
		return design.at(maths.NewSubspace(at, maths.Identity, 0))
	}
	at := maths.V(cursor+high.X, height, near+high.Z)
	turn := maths.Rotation(maths.YAxis, halfTurn*maths.Degree)
	return design.at(maths.NewSubspace(at, turn, 0))
}
