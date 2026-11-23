package manufacture

import (
	"math"
	"sync"

	"github.com/janearc/slayer/maths"
)

// The lattice the rest shape is cut on, in metres. The spacing is
// chosen: finer than a lip groove's width, coarse enough that the field
// is read in seconds.
const (
	orcaLatticeStep = .0125
	orcaLatticeNX   = 513 // points along the body: 6.4 m
	orcaLatticeNY   = 199 // points from belly to the tip of the fin
	orcaLatticeNZ   = 181 // points across the flukes
)

// orcaRestSkin is the animal's rest shape: the surface points as the
// field gives them, the same points with the skin's relief applied, and
// the triangles joining them.
type orcaRestSkin struct {
	points, relieved []maths.Vec
	faces            [][3]int
}

var (
	orcaRestOnce   sync.Once
	orcaRestShared *orcaRestSkin
)

// orcaRest returns the rest shape, cut from the field on first use and
// shared afterwards.
func orcaRest() *orcaRestSkin {
	orcaRestOnce.Do(func() {
		lattice := orcaLattice{
			low:  maths.V(-3.20, -.75, -1.12),
			step: orcaLatticeStep,
			nx:   orcaLatticeNX, ny: orcaLatticeNY, nz: orcaLatticeNZ,
		}
		points, faces := orcaContour(orcaField, lattice)
		normals := orcaVertexNormals(points, faces)
		relieved := make([]maths.Vec, len(points))
		for i, p := range points {
			relieved[i] = p.Add(normals[i].Scale(orcaRelief(p)))
		}
		orcaRestShared = &orcaRestSkin{points: points, relieved: relieved,
			faces: faces}
	})
	return orcaRestShared
}

// orcaVertexNormals is each point's outward direction: the sum of the
// normals of the triangles that meet there, each weighted by its area,
// then made a unit.
func orcaVertexNormals(points []maths.Vec, faces [][3]int) []maths.Vec {
	normals := make([]maths.Vec, len(points))
	for _, f := range faces {
		n := points[f[1]].Sub(points[f[0]]).Cross(points[f[2]].Sub(points[f[0]]))
		for _, v := range f {
			normals[v] = normals[v].Add(n)
		}
	}
	for i, n := range normals {
		if n.Len() > 0 {
			normals[i] = n.Unit()
		}
	}
	return normals
}

// The dermal ridges: faint lines running the length of the body with
// the flow of water over it. The count is chosen; the spacing it gives
// is wider than a real animal's, so that the mesh can carry it.
const (
	orcaRidgeCount  = 64    // ridges round the trunk
	orcaRidgeHeight = .0012 // height of a ridge, in metres
	orcaRidgeWander = 3.    // how far the lines wander, in radians of phase
)

// orcaRidge is the height of the dermal ridges at p, in metres, fading
// out where the trunk is too slender for them to be a mesh apart.
func orcaRidge(p maths.Vec) float64 {
	s := orcaTrunkProfile().at(p.X)
	angle := math.Atan2(p.Z, p.Y-s.centre)
	girth := math.Sqrt(s.high * s.wide)
	phase := orcaRidgeCount*angle + orcaRidgeWander*math.Sin(2.1*p.X+3*angle)
	patchy := .6 + .4*math.Sin(1.3*p.X+2*angle)
	return orcaRidgeHeight * patchy * orcaStep(.30, .45, girth) * math.Sin(phase)
}

// orcaRake is a set of parallel scratches left by another animal's
// teeth, on one flank, running from one end to the other.
type orcaRake struct {
	side         float64
	fromX, fromY float64
	toX, toY     float64
	lines        int
}

// orcaRakes are the rake marks: chosen places, on both flanks.
var orcaRakes = []orcaRake{
	{side: 1, fromX: .55, fromY: -.10, toX: 1.00, toY: .14, lines: 4},
	{side: -1, fromX: 1.15, fromY: .22, toX: 1.55, toY: .04, lines: 3},
	{side: 1, fromX: 1.30, fromY: .30, toX: 1.62, toY: .38, lines: 3},
	{side: -1, fromX: -.85, fromY: .05, toX: -.40, toY: -.10, lines: 4},
	{side: 1, fromX: -1.30, fromY: .12, toX: -.95, toY: .02, lines: 3},
}

