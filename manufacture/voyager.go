package manufacture

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/research"
)

// Voyager is assembled from its data: the published dimensions are asked of
// the research data, and every other dimension, count and placement of the
// craft is asked by name of the book beside this file (voyager-numbers.yaml,
// voyager-points.csv and voyager-looks.csv). What remains here is the order
// the parts are built in and the arithmetic that places them.

// voyagerRows gives both identical twin spacecraft one shared assembly.
// The mission numbers are identical; their separate kinds let sets identify
// the two probes without inventing different hardware.
func voyagerRows() map[string]maker {
	return map[string]maker{
		"voyager-1": voyagerMaker{},
		"voyager-2": voyagerMaker{},
	}
}

// voyagerMaker makes either twin.
type voyagerMaker struct{}

// made is Voyager at the quick detail.
func (maker voyagerMaker) made(from seed) (good, error) {
	return maker.madeAt(from, "quick")
}

// madeAt is Voyager at a detail, or the reason it cannot be made.
func (maker voyagerMaker) madeAt(from seed, detail string) (good, error) {
	object, err := maker.assembled(from, detail)
	if err != nil {
		//ninja:vow feint the maker interface fixes a good beside the error and every maker returns the empty one on refusal
		return good{}, err
	}
	return *object, nil
}

// assembled is the craft with every part built, or nothing and why not: a
// seed that was never sown, a book that is broken, or a name the code asks
// for that the book lacks or the book holds that the code does not ask for.
func (maker voyagerMaker) assembled(from seed, detail string) (*good, error) {
	if !from.made() {
		return nil, errGood
	}
	if err := errors.Join(voyagerBookError, voyagerDressError); err != nil {
		return nil, err
	}
	ask := voyagerBook.asking()
	// This assembly has many capped struts. Keep their shared tessellation
	// within Things' 100k-face flattening limit even at demo complexity.
	chosen := int(spacecraftValue("sides-" + detail))
	around := min(chosen, ask.count("sides-cap"))
	assembly := &instrumentAssembly{
		fine:   cut{around: around},
		object: good{dressed: voyagerLooks()},
	}
	for _, build := range []func(*instrumentAssembly, *asking){
		maker.bus, maker.highGainAntenna, maker.magnetometerBoom,
		maker.radioAntennae, maker.rtgBoom, maker.scienceBoom,
		maker.thrusters,
	} {
		build(assembly, ask)
		if ask.failed() {
			return nil, ask.refusal()
		}
	}
	bounds := voyagerBounds(ask)
	if err := ask.refusal(); err != nil {
		return nil, err
	}
	return assembly.finish(bounds)
}

// voyagerLooks dresses the slots of Voyager from voyager-looks.csv over the
// common spacecraft materials.
func voyagerLooks() looks {
	dressed := spacecraftLooks("spitzer")
	voyagerDress.dress(dressed)
	return dressed
}

// voyagerBounds is the box that encloses the craft with its booms out.
func voyagerBounds(ask *asking) maths.Box {
	return maths.SegmentBox(ask.point("bounds-low"),
		ask.point("bounds-high"), ask.number("bounds-padding"))
}

// placeBox sets a box of a size with its minimum corner at a place.
func placeBox(assembly *instrumentAssembly, size, corner maths.Vec,
	material slot,
) {
	assembly.box(size.X, size.Y, size.Z, corner, material)
}

// placeBoxOn sets a box on the platform: the book names its size and where
// its minimum corner lies against the platform.
func placeBoxOn(assembly *instrumentAssembly, ask *asking,
	platform maths.Vec, size, corner string, material slot,
) {
	at := platform.Add(ask.point(corner))
	placeBox(assembly, ask.point(size), at, material)
}

