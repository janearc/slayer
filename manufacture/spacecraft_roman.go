package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

const romanShade slot = spaceSecondary + 1
const romanSolarString slot = romanShade + 1
const romanSilverBlanket slot = romanSolarString + 1
const romanGoldBlanket slot = romanSilverBlanket + 1

// roman builds the observatory around its published aperture and envelope.
// Source roman-deployed-reference-glb guides the silhouette and finish.
// Source roman-optical-system-spie-2024, slides 8 and 9, distinguishes the
// forward fabric cover from the round baffle vanes inside the outer barrel.
// Source roman-support-systems identifies the six-sided aft bus.
//
// The eight exterior faces and six fabric bays approximate visible folds;
// NASA's written description does not specify those exact panel counts.
func (row spacecraft) roman(assembly *instrumentAssembly,
	size spacecraftSize,
) {
	row.romanOpticalAssembly(assembly, size)
	row.romanBus(assembly, size)
	row.romanSolarArray(assembly, size)
	row.romanHighGainAntenna(assembly, size)
}

// romanOpticalAssembly makes the long outer barrel, its pupil, and the
// primary and secondary mirrors visible from the open end.
func (row spacecraft) romanOpticalAssembly(a *instrumentAssembly,
	size spacecraftSize,
) {
	aperture := size.aperture
	radius := aperture * spacecraftValue("roman-barrel-radius")
	wall := aperture * spacecraftValue("roman-barrel-wall")
	length := aperture * 1.53
	// The source model projects its cover far ahead of the baffle rim.
	front := -size.length*spaceHalf + aperture*1.52
	axis := maths.About(maths.XAxis, math.Pi*spaceHalf)
	pose := maths.NewSubspace(maths.V(spaceZero, spaceZero, front), axis, spaceOne)
	barrel := turnedProfile{points: [][coordinatePair]float64{
		{radius - wall, spaceZero}, {radius, spaceZero},
		{radius, length * spacecraftValue("roman-barrel-shoulder")},
		{radius * spacecraftValue("roman-barrel-rear-taper"), length},
		{radius*spacecraftValue("roman-barrel-rear-taper") - wall, length},
		{radius - wall, length * spacecraftValue("roman-barrel-shoulder")},
	}}
	a.add(barrel, pose, romanShade)
	romanFacetedHousing(a, front, radius, aperture)

	// The baffles and the central secondary supports are dark structural parts.
	previous := a.fine
	a.fine.around = min(a.fine.around, 48)
	for _, fraction := range []float64{.035, .08, .125, .17, .215, .26, .32, .40, .49, .60, .74} {
		at := front + length*fraction
		clearance := wall * 0.05
		ring := turnedProfile{points: [][coordinatePair]float64{
			{radius - wall*spacecraftValue("roman-baffle-opening"), spaceZero},
			{radius - wall*spacecraftValue("roman-baffle-opening"), wall},
			{radius - wall - clearance, wall},
			{radius - wall - clearance, spaceZero},
		}}
		a.add(ring, maths.NewSubspace(maths.V(spaceZero, spaceZero, at),
			axis, spaceOne), spaceStructure)
	}
	a.fine = previous

	primaryAt := maths.V(spaceZero, spaceZero,
		front+length*spacecraftValue("roman-primary-depth"))
	primaryPose := maths.NewSubspace(primaryAt,
		maths.About(maths.XAxis, -math.Pi*spaceHalf), spaceOne)
	a.add(mirrorDish{radius: aperture * spaceHalf,
		depth: aperture * spacecraftValue("roman-mirror-depth")},
		primaryPose, spaceMirror)

	secondaryAt := maths.V(spaceZero, spaceZero,
		front+length*spacecraftValue("roman-secondary-position"))
	secondaryRadius := aperture * spacecraftValue("roman-secondary-radius")
	for spoke := range triangleCorners {
		angle := fullTurn * float64(spoke) / triangleCorners
		outer := maths.V(0.96*radius*math.Cos(angle),
			0.96*radius*math.Sin(angle), secondaryAt.Z)
		inner := maths.V(secondaryRadius*1.2*math.Cos(angle),
			secondaryRadius*1.2*math.Sin(angle), secondaryAt.Z)
		a.rod(outer, inner, aperture*spacecraftValue("roman-spider-radius"),
			spaceStructure)
	}
	secondaryPose := maths.NewSubspace(secondaryAt,
		maths.About(maths.XAxis, -math.Pi*spaceHalf), spaceOne)
	a.add(mirrorDish{radius: secondaryRadius,
		depth: aperture * spacecraftValue("roman-mirror-depth")},
		secondaryPose, spaceSecondary)

	row.romanApertureCover(a, size, front)
}

