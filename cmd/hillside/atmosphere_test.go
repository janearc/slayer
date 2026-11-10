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

//go:embed testdata/atmosphere-block.yaml
var atmosphereBlock string

//go:embed testdata/atmosphere-contract.yaml
var atmosphereAdapterRaw []byte

type atmosphereAdapterContract struct {
	Weather string  `json:"weather"`
	Albedo  float64 `json:"albedo"`
	Depth   float64 `json:"depth"`
	Later   float64 `json:"later"`
}

func TestRegionalAtmosphereReachesTheActualSceneAndRetainsItsTerrain(test *testing.T) {
	var want atmosphereAdapterContract
	if err := yamlfile.Read("testdata/atmosphere-contract.yaml", atmosphereAdapterRaw, &want); err != nil {
		test.Fatal(err)
	}
	given := laserGiven(test)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		test.Fatal(err)
	}
	text := strings.ReplaceAll(string(raw), "  terrain:", atmosphereBlock+"  terrain:")
	sequenceWrite(test, given.set, []byte(text))
	world, _, given, err := staged(given)
	if err != nil {
		test.Fatal(err)
	}
	volume, layer := actualLayer(test, world)
	if volume.Material == nil || volume.Material.Name != want.Weather || volume.Material.Colour != maths.Grey(want.Albedo) {
		test.Fatal("lost regional scattering appearance", volume.Material)
	}
	want.assertSupport(test, world, layer)
	elapsed := time.Duration(want.Later * float64(time.Second))
	later, _, err := given.live.frameAt(elapsed)
	if err != nil {
		test.Fatal(err)
	}
	_, retained := actualLayer(test, later)
	if retained != layer {
		test.Fatal("static terrain field was rebuilt between frames")
	}
}

func actualLayer(test *testing.T, world *thing.World) (*thing.Volume, *medium.Layer) {
	test.Helper()
	var found *thing.Volume
	var layer *medium.Layer
	for _, volume := range world.Volumes {
		if field, ok := volume.Medium.(*medium.Layer); ok {
			if found != nil {
				test.Fatal("duplicate regional atmosphere")
			}
			found, layer = volume, field
		}
	}
	if found == nil {
		test.Fatal("regional atmosphere missing from actual scene")
	}
	return found, layer
}

func (want atmosphereAdapterContract) assertSupport(test *testing.T, world *thing.World, layer *medium.Layer) {
	test.Helper()
	found := false
	for _, object := range world.Things {
		if object.Material != nil && object.Material.Name == "basalt" {
			found = true
			bounds := object.Shape.Bounds()
			bounds.Max.Y += want.Depth
			if layer.Bounds() != bounds {
				test.Fatal("mist does not span terrain plus its declared depth", layer.Bounds(), bounds)
			}
		}
	}
	if !found {
		test.Fatal("integration scene has no actual terrain")
	}
}