// bus builds the electronics bus, its equipment bays and its feet.
func (maker voyagerMaker) bus(assembly *instrumentAssembly, ask *asking) {
	diameter := research.Value("voyager-bus-diameter-metres")
	radius := diameter * spaceHalf
	height := ask.number("bus-height")
	previous := assembly.fine
	sides := ask.count("bus-sides")
	assembly.fine.around = sides
	outline := busProfile(ask.table("bus-profile"), radius, height)
	assembly.add(turnedProfile{points: outline}, maths.Local(), body)
	assembly.fine = previous
	maker.busBays(assembly, ask, radius, sides)
	maker.busFeet(assembly, ask)
}

// busProfile is the turned outline of the bus from the shares of its radius
// and of half its height that its table gives.
func busProfile(rows []maths.Vec, radius, height float64,
) [][coordinatePair]float64 {
	points := make([][coordinatePair]float64, len(rows))
	for place, row := range rows {
		points[place] = [coordinatePair]float64{radius * row.X,
			height * row.Y * spaceHalf}
	}
	return points
}

// busBays builds the equipment bays, one on each side of the bus.
func (voyagerMaker) busBays(assembly *instrumentAssembly, ask *asking,
	radius float64, sides int,
) {
	width := ask.number("bus-bay-width")
	height := ask.number("bus-bay-height")
	depth := ask.number("bus-bay-depth")
	bayRadius := radius + ask.number("bus-bay-standoff")
	for side := range sides {
		angle := fullTurn * (float64(side) + spaceHalf) / float64(sides)
		x, z := bayRadius*math.Cos(angle), -bayRadius*math.Sin(angle)
		at := maths.V(x, -height*spaceHalf, z)
		turn := maths.About(maths.YAxis, angle)
		pose := maths.NewSubspace(at, turn, spaceOne)
		corner := maths.V(-width*spaceHalf, spaceZero, -depth*spaceHalf)
		assembly.boxAt(width, height, depth, corner, pose, spaceFoil)
	}
}

// busFeet builds the triangular feet below the bus and the mast between
// them.
func (voyagerMaker) busFeet(assembly *instrumentAssembly, ask *asking) {
	feet := ask.count("bus-feet")
	sole := ask.point("foot-sole")
	for foot := range feet {
		angle := fullTurn * float64(foot) / float64(feet)
		upperRadius := ask.number("foot-upper-radius")
		upper := maths.V(upperRadius*math.Cos(angle),
			ask.number("foot-upper-height"),
			upperRadius*math.Sin(angle))
		toeRadius := ask.number("foot-toe-radius")
		toe := maths.V(toeRadius*math.Cos(angle),
			ask.number("foot-toe-height"), toeRadius*math.Sin(angle))
		assembly.rod(upper, toe, ask.number("foot-leg-radius"),
			spaceStructure)
		assembly.rod(toe, toe.Add(sole), ask.number("foot-sole-radius"),
			spaceStructure)
	}
	assembly.rod(ask.point("bus-mast-top"), ask.point("bus-mast-bottom"),
		ask.number("bus-mast-radius"), spaceStructure)
}

// highGainAntenna builds the reflector and the ribs behind it, then its
// support tower and feed.
func (maker voyagerMaker) highGainAntenna(assembly *instrumentAssembly,
	ask *asking,
) {
	diameter := research.Value("voyager-hga-diameter-metres")
	radius := diameter * spaceHalf
	centre := ask.point("dish-centre")
	depth, thickness := ask.number("dish-depth"),
		ask.number("dish-thickness")
	turn := maths.About(maths.XAxis, math.Pi*spaceHalf)
	pose := maths.NewSubspace(centre, turn, spaceOne)
	assembly.add(radioDish{radius: radius, depth: depth,
		thickness: thickness}, pose, spaceMirror)
	maker.dishRibs(assembly, ask, centre, radius, depth)
	maker.dishSupport(assembly, ask, centre)
}

