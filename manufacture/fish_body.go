package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// A fish's body is lofted: rings across it, from the snout to the base of
// the tail, each as deep and as wide as the species' proportions put it
// there, closed at either end. Its depth rises from the snout to where the
// species is deepest, falls by a cosine to the tail stalk, and flares a
// little into the tail. A ring is a rounded ellipse; a keel is a ridge
// along the stalk's side where the species has keels.

// How the body's surface is shaped. Each is picked, to read as a fish.
const (
	fishRound      = 2.3  // picked: a ring's power; two is an ellipse
	fishFlare      = .35  // picked: the stalk's widening into the tail
	fishSpineHigh  = .12  // picked: the column's height, of a half depth
	fishEnd        = .004 // picked: the closing point beyond the tail base
	fishKeelRun    = .07  // picked: the median keel's run before the tail
	fishKeelSize   = .25  // picked: its height, of the stalk's depth
	fishKeelSpread = .2   // picked: its spread, of a half depth
	fishSmallKeel  = .08  // picked: a small keel's height, of the stalk
	fishSmallRun   = .02  // picked: a small keel's run
	fishSmallAt    = .45  // picked: its height, of a half depth
	fishHalf       = .5   // defined: the middle of the length
)

// The loft's cut, at a fish alone and at a fish of a school, seen far.
// Chosen: enough rings to bend smoothly at the joints of its column.
const (
	fishRings        = 72
	fishAround       = 40
	fishSchoolRings  = 36
	fishSchoolAround = 18
)

// fishOutline is a fish's proportions as lengths: how long it is, how deep
// and how wide at its deepest and at its stalk, where those fall, how
// blunt its snout is, and its keels.
type fishOutline struct {
	length, depth, width     float64
	stalk, stalkWidth        float64
	deepest, stalkAt, tailAt float64
	snout                    float64
	keels                    int
}

// outline is the fish's proportions, picked from its row.
func (plan fishPlan) outline() fishOutline {
	length := plan.length()
	depth := plan.figure("depth") * length
	stalk := plan.figure("stalk") * length
	return fishOutline{length: length, depth: depth,
		width: plan.figure("width") * depth, stalk: stalk,
		stalkWidth: plan.figure("stalk-width") * stalk,
		deepest:    plan.figure("deepest"), stalkAt: plan.figure("stalk-at"),
		tailAt: plan.figure("tail-base"), snout: plan.figure("snout"),
		keels: plan.whole("keels")}
}

// x is where along the fish a share of its length from the snout lies.
func (outline fishOutline) x(s float64) float64 {
	return outline.length * (fishHalf - s)
}

// shareAt is the share of the length from the snout at which x lies.
func (outline fishOutline) shareAt(x float64) float64 {
	return fishHalf - x/outline.length
}

// section is a size along the body: nothing at the snout, rising to most
// at the deepest, falling to least at the stalk, flaring into the tail.
func (outline fishOutline) section(most, least, s float64) float64 {
	switch {
	case s <= 0:
		return 0
	case s <= outline.deepest:
		u := 1 - s/outline.deepest
		return most * math.Pow(1-u*u, outline.snout)
	case s <= outline.stalkAt:
		u := (s - outline.deepest) / (outline.stalkAt - outline.deepest)
		return least + (most-least)*(1+math.Cos(math.Pi*u))/2
	}
	u := math.Min(1, (s-outline.stalkAt)/(outline.tailAt-outline.stalkAt))
	return least * (1 + fishFlare*u*u)
}

// halfDepth is half the body's depth at share s.
func (outline fishOutline) halfDepth(s float64) float64 {
	return outline.section(outline.depth, outline.stalk, s) / 2
}

// halfWidth is half the body's width at share s, keels aside.
func (outline fishOutline) halfWidth(s float64) float64 {
	return outline.section(outline.width, outline.stalkWidth, s) / 2
}

// spineY is the height of the column at share s.
func (outline fishOutline) spineY(s float64) float64 {
	return fishSpineHigh * outline.halfDepth(s)
}

