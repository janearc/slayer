package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// enclosure gives the instrument its own housing family.
func (row summitInstrument) enclosure(assembly *instrumentAssembly) {
	switch row.Housing {
	case "none":
		return
	case "vented-dome":
		row.ground().enclosure(assembly)
		row.ventBands(assembly)
	case "clamshell":
		row.clamshell(assembly)
	case "calotte":
		row.calotte(assembly)
	case "retracted":
		row.retracted(assembly)
	case "sliding-roof":
		row.slidingRoof(assembly)
	case "screened-house":
		row.screenedHouse(assembly)
	case "dome":
		row.ground().enclosure(assembly)
	default:
		assembly.trouble = fmt.Errorf("manufacture: unknown summit housing %q",
			row.Housing)
	}
}

// ventBands put separate closed panels on the tall cylindrical enclosure.
func (row summitInstrument) ventBands(assembly *instrumentAssembly) {
	count := int(summitValue("vent-panels"))
	radius := row.Diameter*spaceHalf + summitValue("vent-offset")
	for index := range count {
		angle := fullTurn * float64(index) / float64(count)
		orientation := maths.Rotation(maths.YAxis, math.Pi*spaceHalf-angle)
		sine, cosine := math.Sincos(angle)
		at := maths.V(radius*cosine, row.height()*summitValue("vent-seat"),
			radius*sine)
		pose := maths.NewSubspace(at, orientation, spaceOne)
		width := row.Diameter * summitValue("vent-width")
		panel := boxShape(width, row.height()*summitValue("vent-height"),
			summitValue("vent-depth"))
		assembly.add(panel, pose, summitRoof)
	}
}

// clamshell leaves a broad opening above the low base in its parked pose.
func (row summitInstrument) clamshell(assembly *instrumentAssembly) {
	radius := row.Diameter * spaceHalf
	wall := radius * groundValue("wall-share")
	base := row.height() - radius
	assembly.add(groundDrum(radius, radius-wall, base), maths.Local(), body)
	gap := groundValue("enclosure-joint-gap")
	pose := maths.NewSubspace(maths.YAxis.Scale(base+gap),
		maths.Identity, spaceOne)
	canopy := domeShell{radius: radius - gap, thick: wall,
		opening: summitValue("clamshell-opening-deg") * maths.Degree}
	assembly.add(canopy, pose, body)
}

// calotte has a circular observing pupil in a tilted spherical shell.
func (row summitInstrument) calotte(assembly *instrumentAssembly) {
	radius := row.Diameter * spaceHalf
	wall := radius * groundValue("wall-share")
	base := row.height() - radius
	assembly.add(groundDrum(radius, radius-wall, base), maths.Local(), body)
	pose := row.frame()
	pose.Origin = maths.YAxis.Scale(base)
	cap := calotteCap{radius: radius, wall: wall,
		opening: summitValue("calotte-opening-deg") * maths.Degree}
	assembly.add(cap, pose, body)
}

// retracted parks the enclosure panels low, exposing the tracking telescope.
func (row summitInstrument) retracted(assembly *instrumentAssembly) {
	width := row.Diameter
	wall := summitValue("retracted-wall")
	height := row.height() * summitValue("retracted-height")
	for _, sign := range []float64{-spaceOne, spaceOne} {
		corner := maths.V(sign*(width*spaceHalf-wall)-wall*spaceHalf,
			spaceZero, -width*spaceHalf)
		assembly.box(wall, height, width, corner, body)
	}
	for _, sign := range []float64{-spaceOne, spaceOne} {
		corner := maths.V(-width*spaceHalf, spaceZero,
			sign*(width*spaceHalf-wall)-wall*spaceHalf)
		assembly.box(width, height, wall, corner, body)
	}
}

// slidingRoof keeps the authored Mees/SOLARC housing open above its instrument.
func (row summitInstrument) slidingRoof(assembly *instrumentAssembly) {
	row.retracted(assembly)
	width := row.Diameter * summitValue("sliding-roof-width")
	depth := row.Diameter
	corner := maths.V(row.Diameter*spaceHalf,
		row.height()*summitValue("retracted-height"), -depth*spaceHalf)
	assembly.box(width, summitValue("sliding-roof-depth"), depth,
		corner, summitRoof)
}

// screenedHouse suggests JCMT's rotating enclosure with a broad front aperture.
func (row summitInstrument) screenedHouse(assembly *instrumentAssembly) {
	radius := row.Diameter * spaceHalf
	base := row.height() * summitValue("screen-base-height")
	wall := radius * groundValue("wall-share")
	assembly.add(groundDrum(radius, radius-wall, base), maths.Local(), body)
	opening := summitValue("screen-opening-deg") * maths.Degree
	frame := maths.NewSubspace(maths.YAxis.Scale(base+wall),
		maths.Identity, spaceOne)
	upper := facetedShell{base: radius, top: radius, wall: wall,
		height: row.height() - base - wall/spaceHalf, opening: opening,
		sides: int(summitValue("screen-facets"))}
	assembly.add(upper, frame, body)
	frame.Origin.Y = row.height() - wall
	assembly.add(slitCylinder{outer: radius, inner: spaceZero,
		height: wall, opening: opening}, frame, summitRoof)
}
