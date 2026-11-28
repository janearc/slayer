package session

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// A self-contained first scene stages a tree and a flame without terrain.
func TestLarsonCampStagesItsTreeAndFire(t *testing.T) {
	path := "../ferruccio/examples/larson-camp.yaml"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	scene, err := set.Read(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	stage := testStaging()
	stage.recipes = scene
	stage.given.exposure = 1
	if err := stage.afloat(scene); err != nil {
		t.Fatal(err)
	}
	if stage.made.burning != 1 || len(stage.fires) != 1 || len(stage.world.Volumes) == 0 {
		t.Fatal("campfire flame was not staged")
	}
	for _, name := range []string{"larson", "tree", "campfire", "flame"} {
		if _, ok := stage.places[name]; !ok {
			t.Errorf("missing placement for %s", name)
		}
	}
	world, ok := scene.World()
	if !ok || len(world.Imports()) != 1 || world.Imports()[0].Name() != "visitor" {
		t.Fatal("camp is missing its imported visitor")
	}
}

func TestAFlatSceneRefusesToIgnoreFireScale(t *testing.T) {
	path := "../ferruccio/examples/larson-camp.yaml"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(raw), "      flame:\n        recipe: flames\n", "      flame:\n        recipe: flames\n        scale: 2\n", 1)
	if changed == string(raw) {
		t.Fatal("flame fixture did not change")
	}
	scene, err := set.Read(path, []byte(changed))
	if err != nil {
		t.Fatal(err)
	}
	stage := testStaging()
	stage.recipes = scene
	stage.given.exposure = 1
	if err := stage.afloat(scene); err == nil || !strings.Contains(err.Error(), "scale is not yet supported") {
		t.Fatalf("scaled fire was not refused: %v", err)
	}
}
