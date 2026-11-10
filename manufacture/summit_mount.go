package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// mount distinguishes a yoke, counterweighted axis and altitude fork.
func (row summitInstrument) mount(assembly *instrumentAssembly) {
	switch row.Mount {
	case "yoke":
		row.yoke(assembly)
	case "german":
		row.german(assembly)
	case "gimbal":
		row.gimbal(assembly)
	case "radio":
		row.radioMount(assembly)
	case "twin":
		row.twinMount(assembly)
	case "fork":
		row.ground().forkMount(assembly)
	default:
		assembly.trouble = fmt.Errorf("manufacture: unknown summit mount %q", row.Mount)
	}
}

// radioMount puts the bearings behind the bowl, within its projected width.
func (row summitInstrument) radioMount(assembly *instrumentAssembly) {
	aperture := row.aperture()
	pier := row.Seat * groundValue("pier-height")
	width := aperture * summitValue("radio-pier-width")
	assembly.box(width, pier, width,
		maths.V(-width*spaceHalf, spaceZero, -width*spaceHalf), spaceStructure)
	arm := aperture * summitValue("radio-yoke-arm")
	frame := forkFrame{width: aperture * summitValue("radio-yoke-width"),
		height: row.Seat - pier + arm, arm: arm,
		depth: aperture * summitValue("radio-yoke-depth")}
	pose := maths.NewSubspace(maths.YAxis.Scale(pier-arm),
		maths.Identity, spaceOne)
	assembly.add(frame, pose, spaceStructure)
}

// twinMount supports two outside tubes on a shared central polar head.
func (row summitInstrument) twinMount(assembly *instrumentAssembly) {
	aperture := row.aperture()
	width := aperture * groundValue("pier-width")
	pier := row.Seat * groundValue("pier-height")
	assembly.box(width, pier, width,
		maths.V(-width*spaceHalf, spaceZero, -width*spaceHalf), spaceStructure)
	start := maths.V(spaceZero, pier,
		aperture*summitValue("twin-polar-foot"))
	centre := maths.V(spaceZero,
		row.Seat-aperture*summitValue("twin-bearing-lower"),
		aperture*summitValue("twin-bearing-back"))
	girth := aperture * summitValue("twin-polar-radius")
	assembly.rod(start, centre, girth, spaceStructure)
	offset := aperture * summitValue("twin-axis-offset")
	left := centre.Add(maths.XAxis.Scale(-offset))
	right := centre.Add(maths.XAxis.Scale(offset))
	assembly.rod(left, right, girth*spaceHalf, spaceStructure)
}

// yoke gives the polar frame two supports instead of a horseshoe ring.
// The posed frame is scenery; it does not implement equatorial tracking.
func (row summitInstrument) yoke(assembly *instrumentAssembly) {
	aperture := row.aperture()
	width := aperture * summitValue("yoke-width")
	depth := aperture * summitValue("yoke-depth")
	arm := aperture * summitValue("yoke-arm")
	frame := forkFrame{width: width, height: row.Seat,
		arm: arm, depth: depth}
	assembly.add(frame, maths.Local(), spaceStructure)
	centre := maths.YAxis.Scale(row.Seat - arm*spaceHalf)
	left := centre.Add(maths.XAxis.Scale(-width * spaceHalf))
	right := centre.Add(maths.XAxis.Scale(width * spaceHalf))
	assembly.rod(left, right,
		aperture*groundValue("bearing-radius"), spaceStructure)
	row.polarBraces(assembly, width)
}

// polarBraces attach on different seats so their closed caps do not coincide.
func (row summitInstrument) polarBraces(assembly *instrumentAssembly,
	width float64,
) {
	for _, sign := range []float64{-spaceOne, spaceOne} {
		from := maths.V(sign*width*spaceHalf, row.Seat*spaceHalf,
			row.aperture()*summitValue("polar-footprint"))
		to := maths.V(sign*width*spaceHalf,
			row.Seat*summitValue("polar-attachment"), spaceZero)
		assembly.rod(from, to,
			row.aperture()*summitValue("polar-brace-radius"), spaceStructure)
	}
}

// german adds an inclined head and a counterweight opposite the tube.
func (row summitInstrument) german(assembly *instrumentAssembly) {
	aperture := row.aperture()
	width := aperture * summitValue("german-pier-width")
	assembly.box(width, row.Seat*summitValue("german-pier-height"), width,
		maths.V(-width*spaceHalf, spaceZero, -width*spaceHalf), spaceStructure)
	head := maths.YAxis.Scale(row.Seat * summitValue("german-head-height"))
	end := maths.YAxis.Scale(row.Seat)
	end.Z = -aperture * summitValue("german-polar-reach")
	girth := aperture * summitValue("german-axis-radius")
	assembly.rod(head, end, girth, spaceStructure)
	reach := -aperture * summitValue("counter-reach")
	counter := end.Add(maths.XAxis.Scale(reach))
	assembly.rod(end, counter, girth*spaceHalf, spaceStructure)
	depth := -aperture * summitValue("counter-depth")
	weightEnd := counter.Add(maths.XAxis.Scale(depth))
	assembly.rod(counter, weightEnd,
		aperture*summitValue("counter-radius"), spaceStructure)
}

// gimbal adds an outer ring to the ordinary altitude cradle.
func (row summitInstrument) gimbal(assembly *instrumentAssembly) {
	row.ground().forkMount(assembly)
	aperture := row.aperture()
	radius := aperture * summitValue("gimbal-radius")
	thick := aperture * summitValue("gimbal-thickness")
	ring := groundDrum(radius, radius-thick, thick)
	pose := maths.NewSubspace(maths.YAxis.Scale(row.Seat),
		maths.Rotation(maths.XAxis, math.Pi*spaceHalf), spaceOne)
	assembly.add(ring, pose, spaceStructure)
}
