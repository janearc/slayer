package job_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestWhaleJobSnapshotsProfileAndKeys(t *testing.T) {
	dir := t.TempDir()
	_, raw, err := manufacture.EmbeddedAnimalBook("whale")
	if err != nil {
		t.Fatal(err)
	}
	scene := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: orca
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    specimen: {animal: {family: whale, file: whale.yaml, species: orca, pose: swim}}
  set:
    things:
      one: {recipe: specimen, seed: 1, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`
	jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: scene.yaml}
film: {width: 8, height: 8, rays: 1}
outputs: {picture: out.png}
`
	for name, data := range map[string][]byte{"whale.yaml": raw, "scene.yaml": []byte(scene), "job.yaml": []byte(jobText)} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	path, asset := filepath.Join(dir, "job.yaml"), filepath.Join(dir, "whale.yaml")
	first, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(raw, []byte("satin: 0.12"), []byte("satin: 0.4"), 1)
	if bytes.Equal(raw, changed) {
		t.Fatal("profile has no expected sheen")
	}
	station := []byte("[0.70, 0.610, 0.560, 0.030]")
	if !bytes.Contains(changed, station) {
		t.Fatal("profile has no expected body station")
	}
	changed = bytes.Replace(changed, station, []byte("[0.70, 0.610, 0.600, 0.030]"), 1)
	if err := os.WriteFile(asset, changed, 0600); err != nil {
		t.Fatal(err)
	}
	second, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	firstKeys, secondKeys := animalJobKeys(first), animalJobKeys(second)
	for i := range firstKeys {
		if firstKeys[i] == secondKeys[i] {
			t.Fatalf("profile edit did not change key %d", i)
		}
	}
	if whaleSatin(t, first) != .12 || whaleSatin(t, second) != .4 {
		t.Fatal("profile change was not reflected in the recipe")
	}
	firstShape, secondShape := whaleGeometry(t, first), whaleGeometry(t, second)
	if firstShape == secondShape {
		t.Fatal("body station change did not change geometry")
	}
	if err := os.Remove(asset); err != nil {
		t.Fatal(err)
	}
	if whaleSatin(t, first) != .12 || whaleSatin(t, second) != .4 {
		t.Fatal("job did not retain its frozen whale profile")
	}
	if whaleGeometry(t, first) != firstShape || whaleGeometry(t, second) != secondShape {
		t.Fatal("job did not retain its frozen whale geometry")
	}
	if _, err := job.Read(path); err == nil {
		t.Fatal("missing whale profile was accepted")
	}
}

func whaleGeometry(t *testing.T, wanted job.Job) [32]byte {
	t.Helper()
	scene, ok := wanted.ImportedSet()
	if !ok {
		t.Fatal("missing set")
	}
	recipe, ok := scene.Recipe("specimen")
	if !ok {
		t.Fatal("missing orca")
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
	skin, ok := parts[0].Shape.(*mesh.Mesh)
	if !ok {
		t.Fatalf("orca skin is %T", parts[0].Shape)
	}
	h := sha256.New()
	var raw [8]byte
	for _, point := range skin.Points() {
		for _, coordinate := range [...]float64{point.X, point.Y, point.Z} {
			binary.LittleEndian.PutUint64(raw[:], math.Float64bits(coordinate))
			_, _ = h.Write(raw[:])
		}
	}
	var digest [32]byte
	copy(digest[:], h.Sum(nil))
	return digest
}

func whaleSatin(t *testing.T, wanted job.Job) float64 {
	t.Helper()
	scene, ok := wanted.ImportedSet()
	if !ok {
		t.Fatal("missing set")
	}
	recipe, ok := scene.Recipe("specimen")
	if !ok {
		t.Fatal("missing orca")
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
	return parts[0].Material.Satin
}
