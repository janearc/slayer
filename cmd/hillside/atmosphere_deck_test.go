package main

import (
	_ "embed"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

//go:embed testdata/atmosphere-deck-block.yaml
var atmosphereDeckBlock string

//go:embed testdata/atmosphere-deck-contract.yaml
var atmosphereDeckContractRaw []byte

type deckAdapterContract struct {
	Layers []struct {
		Weather             string
		Albedo, Bottom, Top float64
	}
	Later float64
}

func layeredGiven(test *testing.T) asked {
	test.Helper()
	given := laserGiven(test)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		test.Fatal(err)
	}
	text := strings.ReplaceAll(string(raw), "  terrain:", atmosphereDeckBlock+"  terrain:")
	sequenceWrite(test, given.set, []byte(text))
	return given
}

func (contract deckAdapterContract) layersIn(test *testing.T, world *thing.World) []*medium.Layer {
	test.Helper()
	var fields []*medium.Layer
	for _, volume := range world.Volumes {
		if layer, ok := volume.Medium.(*medium.Layer); ok {
			index := len(fields)
			if index >= len(contract.Layers) {
				test.Fatal("extra atmospheric layer")
			}
			want := contract.Layers[index]
			bounds := layer.Bounds()
			if bounds.Min.Y != want.Bottom || bounds.Max.Y != want.Top || volume.Material.Name != want.Weather || volume.Material.Colour != maths.Grey(want.Albedo) {
				test.Fatal("lost deck support or scattering optics", index, bounds, volume.Material)
			}
			fields = append(fields, layer)
		}
	}
	if len(fields) != len(contract.Layers) {
		test.Fatal("missing atmospheric layers", len(fields))
	}
	return fields
}

func TestAtmosphereDeckAndUpperAirReachAndRemainInTheScene(test *testing.T) {
	var contract deckAdapterContract
	if err := yamlfile.Read("testdata/atmosphere-deck-contract.yaml", atmosphereDeckContractRaw, &contract); err != nil {
		test.Fatal(err)
	}
	world, _, given, err := staged(layeredGiven(test))
	if err != nil {
		test.Fatal(err)
	}
	fields := contract.layersIn(test, world)
	elapsed := time.Duration(contract.Later * float64(time.Second))
	later, _, err := given.live.frameAt(elapsed)
	if err != nil {
		test.Fatal(err)
	}
	retained := contract.layersIn(test, later)
	for index, layer := range retained {
		if layer != fields[index] {
			test.Fatal("rebuilt or reordered static layer")
		}
	}
}
