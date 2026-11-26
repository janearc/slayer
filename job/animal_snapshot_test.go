package job_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

func TestAnimalJobFreezesExternalProfileAndChangesRenderKeys(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile("../manufacture/animals/vertebrates/fishes/fish.yaml")
	if err != nil {
		t.Fatal(err)
	}
	scene := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: fish
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    fish: {animal: {family: fish, file: fish.yaml, species: pacific-bluefin-tuna}}
  set:
    things:
      one: {recipe: fish, seed: 1, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`
	jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: scene.yaml}
film: {width: 8, height: 8, rays: 1}
outputs: {picture: out.png}
`
	for name, data := range map[string][]byte{"fish.yaml": raw, "scene.yaml": []byte(scene), "job.yaml": []byte(jobText)} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	path, asset := filepath.Join(dir, "job.yaml"), filepath.Join(dir, "fish.yaml")
	first, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	before := frozenAnimalBounds(t, first)
	old := []byte("length: {least: 1.5, most: 2.4,")
	if !bytes.Contains(raw, old) {
		t.Fatal("expected tuna length not present")
	}
	changed := bytes.Replace(raw, old, []byte("length: {least: 2.5, most: 3.4,"), 1)
	if err := os.WriteFile(asset, changed, 0600); err != nil {
		t.Fatal(err)
	}
	second, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	after := frozenAnimalBounds(t, second)
	if before == after || first.Key() == second.Key() || first.PictureKey() == second.PictureKey() || first.LightKey() == second.LightKey() {
		t.Fatal("profile change did not change geometry and keys")
	}
	if err := os.Remove(asset); err != nil {
		t.Fatal(err)
	}
	if frozenAnimalBounds(t, first) != before || frozenAnimalBounds(t, second) != after {
		t.Fatal("frozen profile changed after deletion")
	}
	if _, err := job.Read(path); err == nil {
		t.Fatal("accepted missing animal book")
	}
}

// Changes which alter each animal's anatomy must affect both construction and
// render keys, without changing either snapshot after its source disappears.
func TestAnimalJobFreezesBirdTheropodAndPinnipedGeometry(t *testing.T) {
	for _, tc := range []struct {
		family, source, species, pose, old, replacement string
	}{
		{"bird", "birds.yaml", "grey-heron", "standing", "length: {least: 0.84, most: 1.02,", "length: {least: 0.90, most: 1.08,"},
		{"theropod", "allosaurus-profiles.yaml", "allosaurus", "", "[-0.2, 1.88, 0.67, 0.49, 1]", "[-0.2, 1.88, 0.77, 0.49, 1]"},
		{"pinniped", "pinniped_profiles.yaml", "harbor-seal", "hauled", "cervical: 7", "cervical: 8"},
	} {
		t.Run(tc.family, func(t *testing.T) {
			dir := t.TempDir()
			_, raw, err := manufacture.EmbeddedAnimalBook(tc.family)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(raw, []byte(tc.old)) {
				t.Fatalf("fixture lacks %q", tc.old)
			}
			changed := bytes.Replace(raw, []byte(tc.old), []byte(tc.replacement), 1)
			pose := ""
			if tc.pose != "" {
				pose = ", pose: " + tc.pose
			}
			scene := fmt.Sprintf(`$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: animal
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    specimen: {animal: {family: %s, file: %s, species: %s%s}}
  set:
    things:
      one: {recipe: specimen, seed: 1, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`, tc.family, tc.source, tc.species, pose)
			jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: scene.yaml}
film: {width: 8, height: 8, rays: 1}
outputs: {picture: out.png}
`
			asset := filepath.Join(dir, tc.source)
			for name, data := range map[string][]byte{tc.source: raw, "scene.yaml": []byte(scene), "job.yaml": []byte(jobText)} {
				if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(dir, "job.yaml")
			first, err := job.Read(path)
			if err != nil {
				t.Fatal(err)
			}
			before, firstKeys := frozenAnimalGeometry(t, first), animalJobKeys(first)
			if err := os.WriteFile(asset, changed, 0600); err != nil {
				t.Fatal(err)
			}
			second, err := job.Read(path)
			if err != nil {
				t.Fatal(err)
			}
			after, secondKeys := frozenAnimalGeometry(t, second), animalJobKeys(second)
			if before == after {
				t.Fatal("anatomy change did not change geometry")
			}
			for i := range firstKeys {
				if firstKeys[i] == secondKeys[i] {
					t.Fatalf("key %d did not change", i)
				}
			}
			if err := os.Remove(asset); err != nil {
				t.Fatal(err)
			}
			if frozenAnimalGeometry(t, first) != before || frozenAnimalGeometry(t, second) != after {
				t.Fatal("frozen geometry changed after deletion")
			}
			if animalJobKeys(first) != firstKeys || animalJobKeys(second) != secondKeys {
				t.Fatal("frozen keys changed after deletion")
			}
			if _, err := job.Read(path); err == nil {
				t.Fatal("accepted missing animal profile")
			}
		})
	}
}

func animalJobKeys(wanted job.Job) [3]string {
	return [3]string{wanted.Key(), wanted.PictureKey(), wanted.LightKey()}
}

// frozenAnimalGeometry fingerprints authored positions and triangle topology,
// placements and analytic primitives rather than merely comparing bounds.
func frozenAnimalGeometry(t *testing.T, wanted job.Job) [32]byte {
	t.Helper()
	scene, ok := wanted.ImportedSet()
	if !ok {
		t.Fatal("animal set was not frozen")
	}
	recipe, ok := scene.Recipe("specimen")
	if !ok {
		t.Fatal("animal recipe missing")
	}
	row, err := recipe.Construction()
	if err != nil {
		t.Fatal(err)
	}
	made, err := manufacture.MakeWithComplexity(*row, manufacture.Sow(1), "quick")
	if err != nil {
		t.Fatal(err)
	}
	parts, err := made.Things(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	writeAnimalValue(t, h, int64(len(parts)))
	for _, part := range parts {
		at := part.At(0)
		writeAnimalVec(t, h, at.Origin)
		writeAnimalValue(t, h, at.Scale)
		for _, axis := range []maths.Vec{maths.XAxis, maths.YAxis, maths.ZAxis} {
			writeAnimalVec(t, h, at.Orientation.Out(axis))
		}
		switch solid := part.Shape.(type) {
		case *mesh.Mesh:
			writeAnimalValue(t, h, byte(1))
			writeAnimalMesh(t, h, solid.Points(), solid.Faces())
		case *shape.Mesh:
			writeAnimalValue(t, h, byte(2))
			writeAnimalMesh(t, h, solid.Vertices, solid.Faces)
		case shape.Sphere:
			writeAnimalValue(t, h, byte(3))
			writeAnimalValue(t, h, solid.Radius)
		case shape.Tube:
			writeAnimalValue(t, h, byte(4))
			writeAnimalValue(t, h, [3]float64{solid.Radius, solid.Thick, solid.Height})
		default:
			t.Fatalf("unhandled animal shape %T", part.Shape)
		}
	}
	var digest [32]byte
	copy(digest[:], h.Sum(nil))
	return digest
}

func writeAnimalVec(t *testing.T, h hash.Hash, p maths.Vec) {
	t.Helper()
	writeAnimalValue(t, h, [3]float64{p.X, p.Y, p.Z})
}

func writeAnimalMesh(t *testing.T, h hash.Hash, points []maths.Vec, faces [][3]int) {
	t.Helper()
	writeAnimalValue(t, h, int64(len(points)))
	for _, point := range points {
		writeAnimalVec(t, h, point)
	}
	writeAnimalValue(t, h, int64(len(faces)))
	for _, face := range faces {
		writeAnimalValue(t, h, [3]int64{int64(face[0]), int64(face[1]), int64(face[2])})
	}
}

func writeAnimalValue(t *testing.T, h hash.Hash, value any) {
	t.Helper()
	if err := binary.Write(h, binary.LittleEndian, value); err != nil {
		t.Fatal(err)
	}
}

func frozenAnimalBounds(t *testing.T, wanted job.Job) maths.Box {
	t.Helper()
	scene, ok := wanted.ImportedSet()
	if !ok {
		t.Fatal("animal set was not frozen")
	}
	recipe, ok := scene.Recipe("fish")
	if !ok {
		t.Fatal("animal recipe missing")
	}
	row, err := recipe.Construction()
	if err != nil {
		t.Fatal(err)
	}
	made, err := manufacture.MakeWithComplexity(*row, manufacture.Sow(1), "quick")
	if err != nil {
		t.Fatal(err)
	}
	return made.Bounds()
}
