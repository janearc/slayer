package shape_test

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// The tables the tests of this file read: points and rays, the scales a
// group is put at, the faces a ray is met by, the angles a group is turned
// by, and the faces of the meshes.
const (
	aGroupWindowContractsStem       = "testdata/a-group-window-contracts-"
	aGroupWindowContractsVectorFile = aGroupWindowContractsStem + "vectors.csv"
	aGroupWindowContractsScaleFile  = aGroupWindowContractsStem + "scales.csv"
	aGroupWindowContractsPlaneFile  = aGroupWindowContractsStem + "planes.csv"
	aGroupWindowContractsAngleFile  = aGroupWindowContractsStem + "angles.csv"
	aGroupWindowContractsFaceFile   = aGroupWindowContractsStem + "faces.csv"
)

// aGroupWindowContractsPrefix starts the name of every number in
// numbers.yaml that the tests of this file choose.
const aGroupWindowContractsPrefix = "a-group-window-contracts-"

// aGroupWindowContractsPicked is the number the tests of this file chose,
// by its name in numbers.yaml less the prefix they all share.
func aGroupWindowContractsPicked(t *testing.T, name string) float64 {
	t.Helper()
	return groupPicked(t, aGroupWindowContractsPrefix+name)
}

// aGroupWindowContractsRows are the rows of a csv file whose name starts
// with prefix, in the order of the file.
func aGroupWindowContractsRows(t *testing.T, file,
	prefix string) []map[string]string {
	t.Helper()
	var chosen []map[string]string
	for _, each := range groupRows(t, file) {
		if strings.HasPrefix(each["name"], prefix) {
			chosen = append(chosen, each)
		}
	}
	return chosen
}

// aGroupWindowContractsFaces are the faces of the mesh whose rows in the
// table of faces start with prefix.
func aGroupWindowContractsFaces(t *testing.T,
	prefix string) [][groupCorners]int {
	t.Helper()
	var faces [][groupCorners]int
	for _, each := range aGroupWindowContractsRows(t,
		aGroupWindowContractsFaceFile, prefix) {
		faces = append(faces, [groupCorners]int{
			groupCellCount(t, each, "first"),
			groupCellCount(t, each, "second"),
			groupCellCount(t, each, "third")})
	}
	return faces
}

