package set_test

import (
	"os"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/set"
)

func TestClusteredCityExampleUsesRealSetAndJobReaders(t *testing.T) {
	source, err := os.ReadFile("../doc/city-clustered-10000.yaml")
	if err != nil {
		t.Fatal(err)
	}
	scene, err := set.Read("../doc/city-clustered-10000.yaml", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(scene.Things()) != 1 || len(scene.Cameras()) != 3 {
		t.Fatal("clustered city example assets/cameras")
	}
	if _, err := job.Read("../doc/examples/jobs/city-clustered-10000.yaml"); err != nil {
		t.Fatal(err)
	}
}

func TestClusteredGreenAndNightExamplesUseNativeRecipeReaders(t *testing.T) {
	for _, variant := range []string{"green", "night"} {
		name := "../doc/city-clustered-" + variant + "-10000.yaml"
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := set.Read(name, source); err != nil {
			t.Fatal(err)
		}
		if _, err := job.Read("../doc/examples/jobs/city-clustered-" + variant + "-10000.yaml"); err != nil {
			t.Fatal(err)
		}
	}
}