// dishRibs builds the ribs behind the reflector. They expose its depth
// without putting capped struts edge to edge around its lip, and their inner
// ends are staggered around a small feed hub instead of meeting at a point.
func (voyagerMaker) dishRibs(assembly *instrumentAssembly, ask *asking,
	centre maths.Vec, radius, depth float64,
) {
	ribs := ask.count("dish-ribs")
	setback := ask.number("dish-rib-setback")
	hub := ask.number("dish-hub-radius")
	back := ask.number("dish-hub-back")
	for rib := range ribs {
		angle := fullTurn * float64(rib) / float64(ribs)
		rim := maths.V(radius*math.Cos(angle),
			centre.Y+radius*math.Sin(angle), centre.Z+depth-setback)
		apex := maths.V(hub*math.Cos(angle),
			centre.Y+hub*math.Sin(angle), centre.Z-back)
		assembly.rod(apex, rim, ask.number("dish-rib-radius"),
			spaceStructure)
	}
}

// dishSupport builds the support tower, the central feed and the circular
// subreflector that keep the dish readable as a communication antenna.
func (voyagerMaker) dishSupport(assembly *instrumentAssembly, ask *asking,
	centre maths.Vec,
) {
	arms := ask.count("dish-arms")
	for arm := range arms {
		angle := fullTurn * float64(arm) / float64(arms)
		startRadius := ask.number("dish-arm-start-radius")
		start := maths.V(startRadius*math.Cos(angle),
			ask.number("dish-arm-start-height"),
			startRadius*math.Sin(angle))
		endRadius := ask.number("dish-arm-end-radius")
		end := maths.V(endRadius*math.Cos(angle),
			centre.Y-ask.number("dish-arm-end-drop"),
			centre.Z-ask.number("dish-arm-end-back"))
		assembly.rod(start, end, ask.number("dish-arm-radius"),
			spaceStructure)
	}
	feedStart := maths.V(spaceZero, centre.Y,
		centre.Z-ask.number("dish-feed-back"))
	feedEnd := maths.V(spaceZero, centre.Y,
		centre.Z+ask.number("dish-feed-front"))
	assembly.rod(feedStart, feedEnd, ask.number("dish-feed-radius"),
		spaceGold)
	nearEnd := maths.V(spaceZero, centre.Y,
		centre.Z+ask.number("dish-sub-near"))
	farEnd := maths.V(spaceZero, centre.Y,
		centre.Z+ask.number("dish-sub-far"))
	assembly.rod(nearEnd, farEnd, ask.number("dish-sub-radius"),
		spaceSecondary)
}

// magnetometerBoom builds the lattice boom and the instrument heads on it.
func (maker voyagerMaker) magnetometerBoom(assembly *instrumentAssembly,
	ask *asking,
) {
	length := research.Value("voyager-magnetometer-boom-metres")
	start, end := -research.Value("voyager-bus-diameter-metres")*spaceHalf,
		-research.Value("voyager-bus-diameter-metres")*spaceHalf-length
	halfWidth := ask.number("boom-half-width")
	halfHeight := ask.number("boom-half-height")
	for _, y := range []float64{-halfHeight, halfHeight} {
		for _, z := range []float64{-halfWidth, halfWidth} {
			assembly.rod(maths.V(start, y, z), maths.V(end, y, z),
				ask.number("boom-rail-radius"), spaceStructure)
		}
	}
	bays := ask.count("boom-bays")
	step := length / float64(bays)
	inset := ask.number("boom-diagonal-inset")
	for bay := range bays {
		// Shorten each diagonal just inside the longitudinal rails. The
		// small visual gap avoids coincident edges between capped struts.
		back := start - float64(bay)*step - inset
		front := start - float64(bay+spaceOne)*step + inset
		for _, side := range []float64{-spaceOne, spaceOne} {
			for _, face := range []float64{-spaceOne, spaceOne} {
				rear := maths.V(back, side*halfHeight, face*halfWidth)
				forward := maths.V(front, side*halfHeight,
					-face*halfWidth)
				assembly.rod(rear, forward,
					ask.number("boom-diagonal-radius"), spaceStructure)
			}
		}
	}
	maker.instrumentHeads(assembly, ask, end, length)
}

