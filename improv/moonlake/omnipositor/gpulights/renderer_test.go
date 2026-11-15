//go:build draw

package gpulights

import (
	"math"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/lights"
)

func TestRendererRefusesUnknownEffect(t *testing.T) {
	renderer, err := New(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	plate := lights.Image{Width: 1, Height: 1, Light: []float64{1, 1, 1}}
	if _, err := renderer.Render(Request{Plate: plate, Master: 1, Effects: Effect(1 << 20)}); err == nil {
		t.Fatal("unknown effect accepted")
	}
}

func TestRendererRefusesFixtureBudget(t *testing.T) {
	renderer, err := NewWithBudget(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	plate := lights.Image{Width: 1, Height: 1, Light: []float64{1, 1, 1}}
	if _, err := renderer.Render(Request{Plate: plate, Master: 1}); err == nil {
		t.Fatal("insufficient buffer budget accepted")
	}
}

func TestRendererRejectsNonfiniteInputs(t *testing.T) {
	renderer, err := New(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	plate := lights.Image{Width: 1, Height: 1, Light: []float64{1, math.NaN(), 1}}
	if _, err := renderer.Render(Request{Plate: plate, Master: 1}); err == nil {
		t.Fatal("nonfinite plate accepted")
	}
}

func TestRendererBoundsDimensionsAndLayers(t *testing.T) {
	if _, err := New(1<<24, 2); err == nil {
		t.Fatal("excessive dimensions accepted")
	}
	renderer, err := New(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	plate := lights.Image{Width: 1, Height: 1, Light: []float64{1, 1, 1}}
	request := Request{Plate: plate, Master: 1, Effects: EffectFixtures, FixtureLayers: make([]FixtureLayer, 65)}
	if _, err := renderer.Render(request); err == nil {
		t.Fatal("excessive fixture count accepted")
	}
}
