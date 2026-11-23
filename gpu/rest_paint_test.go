package gpu

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// restPaintTriangle is one triangle, posed from a rest triangle a metre
// to the side when posed is true.
func restPaintTriangle(t *testing.T, posed bool) *mesh.Mesh {
	t.Helper()
	corners := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(0, 1, 0)}
	faces := [][3]int{{0, 1, 2}}
	if !posed {
		made, err := mesh.New(corners, faces, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		return made
	}
	rest := []maths.Vec{maths.V(0, 0, 1), maths.V(1, 0, 1),
		maths.V(0, 1, 1)}
	made, err := mesh.NewWithRestPoints(corners, rest, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// restPaintWorld is a world of that triangle in material.
func restPaintWorld(made *mesh.Mesh, material *thing.Material) *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: made, Material: material})
	return world
}

// restPainted is a painted material named skin.
func restPainted() *thing.Material {
	return &thing.Material{Name: "skin", Texture: texture.Checker(.1,
		maths.Grey(.1), maths.Grey(.9))}
}

func TestAPosedPaintedMeshIsRefusedForTheProcessor(t *testing.T) {
	world := restPaintWorld(restPaintTriangle(t, true), restPainted())
	err := newScene().things(world, 0)
	if err == nil || !strings.Contains(err.Error(), `"skin"`) ||
		!strings.Contains(err.Error(), "CPU-only") ||
		!strings.Contains(err.Error(), "draw it on the processor") {
		t.Fatalf("a posed painted mesh was not refused in words: %v", err)
	}
}

// Anti-tests: what the refusal must leave alone.

func TestAPosedMeshWithPlainColourStillDraws(t *testing.T) {
	plain := &thing.Material{Name: "plain", Colour: maths.Grey(.5)}
	world := restPaintWorld(restPaintTriangle(t, true), plain)
	if err := newScene().things(world, 0); err != nil {
		t.Fatalf("a posed mesh of plain colour was refused: %v", err)
	}
}

func TestAPaintedMeshAtRestStillDraws(t *testing.T) {
	world := restPaintWorld(restPaintTriangle(t, false), restPainted())
	if err := newScene().things(world, 0); err != nil {
		t.Fatalf("a painted mesh with no rest points was refused: %v", err)
	}
}