// instrumentHeads builds the two visible instrument packages on the outboard
// end of the lattice.
func (voyagerMaker) instrumentHeads(assembly *instrumentAssembly,
	ask *asking, end, length float64,
) {
	size := ask.point("boom-head-size")
	stems := []float64{end + ask.number("boom-head-inboard"),
		end + ask.number("boom-head-outboard-share")*length}
	for index, x := range stems {
		corner := maths.V(x, ask.number("boom-head-drop"),
			ask.number("boom-head-side"))
		placeBox(assembly, size, corner, spaceSecondary)
		base := maths.V(x, ask.number("boom-head-stem-base"), spaceZero)
		top := ask.number("boom-head-stem-top") +
			float64(index)*ask.number("boom-head-stem-step")
		assembly.rod(base, maths.V(x, top, spaceZero),
			ask.number("boom-head-stem-radius"), spaceGold)
	}
}

// radioAntennae builds the two long antennae in a V, their tip fittings and
// the short stays at their hinge.
func (voyagerMaker) radioAntennae(assembly *instrumentAssembly, ask *asking) {
	length := research.Value("voyager-radio-antennas-metres")
	for _, side := range []float64{-spaceOne, spaceOne} {
		root := maths.V(ask.number("radio-root-x"),
			ask.number("radio-root-y"),
			side*ask.number("radio-root-side"))
		run := maths.V(length/math.Sqrt2, spaceZero, side*length/math.Sqrt2)
		assembly.rod(root, root.Add(run), ask.number("radio-rod-radius"),
			spaceStructure)
		tip := run.Scale(ask.number("radio-tip-start"))
		assembly.rod(root.Add(tip), root.Add(run),
			ask.number("radio-tip-radius"), spaceGold)
	}
	// Short, paired stays at the hinge make the V root look deployed.
	for _, side := range []float64{-spaceOne, spaceOne} {
		root := maths.V(ask.number("radio-stay-x"),
			ask.number("radio-stay-y"),
			side*ask.number("radio-stay-side"))
		reach := ask.number("radio-stay-reach")
		lean := maths.V(spaceZero, side*reach, side*reach)
		assembly.rod(root, root.Add(lean),
			ask.number("radio-stay-radius"), spaceStructure)
	}
}

// rtgBoom builds the generator boom with its outer rods and the generator
// cans along it.
func (maker voyagerMaker) rtgBoom(assembly *instrumentAssembly, ask *asking) {
	length := research.Value("voyager-rtg-boom-metres")
	base := ask.point("rtg-base")
	end := base.Add(maths.V(length, spaceZero, spaceZero))
	assembly.rod(base, end, ask.number("rtg-rod-radius"), spaceStructure)
	for _, side := range []float64{-spaceOne, spaceOne} {
		shift := maths.V(spaceZero, spaceZero,
			side*ask.number("rtg-outrigger-offset"))
		assembly.rod(base.Add(shift), end.Add(shift),
			ask.number("rtg-outrigger-radius"), spaceStructure)
	}
	cans := ask.count("rtg-count")
	segment := length / float64(cans)
	inset := ask.number("rtg-can-inset")
	for unit := range cans {
		startX := base.X + float64(unit)*segment + inset
		stopX := base.X + float64(unit+spaceOne)*segment - inset
		assembly.rod(maths.V(startX, base.Y, spaceZero),
			maths.V(stopX, base.Y, spaceZero),
			ask.number("rtg-can-radius"), spaceSecondary)
		maker.canFins(assembly, ask, startX, stopX, base.Y)
	}
}

