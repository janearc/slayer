package cities

import (
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
)

// The measures of a hub bridge, in metres, for its first look. They move
// to the bridge culture's data before the hub bridge lands.
const (
	// hubWide is the middle tower along the span and across it, and
	// hubTall its height.
	hubWide = 250.0
	hubTall = 750.0
	// hubVoid is the height of the void under the tower's body that the
	// bridge passes through, and hubLeg the side of each of the four legs
	// at its corners that carry the body over it.
	hubVoid = 220.0
	hubLeg  = 50.0
	// hubRoad is the width of each deck's roadway, both ways of traffic.
	hubRoad = 60.0
	// hubDeck is the top of the lower deck over the water, hubDecks the
	// upper deck's top over the lower's, and hubThick a deck's slab.
	hubDeck  = 62.0
	hubDecks = 16.0
	hubThick = 3.0
	// hubBody is the width of an arm's building, which hangs hubBodyDeep
	// under the lower deck.
	hubBody     = 100.0
	hubBodyDeep = 36.0
	// hubStorey is the height of one storey of an arm's building, and
	// hubBeside how many storeys stand beside each deck's roadway, on the
	// strips of the arm wider than the road: three beside the lower deck
	// fill the headroom under the upper one.
	hubStorey = 3.5
	hubBeside = 3
	// hubSill and hubGlass are the foot of a storey's ribbon of windows
	// over its floor and the ribbon's height; hubBay is the width of one
	// window along it, hubLitShare the share of windows lit, hubHeat their
	// light's colour temperature and hubGlow their glow against the
	// culture's window glow, as an office's is.
	hubSill     = 0.9
	hubGlass    = 2.0
	hubBay      = 6.0
	hubLitShare = 0.35
	hubHeat     = 3500.0
	hubGlow     = 0.3
	// hubProud is the thickness of a pane, a lit wall and a ceiling light.
	hubProud = 0.05
	// hubRoom is how deep a room runs in from the glass to the core, and
	// hubSlab the thickness of a floor; hubSpandrel is the thickness of the
	// wall under and over the glass.
	hubRoom     = 8.0
	hubSlab     = 0.3
	hubSpandrel = 0.3
	// hubGlazingTone is the clear glass's tint and hubWallTone the rooms'
	// walls'; a lit room's wall glows hubWash times a window's light and
	// its ceiling light, hubPanel square, hubCeiling times, as a tower's
	// office's do.
	hubGlazingTone = 0.9
	hubWallTone    = 0.7
	hubWash        = 1.2
	hubCeiling     = 40.0
	hubPanel       = 1.2
	// hubDeskShare is the share of bays with a desk, hubDesk, hubDeskHigh
	// and hubDeskDeep its size, set hubDeskBack in from the glass.
	hubDeskShare = 0.5
	hubDesk      = 1.6
	hubDeskHigh  = 0.75
	hubDeskDeep  = 0.8
	hubDeskBack  = 1.5
	// hubColumns is how far apart the columns stand that carry the upper
	// deck over the lower along an arm, and hubColumn how thick each is.
	hubColumns = 25.0
	hubColumn  = 1.5
	// hubStays is how many stays fan from each face of the tower in each
	// plane, from hubStayLow up to hubStayHigh on the tower, and
	// hubStayThick how thick each is.
	hubStays     = 8
	hubStayLow   = 420.0
	hubStayHigh  = 700.0
	hubStayThick = 1.2
)

// hubDesign is the bridge as a tower in the middle with an arm out of each
// face. The two decks run unbroken from one shore to the other and through
// the tower's void, open on all four sides, under its body on four legs.
//
// Each arm is a building hung under and beside the decks, running on
// through the void to meet the other in the middle, with columns carrying
// the upper deck over the lower, and stays fanned from high on the tower's
// face to the upper deck in the two planes of the arm's sides. Along the
// span is x, across it z, and up y.
func (bridge *Bridge) hubDesign() Design {
	looks := bridge.looks()
	half := bridge.Span / 2
	arm := half - hubWide/2
	parts := hubTower(looks)
	for _, top := range []float64{hubDeck, hubDeck + hubDecks} {
		parts = append(parts, put("deck", Box{bridge.Span, hubThick, hubRoad},
			looks.street, maths.V(-half, top-hubThick, -hubRoad/2)))
	}
	for _, side := range []float64{-1, 1} {
		parts = append(parts, hubArm(side, arm, looks)...)
		parts = append(parts, bridge.hubStoreys(side, arm, looks)...)
	}
	return Design{Name: "bridge", Elements: parts}
}

