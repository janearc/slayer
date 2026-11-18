package biome_test

import (
	"reflect"
	"testing"

	"github.com/janearc/slayer/biome"
	"github.com/janearc/slayer/maths/hexes"
)

// Reading placement metadata must not regenerate or recolour the legacy map.
func TestBiomeFeaturesExposeExistingWorldWithoutChangingItsPaint(t *testing.T) {
	world := biome.Make(biome.Fields{Tiling: hexes.Sphere{Splits: 24}, Seed: 31,
		Sea: -.08, Scale: .8})
	features := world.Features()
	volcanoes, cities := 0, 0
	for _, feature := range features {
		if feature.Volcano {
			volcanoes++
		}
		if feature.City {
			cities++
		}
		if feature.At != world.Tiling.Centre(feature.Tile) || feature.HeightAboveSea != world.Height(feature.At)-world.Sea {
			t.Fatal("feature coordinates or relief diverged")
		}
	}
	if volcanoes != world.Volcanoes || cities != world.Cities {
		t.Fatal("feature source maps diverged")
	}
	if !reflect.DeepEqual(features, world.Features()) {
		t.Fatal("feature order changed")
	}
	if len(features) == 0 {
		t.Fatal("empty feature test fixture")
	}
	tile := features[0].Tile
	paint, glow := world.Paint(tile), world.Shine().Paint(tile)
	features[0].City = !features[0].City
	if reflect.DeepEqual(features, world.Features()) {
		t.Fatal("metadata exposes mutable storage")
	}
	if world.Paint(tile) != paint || world.Shine().Paint(tile) != glow {
		t.Fatal("metadata changed legacy output")
	}
}
