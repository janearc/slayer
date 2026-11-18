package main

import (
	"bytes"
	"context"
	"errors"
	"github.com/janearc/celestial/measure"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// minimalScene is the smallest scene the bundled schema accepts. Its source
// file is named but never opened by the planning code.
const minimalScene = `height-model: authored-radial-offsets
radius-metres: 6371000
anchor: {longitude-deg: -155.5, latitude-deg: 19.5, height-metres: 0}
sources:
  - file: data/heights.tif
    sha256: "0000000000000000000000000000000000000000000000000000000000000000"
    geographic-crs: 4326
    height-datum: MSL
    height-offset-metres: 0
raster-budget:
  cache-total-bytes: 67108864
  cache-tiles-per-source: 128
  tile-bytes: 33554432
  metadata-bytes: 16777216
region: {west: -156.19, south: 18.81, east: -154.71, north: 20.39}
detail:
  patches: 256
  evaluations: 2048
  depth: 12
  side: 9
  points: 65536
  cell-pixels: 8
  minimum-height-metres: -6000
  maximum-height-metres: 4500
camera:
  at: {longitude-deg: -156.4, latitude-deg: 19.4, height-metres: 300000}
  look-at: {longitude-deg: -155.5, latitude-deg: 19.55, height-metres: 1500}
  near-metres: 0.1
film: {width: 640, height: 360, rays: 2, wide-deg: 65}
sun: {azimuth-deg: 285, altitude-deg: 35, strength-rgb: [2, 1.75, 1.4]}
ground-rgb: [0.12, 0.20, 0.07]
sky-rgb: [0.008, 0.012, 0.025]
ocean:
  height-metres: 0
  colour-rgb: [0.01, 0.06, 0.10]
  gloss: 1
  index: 1.34
exposure: 1.5
`

// writeFile stores text under a fresh name in dir and returns its path.
func writeFile(t *testing.T, dir, name, text string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// runTool runs the command tree and returns its exit code and error text.
func runTool(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errs bytes.Buffer
	code := tool(context.Background()).Run(args, &out, &errs)
	return code, out.String(), errs.String()
}

// mustContain fails the test unless text includes every fragment.
func mustContain(t *testing.T, text string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(text, fragment) {
			t.Errorf("%q does not contain %q", text, fragment)
		}
	}
}

// containsText reports whether text includes fragment.
func containsText(text, fragment string) bool { return strings.Contains(text, fragment) }

// cancelledContext returns a context and the function that cancels it.
func cancelledContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithCancel(t.Context())
}

// metres is a length of the given number of metres.
func metres(value float64) measure.Length { return measure.Metres(value) }

// errBoom is a stand-in failure from a source.
var errBoom = errors.New("boom")