// keel is how far the keels stand out from the side at share s and
// height y: the median keel along the stalk, and the two small ones above
// and below it at the tail's base, where the species has three.
func (outline fishOutline) keel(s, y float64) float64 {
	if outline.keels == 0 {
		return 0
	}
	half := math.Max(outline.halfDepth(s), outline.stalk/2)
	ridge := func(from, to, at, size float64) float64 {
		if s < from || s > to {
			return 0
		}
		spread := fishKeelSpread * half
		off := (math.Abs(y) - at) / spread
		return size * math.Sin(math.Pi*(s-from)/(to-from)) * math.Exp(-off*off)
	}
	out := ridge(outline.stalkAt-fishKeelRun, outline.tailAt, 0,
		fishKeelSize*outline.stalk)
	if outline.keels >= 3 {
		out += ridge(outline.tailAt-fishSmallRun, outline.tailAt,
			fishSmallAt*half, fishSmallKeel*outline.stalk)
	}
	return out
}

// surface is the point of the body's ring at share s and angle round it,
// the angle nought at the left side and a quarter turn at the back.
func (outline fishOutline) surface(s, angle float64) maths.Vec {
	power := 2 / fishRound
	along, up := math.Cos(angle), math.Sin(angle)
	y := outline.halfDepth(s) * math.Copysign(math.Pow(math.Abs(up), power), up)
	z := outline.halfWidth(s) * math.Copysign(math.Pow(math.Abs(along),
		power), along)
	z += math.Copysign(outline.keel(s, y), along)
	return maths.V(outline.x(s), y, z)
}

// side is how far out from the middle the left side is at share s and
// height y, keels aside: where the ring at s passes y.
func (outline fishOutline) side(s, y float64) float64 {
	half := outline.halfDepth(s)
	if half <= 0 || math.Abs(y) >= half {
		return 0
	}
	up := math.Pow(math.Abs(y)/half, fishRound/2)
	return outline.halfWidth(s) * math.Pow(1-up*up, 1/fishRound)
}

// loft is the body's surface at rest, rings across and around many points
// round each, closed by a point at the snout and one past the tail's base.
func (outline fishOutline) loft(rings, around int) ([]maths.Vec, [][3]int) {
	points := []maths.Vec{maths.V(outline.x(0), 0, 0)}
	for ring := 1; ring <= rings; ring++ {
		s := outline.tailAt * (1 - math.Cos(math.Pi*float64(ring)/
			float64(rings))) / 2
		for k := 0; k < around; k++ {
			angle := 2 * math.Pi * float64(k) / float64(around)
			points = append(points, outline.surface(s, angle))
		}
	}
	points = append(points, maths.V(outline.x(outline.tailAt+fishEnd),
		outline.spineY(outline.tailAt), 0))
	return points, fishOutward(points, fishLoftFaces(rings, around))
}

// fishLoftFaces are the triangles of a loft of rings rings of around
// points between a first and a last point.
func fishLoftFaces(rings, around int) [][3]int {
	at := func(ring, k int) int { return 1 + (ring-1)*around + k%around }
	last := 1 + rings*around
	faces := [][3]int{}
	for k := 0; k < around; k++ {
		faces = append(faces, [3]int{0, at(1, k+1), at(1, k)},
			[3]int{last, at(rings, k), at(rings, k+1)})
		for ring := 1; ring < rings; ring++ {
			faces = append(faces,
				[3]int{at(ring, k), at(ring, k+1), at(ring+1, k+1)},
				[3]int{at(ring, k), at(ring+1, k+1), at(ring+1, k)})
		}
	}
	return faces
}

// fishOutward is faces turned inside out if, together, they enclose a
// negative volume, so that a closed surface faces outward.
func fishOutward(points []maths.Vec, faces [][3]int) [][3]int {
	volume := 0.
	for _, face := range faces {
		a, b, c := points[face[0]], points[face[1]], points[face[2]]
		volume += a.Dot(b.Cross(c))
	}
	if volume >= 0 {
		return faces
	}
	turned := make([][3]int, len(faces))
	for i, face := range faces {
		turned[i] = [3]int{face[0], face[2], face[1]}
	}
	return turned
}

// fishNormals is the way a surface faces at each point: the sum of the
// faces round it, each by its area.
func fishNormals(points []maths.Vec, faces [][3]int) []maths.Vec {
	normals := make([]maths.Vec, len(points))
	for _, face := range faces {
		a, b, c := points[face[0]], points[face[1]], points[face[2]]
		facing := b.Sub(a).Cross(c.Sub(a))
		for _, corner := range face {
			normals[corner] = normals[corner].Add(facing)
		}
	}
	for i, normal := range normals {
		if normal.Len() > 0 {
			normals[i] = normal.Unit()
		}
	}
	return normals
}
