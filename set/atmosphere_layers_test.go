package set_test

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/set"
)

// firstAtmosphere and secondAtmosphere name the fixture layers.
// They are defined as Go slice positions.
const firstAtmosphere, secondAtmosphere = 0, 1

//go:embed testdata/atmosphere-layers.yaml
var atmosphereLayersRaw []byte

func TestAtmosphereLayersKeepHeightOrderAndBothReaderPaths(test *testing.T) {
	var data struct {
		Block    string `json:"block"`
		Expected []struct {
			Albedo, Base float64
			Level        bool
		} `json:"expected"`
		Invalid []struct{ Before, After string } `json:"invalid"`
	}
	if err := yamlfile.Read("testdata/atmosphere-layers.yaml", atmosphereLayersRaw, &data); err != nil {
		test.Fatal(err)
	}
	for _, raw := range laserSources() {
		source := strings.ReplaceAll(raw, "  weather:\n", indentAtmosphere(data.Block)+"  weather:\n")
		made, err := set.Read("layers.yaml", []byte(source))
		if err != nil {
			test.Fatal(err)
		}
		layers := made.AtmosphericLayers()
		if len(layers) != len(data.Expected) {
			test.Fatal("lost atmosphere layers", len(layers))
		}
		for index, layer := range layers {
			want := data.Expected[index]
			base, level := layer.Base()
			if layer.Weather() != readAtmosphereContract(test).Weather || layer.Albedo() != want.Albedo || base != want.Base || level != want.Level {
				test.Fatal("lost atmospheric appearance or vertical placement", index, layer, want)
			}
		}
		layers[firstAtmosphere] = layers[secondAtmosphere]
		if made.AtmosphericLayers()[firstAtmosphere].Albedo() != data.Expected[firstAtmosphere].Albedo {
			test.Fatal("caller changed retained layers")
		}
		for _, invalid := range data.Invalid {
			broken := strings.ReplaceAll(source, invalid.Before, invalid.After)
			if broken == source {
				test.Fatal("ineffective layer fault", invalid)
			}
			if _, err := set.Read("bad-layers.yaml", []byte(broken)); err == nil {
				test.Fatal("accepted invalid layers", invalid)
			}
		}
	}
}

func indentAtmosphere(block string) string {
	return "  " + strings.ReplaceAll(strings.TrimSuffix(block, "\n"), "\n", "\n  ") + "\n"
}
