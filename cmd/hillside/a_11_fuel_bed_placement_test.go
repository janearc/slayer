package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/thing"
)

// A 16-by-8 half-metre bed centred at 1050E/8960N has the footprint
// [1046,1054]E by [8958,8962]N. Check actual meshes and patch origins.
func TestFuelBedKeepsItsRequestedTerrainCentreAndRectangularFootprint(t *testing.T) {
	for _, surface := range []string{"fuel", "terrain"} {
		t.Run(surface, func(t *testing.T) {
			given := fuelOnFixture(t)
			raw, err := os.ReadFile(given.set)
			if err != nil {
				t.Fatal(err)
			}
			text := strings.Replace(string(raw), "      width: 8", "      width: 16", 1)
			text = strings.Replace(text, "      model: mandel-grass", "      model: mandel-grass\n      surface: "+surface, 1)
			if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			read := readBoundarySet(t, given.set)
			given.onGPU = false
			given.seed = 7
			stage := staging{given: given, recipes: read, world: thing.NewWorld()}
			if err := stage.terrain(read); err != nil {
				t.Fatal(err)
			}
			if err := stage.prepareBeds(read); err != nil {
				t.Fatal(err)
			}
			defer (&liveScene{beds: stage.beds}).closeBeds()
			if len(stage.beds) != 1 {
				t.Fatal("fixture lost bed")
			}
			bed := stage.beds[0]
			// The shipped height fixture is affine. Build its plane from
			// public sample values rather than calling groundAt as oracle.
			wide, high, step := stage.ground.Size()
			base := stage.ground.At(0, 0)
			sx := (stage.ground.At(1, 0) - base) / step
			sz := (stage.ground.At(0, 1) - base) / step
			far := base + sx*float64(wide-1)*step + sz*float64(high-1)*step
			if !(math.Abs(stage.ground.At(wide-1, high-1)-far) <= 1e-9) {
				t.Fatal("height fixture lost its independently specified affine plane")
			}
			if surface == "fuel" {
				if len(bed.tiles) != 128 {
					t.Fatalf("got%d tiles, want128", len(bed.tiles))
				}
				lowE, highE, lowN, highN := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
				for _, tile := range bed.tiles {
					bounds := tile.Bounds()
					if tile.FaceCount() != 2 || len(tile.Points()) != 4 ||
						!(math.Abs(bounds.Max.X-bounds.Min.X-.5) <= 1e-12) ||
						!(math.Abs(bounds.Max.Z-bounds.Min.Z-.5) <= 1e-12) {
						t.Fatal("fuel cell lost its half-metre four-corner, two-face footprint")
					}
					for _, point := range tile.Points() {
						// The current fuel decal uses an authored 1.5 cm lift.
						ground := base + sx*point.X + sz*point.Z - stage.lowest
						if !(math.Abs(point.Y-ground-.015) <= 1e-9) {
							t.Fatalf("fuel surface height %g lost its clearance above ground %g", point.Y, ground)
						}
						east, north := stage.west+point.X, stage.north-point.Z
						lowE, highE = math.Min(lowE, east), math.Max(highE, east)
						lowN, highN = math.Min(lowN, north), math.Max(highN, north)
					}
				}
				if !(math.Abs(lowE-1046) <= 1e-10 && math.Abs(highE-1054) <= 1e-10 &&
					math.Abs(lowN-8958) <= 1e-10 && math.Abs(highN-8962) <= 1e-10) {
					t.Fatalf("fuel footprint E[%g,%g] N[%g,%g] differs from requested rectangle", lowE, highE, lowN, highN)
				}
			} else if len(bed.tiles) != 0 {
				t.Fatal("terrain surface acquired extra fuel geometry")
			}
			// Final coordinates and seeds come from the separate C producer
			// in testdata. These pin authored replay choices, not fire data.
			vectors := map[int]struct {
				east, north float64
				seed        uint64
			}{
				0:   {1046.401795, 8961.6775500000003, 12918135221727111561},
				1:   {1046.72109, 8961.8605650000009, 16731224329868871185},
				17:  {1046.9155149999999, 8961.26253, 14106777335606723636},
				63:  {1053.763475, 8960.3126850000008, 12123873080370884649},
				96:  {1046.3561199999999, 8958.8837349999994, 7075973594347087881},
				127: {1053.8639250000001, 8958.1774449999994, 9196242371363065103},
			}
			seen := map[[2]int]bool{}
			for _, group := range bed.groups {
				for _, piece := range group.field.Pieces() {
					point := group.anchor.Add(piece.At)
					ground := base + sx*point.X + sz*point.Z - stage.lowest
					if !(math.Abs(point.Y-ground-.025) <= 1e-9) {
						t.Fatal("flame origin lost its authored2.5cm terrain clearance")
					}
					east, north := stage.west+point.X, stage.north-point.Z
					column := int(math.Floor((east - 1046) / .5))
					row := int(math.Floor((8962 - north) / .5))
					cell := [2]int{column, row}
					if column < 0 || column >= 16 || row < 0 || row >= 8 || seen[cell] {
						t.Fatalf("flame patch E%g N%g escaped or duplicated cell%v", east, north, cell)
					}
					seen[cell] = true
					if want, has := vectors[row*16+column]; has {
						fire, _, _, _ := piece.Body.Samples()
						if !(math.Abs(east-want.east) <= 1e-10 && math.Abs(north-want.north) <= 1e-10) || fire.Seed != want.seed {
							t.Fatalf("patch %v lost its authored replay position or flame seed: E%.17g N%.17g seed%d", cell, east, north, fire.Seed)
						}
					}
				}
			}
			if len(seen) != 128 {
				t.Fatalf("retained%d patch cells, want128", len(seen))
			}
		})
	}
}