// canFins builds the cooling ribs and end rings of one generator can. Their
// spacing is authored because the sources give no sizes.
func (voyagerMaker) canFins(assembly *instrumentAssembly, ask *asking,
	startX, stopX, centreY float64,
) {
	ribs := ask.count("rtg-ribs")
	spokes := ask.count("rtg-rib-spokes")
	ring := ask.number("rtg-ring-radius")
	for rib := range ribs {
		x := startX + (stopX-startX)*float64(rib+spaceOne)/
			float64(ribs+spaceOne)
		for spoke := range spokes {
			angle := fullTurn * float64(spoke) / float64(spokes)
			point := maths.V(x, centreY+ring*math.Cos(angle),
				ring*math.Sin(angle))
			fin := maths.V(ask.number("rtg-rib-length"), spaceZero,
				spaceZero)
			assembly.rod(point, point.Add(fin),
				ask.number("rtg-rib-radius"), spaceGold)
		}
	}
	endSpokes := ask.count("rtg-end-spokes")
	for _, x := range []float64{startX, stopX} {
		for spoke := range endSpokes {
			angle := fullTurn * float64(spoke) / float64(endSpokes)
			point := maths.V(x, centreY+ring*math.Cos(angle),
				ring*math.Sin(angle))
			fin := maths.V(ask.number("rtg-end-length"), spaceZero,
				spaceZero)
			assembly.rod(point, point.Add(fin),
				ask.number("rtg-end-radius"), spaceStructure)
		}
	}
}

// scienceBoom builds the science boom and the scan platform on its end.
func (maker voyagerMaker) scienceBoom(assembly *instrumentAssembly,
	ask *asking,
) {
	length := research.Value("voyager-science-boom-metres")
	base := ask.point("science-base")
	end := base.Add(maths.V(spaceZero, spaceZero, -length))
	assembly.rod(base, end, ask.number("science-rod-radius"),
		spaceStructure)
	// Movable scan platform with independent camera and spectrometer heads.
	platform := maths.V(end.X, end.Y+ask.number("platform-rise"), end.Z)
	placeBoxOn(assembly, ask, platform, "platform-size",
		"platform-corner", spaceFoil)
	maker.scanHeads(assembly, ask, platform)
	maker.platformFittings(assembly, ask, platform)
}

// scanHeads builds the camera and spectrometer heads on the platform, each a
// barrel with a lens.
func (voyagerMaker) scanHeads(assembly *instrumentAssembly, ask *asking,
	platform maths.Vec,
) {
	barrel := maths.V(spaceZero, spaceZero, -ask.number("scan-head-length"))
	lens := maths.V(spaceZero, spaceZero, -ask.number("scan-head-reach"))
	for _, mount := range ask.table("scan-heads") {
		at := platform.Add(mount)
		assembly.rod(at, at.Add(barrel), ask.number("scan-head-radius"),
			spaceSecondary)
		assembly.rod(at.Add(barrel), at.Add(lens),
			ask.number("scan-lens-radius"), spaceMirror)
	}
}

// platformFittings builds the box, the plate, the calibration bracket and
// its target on the platform.
func (voyagerMaker) platformFittings(assembly *instrumentAssembly,
	ask *asking, platform maths.Vec,
) {
	placeBoxOn(assembly, ask, platform, "scan-camera-size",
		"scan-camera-corner", spaceSecondary)
	placeBoxOn(assembly, ask, platform, "scan-plate-size",
		"scan-plate-corner", spaceGold)
	// Optical calibration target on a small outboard bracket.
	bracketStart := platform.Add(ask.point("bracket-start"))
	bracketEnd := platform.Add(ask.point("bracket-end"))
	assembly.rod(bracketStart, bracketEnd, ask.number("bracket-radius"),
		spaceStructure)
	placeBoxOn(assembly, ask, platform, "target-size", "target-corner",
		spacePanel)
}

// thrusters builds the attitude control nozzles. The published flight report
// gives their count; their placements are a chosen visual distribution.
func (voyagerMaker) thrusters(assembly *instrumentAssembly, ask *asking) {
	count := ask.count("thruster-count")
	ring := ask.number("thruster-ring-radius")
	reach := ask.number("thruster-run-radius")
	for nozzle := range count {
		angle := fullTurn * float64(nozzle) / float64(count)
		at := maths.V(ring*math.Cos(angle), ask.number("thruster-height"),
			ring*math.Sin(angle))
		run := maths.V(reach*math.Cos(angle),
			ask.number("thruster-run-drop"), reach*math.Sin(angle))
		assembly.rod(at, at.Add(run), ask.number("thruster-radius"),
			spaceGold)
	}
}
