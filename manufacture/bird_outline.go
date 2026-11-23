package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// The bird's outline: the trunk, the neck and the head are one closed
// skin lofted through a table of stations, each a centre and an
// elliptical section, eased between, so the outline is one smooth
// profile from the side and from above.
//
// Each ring of the skin is held by the bones under it: the trunk's by
// the synsacrum, the neck's by the cervicals nearest it, each falling
// from one at a vertebra's middle to nothing a vertebra and a half
// away, the head's by the skull. So the skin bends smoothly at a joint.
//
// The skin is painted by its rest point: the back above a line along
// the trunk, the belly below, the neck, and the crown on the head.

// birdStation is one section: where it lies, as a share along its part,
// its half width and half height as shares of the part's half sizes,
// and for the trunk its centre's depth below the spine.
type birdStation struct{ along, wide, high, centre float64 }

// birdTrunkStations are the trunk's sections, from the vent to the
// breast, as shares of the trunk's length, half width and half depth;
// picked: no measured outline of a wader was reachable tonight.
var birdTrunkStations = []birdStation{
	{along: -.02, wide: .15, high: .15, centre: .15},
	{along: .10, wide: .55, high: .60, centre: .35},
	{along: .25, wide: .85, high: .90, centre: .48},
	{along: .45, wide: 1, high: 1, centre: .52},
	{along: .65, wide: .95, high: .95, centre: .50},
	{along: .80, wide: .75, high: .80, centre: .42},
	{along: .92, wide: .45, high: .55, centre: .28},
}

// birdHeadStations are the head's sections, from behind the condyle to
// the bill's base, as shares of the head's length and of the
// braincase's half sizes times birdHeadShare; picked.
var birdHeadStations = []birdStation{
	{along: .02, wide: .55, high: .55},
	{along: .15, wide: .90, high: 1},
	{along: .45, wide: 1, high: .95},
	{along: .75, wide: .80, high: .72},
	{along: 1.02, wide: .45, high: .42},
}

// Rings of the loft, and how the skin is held; picked.
const (
	birdOutlineSides = 48 // points round each ring
	birdTrunkRings   = 72
	birdJoinRings    = 16 // eased from the breast into the neck
	birdNeckRings    = 96
	birdHeadRings    = 28
	birdNeckClear    = 1.0 // where the neck's skin begins, of the trunk
	birdBackLine     = .70 // the back's edge, of the depth below the spine
	birdHoldSpan     = 1.5 // a vertebra's hold, in its own lengths
	birdHeadJoin     = .2  // the head's share over which C1 still holds
)

// birdRing is one ring of the loft at rest: its centre, its half width
// and half height, and the bones that hold it.
type birdRing struct {
	centre     maths.Vec
	wide, high float64
	held       []skeleton.Influence
}

// birdStationAt is the station table eased to share along by a smooth
// curve through every station (Catmull-Rom), so the outline has no
// crease at a station; held at its ends.
func birdStationAt(stations []birdStation, along float64) birdStation {
	if along <= stations[0].along {
		return stations[0]
	}
	last := len(stations) - 1
	for i := 1; i <= last; i++ {
		a, b := stations[i-1], stations[i]
		if along <= b.along {
			t := (along - a.along) / (b.along - a.along)
			p, q := stations[max(i-2, 0)], stations[min(i+1, last)]
			return birdStation{along: along,
				wide:   birdCurve(p.wide, a.wide, b.wide, q.wide, t),
				high:   birdCurve(p.high, a.high, b.high, q.high, t),
				centre: birdCurve(p.centre, a.centre, b.centre, q.centre, t)}
		}
	}
	return stations[last]
}

// birdCurve is the Catmull-Rom curve through b and c at t, its slopes
// set by a and d.
func birdCurve(a, b, c, d, t float64) float64 {
	return b + t*(c-a)/2 + t*t*(a-2.5*b+2*c-.5*d) +
		t*t*t*(1.5*(b-c)+(d-a)/2)
}

// holding is the influence of the bone named name by weight.
func (body *birdBody) holding(name string, weight float64) skeleton.Influence {
	at, _ := body.bones.Find(name)
	return skeleton.NewInfluence(at, weight)
}

