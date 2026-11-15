package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// TestALayerKeepsItsAsymmetry: a layer and a deck say the asymmetry their
// weather was given, a copy ticked on keeps it, and an even weather's is
// nought.
func TestALayerKeepsItsAsymmetry(t *testing.T) {
	box := maths.Box{Min: maths.V(-5, -5, -5), Max: maths.V(5, 5, 5)}
	ground := func(x, z float64) float64 { return -10 }
	for _, want := range []float64{0, 0.85, -0.4} {
		weather := GroveWeather{Extinction: .05, Depth: 10, Coverage: 1,
			Asymmetry: want}
		layer, err := NewLayer(weather, box, ground)
		if err != nil {
			t.Fatal(err)
		}
		deck, err := NewDeck(weather, box, ground, -5)
		if err != nil {
			t.Fatal(err)
		}
		ticked, ok := deck.Tick(1).(*Layer)
		if !ok {
			t.Fatal("a ticked deck is not a layer")
		}
		for name, got := range map[string]float64{"layer": layer.Asymmetry(),
			"deck": deck.Asymmetry(), "ticked": ticked.Asymmetry()} {
			if got != want {
				t.Errorf("%s: asymmetry %v, want %v", name, got, want)
			}
		}
	}
}

// TestALobePastOneIsRefused: a weather whose asymmetry is one or past it
// either way, or no number, is no mist; one just inside is.
func TestALobePastOneIsRefused(t *testing.T) {
	box := maths.Box{Min: maths.V(-5, -5, -5), Max: maths.V(5, 5, 5)}
	ground := func(x, z float64) float64 { return -10 }
	for asymmetry, want := range map[float64]bool{1: false, -1: false,
		1.5: false, 0.99: true, -0.99: true} {
		weather := GroveWeather{Extinction: .05, Depth: 10, Coverage: 1,
			Asymmetry: asymmetry}
		if _, err := NewLayer(weather, box, ground); (err == nil) != want {
			t.Errorf("asymmetry %v: error %v, want taken %t", asymmetry,
				err, want)
		}
	}
	weather := GroveWeather{Extinction: .05, Depth: 10, Coverage: 1}
	weather.Asymmetry = math.NaN()
	if _, err := NewLayer(weather, box, ground); err == nil {
		t.Error("an asymmetry of no number was taken")
	}
}
