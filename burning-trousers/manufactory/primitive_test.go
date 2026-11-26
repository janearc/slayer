package manufactory_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestAnalyticPrimitivesHaveFiniteInsideAndOutside(t *testing.T) {
	for _, solid := range []shape.Shape{
		shape.Sphere{Radius: 2},
		shape.Sphere{Radius: 0.021 * (2.13 / 1.8)},
		shape.Sphere{Radius: 0.026},
		shape.Tube{Radius: 2, Thick: 0.5, Height: 4},
		shape.Tube{Radius: 0, Thick: 1, Height: 2},
		shape.Tube{Radius: 2, Thick: 0, Height: 0},
	} {
		if err := checkPrimitive(solid); err != nil {
			t.Errorf("%T: %v", solid, err)
		}
	}
}

func TestInvalidPrimitiveMeasuresAreDetected(t *testing.T) {
	for _, solid := range []shape.Shape{
		shape.Sphere{Radius: 0}, shape.Sphere{Radius: math.NaN()},
		shape.Sphere{Radius: math.Inf(1)}, shape.Tube{Radius: -1, Thick: 1, Height: 1},
		shape.Tube{Radius: 1, Thick: math.NaN(), Height: 1},
		shape.Tube{Radius: 1, Thick: 1, Height: -1},
	} {
		if err := checkPrimitive(solid); err == nil {
			t.Errorf("%T %+v accepted invalid measures", solid, solid)
		}
	}
}

func TestNonfinitePrimitivePlacementIsDetected(t *testing.T) {
	box := (shape.Sphere{Radius: 1}).Bounds()
	if !finitePlacedBounds(maths.Local(), box) {
		t.Fatal("ordinary placement was refused")
	}
	if finitePlacedBounds(maths.Subspace{Origin: maths.V(math.Inf(1), 0, 0)}, box) {
		t.Fatal("infinite placement was accepted")
	}
}

func TestMixedAnalyticAndMeshSpecimensRetainBothMeshSkins(t *testing.T) {
	points := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}
	faces := [][3]int{{0, 1, 2}}
	ours, err := mesh.New(points, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	theirs, err := shape.NewMesh(points, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: ours}, {Mesh: theirs}})
	if err != nil {
		t.Fatal(err)
	}
	things := []thing.Thing{{Shape: ours}, {Shape: theirs}, {Shape: group},
		{Shape: shape.Sphere{Radius: 1}}, {Shape: shape.Tube{Radius: 1, Thick: 0.2, Height: 2}}}
	skins := readSkins(t, "mixed", 0, things)
	if len(skins) != 2 {
		t.Fatalf("got %d skins, want the two distinct mesh prototypes", len(skins))
	}
	for _, read := range skins {
		if len(read.points) != 3 || len(read.faces) != 1 || read.unclosed() == nil {
			t.Fatal("an open triangle was not retained and independently read")
		}
	}
}

func TestReferenceMeshClosureIsReadFromFaces(t *testing.T) {
	points := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(0, 1, 0), maths.V(0, 0, 1)}
	faces := [][3]int{{0, 2, 1}, {0, 1, 3}, {0, 3, 2}, {1, 2, 3}}
	object, err := shape.NewMesh(points, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	skins := readSkins(t, "tetrahedron", 0, []thing.Thing{{Shape: object}})
	if len(skins) != 1 || !skins[0].closed || skins[0].unclosed() != nil ||
		math.Abs(skins[0].volume()-1.0/6) > 1e-12 || skins[0].reversed().volume() >= 0 {
		t.Fatalf("reference mesh was not independently read as a closed tetrahedron: %+v", skins)
	}
}
