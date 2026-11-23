package cities

import (
	"fmt"
	"math"
	"slices"
)

// towerCulture is the data every tower is drawn from. Its scopes mean
// something else than a street's: a row is the whole tower, a bay is one
// shaft of it, a storey is one segment of a shaft, and an opening is one
// window or panel.
const towerCulture = "tower"

// Irregularity is how far a tower may stray from a plain block, from
// nought to one, or the word that it is drawn from the skyline's own
// distribution, in which most towers are low and a few are high.
type Irregularity struct {
	// Value is the irregularity, from nought to one, if not Drawn.
	Value float64
	// Drawn says the tower draws its irregularity from its seed, from the
	// distribution the tower's data gives, so a skyline of towers is
	// mostly ordinary with the odd landmark.
	Drawn bool
}

// Pins are features a set fixes for one tower, over what the draw would
// have given. A zero field is not pinned. A pin is an override, not a
// weighting: a tower pinned to the plan y has the plan y whatever its
// irregularity.
type Pins struct {
	// Plan is the plan of every shaft.
	Plan Plan
	// Shafts is how many shafts the tower has.
	Shafts int
	// Taper, Twist, Setbacks, Spire and LitCrown, when given, say that
	// every segment tapers, every segment turns, every segment steps
	// back, every shaft has a spire, and every crown is lit; or that
	// none is.
	Taper, Twist, Setbacks, Spire, LitCrown *bool
	// Stack and Nest, when given, say that every shaft is a stack of
	// flared modules, each wider at the top than at the bottom, or that
	// none is; and that every shaft has a slim core carried on above its
	// wrapped body, or that none has.
	Stack, Nest *bool
	// Span is none, skybridge or deck: what joins the shafts.
	Span SpanKind
	// Room, if over nought, is the width in metres of the block the
	// tower stands in. Shafts are drawn narrower, or fewer, so that the
	// tower, its gaps and its base fit in it.
	Room float64
	// Skin is what the faces are made of, one of SkinNames, and Style
	// how a granite skin is windowed, punched or ribbon, in every
	// segment; Colour is the name of the colour option the skin takes
	// from the data, for the skins that have several. Use is office or
	// home for every segment. Each is empty if not pinned.
	Skin  Skin
	Style string
	// Colour names the stone, the tint or the pastel the skin wears.
	Colour string
	// Use is what every segment is used for, office or home.
	Use Use
	// Cyber and Planted, when given, say that the cyberpunk layer is on
	// every face, or that terraces and balconies hold trees on every
	// face, or that neither is.
	Cyber, Planted *bool
	// Spiral, when given, says that the tower is blocks of a few storeys,
	// each turned on the one below, so that its floors rotate as it rises.
	Spiral *bool
	// Storeys, if over nought, is how many storeys each shaft has, and
	// Segments how many segments a shaft is cut into.
	Storeys, Segments int
	// Width, if over nought, is the width in metres of the plan at its
	// foot, in place of the draw.
	Width float64
}

// SpanKind is what joins the shafts of a tower.
type SpanKind string

// The things that may join shafts.
const (
	// SpanNone joins nothing.
	SpanNone SpanKind = "none"
	// SpanSkybridge is a bridge between neighbouring shafts at one
	// storey.
	SpanSkybridge SpanKind = "skybridge"
	// SpanDeck is a deck laid across the tops of all the shafts.
	SpanDeck SpanKind = "deck"
)

// segment is a stretch of a shaft with one outline at its bottom and
// another at its top, built as a prism of the two, so a taper or a
// setback is only a top smaller than the bottom.
type segment struct {
	Bottom, Top footprintShape
	// Storeys is how many floors the segment has.
	Storeys int
	// Turn is how many degrees about the vertical axis the segment's
	// plan is turned from that of the segment below.
	Turn float64
	// Use is what the segment is for, which lights its windows by night,
	// and Style how a granite segment is windowed.
	Use   Use
	Style string
}

// shaft is one stack of segments on a base, standing on its own centre
// and ending at its own height.
type shaft struct {
	Centre   outlineCorner
	Segments []segment
	// Storeys is all the shaft's floors.
	Storeys int
	// Crown, Penthouse, Deck, Spire and LitCrown say what is on top.
	Crown, Penthouse, Deck, Spire, LitCrown bool
	// Stacked says the shaft is a stack of flared modules, and Nested
	// that it carries a slim core on over its body.
	Stacked, Nested bool
}

