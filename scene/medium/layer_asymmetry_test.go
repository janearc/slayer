package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// layerAsymmetryLobesFile is the table of asymmetries a weather is offered,
// each with whether the weather is taken.
const layerAsymmetryLobesFile = "testdata/layer-asymmetry-lobes.csv"

// layerAsymmetryGround is ground at a height everywhere.
type layerAsymmetryGround struct{ height float64 }

// at is the height of the ground, wherever it is asked.
func (ground layerAsymmetryGround) at(float64, float64) float64 {
	return ground.height
}

// layerAsymmetryBox is the box the layers are made in: a cube about the
// middle of space.
func layerAsymmetryBox(t *testing.T) maths.Box {
	t.Helper()
	edge := picked(t, "layer-asymmetry-box-edge")
	return maths.Box{Min: maths.V(-edge, -edge, -edge),
		Max: maths.V(edge, edge, edge)}
}

// layerAsymmetryWeather is the mist the layers are made of, with an
// asymmetry.
func layerAsymmetryWeather(t *testing.T, asymmetry float64) GroveWeather {
	t.Helper()
	return GroveWeather{Extinction: picked(t, "layer-asymmetry-extinction"),
		Depth: picked(t, "layer-asymmetry-depth"), Coverage: unity,
		Asymmetry: asymmetry}
}

// TestALayerKeepsItsAsymmetry: a layer and a deck say the asymmetry their
// weather was given, a copy ticked on keeps it, and an even weather's is
// nought.
func TestALayerKeepsItsAsymmetry(t *testing.T) {
	box := layerAsymmetryBox(t)
	ground := layerAsymmetryGround{picked(t, "layer-asymmetry-ground")}.at
	level := picked(t, "layer-asymmetry-deck-level")
	for _, want := range []float64{nought,
		picked(t, "layer-asymmetry-forward"),
		picked(t, "layer-asymmetry-backward")} {
		weather := layerAsymmetryWeather(t, want)
		layer, err := NewLayer(weather, box, ground)
		if err != nil {
			t.Fatal(err)
		}
		deck, err := NewDeck(weather, box, ground, level)
		if err != nil {
			t.Fatal(err)
		}
		ticked, ok := deck.Tick(unity).(*Layer)
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
	box := layerAsymmetryBox(t)
	ground := layerAsymmetryGround{picked(t, "layer-asymmetry-ground")}.at
	for _, row := range readSheet(t, layerAsymmetryLobesFile) {
		asymmetry, want := cell(t, row, "asymmetry"), row["taken"] == "true"
		weather := layerAsymmetryWeather(t, asymmetry)
		if _, err := NewLayer(weather, box, ground); (err == nil) != want {
			t.Errorf("asymmetry %v: error %v, want taken %t", asymmetry,
				err, want)
		}
	}
	weather := layerAsymmetryWeather(t, math.NaN())
	if _, err := NewLayer(weather, box, ground); err == nil {
		t.Error("an asymmetry of no number was taken")
	}
}
