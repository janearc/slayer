package set_test

import (
	"os"
	"slices"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/set"
)

// The full city example crosses the real set and job readers, including the
// named camera and still output. No geometry or GPU is opened by this check.
func TestCityExampleIsAReadableSetAndStillJob(t *testing.T) {
	source, err := os.ReadFile("../doc/city-grid-10000.yaml")
	if err != nil {
		t.Fatal(err)
	}
	scene, err := set.Read("../doc/city-grid-10000.yaml", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(scene.Things()) != 1 || !slices.Contains(scene.Cameras(), "traffic circle") {
		t.Fatal("city recipe or close camera is missing")
	}
	picture, err := job.Read("../doc/examples/jobs/city-grid-10000.yaml")
	if err != nil {
		t.Fatal(err)
	}
	camera, has := picture.Camera()
	name, named := camera.Name()
	if !has || !named || name != "overview" {
		t.Fatal("overview camera did not survive the job reader")
	}
	if _, has := picture.Outputs().Picture(); !has {
		t.Fatal("city job does not produce a still")
	}
}
