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

// layerNothing is defined as no density or accumulated difference.
const layerNothing = 0

//go:embed testdata/layer.yaml
var layerRaw []byte

type layerFloor struct {
	East  float64 `json:"east"`
	South float64 `json:"south"`
	Level float64 `json:"level"`
}

func (floor layerFloor) height(x, z float64) float64 {
	return floor.East*x + floor.South*z + floor.Level
}

type layerData struct {
	Weather   medium.GroveWeather `json:"weather"`
	Bounds    maths.Box           `json:"bounds"`
	Ground    layerFloor          `json:"ground"`
	Finest    float64             `json:"finest"`
	Tolerance float64             `json:"tolerance"`
	Samples   []struct {
		At      maths.Vec `json:"at"`
		Density float64   `json:"density"`
	} `json:"samples"`
	Advection struct {
		Seconds  float64     `json:"seconds"`
		Coverage float64     `json:"coverage"`
		Wind     maths.Vec   `json:"wind"`
		Probes   []maths.Vec `json:"probes"`
	} `json:"advection"`
	InvalidWeather []medium.GroveWeather `json:"invalid-weather" yaml:"invalid-weather"`
	InvalidBounds  []maths.Box           `json:"invalid-bounds" yaml:"invalid-bounds"`
}

func readLayerData(test *testing.T) layerData {
	test.Helper()
	var data layerData
	if err := yamlfile.Read("testdata/layer.yaml", layerRaw, &data); err != nil {
		test.Fatal(err)
	}
	return data
}

func TestLayerFollowsTerrainAndExplicitRegionalBounds(test *testing.T) {
	data := readLayerData(test)
	layer, err := medium.NewLayer(data.Weather, data.Bounds, data.Ground.height)
	if err != nil {
		test.Fatal(err)
	}
	if layer.Bounds() != data.Bounds || layer.Finest() != data.Finest {
		test.Fatal("lost bounds or density sampling hint")
	}
	for _, sample := range data.Samples {
		got := layer.Density(sample.At)
		if !(math.Abs(got-sample.Density) <= data.Tolerance) {
			test.Fatal("regional height profile", sample.At, got, sample.Density)
		}
	}
}

func TestLayerAdvectsWithoutChangingTheOriginalFrame(test *testing.T) {
	data := readLayerData(test)
	weather := data.Weather
	weather.Coverage, weather.Wind = data.Advection.Coverage, data.Advection.Wind
	var flat layerFloor
	layer, err := medium.NewLayer(weather, data.Bounds, flat.height)
	if err != nil {
		test.Fatal(err)
	}
	elapsed := time.Duration(data.Advection.Seconds * float64(time.Second))
	moved := layer.Tick(elapsed)
	shift := weather.Wind.Scale(data.Advection.Seconds)
	energy, changed := float64(layerNothing), float64(layerNothing)
	for _, probe := range data.Advection.Probes {
		before := layer.Density(probe)
		at := probe.Add(shift)
		if math.Abs(moved.Density(at)-before) > data.Tolerance {
			test.Fatal("wind did not carry the noise field", probe)
		}
		energy += before
		changed += math.Abs(moved.Density(probe) - before)
	}
	if energy <= layerNothing || changed <= data.Tolerance {
		test.Fatal("advection fixture did not exercise moving nonempty mist", energy, changed)
	}
	if moved.Bounds() != layer.Bounds() {
		test.Fatal("advection moved the weather support")
	}
}

func TestLayerRefusesInvalidWeatherBoundsAndMissingTerrain(test *testing.T) {
	data := readLayerData(test)
	for _, weather := range data.InvalidWeather {
		layer, err := medium.NewLayer(weather, data.Bounds, data.Ground.height)
		if err == nil || layer != nil {
			test.Fatal("accepted invalid regional weather", weather, err)
		}
	}
	for _, box := range data.InvalidBounds {
		layer, err := medium.NewLayer(data.Weather, box, data.Ground.height)
		if err == nil || layer != nil {
			test.Fatal("accepted invalid region", box, err)
		}
	}
	layer, err := medium.NewLayer(data.Weather, data.Bounds, nil)
	if err == nil || layer != nil {
		test.Fatal("accepted missing terrain", err)
	}
}
