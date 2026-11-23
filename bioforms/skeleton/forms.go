package skeleton

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Ring counts along each form, and shares of their sizes. Chosen: enough
// to read the shape at a close camera.
const (
	rodRings   = 8
	bladeRings = 8
	arcRings   = 14
	// rodGap is the share of a rod's length left open at each end, as a
	// joint's cartilage is, so that no two bones meet in one corner.
	rodGap = .02
	// bladeLeast is the least share of its width a blade keeps at its
	// rounded ends.
	bladeLeast = .15
	// arcEnds is the share of its size a curved bone keeps at its ends.
	arcEnds = .7
)

// Rod is a straight bone, round in section, its ends wider than its shaft:
// a limb bone, a digit's bone, a carpal or a sternebra. Shaft is the share
// of the radius the middle keeps.
type Rod struct {
	from, to      maths.Vec
	radius, shaft float64
}

// NewRod is a rod from from to to, radius wide at its ends.
func NewRod(from, to maths.Vec, radius, shaft float64) Rod {
	return Rod{from: from, to: to, radius: radius, shaft: shaft}
}

// Ends is where the rod starts and ends at rest.
func (rod Rod) Ends() (maths.Vec, maths.Vec) { return rod.from, rod.to }

// Middle is the rod's line, end to end.
func (rod Rod) Middle() []maths.Vec { return []maths.Vec{rod.from, rod.to} }

// Build adds the rod to the mesh, short of its ends by a joint's gap.
func (rod Rod) Build(points []maths.Vec, faces [][3]int,
	place func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	span := rod.to.Sub(rod.from)
	start := rod.from.Add(span.Scale(rodGap))
	finish := start.Add(span.Scale(1 - 2*rodGap))
	path := segmentPath(start, finish)
	half := func(t float64) (float64, float64) {
		end := math.Pow(2*t-1, 4)
		r := rod.radius * (rod.shaft + (1-rod.shaft)*end)
		return r, r
	}
	return Sweep(points, faces, rodRings, path, half, maths.V(0, 1, 0), place)
}

// Blade is a flat bone, a shoulder blade or a hip bone: thin through
// facing, widening from start times its width at from to its whole width
// at to.
type Blade struct {
	from, to         maths.Vec
	width, thickness float64
	start            float64
	facing           maths.Vec
}

// NewBlade is a blade from from to to, width wide at to and thickness
// thick, start of its width at from, its flat face toward facing.
func NewBlade(from, to maths.Vec, width, thickness, start float64,
	facing maths.Vec) Blade {
	return Blade{from: from, to: to, width: width, thickness: thickness,
		start: start, facing: facing}
}

// Middle is the blade's line, end to end.
func (blade Blade) Middle() []maths.Vec { return []maths.Vec{blade.from, blade.to} }

// Build adds the blade to the mesh.
func (blade Blade) Build(points []maths.Vec, faces [][3]int,
	place func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	half := func(t float64) (float64, float64) {
		curve := math.Max(0, 1-math.Pow(2*t-1, 8))
		round := math.Max(bladeLeast, math.Sqrt(curve))
		return blade.width * (blade.start + (1-blade.start)*t) * round,
			blade.thickness
	}
	return Sweep(points, faces, bladeRings, segmentPath(blade.from, blade.to),
		half, blade.facing, place)
}

// Arc is a bone whose middle line is the cubic curve through four points,
// a rib, a jaw's branch or a cheek arch: wide across the curve square to
// up and deep square to that, a little thinner at its ends.
type Arc struct {
	points     [4]maths.Vec
	wide, deep float64
	up         maths.Vec
}

// NewArc is the arc through points with those half sizes.
func NewArc(points [4]maths.Vec, wide, deep float64, up maths.Vec) Arc {
	return Arc{points: points, wide: wide, deep: deep, up: up}
}

// At is the point t along the arc, from nought to one.
func (arc Arc) At(t float64) maths.Vec {
	s := 1 - t
	p := arc.points
	return p[0].Scale(s * s * s).Add(p[1].Scale(3 * s * s * t)).
		Add(p[2].Scale(3 * s * t * t)).Add(p[3].Scale(t * t * t))
}

// Middle is the arc's line at five points along it.
func (arc Arc) Middle() []maths.Vec {
	const count = 5
	line := make([]maths.Vec, count)
	for i := range line {
		line[i] = arc.At(float64(i) / (count - 1))
	}
	return line
}

// Build adds the arc to the mesh.
func (arc Arc) Build(points []maths.Vec, faces [][3]int,
	place func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	half := func(t float64) (float64, float64) {
		taper := arcEnds + (1-arcEnds)*math.Sin(math.Pi*t)
		return arc.wide * taper, arc.deep * taper
	}
	return Sweep(points, faces, arcRings, arc.At, half, arc.up, place)
}

// Centrum is one bone of the backbone: a round body from head to tail, a
// neural spine standing spine high above it, and a transverse process
// reaching out reach to either side, the side being z.
type Centrum struct {
	head, tail           maths.Vec
	radius, spine, reach float64
}

// NewCentrum is a vertebra from head to tail with those sizes.
func NewCentrum(head, tail maths.Vec, radius, spine, reach float64) Centrum {
	return Centrum{head: head, tail: tail, radius: radius, spine: spine,
		reach: reach}
}

// Sizes is the vertebra's radius, neural spine and transverse reach.
func (centrum Centrum) Sizes() (radius, spine, reach float64) {
	return centrum.radius, centrum.spine, centrum.reach
}

// Middle is the centrum's line, head to tail.
func (centrum Centrum) Middle() []maths.Vec {
	return []maths.Vec{centrum.head, centrum.tail}
}

// Build adds the centrum and its processes to the mesh. The processes
// start off the centrum's middle so that no two rings lie on one another.
func (centrum Centrum) Build(points []maths.Vec, faces [][3]int,
	place func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	const centrumShaft, spineFoot, processFoot = .85, .8, .5
	length := centrum.tail.Sub(centrum.head).Len()
	points, faces = NewRod(centrum.head, centrum.tail, centrum.radius,
		centrumShaft).Build(points, faces, place)
	middle := centrum.head.Add(centrum.tail).Scale(.5)
	axis := centrum.tail.Sub(centrum.head).Unit()
	up := maths.V(0, 1, 0)
	if centrum.spine > 0 {
		base := middle.Add(up.Scale(centrum.radius * spineFoot))
		top := base.Add(up.Scale(centrum.spine))
		points, faces = Sweep(points, faces, rodRings,
			segmentPath(base, top),
			fixedHalf(centrum.radius*.22, length*.3), axis, place)
	}
	if centrum.reach <= 0 {
		return points, faces
	}
	for _, side := range []float64{-1, 1} {
		out := maths.V(0, 0, side)
		base := middle.Add(out.Scale(centrum.radius * processFoot))
		tip := middle.Add(out.Scale(centrum.radius + centrum.reach))
		points, faces = Sweep(points, faces, rodRings, segmentPath(base, tip),
			fixedHalf(length*.25, centrum.radius*.2), up, place)
	}
	return points, faces
}
