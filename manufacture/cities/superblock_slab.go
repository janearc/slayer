package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The planned block drawn: the podium with its shops, parking and ramp,
// the court's paving, the slabs with their households and decks, and the
// cores. A slab's faces are drawn by the condominium's culture, so that
// its households are drawn as a condominium's are.

// faceKey is the bay a face of a slab is drawn at by the households'
// drawer, so that the two faces of one slab draw different households.
func faceKey(index, side int) int {
	return index*sidesAround + side
}

// podium is the podium drawn: its body, a face of shops under parking on
// each side, the court's paving on its top, and the ramp up its back.
func (growing *accretion) podium() {
	drawing := growing.drawing
	out := growing.out
	concrete := paint("podium", drawing.pick("podium-colour",
		drawing.options("podium-colour"), place{}))
	recess := drawing.fixed("recess")
	body := out.Podium.inset(recess)
	growing.parts = append(growing.parts, put("podium", Box{body.width(),
		out.Deck, body.depth()}, concrete, maths.V(body.X0, 0, body.Z0)))
	paving := paint("paving", drawing.pick("court-paving",
		drawing.options("court-paving"), place{}))
	growing.parts = append(growing.parts, put("court", Box{out.Court.width(),
		drawing.fixed("paving-thickness"), out.Court.depth()}, paving,
		maths.V(out.Court.X0, out.Deck, out.Court.Z0)))
	growing.ramp(concrete)
	for side := range sidesAround {
		growing.shops(side, concrete)
	}
}

// shops is one side of the podium: the shops' windows under the parking's
// openings, and over each shop a canopy and a fascia sign that glows, if
// the pavement is clear there.
func (growing *accretion) shops(side int, concrete *thing.Material) {
	drawing := growing.drawing
	frame, length := wallFrame(growing.out.Podium, side, 0)
	count := int(length / drawing.fixed("shop-pitch"))
	windows := growing.shopRow(side, count, length)
	front := facade{Width: length, Height: growing.out.Deck,
		Recess:           drawing.fixed("recess"),
		FrameWidth:       drawing.fixed("frame-width"),
		GlassThickness:   drawing.fixed("glass-thickness"),
		ShutterThickness: drawing.fixed("shutter-thickness"),
		Wall:             concrete,
		Storeys:          [][]facadeOpening{windows, growing.parking(count, length)}}
	parts, err := front.parts()
	drawing.fail(err)
	growing.parts = append(growing.parts, within(parts, frame)...)
	for shop, window := range windows {
		growing.fascia(place{Row: side, Bay: shop}, frame, window.Left)
	}
}

// shopRow is the shops' windows along one side of the podium, lit or dark.
func (growing *accretion) shopRow(side, count int, length float64) []facadeOpening {
	drawing := growing.drawing
	windows := make([]facadeOpening, count)
	widths := make([]float64, count)
	frames := drawing.options("frame-colour")
	for shop := range windows {
		at := place{Row: side, Bay: shop}
		pane, err := drawing.glass(at)
		drawing.fail(err)
		windows[shop] = facadeOpening{Width: drawing.fixed("shop-window-width"),
			Height: drawing.fixed("shop-window-height"),
			Sill:   drawing.fixed("shop-sill"), Pane: pane,
			Frame: paint("frame", drawing.pick("frame-colour", frames, at))}
		widths[shop] = windows[shop].span()
	}
	return placed(windows, widths, length)
}

// parking is the openings of the parking storey over the shops, one over
// each shop, dark.
func (growing *accretion) parking(count int, length float64) []facadeOpening {
	drawing := growing.drawing
	dark := paint("parking", drawing.pick("carpark-dark",
		drawing.options("carpark-dark"), place{}))
	slots := make([]facadeOpening, count)
	widths := make([]float64, count)
	for slot := range slots {
		slots[slot] = facadeOpening{Width: drawing.fixed("carpark-slot-width"),
			Height: drawing.fixed("carpark-slot-height"),
			Sill:   drawing.fixed("carpark-slot-sill"), Pane: dark,
			Frame: dark}
		widths[slot] = slots[slot].span()
	}
	return placed(slots, widths, length)
}

// placed is a row of openings spread evenly along a length.
func placed(openings []facadeOpening, widths []float64, length float64) []facadeOpening {
	lefts, _ := spread(0, length, widths)
	for index := range openings {
		openings[index].Left = lefts[index]
	}
	return openings
}

