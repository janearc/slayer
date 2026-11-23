package manufacture

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// Enclosed spacecraft keep an open optical baffle and a separate bus.
func (row spacecraft) enclosed(assembly *instrumentAssembly,
	size spacecraftSize,
) {
	aperture := size.aperture
	bus := aperture * spacecraftValue("bus-height")
	busProfile := []ComponentProfilePoint{
		{Radius: measure.Metres(spaceZero), Height: measure.Metres(spaceZero)},
		{Radius: measure.Metres(size.radius), Height: measure.Metres(spaceZero)},
		{Radius: measure.Metres(size.radius), Height: measure.Metres(bus)},
		{Radius: measure.Metres(spaceZero), Height: measure.Metres(bus)},
	}
	ComponentSpec{Form: ComponentDrum, Radius: measure.Metres(size.radius),
		Height: measure.Metres(bus), Cuts: octagonSides, Profile: busProfile,
		exact: &componentMetrics{radius: size.radius, height: bus,
			profile: [][coordinatePair]float64{{spaceZero, spaceZero}, {size.radius, spaceZero},
				{size.radius, bus}, {spaceZero, bus}}}}.appendTo(
		assembly, spaceStructure)
	wall := aperture * spacecraftValue("wall-thickness")
	// the tube is round, and its sides only stand for that: a polished
	// one shaded flat mirrors the world as so many strips, each at its
	// own angle, and draws a sawtooth inside the tube
	tubeStart := maths.V(spaceZero, bus, spaceZero)
	tubePose := maths.NewSubspace(tubeStart, maths.Identity, spaceOne)
	tubeProfile := []ComponentProfilePoint{
		{Radius: measure.Metres(size.radius), Height: measure.Metres(spaceZero)},
		{Radius: measure.Metres(size.radius), Height: measure.Metres(size.length)},
		{Radius: measure.Metres(size.radius - wall), Height: measure.Metres(size.length)},
		{Radius: measure.Metres(size.radius - wall), Height: measure.Metres(spaceZero)},
	}
	ComponentSpec{Form: ComponentDrum, Frame: tubePose,
		Radius: measure.Metres(size.radius), Height: measure.Metres(size.length),
		Smooth: true, Profile: tubeProfile,
		exact: &componentMetrics{radius: size.radius, height: size.length,
			profile: [][coordinatePair]float64{{size.radius, spaceZero},
				{size.radius, size.length}, {size.radius - wall, size.length},
				{size.radius - wall, spaceZero}}}}.appendTo(assembly, body)
	// the primary is one paraboloid, which its rings and strips stand
	// for, and is shaded as one, as the tube is
	lift := bus + aperture*spacecraftValue("mirror-lift")
	mirrorAt := maths.V(spaceZero, lift, spaceZero)
	ComponentSpec{Form: ComponentDish, Position: mirrorAt,
		Radius: measure.Metres(aperture * spaceHalf),
		Depth:  measure.Metres(aperture * spacecraftValue("mirror-depth")),
		Smooth: true, exact: &componentMetrics{radius: aperture * spaceHalf,
			depth: aperture * spacecraftValue("mirror-depth")}}.appendTo(assembly, spaceMirror)
	if row.design == "spitzer" {
		row.spitzerPanel(assembly, size)
	} else {
		row.hubblePanels(assembly, size)
	}
	row.enclosedDetail(assembly, size)
}

// The cells of a Hubble wing are chosen: no source in the tree counts
// them. A wing has this many columns along it and rows across it.
const hubbleCellColumns, hubbleCellRows = 24, 12

const (
	// hubbleCellGap is the gap between cells per aperture diameter. Chosen.
	hubbleCellGap = 0.004
	// hubbleCellRise is a cell's height as a share of the blanket's
	// thickness. Chosen.
	hubbleCellRise = 0.4
	// hubbleBarShare is the share of the gap between two columns that a
	// bus bar fills. Chosen.
	hubbleBarShare = 0.5
	// hubbleInset is how many rail widths the cells stand in from the
	// blanket's edge, to leave room for the bars and booms. Chosen.
	hubbleInset = 1.5
)

// hubblePanels puts a rolled wing on each side of the body, each on a
// yoke from the body's wall to the wing's tip. The craft's span is to the
// outer face of a wing's end bar, so each wing is laid pulled in.
func (row spacecraft) hubblePanels(assembly *instrumentAssembly,
	size spacecraftSize,
) {
	for _, side := range []float64{-spaceOne, spaceOne} {
		hubbleWing(size, side).pulledIn().lay(assembly)
	}
}

// hubbleWing is one wing of Hubble as an array. The blanket's width is
// the span between the body and the wing tip, its height and thickness
// the picked ones, the cells on the face towards +z, and the yoke the
// picked support tube.
func hubbleWing(size spacecraftSize, side float64) solarArray {
	aperture := size.aperture
	thickness := aperture * spacecraftValue("panel-thickness")
	panels := solarPanels{along: spaceOne, across: spaceOne,
		width:     size.span*spaceHalf - size.radius,
		depth:     aperture * spacecraftValue("hubble-panel-height"),
		thickness: thickness}
	cells := solarCellsFilling(panels, panels.wanted()*hubbleInset,
		hubbleCellColumns, hubbleCellRows, aperture*hubbleCellGap)
	cells.height = thickness * hubbleCellRise
	cells.barShare = hubbleBarShare
	middle := maths.V(side*size.radius,
		aperture*spacecraftValue("hubble-panel-lift"), thickness*spaceHalf)
	return solarArray{
		frame: solarFrame(middle, maths.V(side, spaceZero, spaceZero),
			maths.V(spaceZero, spaceZero, spaceOne)),
		panels: panels, cells: cells, rolled: true,
		yoke: solarYoke{length: panels.width,
			radius: aperture * spacecraftValue("strut-radius"),
			across: []float64{spaceZero}},
		looks: solarLooks{cell: leaf, bar: spaceFoil,
			substrate: spaceStructure, frame: spacePanel,
			hardware: spaceStructure},
	}
}
