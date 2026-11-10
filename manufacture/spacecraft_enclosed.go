package manufacture

import (
	"github.com/janearc/slayer/maths"
)

// Enclosed spacecraft keep an open optical baffle and a separate bus.
func (row spacecraft) enclosed(assembly *instrumentAssembly,
	size spacecraftSize,
) {
	aperture := size.aperture
	bus := aperture * spacecraftValue("bus-height")
	foot := maths.Local()
	busProfile := turnedProfile{points: [][coordinatePair]float64{
		{spaceZero, spaceZero}, {size.radius, spaceZero},
		{size.radius, bus}, {spaceZero, bus}}}
	busFine := assembly.fine
	assembly.fine = cut{around: octagonSides}
	assembly.add(busProfile, foot, spaceStructure)
	assembly.fine = busFine
	wall := aperture * spacecraftValue("wall-thickness")
	tube := turnedProfile{points: [][coordinatePair]float64{
		{size.radius, spaceZero}, {size.radius, size.length},
		{size.radius - wall, size.length},
		{size.radius - wall, spaceZero}}}
	tubeStart := maths.V(spaceZero, bus, spaceZero)
	tubePose := maths.NewSubspace(tubeStart, maths.Identity, spaceOne)
	assembly.add(tube, tubePose, body)
	mirror := mirrorDish{radius: aperture * spaceHalf,
		depth: aperture * spacecraftValue("mirror-depth")}
	lift := bus + aperture*spacecraftValue("mirror-lift")
	mirrorAt := maths.V(spaceZero, lift, spaceZero)
	mirrorPose := maths.NewSubspace(mirrorAt, maths.Identity, spaceOne)
	assembly.add(mirror, mirrorPose, spaceMirror)
	if row.design == "spitzer" {
		row.spitzerPanel(assembly, size)
	} else {
		row.hubblePanels(assembly, size)
	}
}

// spitzerPanel keeps the shield behind the telescope housing.
func (row spacecraft) spitzerPanel(assembly *instrumentAssembly,
	size spacecraftSize,
) {
	width := size.aperture * spacecraftValue("spitzer-panel-width")
	depth := size.aperture * spacecraftValue("spitzer-panel-offset")
	assembly.solarPanel(width, size.span, size.aperture,
		maths.V(-width*spaceHalf, spaceZero, -depth))
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