// fascia is one shop's canopy over the pavement and its sign over the
// canopy, which glows, over the shop's window from its left edge, if the
// pavement is clear there.
func (growing *accretion) fascia(at place, frame maths.Subspace,
	left float64) {
	drawing := growing.drawing
	wide := drawing.fixed("shop-window-width")
	thick := drawing.fixed("shop-canopy-thickness")
	rise := drawing.fixed("podium-storey") - drawing.fixed("fascia-height")
	size := Box{wide, drawing.fixed("fascia-height") + thick,
		drawing.fixed("shop-canopy-depth")}
	corner := maths.V(left, rise-thick, 0)
	if !growing.settle(volumeOf("canopy", -1, frame, size, corner)) {
		return
	}
	lamp := drawing.pick("sign-colour", drawing.options("sign-colour"), at)
	sign := &thing.Material{Name: "cities sign " + lamp.Name,
		Colour: colourOf(lamp), Gloss: lamp.Gloss,
		Glow: colourOf(lamp).Scale(drawing.fixed("sign-glow"))}
	canopy := paint("canopy", drawing.pick("box-colour",
		drawing.options("box-colour"), at))
	growing.parts = append(growing.parts, within([]placedElement{
		put("canopy", Box{wide, thick, size.Depth}, canopy, corner),
		put("sign", Box{wide, drawing.fixed("fascia-height"),
			drawing.fixed("fascia-proud")}, sign,
			corner.Add(maths.V(0, thick, 0)))}, frame)...)
}

// ramp is the car ramp: a slab rising along the podium's back, in the
// apron, from the pavement to the parking's floor, if the apron is clear.
func (growing *accretion) ramp(concrete *thing.Material) {
	drawing := growing.drawing
	podium := growing.out.Podium
	length, width := drawing.fixed("ramp-length"), drawing.fixed("ramp-width")
	thick, rise := drawing.fixed("ramp-thickness"),
		drawing.fixed("podium-storey")
	from := podium.X0 + drawing.fixed("ramp-from")
	if from+length > podium.X1 {
		return
	}
	volume := Volume{Kind: "ramp", Plot: -1,
		Low:  [3]float64{from - thick, 0, podium.Z0 - width},
		High: [3]float64{from + length, rise + thick, podium.Z0}}
	if !growing.settle(volume) {
		return
	}
	slope := math.Atan2(rise, length)
	growing.parts = append(growing.parts, placedElement{Element: solid{Name: "ramp",
		Form:     Box{math.Hypot(length, rise), thick, width},
		Material: concrete}, Place: maths.NewSubspace(
		maths.V(from, 0, podium.Z0-width), maths.Rotation(maths.ZAxis,
			slope), 0)})
}

// slab draws a run of flats as it was planned: a face of households to
// the street and another to the court, the body between them, its roof,
// and the decks along its court face.
func (growing *accretion) slab(index int) {
	plot := growing.out.Plots[index]
	court := opposite(plot.Out)
	growing.decked(index)
	shells := map[int]float64{}
	for _, side := range []int{plot.Out, court} {
		shells[side] = growing.slabFace(index, side)
	}
	body := plot.Lot.trimmed(plot.Out, shells[plot.Out]).trimmed(court,
		shells[court])
	growing.parts = append(growing.parts, put("wall", Box{body.width(),
		plot.Height - plot.Base, body.depth()}, growing.walls[index],
		maths.V(body.X0, plot.Base, body.Z0)))
	growing.roofed(index)
	growing.decks(index)
}

// slabFace draws one long face of a slab, its rows of households drawn by
// the condominium's culture, and gives how deep the face is behind its
// plane, which the body stops short of.
func (growing *accretion) slabFace(index, side int) float64 {
	plot := growing.out.Plots[index]
	frame, width := wallFrame(plot.Lot, side, 0)
	shape := massShape{width: width, depth: across(plot.Lot, side) / half,
		storeys: plot.Storeys, door: doorNone}
	front := growing.homes.facade(place{}, shape, growing.walls[index])
	front.Storeys = growing.homesRows(place{Bay: faceKey(index, side)},
		width, plot.Storeys)
	if front.Level == demoLevel {
		front.Cavity = front.cavityFor(shape.depth)
	}
	parts, err := front.parts()
	growing.drawing.fail(err)
	growing.parts = append(growing.parts, within(parts, raised(frame,
		plot.Base))...)
	return front.shell()
}

// across is how deep a rectangle is from one side to the other.
func across(box rect, side int) float64 {
	if side == frontSide || side == backSide {
		return box.depth()
	}
	return box.width()
}

// homesRows are the windows of every storey of a slab's face.
func (growing *accretion) homesRows(at place, width float64,
	storeys int) [][]facadeOpening {
	rows := make([][]facadeOpening, storeys)
	for storey := range rows {
		at.Storey = storey
		rows[storey] = growing.homesRow(at, width)
	}
	return rows
}

// homesRow is the windows of one storey of a slab's face, as many as the
// face has room for at the slab's pitch, split into dwellings of the
// number the condominium draws, each dressed by its household's choices
// where the faces are drawn with households.
func (growing *accretion) homesRow(at place, width float64) []facadeOpening {
	homes := growing.homes
	edge := growing.drawing.fixed("window-edge")
	count := int((width - 2*edge) / growing.drawing.fixed("slab-window-pitch"))
	per := max(homes.count("dwelling-windows", at), 1)
	windows := make([]facadeOpening, count)
	widths := make([]float64, count)
	var home *household
	for index := range windows {
		at.Opening, at.Dwelling = index, index/per
		if homes.homes && index%per == 0 {
			home = homes.household(at)
		}
		windows[index] = homes.window(at)
		if home != nil {
			windows[index].Home = home
			home.dress(&windows[index], index%per == 0,
				index%per == per-1 || index == count-1)
		}
		widths[index] = windows[index].span()
	}
	lefts, _ := spread(edge, width-edge, widths)
	for index := range windows {
		windows[index].Left = lefts[index] + windows[index].ShutterWidth
	}
	return windows
}

