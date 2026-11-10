package main

import (
	"bytes"
	"image"
	"testing"
	"time"

	"github.com/janearc/slayer/camera"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// A static glowing sphere isolates lens changes from movement or lighting.
func TestSequenceZoomChangesCPUFramingWithoutChangingRetainedCamera(
	test *testing.T,
) {
	checkSequenceZoomFraming(test, false)
}

// checkSequenceZoomFraming samples out of order to catch mutable playheads.
func checkSequenceZoomFraming(test *testing.T, onGPU bool) {
	test.Helper()
	plan, given := zoomRenderFixture(test, onGPU)
	var first []byte
	var counts []int
	times := []time.Duration{0, 2 * time.Second, 4 * time.Second, 0}
	for _, elapsed := range times {
		picture := zoomReferenceFrame(test, plan, given, plan.start+elapsed)
		if first == nil {
			first = append([]byte(nil), picture.Pix...)
		} else if elapsed == 0 && !bytes.Equal(first, picture.Pix) {
			test.Fatal("zoom changed when its first frame was replayed")
		}
		counts = append(counts, zoomCoveredPixels(picture))
	}
	if !(counts[0] > counts[1] && counts[1] > counts[2]*2) {
		test.Fatal("wider lens did not shrink the fixed sphere", counts)
	}
	if given.eye != nil || given.wide != 110 {
		test.Fatal("zoom mutated the retained camera or lens")
	}
}

// zoomRenderFixture uses a hold, so all changing geometry comes from the lens.
func zoomRenderFixture(test *testing.T, onGPU bool) (sequencePlan, asked) {
	test.Helper()
	zoom, err := camera.NewZoom(20, 80, 4*time.Second, "linear")
	if err != nil {
		test.Fatal(err)
	}
	from := maths.V(0, 0, 5)
	hold, err := camera.Timed(from, from, 4*time.Second, "")
	if err != nil {
		test.Fatal(err)
	}
	path, err := camera.New([]camera.Move{hold}, maths.Vec{}, maths.Up)
	if err != nil {
		test.Fatal(err)
	}
	plan := sequencePlan{camera: path, zoom: zoom, start: 45 * time.Second,
		knee: 0.7, top: 8}
	given := zoomReferenceScene(test, onGPU)
	return plan, given
}

// zoomReferenceScene contains no time-dependent light or surface motion.
func zoomReferenceScene(test *testing.T, onGPU bool) asked {
	test.Helper()
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Glow: maths.RGB(1, 0, 0)}})
	given := mustParse(test)
	given.width, given.height, given.samples = 33, 33, 1
	given.wide, given.onGPU = 110, onGPU
	given.live = &liveScene{base: *world}
	return given
}

// zoomReferenceFrame requires an actual hardware dispatch for the GPU case.
func zoomReferenceFrame(test *testing.T, plan sequencePlan, given asked,
	at time.Duration,
) *image.RGBA {
	test.Helper()
	picture, profile, _, err := sequencePicture(plan, given, at)
	if err != nil {
		test.Fatal(err)
	}
	if given.onGPU && (profile.Device == "" || profile.TraceGPU <= 0) {
		test.Fatal("zoom did not dispatch to a hardware renderer", profile)
	}
	return picture
}

// zoomCoveredPixels counts the red emitter against a black background.
func zoomCoveredPixels(picture *image.RGBA) int {
	count := 0
	for y := 0; y < picture.Bounds().Dy(); y++ {
		for x := 0; x < picture.Bounds().Dx(); x++ {
			if picture.RGBAAt(x, y).R > 0 {
				count++
			}
		}
	}
	return count
}