// romanApertureCover approximates the deployed fabric canopy and side
// curtains seen in NASA's imagery. Panel proportions are visual estimates.
func (row spacecraft) romanApertureCover(a *instrumentAssembly,
	size spacecraftSize, front float64,
) {
	aperture := size.aperture
	thick := aperture * .008
	// Six fabric bays follow the raised crown and swept sides visible in
	// NASA's deployed model. The bottom and line of sight are open.
	lip := [7]maths.Vec{
		maths.V(-1.57, .04, front-.12), maths.V(-1.46, 1.09, front-2.65),
		maths.V(-.73, 1.56, front-3.30), maths.V(0, 1.83, front-3.54),
		maths.V(.73, 1.55, front-3.26), maths.V(1.43, 1.07, front-2.65),
		maths.V(1.58, .04, front-.12),
	}
	root := [7]maths.Vec{
		maths.V(-1.57, -.20, front+.16), maths.V(-1.50, .95, front+.16),
		maths.V(-.78, 1.42, front+.16), maths.V(0, 1.65, front+.16),
		maths.V(.78, 1.42, front+.16), maths.V(1.50, .95, front+.16),
		maths.V(1.57, -.20, front+.16),
	}
	for bay := range 6 {
		next := bay + 1
		tipRun := lip[next].Sub(lip[bay])
		rootRun := root[next].Sub(root[bay])
		tipStart := lip[bay].Add(tipRun.Scale(.004))
		tipEnd := lip[next].Sub(tipRun.Scale(.004))
		rootStart := root[bay].Add(rootRun.Scale(.004))
		rootEnd := root[next].Sub(rootRun.Scale(.004))
		outward := maths.V((tipStart.X+tipEnd.X)*.5, 1, 0).Unit()
		romanSheet(a, [4]maths.Vec{tipStart, tipEnd, rootEnd, rootStart},
			outward.Scale(thick), romanShade)
	}
	for _, joint := range []int{1, 3, 5} {
		a.rod(lip[joint], root[joint], thick*.72, spaceStructure)
	}
}

// romanFacetedHousing wraps the circular baffles in a clipped-corner shell.
func romanFacetedHousing(a *instrumentAssembly, front, radius, aperture float64) {
	x, y := radius*1.19, radius*1.02
	const corner = .56
	outline := [8]maths.Vec{
		maths.V(-x*corner, y, 0), maths.V(x*corner, y, 0),
		maths.V(x, y*corner, 0), maths.V(x, -y*corner, 0),
		maths.V(x*corner, -y, 0), maths.V(-x*corner, -y, 0),
		maths.V(-x, -y*corner, 0), maths.V(-x, y*corner, 0),
	}
	start, split, end := front+.04, front+aperture*.62, front+aperture*1.21
	for i, left := range outline {
		right := outline[(i+1)%len(outline)]
		edge := right.Sub(left)
		left, right = left.Add(edge.Scale(.006)), right.Sub(edge.Scale(.006))
		offset := left.Add(right).Unit().Scale(aperture * .014)
		romanSheet(a, [4]maths.Vec{left.Add(maths.V(0, 0, start)), right.Add(maths.V(0, 0, start)),
			right.Add(maths.V(0, 0, split-.014)), left.Add(maths.V(0, 0, split-.014))}, offset, romanSilverBlanket)
		romanSheet(a, [4]maths.Vec{left.Add(maths.V(0, 0, split)), right.Add(maths.V(0, 0, split)),
			right.Add(maths.V(0, 0, end)), left.Add(maths.V(0, 0, end))}, offset, spaceFoil)
	}
}