// hubBlock is one storey of an arm's building across part of its width:
// its floor, and its two long faces, at low and high across the arm. A
// block whose storey is the top of its stack has a roof.
type hubBlock struct {
	floor, low, high float64
	roof             bool
}

// hubBlocks are an arm's storeys: across the whole arm under the lower
// deck, and on the strips either side of each deck's roadway beside it.
func hubBlocks() []hubBlock {
	half, road := hubBody/2, hubRoad/2
	under := hubDeck - hubThick
	var blocks []hubBlock
	for floor := under - hubBodyDeep; floor+hubStorey <= under+1e-9; floor += hubStorey {
		blocks = append(blocks, hubBlock{floor: floor, low: -half, high: half,
			roof: floor+2*hubStorey > under+1e-9})
	}
	for _, top := range []float64{hubDeck, hubDeck + hubDecks} {
		for storey := range hubBeside {
			floor := top + float64(storey)*hubStorey
			for _, strip := range [][2]float64{{-half, -road}, {road, half}} {
				blocks = append(blocks, hubBlock{floor: floor, low: strip[0],
					high: strip[1], roof: storey == hubBeside-1})
			}
		}
	}
	return blocks
}

// hubStoreys are one arm's storeys as rooms. Each block has its floor, a
// core down its middle, and along each long face a room hubRoom deep
// behind clear glass, between a spandrel under the glass and one over it.
// A share of each face's bays are lit: the core's wall behind the glass
// glows warm and a light glows on the ceiling, as a tower's office does;
// some bays have a desk. The faces are the arm's two long sides and,
// beside a deck, the faces onto its roadway.
func (bridge *Bridge) hubStoreys(side, arm float64,
	looks deckLooks) []placedElement {
	glow := swatches.Light(swatch.Planckian(hubHeat)).Scale(
		bridge.drawing.fixed("window-glow") * hubGlow)
	glazing := &thing.Material{Name: "cities clear glazing of a hub bridge",
		Colour: maths.RGB(hubGlazingTone, hubGlazingTone, hubGlazingTone),
		Gloss:  1, Through: 1}
	wash := &thing.Material{Name: "cities lit room wall of a hub bridge",
		Colour: maths.RGB(hubWallTone, hubWallTone, hubWallTone),
		Glow:   glow.Scale(hubWash)}
	panel := &thing.Material{Name: "cities ceiling light of a hub bridge",
		Colour: maths.RGB(hubWallTone, hubWallTone, hubWallTone),
		Glow:   glow.Scale(hubCeiling)}
	long := arm + hubWide/2
	start := min(0, side*long)
	bays := int(long / hubBay)
	var parts []placedElement
	for index, block := range hubBlocks() {
		width := block.high - block.low
		room := min(hubRoom, (width-1)/2)
		parts = append(parts,
			put("arm floor", Box{long, hubSlab, width}, looks.deck,
				maths.V(start, block.floor, block.low)),
			put("arm storey", Box{long, hubStorey - hubSlab, width - 2*room},
				looks.deck, maths.V(start, block.floor+hubSlab,
					block.low+room)))
		if block.roof {
			parts = append(parts, put("arm roof", Box{long, hubSlab, width},
				looks.deck, maths.V(start, block.floor+hubStorey-hubSlab,
					block.low)))
		}
		for face, out := range []float64{-1, 1} {
			// in is the place across the arm of something set back from
			// this face by a distance, as thick as given
			in := func(back, thick float64) float64 {
				if out < 0 {
					return block.low + back
				}
				return block.high - back - thick
			}
			over := hubStorey - hubSill - hubGlass
			parts = append(parts,
				put("spandrel", Box{long, hubSill - hubSlab, hubSpandrel},
					looks.deck, maths.V(start, block.floor+hubSlab,
						in(0, hubSpandrel))),
				put("spandrel", Box{long, over, hubSpandrel}, looks.deck,
					maths.V(start, block.floor+hubSill+hubGlass,
						in(0, hubSpandrel))),
				put("window", Box{long, hubGlass, hubProud}, glazing,
					maths.V(start, block.floor+hubSill, in(hubSpandrel/2,
						hubProud))))
			for bay := range bays {
				x := start + float64(bay)*hubBay
				if hubSpot(uint64(bridge.seed), side, index, face,
					bay) < hubLitShare {
					parts = append(parts,
						put("lit window", Box{hubBay * 0.95, hubStorey -
							hubSlab - 0.1, hubProud}, wash, maths.V(x,
							block.floor+hubSlab, in(room-hubProud,
								hubProud))),
						put("ceiling light", Box{hubPanel, hubProud,
							hubPanel}, panel, maths.V(x+hubBay/2-hubPanel/2,
							block.floor+hubStorey-hubSlab-hubProud,
							in(room/2-hubPanel/2, hubPanel))))
				}
				if hubSpot(uint64(bridge.seed), side, index, face+2,
					bay) < hubDeskShare {
					parts = append(parts, put("desk", Box{hubDesk,
						hubDeskHigh, hubDeskDeep}, looks.steel, maths.V(
						x+hubBay/2-hubDesk/2, block.floor+hubSlab,
						in(hubDeskBack, hubDeskDeep))))
				}
			}
		}
	}
	return parts
}

