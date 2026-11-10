package main

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/set"
)

func fireOnFixture(t *testing.T) asked {
	t.Helper()
	given := mustParse(t)
	given.set = onFixture(t, "", "")
	study, err := os.ReadFile("../../doc/fire-study.yaml")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	text = strings.Replace(text, "scene:", "$schema: "+set.EcosystemSchema+"\nscene:\n  renderer: gpu-native", 1)
	text = strings.Replace(text, "light: moon", "light: none", 1)
	text = strings.Replace(text, "kind: spruce", "recipe: wood fire", 1)
	text = strings.Replace(text, "alight: true", "alight: false", 1)
	start, end := strings.Index(string(study), "  recipes:"), strings.Index(string(study), "  set:")
	text = strings.Replace(text, "  terrain:", string(study)[start:end]+"  terrain:", 1)
	if err = os.WriteFile(given.set, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return given
}

func TestFireRecipeStagesFieldsAndOnlyTheirLighting(t *testing.T) {
	given := fireOnFixture(t)
	world, made, got, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if !got.onGPU || !got.nativeMeshes || made.things != 14 || made.burning != 1 || len(world.Volumes) != 2 {
		t.Fatal("fire recipe was not staged as a GPU-ready field")
	}
	body, ok := world.Volumes[0].Medium.(*flame.Body)
	if !ok {
		t.Fatal("staged fire has no flame body")
	}
	spec, _, _, _ := body.Samples()
	if spec.Radius != 0.45 || spec.Height != 1.5 {
		t.Fatal("tree dimensions replaced fire recipe")
	}
	if len(world.Lamps) == 0 {
		t.Fatal("fire has no illumination")
	}
	for _, lamp := range world.Lamps {
		if _, ok := lamp.(emit.Volume); !ok {
			t.Fatal("external lighting added to fire-only scene")
		}
	}
}

func TestFireRecipeDoesNotAcceptAlightASecondTime(t *testing.T) {
	given := fireOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), "alight: false", "alight: true", 1))
	if err = os.WriteFile(given.set, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err = staged(given); err == nil {
		t.Fatal("fire was set alight a second time")
	}
}

func TestFireLayoutStagesOneFlameField(t *testing.T) {
	given := fireOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	block := "      layout:\n        pattern: cluster\n        count: 7\n        radius: {value: 0.55, unit: m}\n"
	text := strings.Replace(string(raw), "      flame:", block+"      flame:", 1)
	if err = os.WriteFile(given.set, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	world, _, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Volumes) != 2 {
		t.Fatal("patches consumed separate world-volume slots")
	}
	field, ok := world.Volumes[0].Medium.(*flame.Field)
	if !ok || len(field.Pieces()) != 7 {
		t.Fatal("shared source field was not staged")
	}
}
