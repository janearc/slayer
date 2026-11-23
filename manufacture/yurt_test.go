package manufacture

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/scene/thing"
)

// drawnBox is a box as package cities draws one, with a material.
func drawnBox(t *testing.T, size float64, look *thing.Material) cities.Rendered {
	t.Helper()
	mesh, err := cities.Box{Width: size, Height: size, Depth: size}.Mesh("quick")
	if err != nil {
		t.Fatal(err)
	}
	return cities.Rendered{Mesh: mesh, Material: look}
}

func TestTheYurtKindIsOneGroupOfMeshesForEachOfFourMaterials(t *testing.T) {
	made := mustMake(t, "yurt", 1)
	if got := len(made.slots()); got != 4 {
		t.Errorf("%d materials, want cover, frame, door and crown", got)
	}
	reach := made.Bounds().Max
	if reach.Y < 2.7 || reach.X < 3 {
		t.Errorf("the yurt reaches only %v", reach)
	}
}

func TestAYurtNeedsASownSeed(t *testing.T) {
	if _, err := (yurtRow{}).made(seed{}); !errors.Is(err, errGood) {
		t.Errorf("an unsown seed made a yurt: %v", err)
	}
}

func TestMeshesThatCompareEqualAreShared(t *testing.T) {
	var meshes sharedMeshes
	one, err := meshes.of(drawnBox(t, 1, nil).Mesh)
	if err != nil {
		t.Fatal(err)
	}
	same, _ := meshes.of(drawnBox(t, 1, nil).Mesh)
	other, _ := meshes.of(drawnBox(t, 2, nil).Mesh)
	if one != same {
		t.Error("two equal meshes were made twice")
	}
	if one == other {
		t.Error("two different meshes were made one")
	}
}

func TestEqualMeshesOfOneMaterialAreOneSlotOfTwoMembers(t *testing.T) {
	look := plain("felt", .5, .5, .5)
	made, err := goodOfRendered([]cities.Rendered{drawnBox(t, 1, look),
		drawnBox(t, 1, look)})
	if err != nil {
		t.Fatal(err)
	}
	if len(made.slots()) != 1 || made.groupParts[0] != 2 {
		slots := len(made.slots())
		t.Errorf("%d slots, %d members", slots, made.groupParts[0])
	}
}

func TestAnythingDrawnWithoutAMaterialIsRefused(t *testing.T) {
	if _, err := goodOfRendered([]cities.Rendered{drawnBox(t, 1, nil)}); !errors.Is(err, errLook) {
		t.Errorf("a drawing with no material was made: %v", err)
	}
}
