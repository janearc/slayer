package shape

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// The files of rows the cached normal test reads.
const (
	aGroupCachedNormalContractsRowsFile = "testdata/" +
		"a-group-cached-normal-contracts-rows.csv"
	aGroupCachedNormalContractsFacesFile = "testdata/" +
		"a-group-cached-normal-contracts-faces.csv"
)

// aGroupCachedNormalContractsSolid is two faces: the z plane the control
// ray meets first, and a plane at x of delta that it meets at a graze.
func aGroupCachedNormalContractsSolid(t *testing.T,
	delta float64) *mesh.Mesh {
	t.Helper()
	faces := triangleFaces(t, aGroupCachedNormalContractsFacesFile, "planes")
	solid, err := mesh.New([]maths.Vec{
		maths.V(-pair, -pair, nought), maths.V(pair, -pair, nought),
		maths.V(nought, pair, nought),
		maths.V(delta, -pair, -unity), maths.V(delta, pair, -unity),
		maths.V(delta, nought, unity),
	}, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return solid
}

// The ray meets z=0 while nearly parallel to a nearby x plane. Converting its
// world hit point back to local coordinates loses enough precision to make a
// new nearest-face search choose the x plane. Retain the actual ray hit.
func TestGroupNormalsKeepTheRayFaceAcrossCoordinateRounding(t *testing.T) {
	shareY := picked(t, "a-group-cached-normal-contracts-origin-y-share")
	shareZ := picked(t, "a-group-cached-normal-contracts-origin-z-share")
	height := picked(t, "a-group-cached-normal-contracts-ray-height")
	near := picked(t, "a-group-cached-normal-contracts-near")
	far := picked(t, "a-group-cached-normal-contracts-far")
	tolerance := picked(t, "a-group-cached-normal-contracts-tolerance")
	for _, row := range readSheet(t, aGroupCachedNormalContractsRowsFile) {
		scale, magnitude := cell(t, row, "scale"), cell(t, row, "magnitude")
		angle, delta := cell(t, row, "angle"), cell(t, row, "delta")
		solid := aGroupCachedNormalContractsSolid(t, delta)
		origin := maths.V(magnitude, shareY*magnitude, -shareZ*magnitude)
		space := maths.NewSubspace(origin,
			maths.About(maths.YAxis, angle), scale)
		group, err := NewMeshGroup([]MeshMember{{Mesh: solid, Space: space}})
		if err != nil {
			t.Fatal(err)
		}
		way := space.Orientation.Out(maths.ZAxis.Neg()).Unit()
		start := space.Out(maths.V(nought, nought, height))
		ray := triangleRay(start, way)
		unit := scale
		if unit == nought {
			unit = unity
		}
		met, hit := solid.Meet(space.RayIn(ray), near, far)
		if !hit || met.Face() != nought {
			t.Fatal("control ray does not meet the z plane first")
		}
		distance, normal, hit := group.HitNormal(ray, near*unit, far*unit)
		want := maths.V(math.Sin(angle), nought, math.Cos(angle))
		if !hit || !(distance > near*unit && distance < far*unit) ||
			!(normal.Sub(want).Len() <= tolerance) {
			t.Fatalf("scale %g: distance %.17g normal %v hit %t, "+
				"want z-plane normal %v", scale, distance, normal, hit, want)
		}
	}
}