// orcaScar is a healed wound on one flank, an oval of shallow scar.
type orcaScar struct {
	side         float64
	atX, atY     float64
	halfX, halfY float64
}

// orcaScars are the scars: chosen places, on both flanks.
var orcaScars = []orcaScar{
	{side: 1, atX: 1.15, atY: .0, halfX: .10, halfY: .035},
	{side: -1, atX: .25, atY: -.05, halfX: .12, halfY: .040},
	{side: 1, atX: -1.20, atY: .22, halfX: .08, halfY: .030},
	{side: -1, atX: 1.80, atY: .30, halfX: .07, halfY: .025},
}

const (
	orcaRakeGap   = .034 // distance between the scratches of a rake
	orcaRakeWidth = .011 // half width of a scratch
	orcaRakeDepth = .005 // depth of a scratch
	orcaScarDepth = .006 // depth of a scar at its middle
)

// orcaFlank is how far p lies on the side of the body, rather than on
// the back or the belly: one on the flank, zero off it.
func orcaFlank(p maths.Vec) float64 {
	s := orcaTrunkProfile().at(p.X)
	return orcaStep(.55, .80, math.Abs(p.Z)/s.wide)
}

// orcaSegmentGap is the distance in the xy plane from (x, y) to the
// segment from (ax, ay) to (bx, by), and how far along it the nearest
// point lies, from zero to one.
func orcaSegmentGap(x, y, ax, ay, bx, by float64) (gap, along float64) {
	dx, dy := bx-ax, by-ay
	along = orcaClamp01(((x-ax)*dx + (y-ay)*dy) / (dx*dx + dy*dy))
	return math.Hypot(x-(ax+along*dx), y-(ay+along*dy)), along
}

// orcaScratchTaper is how a scratch fades toward its ends: one in the
// middle, falling to zero at each end, where a tooth lifts or lands.
func orcaScratchTaper(along float64) float64 {
	return math.Sqrt(math.Sin(math.Pi * along))
}

// orcaRakeMark is how strongly the rake marks touch p: one on a
// scratch's line, falling to zero at the scratch's edge.
func orcaRakeMark(p maths.Vec) float64 {
	mark := 0.
	for _, rake := range orcaRakes {
		if rake.side*p.Z <= 0 {
			continue
		}
		long := math.Hypot(rake.toX-rake.fromX, rake.toY-rake.fromY)
		nx, ny := -(rake.toY-rake.fromY)/long, (rake.toX-rake.fromX)/long
		for line := 0; line < rake.lines; line++ {
			offset := (float64(line) - float64(rake.lines-1)/2) * orcaRakeGap
			// each scratch starts and ends a little differently
			shorten := .07 * float64((line*5)%3)
			gap, along := orcaSegmentGap(p.X, p.Y,
				rake.fromX+nx*offset+shorten*(rake.toX-rake.fromX),
				rake.fromY+ny*offset+shorten*(rake.toY-rake.fromY),
				rake.toX+nx*offset-shorten*(rake.toX-rake.fromX),
				rake.toY+ny*offset-shorten*(rake.toY-rake.fromY))
			mark = math.Max(mark,
				orcaSlit(gap, orcaRakeWidth)*orcaScratchTaper(along))
		}
	}
	return mark * orcaFlank(p)
}

// orcaScarMark is how strongly the scars touch p: one at a scar's
// middle, falling to zero at its edge.
func orcaScarMark(p maths.Vec) float64 {
	mark := 0.
	for _, scar := range orcaScars {
		if scar.side*p.Z <= 0 {
			continue
		}
		x, y := (p.X-scar.atX)/scar.halfX, (p.Y-scar.atY)/scar.halfY
		mark = math.Max(mark, orcaSlit(math.Hypot(x, y), 1))
	}
	return mark * orcaFlank(p)
}

// orcaRelief is how far the skin at rest point p stands off the smooth
// shape, along the outward normal, in metres: ridges up and down, rake
// scratches and scars sunk. It is zero away from the trunk, so the thin
// edges of the fins are left as they are.
func orcaRelief(p maths.Vec) float64 {
	onTrunk := 1 - orcaStep(0, .03, orcaTrunk(p))
	if onTrunk == 0 {
		return 0
	}
	height := orcaRidge(p) - orcaRakeDepth*orcaRakeMark(p) -
		orcaScarDepth*orcaScarMark(p)
	return onTrunk * height
}