// romanSheet adds a closed sheet: a four-cornered rim and the same rim
// moved by an offset, joined along the sides, turned to face out.
func romanSheet(a *instrumentAssembly, rim [4]maths.Vec, offset maths.Vec, material slot) {
	var outer [4]maths.Vec
	for i, corner := range rim {
		outer[i] = corner.Add(offset)
	}
	var mesh surface
	mesh = quad(mesh, rim[0], rim[1], rim[2], rim[3])
	mesh = quad(mesh, outer[0], outer[3], outer[2], outer[1])
	for i := range rim {
		j := (i + 1) % len(rim)
		mesh = quad(mesh, rim[j], rim[i], outer[i], outer[j])
	}
	// A bay may be slightly twisted. Orient the whole closed sheet by
	// signed volume so neighbouring triangles keep opposite edge directions.
	var volume float64
	for _, face := range mesh {
		a, b, c := face[0].Sub(rim[0]), face[1].Sub(rim[0]), face[2].Sub(rim[0])
		volume += a.Dot(b.Cross(c))
	}
	if volume < 0 {
		for i := range mesh {
			mesh[i][1], mesh[i][2] = mesh[i][2], mesh[i][1]
		}
	}
	a.add(fixedSurface(mesh), maths.Local(), material)
}

// romanBus models the avionics bus, instrument carrier, thermal blankets,
// and the visible service panels from NASA's assembled observatory views.
func (row spacecraft) romanBus(a *instrumentAssembly, size spacecraftSize) {
	aperture := size.aperture
	width := aperture * spacecraftValue("roman-bus-width")
	height := aperture * spacecraftValue("roman-bus-height")
	length := aperture * spacecraftValue("roman-bus-length")
	centre := aperture * spacecraftValue("roman-bus-center-z")
	front := centre - length*spaceHalf
	axis := maths.About(maths.XAxis, math.Pi*spaceHalf)
	pose := maths.NewSubspace(maths.V(0, 0, centre+aperture*.03), axis, spaceOne)
	previous := a.fine
	a.fine.around = 6
	// A shallow gold blanket band wraps the six-sided aft bus.
	const busRadius = 1.7
	base := turnedProfile{points: [][2]float64{
		{0, 0}, {busRadius, 0}, {busRadius, length}, {0, length},
	}}
	a.add(base, pose, body)
	gold := turnedProfile{points: [][2]float64{
		{busRadius + .004, length * .78}, {busRadius + .042, length * .78},
		{busRadius + .042, length - .02}, {busRadius + .004, length - .02},
	}}
	a.add(gold, pose, romanGoldBlanket)
	a.fine = previous

	// Six avionics units stand in for the visible module panels and
	// instrument-carrier faces. Their order is an authored layout.
	bayWidth := aperture * spacecraftValue("roman-bay-width")
	bayHeight := aperture * spacecraftValue("roman-bay-height")
	bayDepth := aperture * spacecraftValue("roman-bay-depth")
	for sideIndex := range 2 {
		side := -float64(spaceOne)
		if sideIndex == 1 {
			side = spaceOne
		}
		for bay := range 3 {
			z := front + aperture*spacecraftValue("roman-bay-first-z") +
				float64(bay)*aperture*spacecraftValue("roman-bay-step")
			x := side*width*spaceHalf - side*bayDepth
			a.box(bayDepth, bayHeight, bayWidth,
				maths.V(x-bayDepth*spaceHalf,
					height*spacecraftValue("roman-bay-y"), z), spaceFoil)
			a.box(bayDepth*spacecraftValue("roman-bay-rail"),
				bayHeight*spacecraftValue("roman-bay-rail"), bayWidth,
				maths.V(x-bayDepth*spaceHalf,
					height*spacecraftValue("roman-bay-y")+bayHeight,
					z), spaceStructure)
		}
	}

	// The Wide Field Instrument and Coronagraph sit under the carrier.
	moduleWidth := aperture * spacecraftValue("roman-instrument-width")
	moduleHeight := aperture * spacecraftValue("roman-instrument-height")
	moduleDepth := aperture * spacecraftValue("roman-instrument-depth")
	moduleY := -height*spaceHalf - moduleHeight
	for instrument := range 2 {
		x := -moduleWidth*spaceHalf + float64(instrument)*
			(moduleWidth+aperture*spacecraftValue("roman-instrument-gap"))*spaceHalf
		z := centre + aperture*spacecraftValue("roman-instrument-setback")
		a.box(moduleWidth, moduleHeight, moduleDepth,
			maths.V(x, moduleY, z), spaceStructure)
		// A shallow dark end panel identifies each separate instrument housing.
		a.box(moduleWidth*spacecraftValue("roman-instrument-face"),
			moduleHeight*spacecraftValue("roman-instrument-face"),
			aperture*spacecraftValue("roman-instrument-face-depth"),
			maths.V(x+moduleWidth*spacecraftValue("roman-instrument-face-inset"),
				moduleY+moduleHeight*spacecraftValue("roman-instrument-face-inset"),
				z+moduleDepth), spaceFoil)
	}

	// Pale and warm reflective blankets wrap the two long bus sides.
	blanketHeight := height * spacecraftValue("roman-blanket-height")
	blanketDepth := length * spacecraftValue("roman-blanket-depth") * .52
	blanketThick := aperture * spacecraftValue("roman-blanket-thickness")
	for _, side := range []float64{-spaceOne, spaceOne} {
		x := side * (width*spaceHalf + blanketThick*spaceHalf)
		a.box(blanketThick, blanketHeight, blanketDepth,
			maths.V(x-blanketThick*spaceHalf,
				-blanketHeight*spaceHalf,
				centre-blanketDepth*.77), romanSilverBlanket)
		// The carrier bay in NASA's model exposes its diagonal bracing.
		railX := side * width * .46
		a.fine.around = min(previous.around, 12)
		for _, y := range []float64{-.52, .52} {
			a.rod(maths.V(railX, y, 2.36), maths.V(railX, y, 4.96), .036, spaceStructure)
		}
		a.rod(maths.V(railX, -.49, 2.39), maths.V(railX, .49, 4.91), .031, spaceStructure)
		a.rod(maths.V(railX, .49, 2.39), maths.V(railX, -.49, 4.91), .031, spaceStructure)
		a.fine = previous
	}

	// Lower sunshades make two separate silver planes below the instruments.
	shieldWidth := aperture * spacecraftValue("roman-lower-shield-width")
	shieldDepth := aperture * spacecraftValue("roman-lower-shield-depth")
	shieldGap := aperture * spacecraftValue("roman-lower-shield-gap")
	shieldY := -height*spaceHalf - moduleHeight -
		aperture*spacecraftValue("roman-lower-shield-drop")
	for _, side := range []float64{-spaceOne, spaceOne} {
		a.box(shieldWidth, aperture*spacecraftValue("roman-lower-shield-thickness"),
			shieldDepth, maths.V(side*(shieldWidth+shieldGap)*spaceHalf-
				shieldWidth*spaceHalf, shieldY, centre-shieldDepth*spaceHalf), spaceFoil)
	}
}

