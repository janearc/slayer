package biome_test

import (
	"testing"

	"github.com/janearc/slayer/biome"
	"github.com/janearc/slayer/maths/hexes"
)

// band is the first row of bands.csv a height above the sea and a wet
// put a tile in.
func band(test testing.TB, height, wet float64) map[string]string {
	test.Helper()
	for _, row := range read(test, bandsFile) {
		edge := float(test, row, "edge")
		switch row["when"] {
		case "height below":
			if height < edge {
				return row
			}
		case "height above":
			if height > edge {
				return row
			}
		case "wet below":
			if wet < edge {
				return row
			}
		case "always":
			return row
		default:
			test.Fatalf("%s: %q is no when", bandsFile, row["when"])
		}
	}
	test.Fatalf("%s: no band holds every tile", bandsFile)
	return nil
}

// a tile is painted by the band its middle's height above the sea and
// its wet put it in, and the test map holds every band.
func TestATileIsPaintedByItsBand(t *testing.T) {
	made := fields(t, picked(t, "risen"))
	seen := map[string]bool{}
	for _, tile := range made.Tiling.Tiles() {
		row := band(t, above(made, tile), wet(made, tile))
		seen[row["name"]] = true
		if got, want := made.Paint(tile), terran(whole(t, row, "paint")); got != want {
			t.Errorf("tile %v, %v above the sea and %v wet, is painted %v, "+
				"not %s's %v", tile, above(made, tile), wet(made, tile),
				got, row["name"], want)
		}
	}
	for _, row := range read(t, bandsFile) {
		if !seen[row["name"]] {
			t.Errorf("the test map holds no %s", row["name"])
		}
	}
}

// anti-tests: no tile under the sea is painted as land, and no tile
// above it as sea.
func TestNoTileIsPaintedOnTheWrongSideOfTheSea(t *testing.T) {
	made := fields(t, picked(t, "risen"))
	for _, tile := range made.Tiling.Tiles() {
		under := above(made, tile) < nought
		if water(made.Paint(tile)) != under {
			t.Errorf("tile %v, %v above the sea, is painted %v", tile,
				above(made, tile), made.Paint(tile))
		}
	}
}

// tiles is every tile of a world's tiling, for a loop.
func tiles(made *biome.World) []hexes.Tile { return made.Tiling.Tiles() }
