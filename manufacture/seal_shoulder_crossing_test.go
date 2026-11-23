package manufacture

import (
	"math"
	"sort"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// sealCrossings counts strict triangle penetrations in the two forelimbs
// and torso. Shared edges and coplanar contacts are outside this check.
func sealCrossings(rig *SealRig, skin *mesh.Mesh) int {
	points := skin.Points()
	var triangles []sealTriangle
	for _, face := range skin.Faces() {
		hind := false
		for _, index := range face {
			vertex := rig.vertices[index]
			hind = hind || (vertex.weight > 0 && vertex.limb >= 2)
		}
		if hind {
			continue
		}
		var tri sealTriangle
		tri.indices = face
		for corner, index := range face {
			tri.points[corner] = points[index]
		}
		tri.lo, tri.hi = triangleExtents(tri.points)
		triangles = append(triangles, tri)
	}
	sort.Slice(triangles, func(i, j int) bool {
		return triangles[i].lo.X < triangles[j].lo.X
	})
	crossings := 0
	for index, first := range triangles {
		for _, second := range triangles[index+1:] {
			if second.lo.X > first.hi.X {
				break
			}
			if first.crosses(second) {
				crossings++
			}
		}
	}
	return crossings
}

type sealTriangle struct {
	indices [3]int
	points  [3]maths.Vec
	lo, hi  maths.Vec
}

func triangleExtents(points [3]maths.Vec) (maths.Vec, maths.Vec) {
	a, b, c := points[0], points[1], points[2]
	highX := math.Max(a.X, math.Max(b.X, c.X))
	highY := math.Max(a.Y, math.Max(b.Y, c.Y))
	highZ := math.Max(a.Z, math.Max(b.Z, c.Z))
	return maths.V(min(a.X, b.X, c.X), min(a.Y, b.Y, c.Y), min(a.Z, b.Z, c.Z)),
		maths.V(highX, highY, highZ)
}

func (first sealTriangle) crosses(second sealTriangle) bool {
	if first.lo.Y > second.hi.Y || second.lo.Y > first.hi.Y ||
		first.lo.Z > second.hi.Z || second.lo.Z > first.hi.Z {
		return false
	}
	for _, a := range first.indices {
		for _, b := range second.indices {
			if a == b {
				return false
			}
		}
	}
	return sealTriangleEdges(first.points, second.points) ||
		sealTriangleEdges(second.points, first.points)
}

func sealTriangleEdges(edges, plane [3]maths.Vec) bool {
	for index, origin := range edges {
		end := edges[(index+1)%len(edges)]
		if sealSegmentPierces(origin, end, plane) {
			return true
		}
	}
	return false
}

func sealSegmentPierces(origin, end maths.Vec, tri [3]maths.Vec) bool {
	const tolerance = 1e-7
	direction := end.Sub(origin)
	across, up := tri[1].Sub(tri[0]), tri[2].Sub(tri[0])
	right := direction.Cross(up)
	divisor := across.Dot(right)
	if math.Abs(divisor) <= tolerance*direction.Len()*across.Len()*up.Len() {
		return false
	}
	offset := origin.Sub(tri[0])
	a := offset.Dot(right) / divisor
	cross := offset.Cross(across)
	b := direction.Dot(cross) / divisor
	t := up.Dot(cross) / divisor
	return a > tolerance && b > tolerance && a+b < 1-tolerance &&
		t > tolerance && t < 1-tolerance
}

func TestSealShoulderAvoidsCrossingsDuringReach(t *testing.T) {
	rig := NewSealRig()
	for step := -12; step <= 12; step++ {
		reach := float64(step) / 12
		pose := SealRigPose{Fore: [2]float64{1.15 * reach, 1.15 * reach}}
		skin, err := rig.Skin(pose)
		if err != nil {
			t.Fatal(err)
		}
		if crossings := sealCrossings(rig, skin); crossings != 0 {
			t.Errorf("reach %+.3f: %d crossed triangle pairs", reach, crossings)
		}
	}
}

func TestSealShoulderKeepsTheAuthoredBankingReachClear(t *testing.T) {
	for _, start := range []float64{-.16, .30, .52} {
		rig, err := NewSealRigWithFlex(start)
		if err != nil {
			t.Fatal(err)
		}
		for step := range 13 {
			reach := float64(step) / 12
			skin, err := rig.Skin(SealRigPose{
				Yaw: 1.40 * reach, Pitch: -.40 * reach,
				Fore: [2]float64{-.12 * reach, 1.02 * reach},
			})
			if err != nil {
				t.Fatal(err)
			}
			if crossings := sealCrossings(rig, skin); crossings != 0 {
				t.Errorf("start %.2f m, reach %.3f: %d crossed triangle pairs",
					start, reach, crossings)
			}
		}
	}
}