// The cells of a Roman panel: this many along the panel's run and across
// it, the picked cell gap per aperture diameter on each side of a cell,
// and a cell's height in metres. The grid represents the reference
// appearance, not a flight inventory.
const romanCellColumns, romanCellRows = 12, 16

const (
	// romanCellGap is the gap on each side of a cell per aperture
	// diameter.
	romanCellGap = 0.0015
	// romanCellHeight is a cell's height, in metres.
	romanCellHeight = 0.004
	// romanBarShare is the share of the gap between two columns that a
	// bus bar fills. Chosen.
	romanBarShare = 0.5
)

// romanSolarArray makes the two fixed centre panels and four deployed
// panels as one array, in a single plane, each with a visible fine cell
// grid. The six panels follow NASA's six-panel SASS, two fixed and four
// deployable: https://science.nasa.gov/missions/roman-space-telescope/building-roman/.
// The panels run along the observatory, in as many rows across it as the
// arrangement has, and a rail runs along the middle of each row.
func (row spacecraft) romanSolarArray(a *instrumentAssembly,
	size spacecraftSize,
) {
	romanArray(size).lay(a)
}

// romanArray is the Roman array in the picked arrangement: its columns
// run aft to forward from the first panel, its rows lie across the
// observatory, and the cells face up.
func romanArray(size spacecraftSize) solarArray {
	aperture := size.aperture
	aft := aperture * spacecraftValue("roman-panel-rail-aft")
	panels := solarPanels{
		along:     int(spacecraftValue("roman-panel-columns")),
		across:    int(spacecraftValue("roman-panel-rows")),
		width:     aperture * spacecraftValue("roman-panel-depth"),
		depth:     aperture * spacecraftValue("roman-panel-width"),
		thickness: aperture * spacecraftValue("roman-panel-thickness"),
		gap:       aperture * spacecraftValue("roman-panel-gap"), reach: aft}
	root := maths.V(spaceZero, aperture*spacecraftValue("roman-array-y")+
		panels.thickness*spaceHalf,
		aperture*spacecraftValue("roman-panel-first-z")-aft)
	return solarArray{
		frame: solarFrame(root, maths.V(spaceZero, spaceZero, spaceOne),
			maths.V(spaceZero, spaceOne, spaceZero)),
		panels: panels, cells: romanCells(panels, aperture),
		yoke: romanRails(panels, aperture),
		looks: solarLooks{cell: leaf, bar: romanSilverBlanket,
			substrate: romanShade, frame: solarMetal,
			hardware: solarMetal},
	}
}

