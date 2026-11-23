package manufacture

import (
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

const (
	// walrusTuskSections and walrusTuskRadial are the rings along a tusk
	// and the points round each ring.
	walrusTuskSections = 60
	walrusTuskRadial   = 32
	// walrusTuskBase and walrusTuskTip are the tusk's radius at the root,
	// in the jaw, and just before its worn end, in metres.
	walrusTuskBase = .046
	walrusTuskTip  = .012
	// walrusTuskCap is how far the worn end rounds out beyond the last
	// ring.
	walrusTuskCap = .008
)

// walrusTuskPath is the middle line of a tusk at rest, from its root deep
// in the maxilla at t equal to nought to its worn end at one: the upper
// canine of the skeleton.
func walrusTuskPath(side, t float64) maths.Vec {
	return walrusCanineOf(side).at(t)
}

// walrusTuskRadius is the tusk's radius at t along it and angle round it.
// The surface carries fine growth rings and shallow ridges along its
// length, and narrows to a worn tip.
func walrusTuskRadius(t, angle float64) float64 {
	const (
		taper      = 1.2
		ridgeDepth = .04
		ridgeCount = 6
		ridgeTwist = 2.
		ringDepth  = .015
		ringCount  = 18
		fullTurn   = 2 * math.Pi
	)
	base := walrusTuskTip + (walrusTuskBase-walrusTuskTip)*math.Pow(1-t, taper)
	ridge := 1 - ridgeDepth*(1+math.Cos(ridgeCount*angle+ridgeTwist*t))/2
	ring := 1 + ringDepth*math.Sin(fullTurn*ringCount*t)
	return base * ridge * ring
}

// walrusTusk is one tusk, a closed ring-by-ring mesh whose root sits in
// the pit cut for it in the upper jaw. The last two vertices are the middle
// of the root and the middle of the worn end. Every vertex is moved by skin
// so the tusk follows the head, and keeps its rest place so the ivory's
// stain and rings are read along the tusk as it lies at rest.
func walrusTusk(side float64, skin func(maths.Vec) maths.Vec) (*mesh.Mesh, error) {
	const sections, radial = walrusTuskSections, walrusTuskRadial
	rest := make([]maths.Vec, 0, (sections+1)*radial+2)
	fs := make([][3]int, 0, 2*sections*radial+2*radial)
	path := func(t float64) maths.Vec { return walrusTuskPath(side, t) }
	for i := 0; i <= sections; i++ {
		t := float64(i) / sections
		centre := path(t)
		across, normal := skeleton.SweepFrame(path, t, maths.V(1, 0, 0))
		for j := 0; j < radial; j++ {
			angle := 2 * math.Pi * float64(j) / radial
			r := walrusTuskRadius(t, angle)
			// normal first, then across, so the rings turn the way the
			// faces below are wound.
			ring := normal.Scale(r * math.Cos(angle)).Add(across.Scale(r * math.Sin(angle)))
			rest = append(rest, centre.Add(ring))
		}
	}
	for i := 0; i < sections; i++ {
		for j := 0; j < radial; j++ {
			a := i*radial + j
			b := i*radial + (j+1)%radial
			c := (i+1)*radial + (j+1)%radial
			d := (i+1)*radial + j
			fs = append(fs, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	root := len(rest)
	rest = append(rest, walrusTuskPath(side, 0))
	end := len(rest)
	tip := walrusTuskPath(side, 1)
	rounded := tip.Add(tip.Sub(walrusTuskPath(side, 1-1./sections)).Unit().Scale(walrusTuskCap))
	rest = append(rest, rounded)
	for j := 0; j < radial; j++ {
		k := (j + 1) % radial
		fs = append(fs, [3]int{root, k, j},
			[3]int{end, sections*radial + j, sections*radial + k})
	}
	posed := make([]maths.Vec, len(rest))
	for i, p := range rest {
		posed[i] = skin(p)
	}
	return mesh.NewWithRestPoints(posed, rest, fs, nil, false)
}

// walrusLandmarkReach is how far either side of a landmark's own place the
// hide is looked for, in metres. Chosen: wider than the eye's or a
// nostril's pit.
const walrusLandmarkReach = .10

// walrusSurfaceZ is the depth at which the hide crosses the line at x and
// y, going out on side from near to far (both distances from the middle),
// or false if the hide does not cross between them.
func walrusSurfaceZ(field func(maths.Vec) float64, x, y, side,
	near, far float64) (float64, bool) {
	const steps = 30
	if field(maths.V(x, y, side*near)) >= 0 ||
		field(maths.V(x, y, side*far)) < 0 {
		return 0, false
	}
	low, high := near, far
	for i := 0; i < steps; i++ {
		z := (low + high) / 2
		if field(maths.V(x, y, side*z)) < 0 {
			low = z
		} else {
			high = z
		}
	}
	return side * (low + high) / 2, true
}

// walrusSurfaceY is the height at which the hide crosses the line at x and
// z between bottom and top, looking down from above, or false if it does
// not cross between them.
func walrusSurfaceY(field func(maths.Vec) float64, x, z, bottom,
	top float64) (float64, bool) {
	const steps = 26
	if field(maths.V(x, top, z)) < 0 || field(maths.V(x, bottom, z)) >= 0 {
		return 0, false
	}
	low, high := bottom, top
	for i := 0; i < steps; i++ {
		y := (low + high) / 2
		if field(maths.V(x, y, z)) < 0 {
			low = y
		} else {
			high = y
		}
	}
	return (low + high) / 2, true
}

// walrusGradient is the outward direction of the hide at p.
func walrusGradient(field func(maths.Vec) float64, p maths.Vec) maths.Vec {
	const step = .004
	grad := maths.V(
		field(p.Add(maths.V(step, 0, 0)))-field(p.Add(maths.V(-step, 0, 0))),
		field(p.Add(maths.V(0, step, 0)))-field(p.Add(maths.V(0, -step, 0))),
		field(p.Add(maths.V(0, 0, step)))-field(p.Add(maths.V(0, 0, -step))))
	if grad.Len() == 0 {
		return maths.V(1, 0, 0)
	}
	return grad.Unit()
}
