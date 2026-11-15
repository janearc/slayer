//go:build draw

package hillside

import (
	"bytes"
	"os"
	"testing"

	"github.com/janearc/slayer/maths"
	milestone "github.com/janearc/slayer/scene/smoke"
)

func TestSequenceCameraUsesElapsedClipTimeAndReachesTheRenderer(test *testing.T) {
	path, _ := sequenceFixture(test)
	raw, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, cameraFixture...), 0600); err != nil {
		test.Fatal(err)
	}
	plan, err := readSequence(path)
	if err != nil {
		test.Fatal(err)
	}
	given := cameraScene(test)
	frames := map[string][]byte{}
	var previous []byte
	for _, name := range []string{"start", "middle", "end", "start"} {
		at := sequenceTime(test, name)
		view := given
		err := plan.withCamera(&view, at)
		if err != nil || view.eye == nil {
			test.Fatalf("camera missing at %s: %v", at, err)
		}
		wantZ := sequenceNumber(test, name+"-z")
		if view.eye.Origin != maths.V(sequenceZero, sequenceZero, wantZ) {
			test.Fatalf("scene start offset leaked into camera time: %v", view.eye.Origin)
		}
		picture, _, _, err := sequencePicture(*plan, given, at)
		if err != nil {
			test.Fatal(err)
		}
		if previous != nil && bytes.Equal(previous, picture.Pix) {
			test.Fatal("rendered camera did not move")
		}
		if earlier, exists := frames[name]; exists && !bytes.Equal(earlier, picture.Pix) {
			test.Fatal("reordered frame changed its camera")
		}
		current := append([]byte(nil), picture.Pix...)
		frames[name], previous = current, current
	}
	if given.eye != nil {
		test.Fatal("sampling changed the retained camera")
	}
}

// cameraScene retains an actual smoke world for the camera integration test.
func cameraScene(test *testing.T) asked {
	test.Helper()
	scene, found := milestone.Get(sequenceOne)
	if !found {
		test.Fatal("first milestone disappeared")
	}
	world, _ := scene.Scene()
	given := mustParse(test)
	dimension := int(sequenceNumber(test, "width"))
	given.width, given.height, given.samples = dimension, dimension, sequenceOne
	given.onGPU = false
	given.live = &liveScene{base: *world}
	return given
}
