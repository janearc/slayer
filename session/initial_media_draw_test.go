//go:build draw && darwin

package session

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/trace"
)

func TestMetalDrawMeasuredUsesSetCameraInitialMedia(t *testing.T) {
	read := fixedCameraTransportSet(t, true)
	staged := stagedWorld(read, asked{set: "fixed-camera-water.yaml",
		onGPU: true, width: 1, height: 1, samples: 1})
	if staged.err != nil {
		t.Fatal(staged.err)
	}
	if len(staged.given.initialMedia) != 1 ||
		staged.given.initialMedia[0].ClearMedium == nil {
		t.Fatalf("staged set-camera stack = %+v", staged.given.initialMedia)
	}

	metalFrame, profile, err := drawMeasured(staged.world, staged.given, 0)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatalf("set-camera Metal draw: %v", err)
	}
	if profile.Device == "" {
		t.Fatal("drawMeasured did not report a Metal device")
	}

	eye := *staged.given.eye
	lens := lensFor(staged.given)
	cpuFrame, err := trace.SampledWithInitialMedia(staged.world, eye, lens,
		0, 1, 1, 1, staged.given.initialMedia)
	if err != nil {
		t.Fatal(err)
	}
	for _, channel := range []struct {
		name string
		got  float64
		want float64
	}{
		{"red", float64(metalFrame.At(0, 0).R), float64(cpuFrame.At(0, 0).R)},
		{"green", float64(metalFrame.At(0, 0).G), float64(cpuFrame.At(0, 0).G)},
		{"blue", float64(metalFrame.At(0, 0).B), float64(cpuFrame.At(0, 0).B)},
	} {
		if math.Abs(channel.got-channel.want) > 3e-4 {
			t.Errorf("set-camera %s Metal %.8g differs from CPU %.8g",
				channel.name, channel.got, channel.want)
		}
	}
	if metalFrame.At(0, 0).Cover != cpuFrame.At(0, 0).Cover ||
		metalFrame.At(0, 0).Depth != cpuFrame.At(0, 0).Depth {
		t.Fatalf("set-camera coverage/depth Metal=%+v CPU=%+v",
			metalFrame.At(0, 0), cpuFrame.At(0, 0))
	}

	airRead := fixedCameraTransportSet(t, false)
	airStaged := stagedWorld(airRead, asked{set: "fixed-camera-water.yaml",
		onGPU: true, width: 1, height: 1, samples: 1})
	if airStaged.err != nil {
		t.Fatal(airStaged.err)
	}
	if len(airStaged.given.initialMedia) != 0 {
		t.Fatalf("omitted camera stack became %+v", airStaged.given.initialMedia)
	}
	airMetal, airProfile, err := drawMeasured(airStaged.world,
		airStaged.given, 0)
	if err != nil {
		t.Fatalf("legacy air Metal draw: %v", err)
	}
	legacyMetal, legacyProfile, err := gpu.SampledMeasured(airStaged.world,
		*airStaged.given.eye, lensFor(airStaged.given),
		0, 1, 1, 1)
	if err != nil {
		t.Fatalf("legacy direct GPU control: %v", err)
	}
	if len(airMetal.Pix) != len(legacyMetal.Pix) ||
		airProfile.Device != profile.Device || legacyProfile.Device != profile.Device {
		t.Fatalf("GPU devices differ between camera-medium and legacy air controls: %q, %q, %q",
			profile.Device, airProfile.Device, legacyProfile.Device)
	}
	for index := range airMetal.Pix {
		if airMetal.Pix[index] != legacyMetal.Pix[index] {
			t.Fatalf("omitted initial-media set changed legacy GPU pixel: route=%+v direct=%+v",
				airMetal.Pix[index], legacyMetal.Pix[index])
		}
	}
	stacked := metalFrame.At(0, 0)
	air := airMetal.At(0, 0)
	if math.Abs(float64(stacked.R-air.R))+math.Abs(float64(stacked.G-air.G))+
		math.Abs(float64(stacked.B-air.B)) < 1e-3 {
		t.Fatalf("camera stack did not affect the boundary transport: medium=%+v air=%+v",
			stacked, air)
	}
}
