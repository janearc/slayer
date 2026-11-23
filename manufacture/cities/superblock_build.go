package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The rows of places a superblock draws its accreted plots at, apart from
// the plan's, which are at row nought by the plot's index.
const (
	// infillRow is the row the court's plots are drawn at, by the index
	// of the lot.
	infillRow = 1
	// toppingRow is the row floors added on roofs are drawn at.
	toppingRow = 2
)

// most is the greatest value an element ranging between two may take,
// which is how a bound that holds whatever is drawn is read from data.
func (drawing *drawer) most(name string) float64 {
	each, _ := drawing.found(name, drawRange)
	return each.Most
}

// same is whether two coordinates are one place, to the slack of the
// arithmetic.
func same(first, second float64) bool {
	return math.Abs(first-second) <= meeting
}

// beyond is the strip of air outside one side of the rectangle, as deep
// as given, along the whole of that side.
func (box rect) beyond(side int, proud float64) rect {
	switch side {
	case frontSide:
		return rect{box.X0, box.Z1, box.X1, box.Z1 + proud}
	case eastSide:
		return rect{box.X1, box.Z0, box.X1 + proud, box.Z1}
	case backSide:
		return rect{box.X0, box.Z0 - proud, box.X1, box.Z0}
	}
	return rect{box.X0 - proud, box.Z0, box.X0, box.Z1}
}

// covers is whether a neighbour stands against the whole of one side of
// a lot, with no gap between.
func covers(neighbour, lot rect, side int) bool {
	spansX := neighbour.X0 <= lot.X0+meeting && neighbour.X1 >= lot.X1-meeting
	spansZ := neighbour.Z0 <= lot.Z0+meeting && neighbour.Z1 >= lot.Z1-meeting
	switch side {
	case frontSide:
		return same(neighbour.Z0, lot.Z1) && spansX
	case eastSide:
		return same(neighbour.X0, lot.X1) && spansZ
	case backSide:
		return same(neighbour.Z1, lot.Z0) && spansX
	}
	return same(neighbour.X1, lot.X0) && spansZ
}

// wallFrame is where one wall of a lot's building stands: its own frame,
// with the origin at the foot of its left edge as the outside sees it and
// its face looking out along z, set in the parcel's; and the wall's
// length. The front and back walls run the whole width, and the side
// walls stand between them, so that no corner is drawn twice.
func wallFrame(lot rect, side int, recess float64) (maths.Subspace, float64) {
	width, depth := lot.width(), lot.depth()-2*recess
	switch side {
	case frontSide:
		return standingAt(maths.V(lot.X0, 0, lot.Z1)), width
	case eastSide:
		return maths.NewSubspace(maths.V(lot.X1, 0, lot.Z1-recess),
			maths.Rotation(maths.YAxis, quarterTurn), 0), depth
	case backSide:
		return maths.NewSubspace(maths.V(lot.X1, 0, lot.Z0),
			maths.Rotation(maths.YAxis, math.Pi), 0), width
	}
	return maths.NewSubspace(maths.V(lot.X0, 0, lot.Z0+recess),
		maths.Rotation(maths.YAxis, -quarterTurn), 0), depth
}

// raised is a wall's frame lifted to the foot of the building it is on.
func raised(frame maths.Subspace, foot float64) maths.Subspace {
	frame.Origin = frame.Origin.Add(maths.V(0, foot, 0))
	return frame
}

// cornersOfBox is how many corners a box has.
const cornersOfBox = 8

// volumeOf is the box of air a box takes when it stands at a point of a
// frame, and the frame stands in the parcel: the least and greatest of
// its eight corners. The frames here are turned by whole quarters, so
// the volume is the box exactly.
func volumeOf(kind string, plot int, frame maths.Subspace, size Box,
	at maths.Vec) Volume {
	volume := Volume{Kind: kind, Plot: plot,
		Low:  [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)},
		High: [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}}
	for corner := range cornersOfBox {
		point := frame.Out(at.Add(maths.V(
			size.Width*float64(corner&1), size.Height*float64(corner>>1&1),
			size.Depth*float64(corner>>2&1))))
		for axis, value := range components(point) {
			volume.Low[axis] = math.Min(volume.Low[axis], value)
			volume.High[axis] = math.Max(volume.High[axis], value)
		}
	}
	return volume
}

// meets is whether the insides of two volumes meet: faces that only
// touch do not.
func (volume Volume) meets(other Volume) bool {
	for axis := range volume.Low {
		if math.Min(volume.High[axis], other.High[axis])-
			math.Max(volume.Low[axis], other.Low[axis]) <= meeting {
			return false
		}
	}
	return true
}

// clear is whether a volume stands inside the parcel, over the ground,
// and where nothing else is solid.
func (growing *accretion) clear(volume Volume) bool {
	parcel := growing.out.Parcel
	if volume.Low[0] < parcel.X0-meeting || volume.High[0] > parcel.X1+meeting ||
		volume.Low[2] < parcel.Z0-meeting || volume.High[2] > parcel.Z1+meeting ||
		volume.Low[1] < -meeting {
		return false
	}
	for _, solid := range growing.solids {
		if volume.meets(solid) {
			return false
		}
	}
	return true
}