// hubSpot is a draw from nought to one for one window of a hub bridge,
// the same for the same seed and window every time.
func hubSpot(seed uint64, side float64, storey, face, bay int) float64 {
	mixed := seed ^ uint64(storey)<<40 ^ uint64(face)<<32 ^ uint64(bay)
	if side > 0 {
		mixed ^= 1 << 63
	}
	mixed += 0x9e3779b97f4a7c15
	mixed = (mixed ^ mixed>>30) * 0xbf58476d1ce4e5b9
	mixed = (mixed ^ mixed>>27) * 0x94d049bb133111eb
	mixed ^= mixed >> 31
	return float64(mixed>>11) / float64(uint64(1)<<53)
}

// hubTower is the middle tower as a gate: four legs at its corners from
// the water up to hubVoid, and its body over them to its full height, so
// that the bridge, every storey of it, passes through a void open on all
// four sides, with the sky beyond it whichever way it is seen.
func hubTower(looks deckLooks) []placedElement {
	half := hubWide / 2
	parts := []placedElement{put("tower", Box{hubWide, hubTall - hubVoid,
		hubWide}, looks.deck, maths.V(-half, hubVoid, -half))}
	for _, x := range []float64{-half, half - hubLeg} {
		for _, z := range []float64{-half, half - hubLeg} {
			parts = append(parts, put("tower leg", Box{hubLeg, hubVoid,
				hubLeg}, looks.deck, maths.V(x, 0, z)))
		}
	}
	return parts
}

// hubArm is one arm, on the side of the tower a sign gives, as long as
// given from the tower's face: its building under the decks, the columns
// between the decks, and the stays.
func hubArm(side, arm float64, looks deckLooks) []placedElement {
	face := side * hubWide / 2
	// the columns run on through the tower's void to its middle, as the
	// building does, where the other arm's meet them
	long := arm + hubWide/2
	var parts []placedElement
	edge := hubRoad/2 + hubColumn
	between := hubDecks - hubThick
	for along := hubColumns; along < long; along += hubColumns {
		x := side*along - hubColumn/2
		for _, z := range []float64{-edge, edge - hubColumn} {
			parts = append(parts, put("column", Box{hubColumn, between,
				hubColumn}, looks.steel, maths.V(x, hubDeck, z)))
		}
	}
	for stay := range hubStays {
		share := float64(stay) / float64(hubStays-1)
		height := hubStayLow + (hubStayHigh-hubStayLow)*share
		reach := arm * float64(stay+1) / float64(hubStays)
		for _, z := range []float64{-hubBody / 2, hubBody / 2} {
			parts = append(parts, chord("stay", maths.V(face, height, z),
				maths.V(face+side*reach, hubDeck+hubDecks, z), hubStayThick,
				looks.steel))
		}
	}
	return parts
}