// outlineRings are the rings of the trunk, the join, the neck and the
// head in order, from the vent to the bill's base.
func (body *birdBody) outlineRings() []birdRing {
	m := body.measures
	rings := []birdRing{}
	first, last := birdTrunkStations[0].along,
		birdTrunkStations[len(birdTrunkStations)-1].along
	trunk := []skeleton.Influence{body.holding("synsacrum", 1)}
	for i := 0; i < birdTrunkRings; i++ {
		along := first + (last-first)*float64(i)/(birdTrunkRings-1)
		s := birdStationAt(birdTrunkStations, along)
		rings = append(rings, birdRing{centre: body.at(along, s.centre, 0),
			wide: s.wide * m.width / 2, high: s.high * m.depth / 2,
			held: trunk})
	}
	neck := body.neckRings()
	rings = append(rings, birdJoined(rings[len(rings)-1], neck[0])...)
	rings = append(rings, neck...)
	return append(rings, body.headRings()...)
}

// birdJoined are the rings eased from the breast's last ring into the
// neck's first, smoothly in place, size and hold.
func birdJoined(from, to birdRing) []birdRing {
	rings := []birdRing{}
	for i := 1; i <= birdJoinRings; i++ {
		t := float64(i) / (birdJoinRings + 1)
		t = t * t * (3 - 2*t)
		held := []skeleton.Influence{}
		for _, each := range from.held {
			held = append(held, skeleton.NewInfluence(each.Bone(),
				each.Weight()*(1-t)))
		}
		for _, each := range to.held {
			held = append(held, skeleton.NewInfluence(each.Bone(),
				each.Weight()*t))
		}
		rings = append(rings, birdRing{
			centre: from.centre.Add(to.centre.Sub(from.centre).Scale(t)),
			wide:   from.wide + (to.wide-from.wide)*t,
			high:   from.high + (to.high-from.high)*t, held: held})
	}
	return rings
}

// neckRings are the neck's rings along its line at rest, from where it
// clears the breast to the skull, thinning toward the head.
func (body *birdBody) neckRings() []birdRing {
	m := body.measures
	root, atlas := body.neckRoot(), body.atlasEnd()
	run := atlas.Sub(root)
	clear := (birdNeckClear*m.trunk - root.X) / run.X
	base := birdNeckBase * m.width / 2
	top := birdNeckTop * birdSkullWide * m.head
	rings := []birdRing{}
	for i := 0; i < birdNeckRings; i++ {
		share := clear + (1-clear)*float64(i)/birdNeckRings
		radius := base + (top-base)*share
		rings = append(rings, birdRing{centre: root.Add(run.Scale(share)),
			wide: radius, high: 1.1 * radius, held: body.neckHeld(share)})
	}
	return rings
}

// neckHeld are the bones that hold the neck's skin share of the way from
// the chest to the skull: each cervical by how near its middle is, the
// trunk before the first and the skull after the last.
func (body *birdBody) neckHeld(share float64) []skeleton.Influence {
	n := body.measures.cervicals
	span := birdHoldSpan / float64(n)
	held := []skeleton.Influence{}
	tent := func(name string, middle float64) {
		if weight := 1 - math.Abs(share-middle)/span; weight > 0 {
			held = append(held, body.holding(name, weight))
		}
	}
	tent("synsacrum", -.5/float64(n))
	for j := 0; j < n; j++ {
		tent(fmt.Sprintf("C%d", n-j), (float64(j)+.5)/float64(n))
	}
	tent("cranium", 1+.5/float64(n))
	return held
}

// headRings are the head's rings over the braincase to the bill.
func (body *birdBody) headRings() []birdRing {
	m := body.measures
	condyle := body.atlasEnd()
	wide := birdHeadShare * birdSkullWide * m.head
	high := birdHeadShare * birdSkullDeep * m.head
	lift := birdUp.Scale(birdSkullRaise * birdSkullDeep * m.head)
	first, last := birdHeadStations[0].along,
		birdHeadStations[len(birdHeadStations)-1].along
	rings := []birdRing{}
	for i := 0; i < birdHeadRings; i++ {
		along := first + (last-first)*float64(i)/(birdHeadRings-1)
		s := birdStationAt(birdHeadStations, along)
		centre := condyle.Add(birdAlong.Scale(along * m.head)).Add(lift)
		held := []skeleton.Influence{body.holding("cranium", 1)}
		if along < birdHeadJoin {
			held = append(held, body.holding("C1", 1-along/birdHeadJoin))
		}
		rings = append(rings, birdRing{centre: centre, wide: s.wide * wide,
			high: s.high * high, held: held})
	}
	return rings
}

