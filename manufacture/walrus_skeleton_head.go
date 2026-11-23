package manufacture

import (
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// The tusks are the upper canines, rooted deep in the maxilla. Their
// length and the share of it outside the bone are from the Digimorph
// walrus page: "A single male tusk can achieve a length of one meter"
// (Berta and Sumich 1999) and "roughly 80% of the tusk is exposed above
// the gumline" (Nowak 1991).
const (
	// walrusTuskLength is the whole tusk, root to worn tip, in metres.
	// Chosen inside the one metre the source gives as the most.
	walrusTuskLength = .65
	// walrusTuskExposed is the share of the tusk outside the jaw.
	walrusTuskExposed = .80
	// walrusTuskStart is the angle, in radians from straight down toward
	// the snout in the skull's own frame, at which the tusk leaves its
	// root; walrusTuskCurve is how fast that angle falls along the tusk,
	// in radians a metre, curving the tusk down and back. Both chosen.
	walrusTuskStart = .80
	walrusTuskCurve = -1.15
)

// walrusSkullFrame is the skull's own frame at rest: the occipital condyle,
// the axis toward the snout, and down, square to the axis.
type walrusSkullFrame struct {
	condyle, axis, down maths.Vec
}

// walrusSkullFrameAt is the skull's frame for a condyle at the given point.
func walrusSkullFrameAt(condyle maths.Vec) walrusSkullFrame {
	axis := maths.V(math.Cos(walrusSkullTilt), -math.Sin(walrusSkullTilt), 0)
	return walrusSkullFrame{condyle: condyle, axis: axis,
		down: maths.V(axis.Y, -axis.X, 0)}
}

// at is the point t of the skull's length forward of the condyle, side
// metres to the right and drop metres down.
func (frame walrusSkullFrame) at(t, side, drop float64) maths.Vec {
	return frame.condyle.Add(frame.axis.Scale(walrusSkullLength * t)).
		Add(frame.down.Scale(drop)).Add(maths.V(0, 0, side))
}

// local is where p lies in the skull's frame: the share of the skull's
// length forward of the condyle, the metres to the right and the metres
// down. It undoes at.
func (frame walrusSkullFrame) local(p maths.Vec) (t, side, drop float64) {
	offset := p.Sub(frame.condyle)
	return offset.Dot(frame.axis) / walrusSkullLength, offset.Z,
		offset.Dot(frame.down)
}

// walrusSkull hangs the skull from the atlas, and from the skull the lower
// jaw and the two tusks.
func walrusSkull(body *walrusBones, atlas string) {
	const skullThick, jawThick, tuskThick = .10, .03, .03
	first, _ := body.bone(atlas)
	frame := walrusSkullFrameAt(first.Tail())
	body.add("skull", atlas, frame.condyle, frame.at(1, 0, 0), skullThick,
		walrusCranium{frame})
	jaw := walrusMandible{frame}
	line := jaw.Middle()
	body.add("mandible", "skull", line[0], line[1], jawThick, jaw)
	for _, side := range []float64{-1, 1} {
		canine := walrusCanine{frame: frame, side: side}
		body.add("tusk "+walrusSideName(side), "skull", canine.at(0),
			canine.at(1), tuskThick, canine)
	}
}

// walrusCranium is the skull without its lower jaw: the braincase behind,
// narrow between the eyes, then the broad, deep maxilla that holds the
// tusks, and an arch of bone under each eye.
type walrusCranium struct{ frame walrusSkullFrame }

// walrusCraniumRings is the rings along the skull.
const walrusCraniumRings = 24

// walrusCraniumProfile is the skull's half width, half height and drop
// below its axis at stations along it, from the condyle to the snout.
// Chosen: a broad braincase, a narrow waist between the orbits, a broad
// deep maxilla over the tusk roots.
var walrusCraniumProfile = [][4]float64{
	{0, .11, .08, 0}, {.2, .15, .11, 0}, {.45, .09, .08, -.01},
	{.7, .15, .12, .03}, {.88, .13, .11, .04}, {1, .08, .06, .03},
}

// Build adds the skull and its two arches to the mesh.
func (skull walrusCranium) Build(vs []maths.Vec, fs [][3]int,
	place func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	frame := skull.frame
	path := func(t float64) maths.Vec {
		return frame.at(t, 0, walrusProfileAt(t, 3))
	}
	half := func(t float64) (float64, float64) {
		return walrusProfileAt(t, 1), walrusProfileAt(t, 2)
	}
	vs, fs = skeleton.Sweep(vs, fs, walrusCraniumRings, path, half,
		maths.V(0, 1, 0), place)
	for _, side := range []float64{-1, 1} {
		arch := skeleton.NewArc([4]maths.Vec{frame.at(.70, side*.12, .05),
			frame.at(.60, side*.17, .06), frame.at(.40, side*.17, .05),
			frame.at(.30, side*.12, .03)}, .012, .02, maths.V(0, 1, 0))
		vs, fs = arch.Build(vs, fs, place)
	}
	return vs, fs
}

// Middle is the skull's axis, condyle to snout.
func (skull walrusCranium) Middle() []maths.Vec {
	return []maths.Vec{skull.frame.at(0, 0, 0), skull.frame.at(1, 0, 0)}
}

// walrusProfileAt is column of walrusCraniumProfile at t, eased between
// stations.
func walrusProfileAt(t float64, column int) float64 {
	rows := walrusCraniumProfile
	for i := 1; i < len(rows); i++ {
		if t <= rows[i][0] {
			share := maths.Smooth(rows[i-1][0], rows[i][0], t)
			return rows[i-1][column] + (rows[i][column]-rows[i-1][column])*share
		}
	}
	return rows[len(rows)-1][column]
}

// walrusMandible is the lower jaw: two branches from the jaw joints under
// the braincase, meeting at the chin between the tusks. Sizes chosen.
type walrusMandible struct{ frame walrusSkullFrame }

// Build adds the jaw to the mesh.
func (jaw walrusMandible) Build(vs []maths.Vec, fs [][3]int,
	place func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	frame := jaw.frame
	for _, side := range []float64{-1, 1} {
		branch := skeleton.NewArc([4]maths.Vec{frame.at(.22, side*.11, .10),
			frame.at(.38, side*.11, .16), frame.at(.58, side*.06, .19),
			frame.at(.72, side*.025, .17)}, .016, .035, maths.V(0, 1, 0))
		vs, fs = branch.Build(vs, fs, place)
	}
	chin := skeleton.NewRod(frame.at(.66, 0, .17), frame.at(.76, 0, .16), .03, .8)
	return chin.Build(vs, fs, place)
}

// Middle is the jaw's line, joint to chin.
func (jaw walrusMandible) Middle() []maths.Vec {
	return []maths.Vec{jaw.frame.at(.22, 0, .10), jaw.frame.at(.72, 0, .17)}
}

// walrusCanine is one tusk on the side given, laid in the skull's frame.
type walrusCanine struct {
	frame walrusSkullFrame
	side  float64
}

// walrusCanineRings is the rings along a tusk.
const walrusCanineRings = 30

// at is the middle line of the tusk at t, from its root in the maxilla at
// nought to its worn tip at one. The tusk turns at a steady rate in the
// skull's middle plane, so its line is an arc, and leans out a little and
// back in toward the tip across it.
func (canine walrusCanine) at(t float64) maths.Vec {
	const (
		rootStation, rootSide, rootDrop = .60, .10, .05
		outward, inward                 = .04, .06
	)
	frame := canine.frame
	s := walrusTuskLength * t
	turned := walrusTuskStart + walrusTuskCurve*s
	forward := (math.Cos(walrusTuskStart) - math.Cos(turned)) / walrusTuskCurve
	down := (math.Sin(turned) - math.Sin(walrusTuskStart)) / walrusTuskCurve
	root := frame.at(rootStation, canine.side*rootSide, rootDrop)
	lean := canine.side * (outward*t - inward*t*t)
	return root.Add(frame.axis.Scale(forward)).Add(frame.down.Scale(down)).
		Add(maths.V(0, 0, lean))
}

// Build adds the tusk to the mesh, its section that of the hide's tusk.
func (canine walrusCanine) Build(vs []maths.Vec, fs [][3]int,
	place func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	half := func(t float64) (float64, float64) {
		r := walrusTuskRadius(t, 0)
		return r, r
	}
	return skeleton.Sweep(vs, fs, walrusCanineRings, canine.at, half,
		maths.V(1, 0, 0), place)
}

// Middle is the tusk's line, root to tip.
func (canine walrusCanine) Middle() []maths.Vec {
	return []maths.Vec{canine.at(0), canine.at(.5), canine.at(1)}
}
