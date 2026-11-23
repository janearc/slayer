package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// TestReadPlanAcceptsMinimalScene checks the decoded fields, the resolved
// source path and the retained bytes and hash of a valid scene.
func TestReadPlanAcceptsMinimalScene(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "scene.yaml", minimalScene)
	got, err := readPlan(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "data", "heights.tif"); got.Sources[0].File != want {
		t.Errorf("source path = %q, want %q", got.Sources[0].File, want)
	}
	if got.Radius != 6371000 || got.Film.Width != 640 || got.Detail.Side != 9 {
		t.Errorf("decoded radius %v width %v side %v", got.Radius, got.Film.Width, got.Detail.Side)
	}
	if string(got.raw) != minimalScene {
		t.Error("raw bytes were not retained exactly")
	}
	if got.hash != configHash([]byte(minimalScene)) || len(got.hash) != 64 {
		t.Errorf("hash = %q", got.hash)
	}
}

// TestReadPlanKeepsAbsoluteSourcePath checks that an absolute source file is
// left exactly as authored.
func TestReadPlanKeepsAbsoluteSourcePath(t *testing.T) {
	dir := t.TempDir()
	absolute := filepath.Join(dir, "elsewhere", "heights.tif")
	text := strings.Replace(minimalScene, "file: data/heights.tif", "file: "+absolute, 1)
	got, err := readPlan(writeFile(t, dir, "scene.yaml", text))
	if err != nil {
		t.Fatal(err)
	}
	if got.Sources[0].File != absolute {
		t.Errorf("source path = %q, want %q", got.Sources[0].File, absolute)
	}
}

// TestReadPlanRefusals feeds a missing file and several bad scenes and checks
// the refusal each one earns.
func TestReadPlanRefusals(t *testing.T) {
	cases := []struct {
		name, from, to, want string
	}{
		{"lattice outside the schema list", "side: 9", "side: 10", "/detail/side"},
		{"region east before west", "west: -156.19", "west: -150", "invalid terrain region"},
		{"region north before south", "south: 18.81", "south: 25", "invalid terrain region"},
		{"height range reversed", "minimum-height-metres: -6000", "minimum-height-metres: 9000", "invalid terrain region"},
		{"schema type error", "radius-metres: 6371000", "radius-metres: wide", "radius-metres"},
		{"composition refusal", "exposure: 1.5", "exposure: 1.5\nlandcover:\n  - {below-metres: 500, colour-rgb: [1, 0, 0]}\n  - {below-metres: 100, colour-rgb: [0, 1, 0]}", "landcover height boundaries must increase"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := strings.Replace(minimalScene, tc.from, tc.to, 1)
			scenePath := writeFile(t, t.TempDir(), "scene.yaml", text)
			_, err := readPlan(scenePath)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want one containing %q", err, tc.want)
			}
		})
	}
	absent := filepath.Join(t.TempDir(), "absent.yaml")
	if _, err := readPlan(absent); err == nil {
		t.Error("a missing scene file was accepted")
	}
	unparsable := writeFile(t, t.TempDir(), "scene.yaml", "{")
	if _, err := readPlan(unparsable); err == nil {
		t.Error("unparsable YAML was accepted")
	}
}

// TestLocationGeographic checks that authored degrees and metres reach the
// frame's units.
func TestLocationGeographic(t *testing.T) {
	got := location{Longitude: 90, Latitude: -45, Height: 1234}.geographic()
	if got.Longitude.InDegrees() != 90 || got.Latitude.InDegrees() != -45 || got.Height.InMetres() != 1234 {
		t.Errorf("geographic = %v %v %v", got.Longitude.InDegrees(), got.Latitude.InDegrees(), got.Height.InMetres())
	}
}

// TestRGBOrdersChannels checks that the three authored channels land in red,
// green, blue order.
func TestRGBOrdersChannels(t *testing.T) {
	if got, want := rgb([3]float64{0.25, 0.5, 0.75}), maths.RGB(0.25, 0.5, 0.75); got != want {
		t.Errorf("rgb = %v, want %v", got, want)
	}
}
