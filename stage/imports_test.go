package stage_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

const importScene = `scene:
  world:
    materials:
      matte:
        colour: {srgb_linear: [0.32, 0.38, 0.42]}
        source: eye
    imports:
      first:
        file: two-metre-human.gltf
        height: {value: 2, unit: m}
        at: [7, 3, 5]
        materials: {material/0/human-fixture-slot: matte}
        source: "Self-authored CC0 mannequin"
        license: CC0-1.0
      second:
        file: two-metre-human.gltf
        height: {value: 2, unit: m}
        at: [10, 3, 5]
        materials: {material/0/human-fixture-slot: matte}
        source: "Same immutable source mesh"
        license: CC0-1.0
`

// importWorld reads actual source bytes and retains the checked world.
func importWorld(t *testing.T, authored ...string) set.World {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"two-metre-human.gltf", "two-metre-human.bin"} {
		data, err := os.ReadFile(filepath.Join("..", "oakland", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	source := importScene
	if len(authored) != 0 {
		source = authored[0]
	}
	read, err := set.Read(filepath.Join(dir, "set.yaml"), []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	// Staging must consume the immutable snapshot, not reopen source files.
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	return world
}

func TestImportedWorldKeepsAuthoredBeam(t *testing.T) {
	withBeam := strings.Replace(importScene, "    imports:\n",
		"    beams:\n      column:\n"+
			"        at: [1, 2, 3]\n        aim: [0, 2, 0]\n"+
			"        radius: 0.5\n        reach: 8\n"+
			"        divergence: 0.8\n        temperature: 5200\n"+
			"        strength: 4\n        source: eye\n    imports:\n", 1)
	if withBeam == importScene {
		t.Fatal("import fixture has no imports block")
	}
	world, err := stage.World(importWorld(t, withBeam))
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Things) != 16 {
		t.Fatalf("got %d imported things", len(world.Things))
	}
	for _, lamp := range world.Lamps {
		if _, ok := lamp.(*emit.Beam); ok {
			return
		}
	}
	t.Fatal("staging imports silently dropped authored beam")
}

// TestImportedMeshesShareGeometryAndUseNativeMaterials checks the native join.
func TestImportedMeshesShareGeometryAndUseNativeMaterials(t *testing.T) {
	said := importWorld(t)
	world, err := stage.World(said)
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Things) != 16 {
		t.Fatalf("got %d things, want two eight-part figures", len(world.Things))
	}
	for index, part := range world.Things[:8] {
		if _, ok := part.Shape.(*mesh.Mesh); !ok {
			t.Fatalf("non-native geometry %T", part.Shape)
		}
		if part.Shape != world.Things[index+8].Shape {
			t.Fatal("repeated import duplicated geometry")
		}
		if part.Material.Name != "matte" || part.Material.Colour != maths.RGB(.32, .38, .42) {
			t.Fatal(part.Material)
		}
		if part.Material.Glow != (maths.Light{}) || part.Material.Gloss != 0 {
			t.Fatal("invented material optics")
		}
		if difference := world.Things[index+8].At(0).Origin.Sub(part.At(0).Origin); !difference.Near(maths.V(3, 0, 0), 1e-12) {
			t.Fatal(difference)
		}
	}
}

// TestImportedMeshesRespectExistingWorldAndShift checks composition with native pieces.
func TestImportedMeshesRespectExistingWorldAndShift(t *testing.T) {
	said := importWorld(t)
	plain, err := stage.World(said)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	marker := world.Add(thing.Thing{Material: &thing.Material{Name: "existing"}})
	shift := maths.V(-2, 4, 6)
	if err := stage.Add(world, said, shift); err != nil {
		t.Fatal(err)
	}
	if world.Things[0] != marker {
		t.Fatal("replaced existing native object")
	}
	for index, part := range world.Things[1:] {
		if delta := part.At(0).Origin.Sub(plain.Things[index].At(0).Origin); !delta.Near(shift, 1e-12) {
			t.Fatal(delta)
		}
	}
	// The production renderer packages remain independent of the exchange port.
	for _, folder := range []string{"trace", "gpu", "manufacture"} {
		files, err := filepath.Glob(filepath.Join("..", folder, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), `"github.com/janearc/slayer/oakland"`) {
				t.Fatalf("core package depends on Oakland: %s", file)
			}
		}
	}
}
