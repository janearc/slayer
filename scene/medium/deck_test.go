package medium_test

import (
	_ "embed"
	"math"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// deckPositiveSign is defined as the positive sign for math.Inf.
const deckPositiveSign = 1

//go:embed testdata/deck.yaml
var deckRaw []byte

type deckSample struct {
	At      maths.Vec `json:"at"`
	Density float64   `json:"density"`
}

type deckFixture struct {
	Ground      layerFloor   `json:"ground"`
	Base        float64      `json:"base"`
	Samples     []deckSample `json:"samples"`
	Clipped     maths.Box    `json:"clipped"`
	ClipSamples []deckSample `json:"clip-samples" yaml:"clip-samples"`
}

func readDeckFixture(test *testing.T) deckFixture {
	test.Helper()
	var fixture deckFixture
	if err := yamlfile.Read("testdata/deck.yaml", deckRaw, &fixture); err != nil {
		test.Fatal(err)
	}
	return fixture
}

func TestDeckKeepsALevelProfileAndClipsTheLandscape(test *testing.T) {
	data := readLayerData(test)
	fixture := readDeckFixture(test)
	floor := fixture.Ground
	deck, err := medium.NewDeck(data.Weather, data.Bounds, floor.height, fixture.Base)
	if err != nil {
		test.Fatal(err)
	}
	for _, sample := range fixture.Samples {
		if got := deck.Density(sample.At); !(math.Abs(got-sample.Density) <= data.Tolerance) {
			test.Fatal("deck moved with terrain or filled solid ground", sample, got)
		}
	}
	elapsed := time.Duration(data.Advection.Seconds * float64(time.Second))
	later := deck.Tick(elapsed)
	for _, sample := range fixture.Samples {
		if later.Density(sample.At) != deck.Density(sample.At) {
			test.Fatal("Tick lost the level base", sample)
		}
	}
	for _, base := range []float64{math.NaN(), math.Inf(deckPositiveSign), math.Inf(-deckPositiveSign)} {
		if layer, err := medium.NewDeck(data.Weather, data.Bounds, floor.height, base); err == nil || layer != nil {
			test.Fatal("accepted non-finite deck base", base, layer, err)
		}
	}
	if layer, err := medium.NewDeck(data.Weather, data.Bounds, nil, fixture.Base); err == nil || layer != nil {
		test.Fatal("deck bypassed terrain validation", layer, err)
	}
}

func TestDeckRetainsDensityAtClippedSupportFaces(test *testing.T) {
	data := readLayerData(test)
	fixture := readDeckFixture(test)
	deck, err := medium.NewDeck(data.Weather, fixture.Clipped, fixture.Ground.height, fixture.Base)
	if err != nil {
		test.Fatal(err)
	}
	for _, sample := range fixture.ClipSamples {
		if got := deck.Density(sample.At); !(math.Abs(got-sample.Density) <= data.Tolerance) {
			test.Fatal("clipped deck lost its boundary density", sample, got)
		}
	}
}
