package manufacture

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

func TestHatzGargoyleStandsInAnActualRoofBerth(t *testing.T) {
	ask := roofParcel
	ask.Parts = map[string]float64{"hatz-gargoyle": 1}
	made, err := MakeWithComplexity(asked(t, ask), sow(roofSeed), "hull")
	if err != nil {
		t.Fatal(err)
	}
	if len(made.native) == 0 {
		t.Fatal("superblock had berths but no gargoyle stood on a roof")
	}
	for _, part := range made.native {
		if part.Material == nil || part.Material.Name != "hatz gargoyle stone" {
			t.Fatalf("unexpected roof part material: %+v", part.Material)
		}
		if part.At(0).Origin.Y <= 0 {
			t.Fatalf("gargoyle was not placed above ground: %+v", part.At(0))
		}
	}
}

func TestHatzGargoyleFitsAndTouchesBerthFloor(t *testing.T) {
	row := superblockRow{ask: SuperblockAsk{Parts: map[string]float64{"hatz-gargoyle": 1}}}
	berth := cities.Volume{Low: [3]float64{10, 30, 20}, High: [3]float64{20, 35, 30}}
	source := rand.NewPCG(2, 3)
	part, fits, err := row.partFor(berth, rand.New(source), "hull")
	if err != nil || !fits {
		t.Fatalf("roof part: fits %v, error %v", fits, err)
	}
	box := part.Bounds()
	t.Logf("gargoyle roof bounds: %+v", box)
	if math.Abs(box.Min.Y-berth.Low[1]) > 1e-8 || box.Min.X < berth.Low[0] || box.Max.X > berth.High[0] || box.Min.Z < berth.Low[2] || box.Max.Z > berth.High[2] || box.Max.Y > berth.High[1] {
		t.Fatalf("placed box %+v does not stand within berth %+v", box, berth)
	}
}

func TestNativeRoofTrianglesRespectAllPlacements(t *testing.T) {
	points := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}
	faces := [][3]int{{0, 1, 2}}
	indexed, err := mesh.New(points, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	native, err := shape.NewMesh(points, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	parent := maths.NewSubspace(maths.V(4, 5, 6), maths.About(maths.YAxis, .7), 2)
	member := maths.NewSubspace(maths.V(-1, .3, .5), maths.About(maths.XAxis, .4), .7)
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: native, Space: member}, {Mesh: indexed, Space: member}})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		shape shape.Shape
		at    maths.Subspace
		want  maths.Subspace
		count int
	}{{indexed, parent, parent, 1}, {native, parent, parent, 1}, {group, parent, member.Within(parent), 2}} {
		got, err := nativeTriangles(item.shape, item.at)
		if err != nil || len(got) != item.count {
			t.Fatalf("%T: %d faces, %v", item.shape, len(got), err)
		}
		for _, face := range got {
			for i := range points {
				if face[i].Sub(item.want.Out(points[i])).Len() > 1e-9 {
					t.Fatalf("%T corner %d: got %v, want %v", item.shape, i, face[i], item.want.Out(points[i]))
				}
			}
		}
	}
	if _, err := nativeTriangles(shape.Sphere{Radius: 1}, parent); err == nil {
		t.Fatal("unsupported native primitive was silently omitted")
	}
	var missing *shape.Mesh
	if _, err := nativeTriangles(missing, parent); err == nil {
		t.Fatal("typed nil native mesh was accepted")
	}
	large := &shape.Mesh{Vertices: []maths.Vec{maths.V(0, 0, 0),
		maths.V(1, 0, 0), maths.V(0, 1, 0)}, Faces: make([][3]int, mostFaces+1)}
	for index := range large.Faces {
		large.Faces[index] = [3]int{0, 1, 2}
	}
	expanded, err := nativeTriangles(large, parent)
	if err != nil || len(expanded) != mostFaces+1 {
		t.Fatalf("a native mesh past the budget gave %d triangles: %v",
			len(expanded), err)
	}
}
