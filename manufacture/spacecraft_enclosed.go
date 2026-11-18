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

// hubblePanels uses bilateral wings and a continuous support bar.
func (row spacecraft) hubblePanels(assembly *instrumentAssembly,
	size spacecraftSize,
) {
	width := size.span*spaceHalf - size.radius
	height := size.aperture * spacecraftValue("hubble-panel-height")
	lift := size.aperture * spacecraftValue("hubble-panel-lift")
	for _, sign := range []float64{-spaceOne, spaceOne} {
		middle := sign * (size.span*spaceHalf + size.radius) * spaceHalf
		at := maths.V(middle-width*spaceHalf,
			lift-height*spaceHalf, spaceZero)
		assembly.solarPanel(width, height, size.aperture, at)
	}
	assembly.rod(maths.V(-size.span*spaceHalf, lift, spaceZero),
		maths.V(size.span*spaceHalf, lift, spaceZero),
		size.aperture*spacecraftValue("strut-radius"), spaceStructure)
}

// solarPanel uses closed boxes for both the panel and its readable grid.
func (assembly *instrumentAssembly) solarPanel(width, height, aperture float64,
	at maths.Vec,
) {
	thickness := aperture * spacecraftValue("panel-thickness")
	assembly.box(width, height, thickness, at, leaf)
	bar := aperture * spacecraftValue("solar-grid-width")
	columns := int(spacecraftValue("solar-columns"))
	rows := int(spacecraftValue("solar-rows"))
	for index := range columns - spaceOne {
		column := index + spaceOne
		x := (width - bar) * float64(column) / float64(columns)
		start := at.Add(maths.V(x, spaceZero, -bar*spaceHalf))
		assembly.box(bar, height, thickness+bar, start, spaceStructure)
	}
	for index := range rows - spaceOne {
		line := index + spaceOne
		y := (height - bar) * float64(line) / float64(rows)
		start := at.Add(maths.V(spaceZero, y, -bar*spaceHalf))
		assembly.box(width, bar, thickness+bar, start, spaceStructure)
	}
}