// settle makes a volume solid and keeps it as a fixture, if it is clear,
// and says whether it did.
func (growing *accretion) settle(volume Volume) bool {
	if !growing.clear(volume) {
		return false
	}
	growing.solids = append(growing.solids, volume)
	growing.out.Fixtures = append(growing.out.Fixtures, volume)
	return true
}

// storeysOf is how many storeys a building of the court has: near the
// ceiling or just under it by the culture's chance, and otherwise a share
// of it, never fewer than the least; and no more than the accretion has
// let the court's buildings grow to, which is the least at nought and the
// ceiling at one.
func (growing *accretion) storeysOf(at place) int {
	drawing := growing.drawing
	ceiling := float64(growing.ask.Ceiling)
	storeys := ceiling * drawing.number("short-fraction", at)
	if drawing.chance("near-ceiling", at) {
		storeys = ceiling - float64(drawing.count("near-drop", at))
	}
	least := min(float64(drawing.count("least-storeys", place{})), ceiling)
	grown := math.Round(least + (ceiling-least)*growing.grown)
	return int(min(max(least, math.Floor(storeys)), grown))
}

// raise gives the plan its wall and makes it solid, then builds over each
// plot of the court the accretion's gate lets through, with its storeys,
// height and wall, and keeps the rest as yards.
func (growing *accretion) raise() {
	drawing := growing.drawing
	palette, cladding := drawing.options("palette"),
		drawing.options("cladding")
	planned := wall(drawing.pick("palette", palette, place{}),
		drawing.pick("cladding", cladding, place{}))
	for index, plot := range growing.out.Plots {
		growing.walls = append(growing.walls, planned)
		growing.solids = append(growing.solids, plot.volume(index))
	}
	for index, lot := range growing.lots {
		at := place{Row: infillRow, Bay: index}
		if !drawing.gate("infill-share", at, growing.grown, nil) {
			growing.out.Yards = append(growing.out.Yards, lot)
			continue
		}
		storeys := growing.storeysOf(at)
		growing.add(plot{Lot: lot, Use: useInfill, Base: growing.out.Deck,
			Storeys: storeys, Height: growing.out.Deck + float64(storeys)*
				drawing.number("storey-height", at)}, at)
	}
}

// add keeps a plot built by the accretion, in a wall of its own, and makes
// it solid.
func (growing *accretion) add(plot plot, at place) {
	drawing := growing.drawing
	growing.walls = append(growing.walls, wall(
		drawing.pick("palette", drawing.options("palette"), at),
		drawing.pick("cladding", drawing.options("cladding"), at)))
	growing.solids = append(growing.solids, plot.volume(len(
		growing.out.Plots)))
	growing.out.Plots = append(growing.out.Plots, plot)
}

// top adds floors on the roofs of the slabs: each roof, inside its
// parapet, cut along its length into stretches, and on each the gate lets
// through, a few storeys of a height that matches nothing below, kept
// under the ceiling.
func (growing *accretion) top() {
	drawing := growing.drawing
	wall := drawing.fixed("parapet-wall")
	for index := range growing.out.Plots {
		slab := growing.out.Plots[index]
		if slab.Use != useFlats {
			continue
		}
		roof := slab.Lot.inset(wall)
		along := roof.width() >= roof.depth()
		start, end := roof.Z0, roof.Z1
		if along {
			start, end = roof.X0, roof.X1
		}
		for piece := 0; start < end-drawing.fixed("plot-least"); piece++ {
			at := place{Row: toppingRow, Bay: index, Storey: piece}
			length := min(drawing.number("topping-length", at), end-start)
			growing.topping(at, slab, roof.span(along, start, start+length))
			start += length
		}
	}
}

// topping is floors added on a stretch of a slab's roof, if the gate lets
// them through and one of them fits under the ceiling.
func (growing *accretion) topping(at place, slab plot, lot rect) {
	drawing := growing.drawing
	if !drawing.gate("topping-share", at, growing.grown, nil) {
		return
	}
	base := slab.Height + drawing.fixed("slab-thickness")
	high := drawing.number("topping-storey-height", at)
	room := int(math.Floor((growing.out.Ceiling - base) / high))
	storeys := min(drawing.reach("topping-storeys", at, growing.grown), room)
	if storeys < 1 {
		return
	}
	growing.add(plot{Lot: lot, Use: useTopping, Base: base, Storeys: storeys,
		Height: base + float64(storeys)*high}, at)
}

// build draws the podium, then every building as what it is, then lets
// each carry out a jetty in the order the seed gives, a later one yielding
// to whatever is there.
func (growing *accretion) build() {
	order := growing.order.Perm(len(growing.out.Plots))
	for rank, index := range order {
		growing.out.Plots[index].Order = rank
	}
	growing.podium()
	for index, plot := range growing.out.Plots {
		switch plot.Use {
		case useFlats:
			growing.slab(index)
		case useCore:
			growing.core(index)
		default:
			growing.house(index)
		}
	}
	for _, index := range order {
		growing.jetty(index)
	}
}

