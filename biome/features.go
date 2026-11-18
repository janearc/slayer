package biome

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// Feature exposes existing authored world metadata for geometry placement.
// At uses the tiling's coordinates, not metres: unit directions on Sphere,
// surface positions on Band/Flat. HeightAboveSea is the dimensionless field
// value, which the caller maps to its chosen physical relief. River marks
// path tiles; Lake marks their terminal shallow-water tiles. Highland marks
// existing rock/snow/ash classification, not a measured mountain summit.
type Feature struct {
	Tile                                 hexes.Tile
	At                                   maths.Vec
	HeightAboveSea                       float64
	River, Lake, Volcano, City, Highland bool
}

// Features returns owned records in stable tiling order, leaving generation,
// Paint and Shine unchanged. Volcano and City are the exact retained source
// maps; river tile counts differ from the number of river-source walks.
func (world *World) Features() []Feature {
	var features []Feature
	for _, tile := range world.Tiling.Tiles() {
		kind := world.biome[tile]
		feature := Feature{Tile: tile, At: world.Tiling.Centre(tile),
			River: world.river[tile], Lake: world.river[tile] && kind == shallowSea,
			Volcano: world.lava[tile], City: world.city[tile],
			Highland: kind == rock || kind == snow || kind == ash}
		if feature.River || feature.Volcano || feature.City || feature.Highland {
			feature.HeightAboveSea = world.Height(feature.At) - world.Sea
			features = append(features, feature)
		}
	}
	return features
}

// WaterAt reports mapped ocean, lake or river at a world position. It uses the
// retained hydrology, so added physical relief can leave those surfaces open.
func (world *World) WaterAt(at maths.Vec) bool {
	tile := world.Tiling.Nearest(at)
	kind, known := world.biome[tile]
	return !known || kind <= shallowSea || world.river[tile]
}
