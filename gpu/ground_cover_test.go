package gpu

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// A ground cover is read by slope, which the GPU's paint is not asked, so
// the GPU refuses it by name and says which renderer draws it.
func TestAGroundCoverIsRefusedAndNamesTheProcessorRenderer(t *testing.T) {
	cover := texture.NewCover(maths.Grey(0.2), 0)
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Name: "ground cover", Texture: cover}})
	err := newScene().things(world, 0)
	if err == nil || !strings.Contains(err.Error(), "ground cover") ||
		!strings.Contains(err.Error(), "--renderer cpu") {
		t.Fatalf("the GPU took a ground cover or the words are unclear: %v",
			err)
	}
}

// Anti: another painted material is not refused for being painted.
func TestAnotherPaintedMaterialIsNotRefusedAsAGroundCover(t *testing.T) {
	world := thing.NewWorld()
	dark := maths.Grey(0.2)
	light := maths.Grey(0.8)
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Name: "checked", Texture: texture.Checker(1, dark,
			light)}})
	err := newScene().things(world, 0)
	if err != nil && strings.Contains(err.Error(), "ground cover") {
		t.Fatalf("a checkered plane was refused as a ground cover: %v", err)
	}
}
