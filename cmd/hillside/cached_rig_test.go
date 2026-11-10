package main

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/ninjafilm/lens"
	"github.com/janearc/slayer/set"
)

func TestCachedLightingRefusesRigBeforeDrawing(t *testing.T) {
	path := onFixtureRigged(t, "0.5", false)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), "scene:",
		"scene:\n  renderer: gpu-native\n  volume-lighting: cached", 1))
	read, err := set.Read(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = filmed(read, mustParse(t)); err == nil ||
		!strings.Contains(err.Error(), "requires a pinhole film") {
		t.Fatalf("cached rig was silently accepted: %v", err)
	}
	// A direct caller must receive the same refusal before world traversal.
	given := mustParse(t)
	given.glass, given.cachedLighting = &lens.Thin{}, true
	if _, _, err = drawMeasured(nil, given, 1); err == nil ||
		!strings.Contains(err.Error(), "requires a pinhole film") {
		t.Fatalf("draw bypassed the refusal: %v", err)
	}
}
