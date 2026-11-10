//go:build darwin

package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/metal"
)

func TestCommandReportsMeasuredGPUWorkForEveryPinholeMode(t *testing.T) {
	requireCommandDevice(t)
	for _, mode := range []string{"gpu", "gpu-native-shadows", "gpu-native", "cached"} {
		t.Run(mode, func(t *testing.T) {
			given := fireOnFixture(t)
			raw, err := os.ReadFile(given.set)
			if err != nil {
				t.Fatal(err)
			}
			renderer := mode
			if mode == "cached" {
				renderer = "gpu-native"
			}
			text := strings.Replace(string(raw), "renderer: gpu-native", "renderer: "+renderer, 1)
			if mode == "cached" {
				text = strings.Replace(text, "renderer: gpu-native", "renderer: gpu-native\n  volume-lighting: cached", 1)
			}
			if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			var said, wrong bytes.Buffer
			output := filepath.Join(t.TempDir(), "measured.png")
			if code := run([]string{"-set", given.set, "-out", output}, &said, &wrong); code != 0 {
				t.Fatalf("mode %s: %d %s", mode, code, wrong.String())
			}
			if !strings.Contains(said.String(), "sample output") || !strings.Contains(said.String(), "gpu ") {
				t.Fatalf("device work not reported: %s", said.String())
			}
			if mode == "gpu-native" && (!strings.Contains(said.String(), "native closest:") || !strings.Contains(said.String(), "software fallbacks") || !strings.Contains(said.String(), "native hierarchy:")) {
				t.Fatalf("native work not reported: %s", said.String())
			}
			if mode == "cached" && !strings.Contains(said.String(), "smoke lighting:") {
				t.Fatalf("lighting work not reported: %s", said.String())
			}
			if info, err := os.Stat(output); err != nil || info.Size() == 0 {
				t.Fatalf("missing output: %v", err)
			}
		})
	}
}

func TestRigSamplesEveryGPUBackendAndPropagatesDrawFailures(t *testing.T) {
	for _, mode := range []string{"gpu", "gpu-native-shadows", "gpu-native", "invalid shutter", "invalid world", "replay"} {
		t.Run(mode, func(t *testing.T) {
			given := mustParse(t)
			given.set = onFixtureRigged(t, "0.5", false)
			world, made, given, err := staged(given)
			if err != nil {
				t.Fatal(err)
			}
			given.width, given.height = 4, 3
			given.onGPU = true
			given.nativeShadows = mode == "gpu-native-shadows"
			given.nativeMeshes = mode == "gpu-native"
			switch mode {
			case "invalid shutter":
				given.glimpses = 0
				if _, err := exposed(world, given, *given.eye); err == nil {
					t.Fatal("invalid shutter drawn")
				}
				if _, _, err := replayed(world, given, *given.eye, 0, 0); err == nil {
					t.Fatal("invalid shutter replayed")
				}
				return
			case "invalid world":
				world.Haze.Density = 1
				if _, err := exposed(world, given, *given.eye); err == nil || !strings.Contains(err.Error(), "haze") {
					t.Fatal("invalid world drawn")
				}
				return
			case "replay":
				if _, _, err := replayed(world, given, *given.eye, -1, 0); err == nil {
					t.Fatal("outside replay pixel accepted")
				}
				return
			}
			picture, _, err := drawMeasured(world, given, made.across)
			if errors.Is(err, metal.ErrNoDevice) {
				t.Skip(err)
			}
			if err != nil {
				t.Fatal(err)
			}
			if picture.W != 4 || picture.H != 3 || len(picture.Pix) != 12 {
				t.Fatal("rig lost its frame dimensions")
			}
		})
	}
}

func requireCommandDevice(t *testing.T) {
	t.Helper()
	device, err := metal.OpenSession(
		"#include <metal_stdlib>\nusing namespace metal;\n"+
			"kernel void command_probe(device float *out [[buffer(0)]], "+
			"uint i [[thread_position_in_grid]]) { out[i] = 0; }\n",
		"command_probe")
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	device.Close()
}
