package workspace

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

func TestImportHeightEditReopensThroughSlayer(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"two-metre-human.gltf", "two-metre-human.bin"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "oakland", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	source := `# keep this comment
"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: imported visitor
  film: {width: 32, height: 24, rays: 1, wide: 45, exposure: 1}
  cameras:
    view:
      location: {at: [0, -5, 2]}
      orientation: {look_at: [0, 0, 1]}
  world:
    materials:
      matte: {colour: {srgb_linear: [0.3, 0.4, 0.5]}, source: eye}
    imports:
      visitor:
        file: two-metre-human.gltf
        height: {value: 2, unit: m} # keep the height comment
        at: [0, 0, 0]
        materials: {material/0/human-fixture-slot: matte}
        source: eye
        license: CC0-1.0
`
	path := filepath.Join(dir, "set.yaml")
	document, err := draft.Open(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if err := document.SetImportHeight("scene.world.imports/visitor", 3); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(document.Text(), "keep the height comment") || !strings.Contains(document.Text(), "keep this comment") {
		t.Fatal("comment lost")
	}
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := set.Read(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	world, ok := reopened.World()
	if !ok || len(world.Imports()) != 1 {
		t.Fatal("import disappeared")
	}
	normalization, ok := world.Imports()[0].Asset().HeightNormalization()
	if !ok || normalization.TargetHeight() != 3 || normalization.HeightUnit() != "m" {
		t.Fatalf("normalization = %+v %v", normalization, ok)
	}
	staged, err := stage.World(world)
	if err != nil {
		t.Fatal(err)
	}
	low, high := math.Inf(1), math.Inf(-1)
	for _, part := range staged.Things {
		shape, ok := part.Shape.(*mesh.Mesh)
		if !ok {
			t.Fatalf("unexpected staged shape %T", part.Shape)
		}
		bounds := shape.Bounds()
		for bits := 0; bits < 8; bits++ {
			point := bounds.Min
			if bits&1 != 0 {
				point.X = bounds.Max.X
			}
			if bits&2 != 0 {
				point.Y = bounds.Max.Y
			}
			if bits&4 != 0 {
				point.Z = bounds.Max.Z
			}
			y := part.At(0).Out(point).Y
			low, high = math.Min(low, y), math.Max(high, y)
		}
	}
	if math.Abs(high-low-3) > 1e-6 {
		t.Fatalf("staged geometry height = %v, want 3 m", high-low)
	}
}
