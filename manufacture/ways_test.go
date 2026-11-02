package manufacture

// the ways of the trees: what is true of a tree that looks like one, as
// tests. they know nothing of how a tree is made, only what it is, so a
// tree made any way at all is held to them. written by ninja-gaiden.

import (
	"math"
	"testing"

	"github.com/janearc/slayer/geom"
)

func unitOf(v geom.Vec3) geom.Vec3 { return v.Scale(1 / v.Length()) }

// angleBetween is the angle between two directions, in degrees, with no
// regard to which way along its line each one points
func angleBetween(a, b geom.Vec3) float64 {
	cos := math.Abs(unitOf(a).Dot(unitOf(b)))
	return math.Acos(math.Min(1, cos)) * 180 / math.Pi
}

// pieceDirections are the three ways a branch points, a piece at a time:
// it starts as the branch says, and sags a little more with each piece
func pieceDirections(b Branch) [3]geom.Vec3 {
	var found [3]geom.Vec3
	dir := b.Dir
	for piece := range found {
		found[piece] = dir
		dir = unitOf(dir.Add(geom.Vec3{0, 0, -0.22}))
	}
	return found
}

// way 1, a branch points where it points: the long edges of its bark, which
// run along the tube, lie along one of its three pieces, to within how much
// a diagonal of the tube can lean: its width across its length. a short
// branch is too thick for its pieces to tell a direction, and is not asked.
// a vertical stub at the right place fails this, as does any tube drawn
// the wrong way.
func TestWayABranchPointsWhereItPoints(t *testing.T) {
	p := DefaultProperConiferParams()
	asked := 0
	for _, whorl := range []int{0, 2, 4, 6, 9, 12, 15} {
		b := BranchOf(p, whorl, 0)
		step := b.Length / 3
		lean := math.Atan2(2*b.Radius, step)*180/math.Pi + 5
		if lean > 40 {
			continue
		}
		asked++
		wrong, checked := 0, 0
		for _, face := range branchGeometry(geom.Vec3{}, b) {
			if face.Material != barkMaterial {
				continue
			}
			for corner := range 3 {
				edge := face.V[(corner+1)%3].Sub(face.V[corner])
				if edge.Length() < 0.5*step {
					continue
				}
				checked++
				best := 90.0
				for _, along := range pieceDirections(b) {
					best = math.Min(best, angleBetween(edge, along))
				}
				if best > lean {
					wrong++
				}
			}
		}
		if checked == 0 || wrong > checked/20 {
			t.Errorf("whorl %d: %d of %d long edges are off every piece "+
				"by more than %.0f degrees", whorl, wrong, checked, lean)
		}
	}
	if asked < 3 {
		t.Errorf("only %d branches were thin enough to ask", asked)
	}
}

// way 2, the low branches droop and the high ones reach: the lowest whorl
// ends below where it left the trunk by a tenth of its length, and the
// highest ends above it by a fifth
func TestWayLowBranchesDroopAndHighOnesReach(t *testing.T) {
	p := DefaultProperConiferParams()
	reach := func(whorl int) (b Branch, down, up float64) {
		b = BranchOf(p, whorl, 0)
		for _, face := range branchGeometry(geom.Vec3{}, b) {
			for _, corner := range face.V {
				down = math.Max(down, b.Base[2]-corner[2])
				up = math.Max(up, corner[2]-b.Base[2])
			}
		}
		return b, down, up
	}
	if b, down, _ := reach(0); down < 0.1*b.Length {
		t.Errorf("the lowest branch reaches down %v of its %v", down, b.Length)
	}
	if b, _, up := reach(p.Whorls - 1); up < 0.2*b.Length {
		t.Errorf("the highest branch reaches up %v of its %v", up, b.Length)
	}
}

// way 3, no ring sits on the one below: each whorl is turned from the
// last. of the pairs of branches, one from a ring and one from the next,
// few are straight above each other: about one in a hundred by chance,
// and about one in seven if the rings were stacked
func TestWayNoWhorlSitsOnTheOneBelow(t *testing.T) {
	p := DefaultProperConiferParams()
	azimuth := func(whorl, k int) float64 {
		d := BranchOf(p, whorl, k).Dir
		return math.Atan2(d[1], d[0])
	}
	aligned, pairs := 0, 0
	for whorl := 0; whorl+1 < p.Whorls; whorl++ {
		for k := range BranchCount(p, whorl) {
			for j := range BranchCount(p, whorl+1) {
				gap := math.Mod(math.Abs(azimuth(whorl, k)-
					azimuth(whorl+1, j)), 2*math.Pi)
				pairs++
				if math.Min(gap, 2*math.Pi-gap) < 0.03 {
					aligned++
				}
			}
		}
	}
	if share := float64(aligned) / float64(pairs); share > 0.03 {
		t.Errorf("%d of %d pairs of branches are stacked: %.1f%%", aligned,
			pairs, 100*share)
	}
}

// way 4, the tree narrows as it rises: branches are long low and short
// high, which is the outline of a conifer
func TestWayTheTreeNarrowsAsItRises(t *testing.T) {
	p := DefaultProperConiferParams()
	mean := func(from, to int) float64 {
		sum, n := 0.0, 0
		for whorl := from; whorl < to; whorl++ {
			for k := range BranchCount(p, whorl) {
				sum += BranchOf(p, whorl, k).Length
				n++
			}
		}
		return sum / float64(n)
	}
	third := p.Whorls / 3
	if low, high := mean(0, third), mean(p.Whorls-third, p.Whorls); low < 2*high {
		t.Errorf("low branches %v long, high ones %v: not a cone", low, high)
	}
}

// way 5, no two trees are alike unless they have the same seed
func TestWayNoTwoSeedsMakeTheSameTree(t *testing.T) {
	one, other := DefaultProperConiferParams(), DefaultProperConiferParams()
	one.Seed, other.Seed = 1, 2
	same, again := 0, DefaultProperConiferParams()
	again.Seed = 1
	for whorl := range one.Whorls {
		for k := range BranchCount(one, whorl) {
			if BranchOf(one, whorl, k) == BranchOf(other, whorl, k) {
				same++
			}
			if BranchOf(one, whorl, k) != BranchOf(again, whorl, k) {
				t.Fatalf("seed 1 made whorl %d branch %d two ways", whorl, k)
			}
		}
	}
	if same > 0 {
		t.Errorf("%d branches are the same under two seeds", same)
	}
}

// way 6, a branch is one piece: each piece starts where the last ended, so
// nothing floats. the bark's lowest ring of the next piece is the old top.
func TestWayABranchIsOneConnectedThing(t *testing.T) {
	p := DefaultProperConiferParams()
	b := BranchOf(p, 5, 0)
	step := b.Length / 3
	var barkPoints []geom.Vec3
	for _, face := range branchGeometry(geom.Vec3{}, b) {
		if face.Material == barkMaterial {
			barkPoints = append(barkPoints, face.V[0], face.V[1], face.V[2])
		}
	}
	reach := 0.0
	for _, point := range barkPoints {
		reach = math.Max(reach, point.Sub(b.Base).Length())
	}
	if reach < 2.5*step || reach > 3.4*step {
		t.Errorf("the bark reaches %v from the trunk; a branch of length %v "+
			"should reach about its length", reach, b.Length)
	}
}
