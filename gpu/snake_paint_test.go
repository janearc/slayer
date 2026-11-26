package gpu

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles/snakes"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

func gpuSnake(t *testing.T, name string) snakes.Result {
	t.Helper()
	file, raw := snakes.Embedded()
	book, err := snakes.Read(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := book.Profile(name)
	if err != nil {
		t.Fatal(err)
	}
	row, err := snakes.Build(profile)
	if err != nil {
		t.Fatal(err)
	}
	return row
}

// Two real profile bodies must be accepted as unposed painted meshes and
// retain separate species paints in the GPU scene's finishing records.
func TestSnakePaintSceneEncodingKeepsDistinctSpecies(t *testing.T) {
	world := thing.NewWorld()
	boa := gpuSnake(t, "boa-constrictor-occidentalis")
	viper := gpuSnake(t, "bitis-gabonica")
	for _, body := range []thing.Thing{boa.Parts[0], viper.Parts[0]} {
		skin, ok := body.Shape.(*mesh.Mesh)
		if !ok || skin.Posed() || body.Material.Texture == nil {
			t.Fatal("expected a real unposed painted snake mesh")
		}
		world.Add(body)
	}
	encoded := newScene()
	if err := encoded.things(world, 0); err != nil {
		t.Fatalf("static snake encoding: %v", err)
	}
	first, ok := encoded.materialOf[boa.Parts[0].Material]
	if !ok {
		t.Fatal("boa material missing")
	}
	second, ok := encoded.materialOf[viper.Parts[0].Material]
	if !ok || first == second {
		t.Fatal("viper material was lost or collapsed")
	}
	point := maths.V(0, 0, 0)
	a, b := encoded.paints[first].paint, encoded.paints[second].paint
	if a == nil || b == nil || a.At(point) != boa.Parts[0].Material.Texture.At(point) || b.At(point) != viper.Parts[0].Material.Texture.At(point) || a.At(point) == b.At(point) {
		t.Fatal("distinct checked snake paints were not retained")
	}
}

// Anti-test: adding even identical rest points must be refused for a painted
// mesh, rather than silently painting a deformed place on the GPU.
func TestSnakePaintSceneRefusesRestPointMesh(t *testing.T) {
	row := gpuSnake(t, "boa-constrictor-occidentalis")
	body := row.Parts[0]
	skin := body.Shape.(*mesh.Mesh)
	points := skin.Points()
	posed, err := mesh.NewWithRestPoints(points, points, skin.Faces(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	body.Shape = posed
	world := thing.NewWorld()
	world.Add(body)
	err = newScene().things(world, 0)
	if err == nil || !strings.Contains(err.Error(), "CPU-only") {
		t.Fatalf("posed snake paint accepted: %v", err)
	}
}
