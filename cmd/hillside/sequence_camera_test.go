package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

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

func TestSequenceSchemaRejectsCameraTyposUnitsAndConflictingTiming(test *testing.T) {
	path, _ := sequenceFixture(test)
	raw, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	for _, change := range []struct{ before, after string }{
		{"unit: m\n", "unit: feet\n"},
		{"unit: m/s", "unit: km/h"},
		{"speed:", "speeed:"},
		{"value: 2", "value: 0"},
		{"speed: {value: 2, unit: m/s}", "speed: {value: 2, unit: m/s}\n      duration: {value: 2, unit: s}"},
		{"speed: {value: 2, unit: m/s}", "ease: smooth\n      speed: {value: 2, unit: m/s}"},
		{"z: 2}", "z: -2}"},
		{"{x: 0, y: 0, z: 2}", "{x: 0, y: 0}"},
	} {
		text := string(raw) + strings.Replace(cameraFixture, change.before, change.after, sequenceOne)
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			test.Fatal(err)
		}
		if _, err := readSequence(path); err == nil {
			test.Fatalf("accepted invalid camera %q", change.after)
		}
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

// moveLook is the fixture's move given its own look: from the path's
// target at its start to a metre east of it at its end.
const moveLook = "speed: {value: 2, unit: m/s}\n" +
	"      look: {from: {x: 0, y: 1, z: 0}, to: {x: 1, y: 1, z: 0}}"

// the test: a move's own look reaches the camera: the camera faces the
// look's first target as the move starts and its second as it ends.
func TestSequenceCameraTakesAMovesOwnLook(test *testing.T) {
	path, _ := sequenceFixture(test)
	raw, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	camera := strings.Replace(cameraFixture,
		"speed: {value: 2, unit: m/s}", moveLook, sequenceOne)
	sequenceWrite(test, path, append(raw, camera...))
	plan, err := readSequence(path)
	if err != nil {
		test.Fatal(err)
	}
	for _, each := range []struct {
		at     time.Duration
		target maths.Vec
	}{
		{0, maths.V(sequenceZero, sequenceOne, sequenceZero)},
		{2 * time.Second, maths.V(sequenceOne, sequenceOne, sequenceZero)},
	} {
		eye, err := plan.camera.At(each.at)
		if err != nil {
			test.Fatal(err)
		}
		toward := each.target.Sub(eye.Origin).Unit()
		if !eye.Forward().Near(toward, 1e-9) {
			test.Errorf("at %s the camera does not face %v", each.at,
				each.target)
		}
	}
}

// the anti-test: a look that says only one of its two targets is
// refused by the schema before any frame.
func TestSequenceSchemaRefusesALookHalfSaid(test *testing.T) {
	path, _ := sequenceFixture(test)
	raw, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	half := strings.Replace(moveLook, ", to: {x: 1, y: 1, z: 0}", "",
		sequenceOne)
	camera := strings.Replace(cameraFixture,
		"speed: {value: 2, unit: m/s}", half, sequenceOne)
	sequenceWrite(test, path, append(raw, camera...))
	if _, err := readSequence(path); err == nil {
		test.Error("a look without its second target was taken")
	}
}

func TestSequenceCameraAcceptsTimedEasingAndExplicitUp(test *testing.T) {
	path, _ := sequenceFixture(test)
	raw, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	camera := strings.Replace(cameraFixture,
		"speed: {value: 2, unit: m/s}",
		"duration: {value: 2, unit: s}\n      ease: smooth", sequenceOne)
	camera = strings.Replace(camera, "  unit: m\n",
		"  unit: m\n  up: {x: 1, y: 0, z: 0}\n", sequenceOne)
	sequenceWrite(test, path, append(raw, camera...))
	plan, err := readSequence(path)
	if err != nil {
		test.Fatal(err)
	}
	at := sequenceTime(test, "smooth-quarter-seconds")
	eye, err := plan.camera.At(at)
	if err != nil {
		test.Fatal(err)
	}
	want := sequenceNumber(test, "smooth-quarter-z")
	if eye.Origin != maths.V(sequenceZero, sequenceZero, want) {
		test.Fatal("duration/easing units changed the hand-worked position", eye.Origin)
	}
	if eye.Orientation.Y != maths.XAxis {
		test.Fatal("lost explicit camera up")
	}
}