// aGroupWindowContractsSolid is two nearly touching faces that have
// different normals.
func aGroupWindowContractsSolid(t *testing.T) *mesh.Mesh {
	t.Helper()
	vectors := groupVectorsIn(t, aGroupWindowContractsVectorFile)
	points := []maths.Vec{vectors.point("pair-first"),
		vectors.point("pair-second"), vectors.point("pair-third"),
		vectors.point("pair-fourth"), vectors.point("pair-fifth"),
		vectors.point("pair-sixth")}
	faces := aGroupWindowContractsFaces(t, "pair-")
	solid, err := mesh.New(points, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return solid
}

// aGroupWindowContractsFace checks that a window about the face at z of the
// pair, set at the factor of its scale, meets that face with its own normal
// and no other, and that Hit gives the same distance.
func aGroupWindowContractsFace(t *testing.T, group *shape.MeshGroup,
	space maths.Subspace, ray maths.Ray, factor float64,
	face map[string]string) {
	t.Helper()
	height, sign := groupCell(t, face, "z"), groupCell(t, face, "sign")
	distance := (aGroupWindowContractsPicked(t, "start-height") -
		height) * factor
	epsilon := aGroupWindowContractsPicked(t, "half-window") * factor
	near, far := distance-epsilon, distance+epsilon
	got, normal, hit := group.HitNormal(ray, near, far)
	wantNormal := space.Orientation.Out(maths.ZAxis.Scale(sign))
	distanceTolerance := aGroupWindowContractsPicked(t, "distance-tolerance")
	normalTolerance := aGroupWindowContractsPicked(t, "normal-tolerance")
	if !hit || !(math.Abs(got-distance) <= distanceTolerance) ||
		!(normal.Sub(wantNormal).Len() <= normalTolerance) {
		t.Fatalf("scale %g z %g hit %t distance %g normal %v", space.Scale,
			height, hit, got, normal)
	}
	if found, ok := group.Hit(ray, near, far); !ok ||
		!(math.Abs(found-distance) <= distanceTolerance) {
		t.Fatalf("scale %g hit window lost distance: %g/%t", space.Scale,
			found, ok)
	}
}

func TestGroupPreservesFiniteRayWindowsAtEveryScale(t *testing.T) {
	// Two nearly touching faces have different normals. A ray meets both;
	// a narrow window selects exactly one, so a fresh unrestricted normal
	// search cannot accidentally select the neighbour.
	vectors := groupVectorsIn(t, aGroupWindowContractsVectorFile)
	solid := aGroupWindowContractsSolid(t)
	turn := aGroupWindowContractsPicked(t, "turn")
	for _, row := range aGroupWindowContractsRows(t,
		aGroupWindowContractsScaleFile, "window-") {
		scale := groupCell(t, row, "scale")
		factor := scale
		if factor == groupNought {
			factor = groupUnity
		}
		axis := vectors.point("placement-axis")
		space := maths.NewSubspace(vectors.point("placement-origin"),
			maths.Rotation(axis, turn), scale)
		group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: solid,
			Space: space}})
		if err != nil {
			t.Fatal(err)
		}
		origin := space.Out(vectors.point("ray-start"))
		direction := space.Orientation.Out(maths.ZAxis.Neg())
		ray := maths.Ray{At: origin, Dir: direction}
		for _, face := range groupRows(t, aGroupWindowContractsPlaneFile) {
			aGroupWindowContractsFace(t, group, space, ray, factor, face)
		}
		aGroupWindowContractsOutside(t, group, ray, factor)
	}
}

// aGroupWindowContractsOutside checks that a window before the first face
// and a window after the last meet nothing.
func aGroupWindowContractsOutside(t *testing.T, group *shape.MeshGroup,
	ray maths.Ray, factor float64) {
	t.Helper()
	beforeNear := aGroupWindowContractsPicked(t, "before-near")
	beforeFar := aGroupWindowContractsPicked(t, "before-far")
	afterNear := aGroupWindowContractsPicked(t, "after-near")
	afterFar := aGroupWindowContractsPicked(t, "after-far")
	if _, hit := group.Hit(ray, beforeNear*factor, beforeFar*factor); hit {
		t.Fatal("surface before window")
	}
	if _, hit := group.Hit(ray, afterNear*factor, afterFar*factor); hit {
		t.Fatal("surface after window")
	}
}

func TestGroupDistanceKeepsScaledNearestSurfaceAwayFromHits(t *testing.T) {
	triangle := groupTriangle(t)
	height := aGroupWindowContractsPicked(t, "nearest-height")
	offset := aGroupWindowContractsPicked(t, "nearest-offset")
	tolerance := aGroupWindowContractsPicked(t, "nearest-tolerance")
	for _, row := range aGroupWindowContractsRows(t,
		aGroupWindowContractsScaleFile, "distance-") {
		scale := groupCell(t, row, "scale")
		space := maths.Local()
		space.Scale = scale
		group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: triangle,
			Space: space}})
		if err != nil {
			t.Fatal(err)
		}
		point := maths.V(offset*scale, offset*scale, height)
		if gap := group.Distance(point); !(math.Abs(gap-height) <= tolerance) {
			t.Fatalf("scale %g distance %g", scale, gap)
		}
	}
}