// A rectangle may touch each terrain edge but may not cross any one edge.
// Keep the other axis central so an incorrect AND cannot mask a refusal.
func TestFuelBedChecksEveryTerrainEdgeIndependently(t *testing.T) {
	given := fuelOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	base := strings.Replace(string(raw), "      width: 8", "      width: 16", 1)
	read := readBoundarySet(t, given.set)
	ground := staging{given: given, world: thing.NewWorld()}
	if err := ground.terrain(read); err != nil {
		t.Fatal(err)
	}
	w, h, spacing := ground.ground.Size()
	minE, maxE := ground.west, ground.west+float64(w-1)*spacing
	maxN, minN := ground.north, ground.north-float64(h-1)*spacing
	centreE, centreN := (minE+maxE)/2, (minN+maxN)/2
	for _, edge := range []struct {
		name                string
		east, north, dx, dn float64
	}{
		{"west", minE + 4, centreN, -.25, 0},
		{"east", maxE - 4, centreN, .25, 0},
		{"north", centreE, maxN - 2, 0, .25},
		{"south", centreE, minN + 2, 0, -.25},
	} {
		for _, cross := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/cross=%v", edge.name, cross), func(t *testing.T) {
				east, north := edge.east, edge.north
				if cross {
					east += edge.dx
					north += edge.dn
				}
				text := strings.Replace(base, "\n      east: 1050\n", fmt.Sprintf("\n      east: %.17g\n", east), 1)
				text = strings.Replace(text, "\n      north: 8960\n", fmt.Sprintf("\n      north: %.17g\n", north), 1)
				if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
				parsed := readBoundarySet(t, given.set)
				beds := parsed.FuelBeds()
				if len(beds) != 1 || beds[0].East() != east || beds[0].North() != north || beds[0].Width() != 16 {
					t.Fatal("fixture did not encode the requested boundary rectangle")
				}
				asked := given
				asked.onGPU = false
				stage := staging{given: asked, recipes: parsed, world: thing.NewWorld()}
				if err := stage.terrain(parsed); err != nil {
					t.Fatal(err)
				}
				err := stage.prepareBeds(parsed)
				defer (&liveScene{beds: stage.beds}).closeBeds()
				if cross {
					if err == nil || !strings.Contains(err.Error(), "fuel bed is off the ground") {
						t.Fatalf("rectangle crossing only%s edge should be refused: %v", edge.name, err)
					}
				} else if err != nil {
					t.Fatalf("rectangle touching%s edge should fit: %v", edge.name, err)
				}
			})
		}
	}
}
