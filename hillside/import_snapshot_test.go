package hillside

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/job"
)

// TestImportedSessionStagesTheJobsFrozenBytes guards the key-to-geometry boundary.
func TestImportedSessionStagesTheJobsFrozenBytes(t *testing.T) {
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
	setText := `scene:
  film: {width: 32, height: 24, rays: 1, wide: 45, exposure: 1}
  cameras:
    view:
      location: {at: [0, 5, 1]}
      orientation: {look_at: [0, 0, 1]}
  world:
    materials:
      matte: {colour: {srgb_linear: [0.3, 0.4, 0.5]}, source: eye}
    imports:
      human:
        file: two-metre-human.gltf
        height: {value: 2, unit: m}
        at: [0, 0, 0]
        materials: {material/0/human-fixture-slot: matte}
        source: "Self-authored CC0 mannequin"
        license: CC0-1.0
`
	jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: set.yaml}
film: {width: 32, height: 24, rays: 1}
renderer: cpu
outputs: {picture: picture.png}
`
	for name, text := range map[string]string{"set.yaml": setText, "job.yaml": jobText} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	wanted, err := job.Read(filepath.Join(dir, "job.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	key := wanted.PictureKey()
	// Removing all input files proves that Open cannot silently read changed bytes.
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	session, err := Open(context.Background(), wanted, Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	source, ok := session.source.(stillFramer)
	if !ok || len(source.world.Things) != 8 {
		t.Fatal("session did not stage frozen native geometry")
	}
	if wanted.PictureKey() != key {
		t.Fatal("staging changed the job identity")
	}
}
