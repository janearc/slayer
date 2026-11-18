package main

import (
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// TestLocalLightsReachTheScene checks the retained YAML through real lamp
// construction and validates refusal before a draw, not only a decoded struct.
func TestLocalLightsReachTheScene(t *testing.T) {
	raw := append(append([]byte{}, planRaw...), []byte("\nlocal-lights:\n  - at: {x: 2, y: 3, z: 4}\n    strength: {r: 8, g: 4, b: 2}\n")...)
	plan, err := readPlan("local.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	if err := plan.illuminate(world, nil); err != nil {
		t.Fatal(err)
	}
	lamp, ok := world.Lamps[len(world.Lamps)-1].(emit.Lit)
	if !ok || lamp.At != maths.V(2, 3, 4) || lamp.Strength != maths.RGB(8, 4, 2) {
		t.Fatal("local source lost in scene", world.Lamps)
	}
	// Direct construction must refuse nonfinite light before opening a GPU.
	plan.LocalLights[0].Strength.R = math.NaN()
	if plan.check() == nil {
		t.Fatal("accepted nonfinite local source")
	}
}

// TestLocalLightsRequirePosition rejects an unsupported or missing position.
func TestLocalLightsRequirePosition(t *testing.T) {
	for _, suffix := range []string{
		"local-lights:\n  - strength: {r: 8, g: 4, b: 2}\n",
		"local-lights:\n  - at: {x: 2, y: 3, z: 4}\n    strength: {r: -1, g: 4, b: 2}\n",
	} {
		raw := append(append([]byte{}, planRaw...), []byte("\n"+suffix)...)
		if _, err := readPlan("bad-local.yaml", raw); err == nil {
			t.Fatal("accepted invalid lamp")
		}
	}
}
