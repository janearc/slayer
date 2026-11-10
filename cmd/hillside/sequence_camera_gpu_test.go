//go:build darwin

package main

import (
	"bytes"
	_ "embed"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/metal"
)

//go:embed testdata/camera-gpu.yaml
var gpuCameraFixture string

func TestSequenceCameraReachesTheGPUKernel(test *testing.T) {
	path, _ := sequenceFixture(test)
	raw, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	sequenceWrite(test, path, append(raw, gpuCameraFixture...))
	plan, err := readSequence(path)
	if err != nil {
		test.Fatal(err)
	}
	given := gpuCameraScene(test)
	seen := map[string][]byte{}
	var previous []byte
	for _, name := range []string{"start", "middle", "end", "start"} {
		at := sequenceTime(test, name)
		picture, profile, _, err := sequencePicture(*plan, given, at)
		if errors.Is(err, metal.ErrNoDevice) {
			test.Skip("no Metal device")
		}
		if err != nil {
			test.Fatal(err)
		}
		if profile.Device == "" || strings.Contains(profile.Device, "software") {
			test.Fatalf("no hardware kernel: %+v", profile)
		}
		if profile.TraceGPU <= sequenceZero {
			test.Fatal("no measured GPU dispatch")
		}
		if previous != nil && bytes.Equal(previous, picture.Pix) {
			test.Fatal("GPU camera did not move")
		}
		if before, exists := seen[name]; exists && !bytes.Equal(before, picture.Pix) {
			test.Fatal("GPU camera changed when sampled out of order")
		}
		current := append([]byte(nil), picture.Pix...)
		seen[name], previous = current, current
		test.Logf("%s: %s, GPU dispatch %s", name, profile.Device, profile.TraceGPU)
	}
}

// gpuCameraScene uses the actual procedural factory and supported lamps.
func gpuCameraScene(test *testing.T) asked {
	test.Helper()
	arguments := only(map[string]string{"boulder": "1"})
	given := mustParse(test, arguments...)
	dimension := int(sequenceNumber(test, "width"))
	given.width, given.height = dimension, dimension
	world, made, err := hillside(given)
	if err != nil {
		test.Fatal(err)
	}
	given.live = &liveScene{base: *world, made: made}
	given.onGPU = true
	return given
}
