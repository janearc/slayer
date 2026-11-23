package cities

import (
	"fmt"
	"math"
)

// podiumStoreys is how many storeys the podium has: the shops, and the
// parking over them.
const podiumStoreys = 2

// opposite is the side that faces the other way.
func opposite(side int) int {
	return (side + half) % sidesAround
}

// trimmed is the rectangle with a strip of a depth taken off one side.
func (box rect) trimmed(side int, depth float64) rect {
	switch side {
	case frontSide:
		box.Z1 -= depth
	case eastSide:
		box.X1 -= depth
	case backSide:
		box.Z0 += depth
	default:
		box.X0 += depth
	}
	return box
}

// edge is the strip of a depth inside the rectangle along one side.
func (box rect) edge(side int, depth float64) rect {
	switch side {
	case frontSide:
		return rect{box.X0, box.Z1 - depth, box.X1, box.Z1}
	case eastSide:
		return rect{box.X1 - depth, box.Z0, box.X1, box.Z1}
	case backSide:
		return rect{box.X0, box.Z0, box.X1, box.Z0 + depth}
	}
	return rect{box.X0, box.Z0, box.X0 + depth, box.Z1}
}

// inset is the rectangle with a strip of a depth taken off every side.
func (box rect) inset(depth float64) rect {
	return rect{box.X0 + depth, box.Z0 + depth, box.X1 - depth,
		box.Z1 - depth}
}

// floorOf is the height of the floor of one storey of a plot's building:
// its foot, the taller ground storey, and the storeys under it.
func (plot plot) floorOf(storey int) float64 {
	if storey == 0 {
		return plot.Base
	}
	high := (plot.Height - plot.Base - plot.Ground) / float64(plot.Storeys)
	return plot.Base + plot.Ground + float64(storey)*high
}

// plan lays the block out as it was planned: the podium on the parcel
// less its apron, a core at each corner of the podium, a slab along each
// side between them cut into runs by cores, and the court inside; then
// raises the slabs to the storeys their people need.
func (growing *accretion) plan() {
	drawing := growing.drawing
	deep := drawing.fixed("slab-depth")
	podium := growing.out.Parcel.inset(drawing.fixed("apron-width"))
	growing.out.Podium = podium
	growing.out.Deck = podiumStoreys * drawing.fixed("podium-storey")
	growing.out.Court = podium.inset(deep)
	growing.out.PerPerson = drawing.fixed("floor-per-person")
	growing.out.Allowance = drawing.fixed("clutter-allowance")
	court := growing.out.Court
	for _, corner := range []rect{{podium.X0, podium.Z0, court.X0, court.Z0},
		{court.X1, podium.Z0, podium.X1, court.Z0},
		{podium.X0, court.Z1, court.X0, podium.Z1},
		{court.X1, court.Z1, podium.X1, podium.Z1}} {
		growing.planned(useCore, corner, frontSide)
	}
	growing.run(rect{court.X0, podium.Z0, court.X1, court.Z0}, backSide)
	growing.run(rect{court.X0, court.Z1, court.X1, podium.Z1}, frontSide)
	growing.run(rect{podium.X0, court.Z0, court.X0, court.Z1}, westSide)
	growing.run(rect{court.X1, court.Z0, podium.X1, court.Z1}, eastSide)
	growing.demand()
}

// planned adds a plot of the plan, standing on the podium, facing out of
// the block on a side, in the planned block's wall.
func (growing *accretion) planned(use string, lot rect, out int) {
	growing.out.Plots = append(growing.out.Plots, plot{Lot: lot, Use: use,
		Out: out, Base: growing.out.Deck})
}

// run is one side's slab: its ground cut along its length into runs of
// flats no longer than the culture's most, with a core between each two.
func (growing *accretion) run(slab rect, out int) {
	drawing := growing.drawing
	along := out == frontSide || out == backSide
	start, end := slab.Z0, slab.Z1
	if along {
		start, end = slab.X0, slab.X1
	}
	core := drawing.fixed("core-length")
	runs := math.Ceil((end - start) / drawing.fixed("slab-run-most"))
	length := (end - start - (runs-1)*core) / runs
	for index := range int(runs) {
		from := start + float64(index)*(length+core)
		growing.planned(useFlats, slab.span(along, from, from+length), out)
		if float64(index) < runs-1 {
			growing.planned(useCore, slab.span(along, from+length,
				from+length+core), out)
		}
	}
}

// span is the stretch of the rectangle between two places along x, or
// along z.
func (box rect) span(along bool, from, to float64) rect {
	if along {
		return rect{from, box.Z0, to, box.Z1}
	}
	return rect{box.X0, from, box.X1, to}
}

// demand raises the slabs: their storeys from the people and the floor of
// a storey of every run, at the crowding tolerated, and refused over the
// ceiling with how many do not fit; and the cores one storey over them.
func (growing *accretion) demand() {
	drawing, homes := growing.drawing, growing.homes
	need := demand{Occupants: growing.ask.Occupants,
		Crowding: growing.ask.Crowding, PerPerson: growing.out.PerPerson,
		Crush: drawing.fixed("crush-limit")}
	for _, plot := range growing.out.Plots {
		if plot.Use == useFlats {
			need.Footprint += plot.Lot.area()
		}
	}
	storeys, err := need.storeys()
	drawing.fail(err)
	if err == nil && storeys > growing.ask.Ceiling {
		drawing.fail(fmt.Errorf("cities: %d people at a crowding of %v need "+
			"%d storeys, over the ceiling of %d: %d do not fit",
			need.Occupants, need.Crowding, storeys, growing.ask.Ceiling,
			need.Occupants-need.held(growing.ask.Ceiling)))
	}
	growing.out.Storeys = storeys
	high := homes.number("storey-height", place{})
	ground := growing.ground()
	tallest := max(high, drawing.most("storey-height"))
	growing.out.Ceiling = growing.out.Deck + ground +
		float64(growing.ask.Ceiling+1)*tallest
	for index := range growing.out.Plots {
		plot := &growing.out.Plots[index]
		plot.Storeys, plot.Ground = storeys, ground
		if plot.Use == useCore {
			plot.Storeys++
		}
		plot.Height = plot.Base + ground + float64(plot.Storeys)*high
	}
}

// ground is how much taller the ground storey of a slab is than the rest:
// what the condominium's faces give it at the demo level, and nought at
// any other.
func (growing *accretion) ground() float64 {
	if growing.homes.level == demoLevel {
		return growing.homes.fixed("ground-extra")
	}
	return 0
}
