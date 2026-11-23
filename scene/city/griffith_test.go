package city

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestAddGriffithBasinBuildsAndPlacesSharedLandscape(t *testing.T) {
	world := thing.NewWorld()
	at := maths.V(-1300, 0, -346)
	options := ShotOptions{CityForward: -800, RidgeForward: 3000, CityDetail: .25, TerrainSamples: 257, LightScale: 6, LightFootprint: 4}
	if err := AddGriffithBasinWithOptions(world, at, 30, 0.5, 0.25, options); err != nil {
		t.Fatal(err)
	}
	if len(world.Things) < 10 || len(world.Volumes) != 0 {
		t.Fatalf("built %d surfaces and %d volumes", len(world.Things), len(world.Volumes))
	}
	if got := world.Things[0].At(0).Origin; got != at {
		t.Errorf("ridge origin = %v, want %v", got, at)
	}
	orientation := world.Things[0].At(0).Orientation
	if got := orientation.X.Cross(orientation.Y); got.Sub(orientation.Z).Len() > 1e-12 {
		t.Errorf("placement is reflected: X cross Y = %v, Z = %v", got, orientation.Z)
	}
	localNorth := maths.V(0, 0, 1)
	want := maths.V(math.Sin(math.Pi/6), 0, -math.Cos(math.Pi/6))
	if got := orientation.Out(localNorth); got.Sub(want).Len() > 1e-12 {
		t.Errorf("local city axis maps to %v, want %v", got, want)
	}
	for _, each := range world.Things {
		pose := each.At(0)
		for _, value := range []float64{pose.Origin.X, pose.Origin.Y, pose.Origin.Z} {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				t.Fatalf("invalid placed city pose: %v", pose)
			}
		}
	}
	var faces int
	for _, each := range world.Things {
		mesh, ok := each.Shape.(*shape.Mesh)
		if !ok {
			t.Fatalf("landscape shape is %T, want *shape.Mesh", each.Shape)
		}
		faces += len(mesh.Faces)
	}
	land, ok := world.Things[0].Shape.(*shape.Mesh)
	if !ok {
		t.Fatalf("coastal terrain shape is %T, want *shape.Mesh", world.Things[0].Shape)
	}
	if len(land.Faces) != 2*256*256 {
		t.Fatalf("257-sample coastal terrain has %d faces, want %d", len(land.Faces), 2*256*256)
	}
	t.Logf("shot landscape: %d grouped instances, %d total faces; coastal terrain %d faces at 257 samples", len(world.Things), faces, len(land.Faces))
}

func TestAddGriffithBasinAllowsDimNightLights(t *testing.T) {
	world := thing.NewWorld()
	options := ShotOptions{CityDetail: .25, TerrainSamples: 129,
		LightScale: 1e-7}
	if err := AddGriffithBasinWithOptions(world, maths.V(0, 0, 0), 0,
		.5, .25, options); err != nil {
		t.Fatalf("small positive night light scale was refused: %v", err)
	}
	options.LightScale = 1e-10
	if err := AddGriffithBasinWithOptions(thing.NewWorld(), maths.V(0, 0, 0),
		0, .5, .25, options); err == nil {
		t.Fatal("light scale below 1e-9 was accepted")
	}
}

func TestLegacyGriffithLandscapeKeepsFullDetailDefaults(t *testing.T) {
	world := thing.NewWorld()
	if err := AddGriffithBasinWithOptions(world, maths.V(0, 0, 0), 0, 0.5,
		0.25, ShotOptions{}); err != nil {
		t.Fatal(err)
	}
	var faces int
	for _, each := range world.Things {
		mesh, ok := each.Shape.(*shape.Mesh)
		if !ok {
			t.Fatalf("landscape shape is %T, want *shape.Mesh", each.Shape)
		}
		faces += len(mesh.Faces)
	}
	if len(world.Things) != 14 {
		t.Fatalf("full-detail shared landscape has %d grouped instances, want 14", len(world.Things))
	}
	t.Logf("default landscape: %d grouped instances, %d total faces", len(world.Things), faces)
}