// deckStoreys are the storeys of a slab that an access deck stands at:
// every few, from the first of them, under the top.
func (growing *accretion) deckStoreys(plot plot) []int {
	every := growing.drawing.count("deck-every", place{})
	var storeys []int
	for storey := every; storey < plot.Storeys; storey += every {
		storeys = append(storeys, storey)
	}
	return storeys
}

// decked tells the households of a slab's court face which storeys stand
// on a deck, so that none of them has a balcony there.
func (growing *accretion) decked(index int) {
	plot := growing.out.Plots[index]
	homes := growing.homes
	if homes.onDecks == nil {
		homes.onDecks = map[place]bool{}
	}
	for _, storey := range growing.deckStoreys(plot) {
		homes.onDecks[place{Bay: faceKey(index, opposite(plot.Out)),
			Storey: storey}] = true
	}
}

// decks are the access decks along a slab's court face: at each deck
// storey a slab standing out of the face at its floor, with a rail along
// its edge, if the air there is clear.
func (growing *accretion) decks(index int) {
	drawing := growing.drawing
	plot := growing.out.Plots[index]
	court := opposite(plot.Out)
	thick, rail := drawing.fixed("deck-thickness"), drawing.fixed("rail-height")
	bar := drawing.fixed("rail-bar")
	concrete := paint("deck", drawing.pick("podium-colour",
		drawing.options("podium-colour"), place{}))
	steel := paint("rail", drawing.pick("cage-colour",
		drawing.options("cage-colour"), place{Bay: index}))
	strip := plot.Lot.beyond(court, drawing.fixed("deck-depth"))
	edge := strip.edge(court, bar)
	for _, storey := range growing.deckStoreys(plot) {
		floor := plot.floorOf(storey)
		if !growing.settle(Volume{Kind: "deck", Plot: index,
			Low:  [3]float64{strip.X0, floor - thick, strip.Z0},
			High: [3]float64{strip.X1, floor + rail, strip.Z1}}) {
			continue
		}
		growing.parts = append(growing.parts,
			put("deck", Box{strip.width(), thick, strip.depth()}, concrete,
				maths.V(strip.X0, floor-thick, strip.Z0)),
			put("rail", Box{edge.width(), rail, edge.depth()}, steel,
				maths.V(edge.X0, floor, edge.Z0)))
	}
}

// core draws a stair and lift core: its body, set in by the recess on
// each side no neighbour stands against, a narrow window at each landing
// on those sides, and its roof.
func (growing *accretion) core(index int) {
	drawing := growing.drawing
	plot := growing.out.Plots[index]
	recess := drawing.fixed("recess")
	body := plot.Lot
	for side := range sidesAround {
		if growing.open(index, side) {
			body = body.trimmed(side, recess)
		}
	}
	growing.parts = append(growing.parts, put("wall", Box{body.width(),
		plot.Height - plot.Base, body.depth()}, growing.walls[index],
		maths.V(body.X0, plot.Base, body.Z0)))
	for side := range sidesAround {
		if growing.open(index, side) {
			growing.landings(index, side)
		}
	}
	growing.roofed(index)
}

// open is whether no other plot stands against all of one side of a
// plot, whatever its height.
func (growing *accretion) open(index, side int) bool {
	plot := growing.out.Plots[index]
	for other, neighbour := range growing.out.Plots {
		if other != index && neighbour.Base <= plot.Base+meeting &&
			covers(neighbour.Lot, plot.Lot, side) {
			return false
		}
	}
	return true
}

// landings is one open face of a core: a narrow window in the middle of
// each storey, one over each landing.
func (growing *accretion) landings(index, side int) {
	drawing := growing.drawing
	plot := growing.out.Plots[index]
	frame, width := wallFrame(plot.Lot, side, 0)
	front := growing.plain(index, width, plot.Height-plot.Base)
	front.Ground = plot.Ground
	front.Storeys = make([][]facadeOpening, plot.Storeys)
	for storey := range front.Storeys {
		slit := drawing.window(place{Row: side, Bay: index, Storey: storey})
		slit.Width = drawing.fixed("core-slit-width")
		slit.Left = (width - slit.Width) / half
		front.Storeys[storey] = []facadeOpening{slit}
	}
	parts, err := front.parts()
	drawing.fail(err)
	growing.parts = append(growing.parts, within(parts, raised(frame,
		plot.Base))...)
}
