package main

import (
	"strings"
	"testing"
)

func TestExperimentalGPUOptions(t *testing.T) {
	args := stretchArgs([]string{"-from", "3370", "-to", "3442", "-settings", "night.yaml", "-out", "film", "-experimental-gpu", "-trace", "trace.json", "-encode", "piece.mp4", "-ffmpeg", "/usr/bin/ffmpeg", "--dry-run"})
	code, output := run(args...)
	if code != 0 || !strings.Contains(output, "experimental hybrid GPU") {
		t.Fatalf("GPU dry run: %d %s", code, output)
	}
}

func TestGPUOceanRequiresExperimentalGPU(t *testing.T) {
	args := stretchArgs([]string{"-settings", "night.yaml", "-out", "film", "-experimental-gpu-ocean", "-encode", "piece.mp4", "-ffmpeg", "/usr/bin/ffmpeg", "--dry-run"})
	code, output := run(args...)
	if code == 0 || !strings.Contains(output, "requires --experimental-gpu") {
		t.Fatalf("GPU ocean without GPU path: %d %s", code, output)
	}
}

func TestExperimentalGPURefusesUnboundPNG(t *testing.T) {
	args := stretchArgs([]string{"-settings", "night.yaml", "-out", "film", "-experimental-gpu", "-png", "--dry-run"})
	code, output := run(args...)
	if code == 0 || !strings.Contains(output, "GPU identity and resume are unavailable") {
		t.Fatalf("GPU PNG was not refused: %d %s", code, output)
	}
}