// Tower is a tall building drawn from a seed. It is one or more shafts
// on a base; each shaft is a stack of segments, each a prism of one
// outline to another; the shafts may be joined; each shaft has a top.
// Windows, a core, and mechanical floors are not drawn but derived from
// the stack: a window module across each face and one a storey up, a core
// sized from the storeys, a band of louvres every so many storeys.
//
// Every feature is drawn from the seed by NewTower, and every draw is
// weighted by the irregularity. At nought a tower is a plain block: a
// rectangle in plan, one segment, mechanical bands, a crown and beacons.
//
// As the irregularity rises each feature becomes likelier and they combine:
// plans round, polygonal, star, Y and sail; tapers; setbacks, which on
// a Y step one wing at a time and so spiral; twisting segments; several
// shafts, with a skybridge or a deck across their tops; an observation
// deck, a penthouse, a spire; a crown that lights in a travelling band.
//
// At one, several of these come at once. The landmarks of the world are
// not in the data: they are the range one generator reaches, and a set
// pins a few features to ask for one.
type Tower struct {
	Irregularity float64
	Plan         Plan
	Shafts       []shaft
	Span         SpanKind
	// SpanStorey is the storey of the skybridges, counted from the
	// shaft's own first floor.
	SpanStorey int
	// Skin is what the faces are made of, Cyber says the cyberpunk layer
	// is over it, and Planted that terraces and balconies hold trees.
	Skin           Skin
	Cyber, Planted bool
	seed           Seed
	drawing        *drawer
	colour, style  string
	// grown is the trees each segment held when it was last drawn, by
	// the band of it they stand in, the whole segment as noBand.
	grown []map[int][]Tree
	// finest is the level the design was made for, the finest any part is
	// drawn at; a part drawn coarser keeps its lights.
	finest Detail
	width  float64
	// scale is how much narrower than drawn the plans are made to fit
	// the room, and room the width of the block, or nought.
	scale, room float64
	// foot is the height of a storey at the foot of a single shaft left
	// open for shops to fill, which the shaft stands on: nought for none.
	foot float64
}

// gate is whether a feature that comes with a share, at an irregularity
// of one, comes at this irregularity: the share, taken down by the
// irregularity raised to the culture's gate power.
func (drawing *drawer) gate(name string, at place, irregularity float64,
	pinned *bool) bool {
	if pinned != nil {
		return *pinned
	}
	each, found := drawing.found(name, drawShare)
	power := drawing.fixed("gate-power")
	reachable := each.Share * math.Pow(irregularity, power)
	return found && drawing.share(each, each.Scope, "", at) < reachable
}

// reach is a count drawn from the least of its range up to as far toward
// the most as the irregularity takes it: at nought always the least.
func (drawing *drawer) reach(name string, at place,
	irregularity float64) int {
	each, found := drawing.found(name, drawRange)
	if !found {
		return 0
	}
	most := each.Least + (each.Most-each.Least)*irregularity
	wide := most - each.Least + 1
	return int(each.Least + math.Min(math.Floor(
		drawing.share(each, each.Scope, "", at)*wide), wide-1))
}

// resolved is the irregularity as a number from nought to one.
func (drawing *drawer) resolved(given Irregularity) (float64, error) {
	if !given.Drawn {
		if !(given.Value >= 0 && given.Value <= 1) {
			return 0, fmt.Errorf("cities: irregularity is from nought to "+
				"one, not %v", given.Value)
		}
		return given.Value, nil
	}
	each, found := drawing.found("skyline-draw", drawRange)
	share := drawing.share(each, each.Scope, "", place{})
	value := each.Least + (each.Most-each.Least)*
		math.Pow(share, drawing.fixed("skyline-power"))
	return value, boolError(found, "skyline-draw")
}

// boolError is nothing if an element was found, else why not.
func boolError(found bool, name string) error {
	if found {
		return nil
	}
	return fmt.Errorf("cities: the tower data has no %s", name)
}

// NewTower draws a tower from a seed at an irregularity, with some
// features pinned by the set. It refuses an irregularity outside nought
// to one, a plan or span that is not one there is, and a count of shafts
// that is not from one to the most the data allows.
func NewTower(seed Seed, irregularity Irregularity,
	pins Pins) (*Tower, error) {
	drawing, err := newDrawer(towerCulture, seed)
	if err != nil {
		return nil, err
	}
	reach, err := drawing.resolved(irregularity)
	if err != nil {
		return nil, err
	}
	tower := &Tower{Irregularity: reach, seed: seed, drawing: drawing,
		scale: 1, room: pins.Room}
	if err := tower.draw(pins); err != nil {
		return nil, err
	}
	return tower, drawing.trouble
}