// house is one building of the accretion drawn: a body set in from its lot
// by the recess all round, a facade on each wall that is not buried
// against a neighbour, and a flat roof.
func (growing *accretion) house(index int) {
	drawing := growing.drawing
	plot, wall := growing.out.Plots[index], growing.walls[index]
	lot, recess := plot.Lot, drawing.fixed("recess")
	body := put("wall", Box{lot.width() - 2*recess, plot.Height - plot.Base,
		lot.depth() - 2*recess}, wall,
		maths.V(lot.X0+recess, plot.Base, lot.Z0+recess))
	growing.parts = append(growing.parts, body)
	for side := range sidesAround {
		growing.face(index, side)
	}
	growing.roofed(index)
}

// roofed is a building's flat roof, on its top.
func (growing *accretion) roofed(index int) {
	plot := growing.out.Plots[index]
	lot := plot.Lot
	shape := massShape{width: lot.width(), depth: lot.depth()}
	roof, err := growing.drawing.roof(place{Bay: index}, shape,
		growing.walls[index]).parts()
	growing.drawing.fail(err)
	growing.parts = append(growing.parts, within(roof, standingAt(
		maths.V(lot.X0, plot.Height, lot.Z0)))...)
}

// buried is whether a wall has a neighbour standing against all of it,
// from below its foot to above its top, so that it is never seen.
func (growing *accretion) buried(index, side int) bool {
	plot := growing.out.Plots[index]
	for other, neighbour := range growing.out.Plots {
		if other != index && neighbour.Base <= plot.Base+meeting &&
			neighbour.Height >= plot.Height &&
			covers(neighbour.Lot, plot.Lot, side) {
			return true
		}
	}
	return false
}

// face is the facade of one wall of a building, as the facade code
// draws it from its rows of windows, set in the wall's frame.
func (growing *accretion) face(index, side int) {
	if growing.buried(index, side) {
		return
	}
	drawing := growing.drawing
	plot := growing.out.Plots[index]
	frame, width := wallFrame(plot.Lot, side, drawing.fixed("recess"))
	front := growing.plain(index, width, plot.Height-plot.Base)
	front.Storeys = growing.rows(index, side, width)
	parts, err := front.parts()
	drawing.fail(err)
	growing.parts = append(growing.parts, within(parts, raised(frame,
		plot.Base))...)
}

// plain is a facade of the superblock's own, a width and a height, with
// no openings yet.
func (growing *accretion) plain(index int, width, height float64) facade {
	drawing := growing.drawing
	return facade{Width: width, Height: height,
		Recess:           drawing.fixed("recess"),
		FrameWidth:       drawing.fixed("frame-width"),
		GlassThickness:   drawing.fixed("glass-thickness"),
		ShutterThickness: drawing.fixed("shutter-thickness"),
		Wall:             growing.walls[index]}
}

// rows are the windows of each storey of one wall, from the ground up.
func (growing *accretion) rows(index, side int, width float64) [][]facadeOpening {
	rows := make([][]facadeOpening, growing.out.Plots[index].Storeys)
	for storey := range rows {
		rows[storey] = growing.row(place{Row: side, Bay: index,
			Storey: storey}, width)
	}
	return rows
}

// row is the windows of one storey of a wall: as many as the wall has
// room for at the culture's pitch, all one size, set at the same gap
// from each other and from the wall's ends.
func (growing *accretion) row(at place, width float64) []facadeOpening {
	drawing := growing.drawing
	edge := drawing.fixed("window-edge")
	count := int((width - 2*edge) / drawing.fixed("window-pitch"))
	windows := make([]facadeOpening, count)
	widths := make([]float64, count)
	for index := range windows {
		at.Opening = index
		windows[index] = drawing.window(at)
		widths[index] = windows[index].span()
	}
	lefts, _ := spread(edge, width-edge, widths)
	for index := range windows {
		windows[index].Left = lefts[index]
	}
	return windows
}

// jetty carries the top storeys of a building out over the air beside
// one wall, if the gate says so and nothing is there. The side is drawn
// whether or not the building has a jetty, so that the order's stream
// does not depend on the chances.
func (growing *accretion) jetty(index int) {
	drawing := growing.drawing
	at := place{Bay: index}
	plot := growing.out.Plots[index]
	side := growing.order.IntN(sidesAround)
	if !drawing.gate("jetty-chance", at, growing.grown, nil) ||
		plot.Storeys < 2 {
		return
	}
	carried := min(drawing.count("jetty-storeys", at), plot.Storeys-1)
	strip := plot.Lot.beyond(side, drawing.number("jetty-proud", at))
	low := plot.floorOf(plot.Storeys - carried)
	volume := Volume{Kind: "jetty", Plot: index,
		Low:  [3]float64{strip.X0, low, strip.Z0},
		High: [3]float64{strip.X1, plot.Height, strip.Z1}}
	if growing.settle(volume) {
		growing.parts = append(growing.parts, put("jetty", Box{
			strip.width(), plot.Height - low, strip.depth()},
			growing.walls[index], maths.V(strip.X0, low, strip.Z0)))
	}
}
