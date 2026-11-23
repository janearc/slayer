package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// enclosedDetail adds service hardware around the optical opening. NASA
// and Spitzer mission diagrams establish the component families; their
// display sizes and placements are authored in spacecraft-choices.yaml.
// See doc/model/spacecraft/space-instrument-studies.md for the references
// and limits.
func (row spacecraft) enclosedDetail(a *instrumentAssembly, size spacecraftSize) {
	aperture := size.aperture
	bus := aperture * spacecraftValue("bus-height")
	relief := aperture * spacecraftValue("detail-relief")
	for _, fraction := range []float64{spaceZero, spaceHalf, spaceOne} {
		a.spaceBand(size.radius, relief, bus+fraction*(size.length-relief))
	}
	row.serviceBays(a, size, bus)
	row.opticalSpider(a, size, bus)
	if row.design == "hubble" {
		row.hubbleDoor(a, size, bus)
		row.hubbleAntennas(a, size, bus)
	} else {
		// A thermal break separates the warm bus and the cold housing.
		a.spaceBand(size.radius, relief, bus+aperture*
			spacecraftValue("spitzer-thermal-break"))
	}
}

// spaceBand closes a narrow annulus around the outside of a housing.
func (a *instrumentAssembly) spaceBand(radius, relief, height float64) {
	band := turnedProfile{points: [][coordinatePair]float64{
		{radius, spaceZero}, {radius + relief, spaceZero},
		{radius + relief, relief}, {radius, relief}}}
	a.add(band, maths.NewSubspace(maths.Up.Scale(height),
		maths.Identity, spaceOne), spaceStructure)
}

// serviceBays places raised equipment doors on the bus facets, with narrow
// rails above them. These are closed relief meshes, not painted outlines.
func (row spacecraft) serviceBays(a *instrumentAssembly, size spacecraftSize, bus float64) {
	wide := size.radius * spacecraftValue("service-bay-width")
	tall := bus * spacecraftValue("service-bay-height")
	depth := size.aperture * spacecraftValue("detail-relief")
	fine := a.fine
	a.fine.around = octagonSides
	defer func() { a.fine = fine }()
	for bay := range octagonSides {
		angle := fullTurn * (float64(bay) + spaceHalf) / octagonSides
		turn := maths.About(maths.YAxis, angle)
		at := maths.V(spaceZero, (bus-tall)*spaceHalf,
			size.radius*math.Cos(math.Pi/octagonSides))
		a.add(boxShape(wide, tall, depth), maths.NewSubspace(
			turn.Out(at), turn, spaceOne), spaceFoil)
		left := turn.Out(at.Add(maths.V(-wide*spaceHalf, tall+depth, depth)))
		right := turn.Out(at.Add(maths.V(wide*spaceHalf, tall+depth, depth)))
		a.rod(left, right, depth*spaceHalf, body)
	}
}

// opticalSpider leaves a real entrance pupil around the secondary. It is
// a visible support model, not a calibrated Cassegrain optical prescription.
func (row spacecraft) opticalSpider(a *instrumentAssembly, size spacecraftSize, bus float64) {
	lift := bus + size.length*spacecraftValue("enclosed-secondary-height")
	radius := size.aperture * spacecraftValue("secondary-radius")
	fine := a.fine
	a.fine.around = octagonSides
	for arm := range triangleCorners {
		angle := fullTurn * float64(arm) / triangleCorners
		axis := maths.V(math.Cos(angle), spaceZero, math.Sin(angle))
		a.rod(axis.Scale(radius).Add(maths.Up.Scale(lift)),
			axis.Scale(size.radius-size.aperture*spacecraftValue("wall-thickness")).Add(maths.Up.Scale(lift)),
			size.aperture*spacecraftValue("spider-radius"), spaceStructure)
	}
	a.fine = fine
	a.rod(maths.V(spaceZero, lift, spaceZero),
		maths.V(spaceZero, lift+size.aperture*spacecraftValue("detail-relief"), spaceZero),
		radius, spaceSecondary)
}

// hubbleDoor stands beside the entrance with a hinge at the rim. The
// opened silhouette is the NASA diagram's; this angle is a display pose.
func (row spacecraft) hubbleDoor(a *instrumentAssembly, size spacecraftSize, bus float64) {
	relief := size.aperture * spacecraftValue("detail-relief")
	centre := maths.V(spaceZero, bus+size.length+size.radius,
		size.radius*spacecraftValue("hubble-door-out"))
	disc := turnedProfile{points: [][coordinatePair]float64{
		{spaceZero, spaceZero}, {size.radius, spaceZero},
		{size.radius, relief}, {spaceZero, relief}}}
	a.add(disc, maths.NewSubspace(centre,
		maths.About(maths.XAxis, math.Pi*spaceHalf), spaceOne), spaceFoil)
	a.rod(maths.V(-size.radius*spaceHalf, bus+size.length, size.radius),
		maths.V(size.radius*spaceHalf, bus+size.length, size.radius), relief, spaceStructure)
}

// hubbleAntennas separates the two communication dishes from the solar
// wings. Each dish is on its own short boom above the equipment section.
func (row spacecraft) hubbleAntennas(a *instrumentAssembly, size spacecraftSize, bus float64) {
	for _, side := range []float64{-spaceOne, spaceOne} {
		start := maths.V(side*size.radius, bus, spaceZero)
		end := maths.V(side*size.span*spacecraftValue("hubble-antenna-out"),
			bus+size.aperture*spacecraftValue("hubble-antenna-rise"), spaceZero)
		a.rod(start, end, size.aperture*spacecraftValue("strut-radius"), spaceStructure)
		dish := mirrorDish{radius: size.aperture * spacecraftValue("hubble-antenna-radius"),
			depth: size.aperture * spacecraftValue("antenna-depth")}
		a.add(dish, maths.NewSubspace(end, maths.Identity, spaceOne), spaceFoil)
	}
}