// romanCells is the grid on a Roman panel: each cell its pitch less the
// gap on both sides of it, and the second string the second half of the
// rows across the run.
func romanCells(panels solarPanels, aperture float64) solarCells {
	gap := aperture * romanCellGap * 2
	cells := solarCells{columns: romanCellColumns, rows: romanCellRows,
		height: romanCellHeight, barShare: romanBarShare,
		pitchX: panels.width / romanCellColumns,
		pitchZ: panels.depth / romanCellRows,
		second: romanSolarString, secondFrom: romanCellRows / 2}
	cells.width, cells.depth = cells.pitchX-gap, cells.pitchZ-gap
	return cells
}

// romanRails is the rail under each row of panels, from the aft extension
// to the forward one, with its centre on the panels' back face.
func romanRails(panels solarPanels, aperture float64) solarYoke {
	run := float64(panels.along)*panels.width +
		float64(panels.along-spaceOne)*panels.gap
	rails := solarYoke{
		length: panels.reach + run +
			aperture*spacecraftValue("roman-panel-rail-forward"),
		radius:     aperture * spacecraftValue("roman-array-rail-radius"),
		rootOffset: panels.thickness * spaceHalf,
		tipOffset:  panels.thickness * spaceHalf}
	for index := range panels.across {
		rails.across = append(rails.across, panels.middle(index))
	}
	return rails
}

// romanHighGainAntenna places the dish aft of the Solar Array Sun Shield.
func (row spacecraft) romanHighGainAntenna(a *instrumentAssembly,
	size spacecraftSize,
) {
	aperture := size.aperture
	base := maths.V(aperture*spacecraftValue("roman-antenna-x"),
		aperture*spacecraftValue("roman-antenna-base-y"),
		aperture*spacecraftValue("roman-antenna-base-z"))
	centre := maths.V(aperture*spacecraftValue("roman-antenna-x"),
		aperture*spacecraftValue("roman-antenna-height"),
		aperture*spacecraftValue("roman-antenna-z"))
	a.rod(base, centre, aperture*spacecraftValue("roman-antenna-mast-radius"),
		spaceStructure)
	dish := mirrorDish{radius: aperture * spacecraftValue("roman-antenna-radius"),
		depth: aperture * spacecraftValue("roman-antenna-depth")}
	pose := maths.NewSubspace(centre,
		maths.About(maths.ZAxis, -math.Pi*spaceHalf), spaceOne)
	a.add(dish, pose, spaceFoil)
	for spoke := range int(spacecraftValue("roman-antenna-spokes")) {
		angle := fullTurn * float64(spoke) /
			spacecraftValue("roman-antenna-spokes")
		innerRadius := aperture * spacecraftValue("roman-antenna-spoke-inner")
		start := centre.Add(maths.V(innerRadius*math.Cos(angle), spaceZero,
			innerRadius*math.Sin(angle)))
		end := centre.Add(maths.V(spacecraftValue("roman-antenna-radius")*aperture*
			math.Cos(angle), spaceZero,
			spacecraftValue("roman-antenna-radius")*aperture*math.Sin(angle)))
		a.rod(start, end, aperture*spacecraftValue("roman-antenna-spoke-radius"),
			spaceStructure)
	}
}
