package set_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

const importedSet = `scene:
  name: imported visitor
  film: {width: 32, height: 24, rays: 1, wide: 45, exposure: 1}
  cameras:
    view:
      location: {at: [0, 5, 1]}
      orientation: {look_at: [0, 0, 1]}
  world:
    unit: 2
    materials:
      matte:
        colour: {srgb_linear: [0.32, 0.38, 0.42]}
        source: eye
    imports:
      visitor:
        file: two-metre-human.gltf
        height: {value: 2, unit: m}
        at: [7, 3, 5]
        heading: {value: 90, unit: deg}
        materials: {material/0/human-fixture-slot: matte}
        source: "Self-authored CC0 mannequin"
        license: CC0-1.0
`

// importedPath copies legal source fixtures beside a set in an isolated root.
func importedPath(t *testing.T) string {
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
	return filepath.Join(dir, "set.yaml")
}

// TestImportedGeometryKeepsNativeBindingsAndMapPlacement exercises the set boundary.
func TestImportedGeometryKeepsNativeBindingsAndMapPlacement(t *testing.T) {
	read, err := set.Read(importedPath(t), []byte(importedSet))
	if err != nil {
		t.Fatal(err)
	}
	world, has := read.World()
	if !has || len(world.Imports()) != 1 {
		t.Fatal("missing imported visitor")
	}
	part := world.Imports()[0]
	if part.Name() != "visitor" || part.Asset() == nil {
		t.Fatal("missing checked asset")
	}
	if len(part.Asset().SourceClosure()) != 2 {
		t.Fatal("source closure must include glTF and buffer")
	}
	bindings := part.Bindings()
	if bindings["material/0/human-fixture-slot"] != "matte" {
		t.Fatal(bindings)
	}
	bindings["material/0/human-fixture-slot"] = "changed"
	if part.Bindings()["material/0/human-fixture-slot"] != "matte" {
		t.Fatal("bindings are mutable")
	}
	pose := part.Placement(world.Unit())
	if pose.Origin != maths.V(7, 5, -3) || pose.Scale != .5 {
		t.Fatal(pose)
	}
	// A local metre forward (-z) turns east at heading 90, and is half a world unit.
	if got := pose.Out(maths.V(0, 0, -1)); got.Sub(maths.V(7.5, 5, -3)).Len() > 1e-12 {
		t.Fatal(got)
	}
	world.Imports()[0] = set.Import{}
	if world.Imports()[0].Name() != "visitor" {
		t.Fatal("imports slice is mutable")
	}
}

// TestImportedGeometryRefusesIncompleteNativeBindings checks useful diagnostics.
func TestImportedGeometryRefusesIncompleteNativeBindings(t *testing.T) {
	for index, change := range []struct{ before, after, diagnostic string }{
		{"material/0/human-fixture-slot: matte", "material/0/human-fixture-slot: absent", "absent"},
		{"material/0/human-fixture-slot: matte", "wrong-slot: matte", "material/0/human-fixture-slot"},
		{"file: two-metre-human.gltf", "file: ../outside.gltf", "escapes source root"},
		{"value: 2, unit: m", "value: 0, unit: m", "number from 5e-324"},
		{"at: [7, 3, 5]", "at: [.nan, 3, 5]", "at"},
	} {
		t.Run(fmt.Sprintf("case-%d", index), func(t *testing.T) {
			text := strings.Replace(importedSet, change.before, change.after, 1)
			_, err := set.Read(importedPath(t), []byte(text))
			diagnostic := ""
			if err != nil {
				_, diagnostic, _ = strings.Cut(err.Error(), "set.yaml:")
			}
			if err == nil || !strings.Contains(diagnostic, change.diagnostic) {
				t.Fatalf("wanted %q refusal, got %v", change.diagnostic, err)
			}
		})
	}
}

// TestImportedGeometryIsInBothPublishedSetSchemas keeps editor and reader aligned.
func TestImportedGeometryIsInBothPublishedSetSchemas(t *testing.T) {
	for _, version := range []string{"v1", "v2"} {
		t.Run(version, func(t *testing.T) {
			text := "$schema: https://github.com/janearc/slayer/schemas/ecosystem-" + version + ".json\n" + importedSet
			if _, err := set.Read(importedPath(t), []byte(text)); err != nil {
				t.Fatal(err)
			}
		})
	}
}
