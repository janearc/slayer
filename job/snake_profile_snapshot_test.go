package job_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
	"gopkg.in/yaml.v3"
)

func TestSnakeJobFreezesExternalProfileAndGeometry(t *testing.T) {
	_, raw, err := manufacture.EmbeddedAnimalBook("snake")
	if err != nil {
		t.Fatal(err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	field := func(node *yaml.Node, name string) *yaml.Node {
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == name {
				return node.Content[i+1]
			}
		}
		return nil
	}
	rows := field(doc.Content[0], "profiles")
	if rows == nil {
		t.Fatal("missing profiles")
	}
	var length *yaml.Node
	for _, row := range rows.Content {
		kind := field(row, "kind")
		if kind != nil && kind.Value == "eunectes-murinus" {
			length = field(row, "length")
		}
	}
	if length == nil {
		t.Fatal("missing anaconda length")
	}
	old, err := strconv.ParseFloat(length.Value, 64)
	if err != nil {
		t.Fatal(err)
	}
	length.Value = strconv.FormatFloat(old+0.5, 'f', -1, 64)
	changed, err := yaml.Marshal(&doc)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	scene := `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: snake
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras: {view: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}}
  recipes:
    specimen: {animal: {family: snake, file: snake.yaml, species: eunectes-murinus}}
  set:
    things:
      snake: {recipe: specimen, seed: 3, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`
	jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: scene.yaml}
film: {width: 8, height: 8, rays: 1}
outputs: {picture: out.png}
`
	for name, data := range map[string][]byte{"scene.yaml": []byte(scene), "job.yaml": []byte(jobText), "snake.yaml": raw} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	path, asset := filepath.Join(dir, "job.yaml"), filepath.Join(dir, "snake.yaml")
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
		t.Fatal("length change did not reach the snake geometry")
	}
	for i := range firstKeys {
		if firstKeys[i] == secondKeys[i] {
			t.Fatalf("profile change did not change key %d", i)
		}
	}
	if err := os.Remove(asset); err != nil {
		t.Fatal(err)
	}
	if frozenAnimalGeometry(t, first) != before || frozenAnimalGeometry(t, second) != after || animalJobKeys(first) != firstKeys || animalJobKeys(second) != secondKeys {
		t.Fatal("frozen geometry or keys changed after source deletion")
	}
	if _, err := job.Read(path); err == nil {
		t.Fatal("fresh job accepted a missing snake book")
	}
}
