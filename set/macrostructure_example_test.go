package set_test

import (
	"os"
	"slices"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/set"
)

func TestMacrostructureExamplePassesTheSetAndStillJobReaders(t *testing.T) {
	source, err := os.ReadFile("../doc/macrostructures.yaml")
	if err != nil {
		t.Fatal(err)
	}
	scene, err := set.Read("../doc/macrostructures.yaml", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(scene.Things()) != 5 || !slices.Contains(scene.Cameras(),
		"volcano") || !slices.Contains(scene.Cameras(), "arcology") {
		t.Fatal("missing example recipes/cameras")
	}
	picture, err := job.Read("../doc/examples/jobs/macrostructures.yaml")
	if err != nil {
		t.Fatal(err)
	}
	camera, has := picture.Camera()
	name, named := camera.Name()
	if !has || !named || name != "overview" {
		t.Fatal("macrostructure named camera")
	}
	if _, has := picture.Outputs().Picture(); !has {
		t.Fatal("macrostructure still output")
	}
}