// outlineRest is the loft at rest: the points of every ring, then the
// two ends' middles, the bones that hold each point, and the faces
// joining them.
func (body *birdBody) outlineRest() ([]maths.Vec, [][]skeleton.Influence,
	[][3]int) {
	rings := body.outlineRings()
	points := []maths.Vec{}
	held := [][]skeleton.Influence{}
	for i, ring := range rings {
		before, after := rings[max(i-1, 0)], rings[min(i+1, len(rings)-1)]
		along := after.centre.Sub(before.centre).Unit()
		side := along.Cross(birdUp).Unit()
		normal := side.Cross(along).Unit()
		for k := 0; k < birdOutlineSides; k++ {
			angle := 2 * math.Pi * float64(k) / birdOutlineSides
			points = append(points, ring.centre.
				Add(side.Scale(ring.wide*math.Cos(angle))).
				Add(normal.Scale(ring.high*math.Sin(angle))))
			held = append(held, ring.held)
		}
	}
	ends := rings[len(rings)-1]
	points = append(points, rings[0].centre, ends.centre)
	held = append(held, rings[0].held, ends.held)
	return points, held, birdLoftFaces(len(rings))
}

// birdLoftFaces are the faces of a loft of count rings of
// birdOutlineSides points, and a fan at each end to its middle.
func birdLoftFaces(count int) [][3]int {
	n := birdOutlineSides
	faces := [][3]int{}
	for r := 0; r+1 < count; r++ {
		for k := 0; k < n; k++ {
			a, b := r*n+k, r*n+(k+1)%n
			faces = append(faces, [3]int{a, a + n, b}, [3]int{b, a + n, b + n})
		}
	}
	back, front := count*n, count*n+1
	last := (count - 1) * n
	for k := 0; k < n; k++ {
		faces = append(faces, [3]int{back, k, (k + 1) % n},
			[3]int{front, last + (k+1)%n, last + k})
	}
	return faces
}

// birdVertebraOf is the region of a vertebra's name, T, Ca or C, and
// whether it is one.
func birdVertebraOf(name string) (string, bool) {
	for _, region := range []string{"Ca", "C", "T"} {
		rest := name[min(len(region), len(name)):]
		if len(name) > len(region) && name[:len(region)] == region &&
			rest[0] >= '0' && rest[0] <= '9' {
			return region, true
		}
	}
	return "", false
}

// outlineThing is the outline carried by frames: one closed skin,
// painted by rest point.
func (body *birdBody) outlineThing(frames []skeleton.Frame,
	at maths.Subspace) (thing.Thing, []maths.Vec, error) {
	rest, held, faces := body.outlineRest()
	posed := make([]maths.Vec, len(rest))
	for i, p := range rest {
		posed[i] = skeleton.Carry(p, frames, held[i])
	}
	skin, err := mesh.NewWithRestPoints(posed, rest, faces,
		orcaVertexNormals(posed, faces), true)
	if err != nil {
		return thing.Thing{}, nil, err
	}
	return thing.Thing{Shape: skin, Material: &thing.Material{
		Name:    body.species.Name + " plumage",
		Texture: body.paint(), Colour: body.colour("belly")},
		Motion: thing.Still(at)}, posed, nil
}

// colour is a region's colour from the row.
func (body *birdBody) colour(region string) maths.Light {
	c := body.species.Plumage[region].Colour
	return maths.RGB(c[0], c[1], c[2])
}

// birdPaint colours the outline by its rest point.
type birdPaint struct {
	back, belly, neck, crown maths.Light
	neckFrom, headFrom, line float64
}

// paint is the outline's paint for this bird.
func (body *birdBody) paint() birdPaint {
	m := body.measures
	return birdPaint{back: body.colour("back"), belly: body.colour("belly"),
		neck: body.colour("neck"), crown: body.colour("crown"),
		neckFrom: birdNeckClear * m.trunk,
		headFrom: body.atlasEnd().X - .1*m.head, line: -birdBackLine * m.depth}
}

// At is the colour of the outline at rest point p.
func (paint birdPaint) At(p maths.Vec) maths.Light {
	switch {
	case p.X >= paint.headFrom:
		return paint.crown
	case p.X >= paint.neckFrom:
		return paint.neck
	case p.Y >= paint.line:
		return paint.back
	}
	return paint.belly
}