// draw is every feature of the tower that is not on a face, drawn.
func (tower *Tower) draw(pins Pins) error {
	drawing, reach := tower.drawing, tower.Irregularity
	tower.Plan = tower.planOf(pins)
	if pins.Plan != "" && !slices.Contains(PlanNames(), pins.Plan) {
		return fmt.Errorf("cities: no plan %q, only %v", pins.Plan,
			PlanNames())
	}
	if err := tower.dressed(pins); err != nil {
		return err
	}
	count := drawing.reach("shafts", place{}, reach)
	if pins.Shafts != 0 {
		count = pins.Shafts
	}
	if count >= 1 {
		count = tower.fitted(count)
	}
	if count < 1 || count > int(drawing.fixed("most-shafts")) {
		return fmt.Errorf("cities: a tower has from one to %v shafts, not "+
			"%d", drawing.fixed("most-shafts"), count)
	}
	tower.Span = tower.spanOf(pins, count)
	if err := tower.stack(pins, count); err != nil {
		return err
	}
	tower.place()
	return tower.check()
}

// fitted is how many shafts fit in the room the tower was given, no more
// than it was asked for, and sets the scale the plans are drawn at so that
// the shafts, the gaps between them and the base round them fit. A tower
// with no room has as many as it wants, at the size it was drawn.
func (tower *Tower) fitted(count int) int {
	if !(tower.room > 0) {
		return count
	}
	drawing := tower.drawing
	margin, gap := drawing.fixed("podium-margin"), drawing.fixed("fit-gap")
	across := 2 * tower.baseShape(tower.Plan).radius()
	for ; count > 1; count-- {
		scale := (tower.room - 2*margin - float64(count-1)*gap) /
			(float64(count) * across)
		if scale*across >= drawing.fixed("least-room-width") {
			tower.scale = math.Min(1, scale)
			return count
		}
	}
	tower.scale = math.Min(1, tower.room/across)
	return 1
}

// planOf is the tower's plan: pinned, or drawn from the irregular plans
// if the gate lets the tower stray, else a rectangle.
func (tower *Tower) planOf(pins Pins) Plan {
	drawing := tower.drawing
	if pins.Plan != "" {
		return pins.Plan
	}
	if !drawing.gate("plan-irregular", place{}, tower.Irregularity, nil) {
		return planRectangle
	}
	return Plan(drawing.pick("plan", drawing.options("plan"), place{}).Name)
}

// spanOf is what joins the shafts, which is nothing for a single shaft.
func (tower *Tower) spanOf(pins Pins, shafts int) SpanKind {
	drawing := tower.drawing
	if shafts < 2 {
		return SpanNone
	}
	if pins.Span != "" {
		return pins.Span
	}
	if !drawing.gate("span", place{}, tower.Irregularity, nil) {
		return SpanNone
	}
	kinds := drawing.options("span-kind")
	return SpanKind(drawing.pick("span-kind", kinds, place{}).Name)
}

// stack draws each shaft's segments. Shafts end at their own heights,
// except under a deck, which needs them level.
func (tower *Tower) stack(pins Pins, count int) error {
	drawing, reach := tower.drawing, tower.Irregularity
	total := drawing.count("base-storeys", place{}) +
		drawing.reach("extra-storeys", place{}, reach)
	if pins.Storeys > 0 {
		total = pins.Storeys
	}
	for index := range count {
		at := place{Bay: index}
		storeys := total
		if index > 0 && tower.Span != SpanDeck {
			storeys = int(float64(total) * drawing.number("shaft-height-share",
				at))
		}
		shaft, err := tower.shaft(at, pins, storeys)
		if err != nil {
			return err
		}
		tower.Shafts = append(tower.Shafts, shaft)
	}
	tower.SpanStorey = int(drawing.number("span-height", place{}) *
		float64(tower.lowest()))
	return drawing.trouble
}

// lowest is how many storeys the lowest shaft has.
func (tower *Tower) lowest() int {
	least := math.MaxInt
	for _, shaft := range tower.Shafts {
		least = min(least, shaft.Storeys)
	}
	return least
}