// aGroupWindowContractsReach is how far a box of the tetrahedron's half
// widths, turned by turn and put at the scale of the placement, reaches
// either side of its centre, along each axis.
func aGroupWindowContractsReach(t *testing.T,
	turn maths.Orientation) maths.Vec {
	t.Helper()
	scale := aGroupWindowContractsPicked(t, "bounds-scale")
	halfX := aGroupWindowContractsPicked(t, "half-x")
	halfY := aGroupWindowContractsPicked(t, "half-y")
	halfZ := aGroupWindowContractsPicked(t, "half-z")
	return maths.V(
		scale*(halfX*math.Abs(turn.X.X)+halfY*math.Abs(turn.Y.X)+
			halfZ*math.Abs(turn.Z.X)),
		scale*(halfX*math.Abs(turn.X.Y)+halfY*math.Abs(turn.Y.Y)+
			halfZ*math.Abs(turn.Z.Y)),
		scale*(halfX*math.Abs(turn.X.Z)+halfY*math.Abs(turn.Y.Z)+
			halfZ*math.Abs(turn.Z.Z)))
}

func TestGroupBoundsAgreeWithRotatedBoxCentreAndHalfWidths(t *testing.T) {
	vectors := groupVectorsIn(t, aGroupWindowContractsVectorFile)
	points := []maths.Vec{vectors.point("tetrahedron-first"),
		vectors.point("tetrahedron-second"),
		vectors.point("tetrahedron-third"),
		vectors.point("tetrahedron-fourth")}
	faces := aGroupWindowContractsFaces(t, "tetrahedron-")
	solid, err := mesh.New(points, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	scale := aGroupWindowContractsPicked(t, "bounds-scale")
	tolerance := aGroupWindowContractsPicked(t, "bounds-tolerance")
	axes := []maths.Vec{maths.XAxis, maths.YAxis, maths.ZAxis,
		vectors.point("tetrahedron-axis")}
	for _, axis := range axes {
		for _, row := range groupRows(t, aGroupWindowContractsAngleFile) {
			angle := groupCell(t, row, "angle")
			space := maths.NewSubspace(vectors.point("tetrahedron-origin"),
				maths.Rotation(axis, angle), scale)
			group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: solid,
				Space: space}})
			if err != nil {
				t.Fatal(err)
			}
			centre := space.Out(vectors.point("tetrahedron-centre"))
			reach := aGroupWindowContractsReach(t, space.Orientation)
			want := maths.Box{Min: centre.Sub(reach), Max: centre.Add(reach)}
			got := group.Bounds()
			if !(got.Min.Sub(want.Min).Len() <= tolerance) ||
				!(got.Max.Sub(want.Max).Len() <= tolerance) {
				t.Fatalf("axis %v angle %g bounds %v, want %v", axis, angle,
					got, want)
			}
		}
	}
}

func TestGroupsAcceptFlatNormalsAndRefuseEveryNonemptySmoothNormalSet(t *testing.T) {
	vectors := groupVectorsIn(t, aGroupWindowContractsVectorFile)
	points := groupTrianglePoints(t)
	faces := groupTriangleFaces()
	normals := []maths.Vec{maths.ZAxis, maths.ZAxis, maths.ZAxis}
	far := aGroupWindowContractsPicked(t, "flat-far")
	ray := maths.Ray{At: vectors.point("flat-start"), Dir: maths.ZAxis.Neg()}
	for _, row := range []struct {
		normals []maths.Vec
		smooth  bool
	}{{normals, false}, {nil, true}} {
		reference, err := shape.NewMesh(points, faces, row.normals, row.smooth)
		if err != nil {
			t.Fatal(err)
		}
		group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: reference}})
		if err != nil {
			t.Fatalf("flat reference mesh refused: %v", err)
		}
		_, normal, hit := group.HitNormal(ray, groupNought, far)
		if !hit || normal != maths.ZAxis {
			t.Fatal("flat reference normal changed")
		}
	}
	// Mesh exposes its fields. A malformed one-normal smooth input must be
	// refused by the group's flat-only boundary before any normal is indexed.
	malformed := &shape.Mesh{Vertices: points, Faces: faces,
		Normals: []maths.Vec{maths.ZAxis}, Smooth: true}
	members := []shape.MeshMember{{Mesh: malformed}}
	if _, err := shape.NewMeshGroup(members); err == nil {
		t.Fatal("accepted nonempty smooth normals")
	}
}
