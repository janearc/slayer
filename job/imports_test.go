package job_test

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/job"
)

// importJobFiles writes a real asset and the two documents that consume it.
func importJobFiles(t *testing.T) (string, string) {
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
	setText := `scene:
  world:
    materials:
      matte:
        colour: {srgb_linear: [0.3, 0.4, 0.5]}
        source: eye
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
outputs: {picture: picture.png}
`
	for name, data := range map[string]string{"set.yaml": setText, "job.yaml": jobText} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, "job.yaml"), filepath.Join(dir, "two-metre-human.bin")
}

// TestImportedJobIdentityIncludesExactBufferBytes catches stale picture/resume keys.
func TestImportedJobIdentityIncludesExactBufferBytes(t *testing.T) {
	path, buffer := importJobFiles(t)
	first, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	before := first.Canonical()
	read, has := first.ImportedSet()
	if !has || len(read.ImportSources()) != 2 {
		t.Fatal("job did not freeze the source closure")
	}
	original, err := os.ReadFile(buffer)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Clone(original)
	// Narrow a complete box within its declared bounds, preserving topology.
	for offset := 0; offset < 288; offset += 12 {
		x := math.Float32frombits(binary.LittleEndian.Uint32(changed[offset:]))
		binary.LittleEndian.PutUint32(changed[offset:], math.Float32bits(x*.75))
	}
	if err := os.WriteFile(buffer, changed, 0600); err != nil {
		t.Fatal(err)
	}
	second, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if first.Key() == second.Key() || first.PictureKey() == second.PictureKey() || first.LightKey() == second.LightKey() {
		t.Fatal("changed external buffer retained a render key")
	}
	if !bytes.Equal(first.Canonical(), before) {
		t.Fatal("existing job changed its immutable snapshot")
	}
	if err := os.Remove(buffer); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Canonical(), before) {
		t.Fatal("job reread deleted input under old key")
	}
	if _, err := job.Read(path); err == nil {
		t.Fatal("fresh job accepted missing imported bytes")
	}
}

// TestImportedJobIdentityIncludesNativeMaterialAndNormalization prevents appearance reuse.
func TestImportedJobIdentityIncludesNativeMaterialAndNormalization(t *testing.T) {
	path, _ := importJobFiles(t)
	first, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	setPath := filepath.Join(filepath.Dir(path), "set.yaml")
	original, err := os.ReadFile(setPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{"[0.3, 0.4, 0.5]", "[0.5, 0.4, 0.3]"}, {"value: 2, unit: m", "value: 3, unit: m"}} {
		changed := bytes.Replace(original, []byte(pair[0]), []byte(pair[1]), 1)
		if bytes.Equal(changed, original) {
			t.Fatal("mutation did not change the fixture")
		}
		if err := os.WriteFile(setPath, changed, 0600); err != nil {
			t.Fatal(err)
		}
		next, err := job.Read(path)
		if err != nil {
			t.Fatal(err)
		}
		if first.PictureKey() == next.PictureKey() {
			t.Fatalf("key ignored %s", pair[0])
		}
	}
}
