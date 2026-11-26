package stage_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

func TestStageKeepsBoxAndSphereAuthorReferences(t *testing.T) {
	source := []byte(`scene:
  world:
    materials:
      matte: {colour: {srgb_linear: [0.3, 0.4, 0.5]}, source: eye}
    boxes:
      shared: {material: matte, middle: [10, 20, 3], size: [2, 4, 6], source: eye}
    spheres:
      shared: {material: matte, middle: [30, 40, 5], radius: 2, source: eye}
`)
	read, err := set.Read("world.yaml", source)
	if err != nil {
		t.Fatal(err)
	}
	said, ok := read.World()
	if !ok {
		t.Fatal("missing world")
	}
	world := thing.NewWorld()
	existing := world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}})
	if err := stage.Add(world, said, maths.V(1, 2, 3)); err != nil {
		t.Fatal(err)
	}
	if _, ok := world.AuthorOf(existing); ok {
		t.Fatal("unrelated existing surface gained an author")
	}
	found := map[string]bool{}
	for _, part := range world.Things[1:] {
		author, ok := world.AuthorOf(part)
		if !ok || author.Name != "shared" {
			t.Fatalf("missing authored identity: %+v %v", author, ok)
		}
		found[author.Collection] = true
		switch author.Collection {
		case "scene.world.boxes":
			if got := part.At(0).Origin; got != maths.V(11, 5, -17) {
				t.Fatalf("box shifted to %v", got)
			}
		case "scene.world.spheres":
			if got := part.At(0).Origin; got != maths.V(31, 7, -37) {
				t.Fatalf("sphere shifted to %v", got)
			}
		default:
			t.Fatalf("wrong collection %q", author.Collection)
		}
	}
	if !found["scene.world.boxes"] || !found["scene.world.spheres"] {
		t.Fatalf("found %v", found)
	}
}

func TestStageKeepsEachImportedGroupDistinct(t *testing.T) {
	world, err := stage.World(importWorld(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Things) != 16 {
		t.Fatalf("parts = %d", len(world.Things))
	}
	for i, part := range world.Things {
		author, ok := world.AuthorOf(part)
		want := "first"
		if i >= 8 {
			want = "second"
		}
		if !ok || author.Collection != "scene.world.imports" || author.Name != want {
			t.Fatalf("part %d author = %+v, %v", i, author, ok)
		}
		if i < 8 && part.Shape != world.Things[i+8].Shape {
			t.Fatal("authorship broke shared geometry")
		}
	}
}
