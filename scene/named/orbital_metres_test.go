package named

import (
	"github.com/janearc/slayer/scene/thing"
	"io"
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
)

func TestOrbitalMetreFloorUsesPhysicalCellWidthOnBothBands(t *testing.T) {
	for _, radius := range []float64{orbitalRadius, orbitalRadius * innerShare} {
		floor := newOrbitalMetreFloor(radius, 10, orbitalSeed)
		if got := floor.tiles.Spacing() * 1e6; math.Abs(got-10) > 1e-7 {
			t.Fatalf("cell across flats = %.12g m, want 10 m", got)
		}
		// Nearest/Centre work on the implicit lattice. Enumerating Tiles at
		// this physical scale would require quadrillions of entries.
		at := maths.V(0, 0, radius)
		first := floor.tiles.Nearest(at)
		second := floor.tiles.Nearest(maths.V(1e-5, 0, radius))
		if first == second {
			t.Fatal("moving ten metres did not reach a new cell")
		}
	}
}

func TestOrbitalMetreFloorFiltersUnresolvedCells(t *testing.T) {
	floor := newOrbitalMetreFloor(orbitalRadius, 10, orbitalSeed)
	for _, glowing := range []bool{false, true} {
		floor.glowing = glowing
		for i := 0; i < 20; i++ {
			a := float64(i) * .3
			p := maths.V(orbitalRadius*math.Sin(a), 0, orbitalRadius*math.Cos(a))
			q := p.Add(maths.V(1e-5, 0, 0))
			one := texture.Sample(floor, p, 3)
			two := texture.Sample(floor, q, 3)
			for _, delta := range []float64{one.R - two.R, one.G - two.G, one.B - two.B} {
				if math.IsNaN(delta) || math.Abs(delta) > 1e-5 {
					t.Fatalf("unresolved ten-metre cell caused sparkle: %v vs %v", one, two)
				}
			}
		}
	}
}

func TestOrbitalMetreParameterIsBounded(t *testing.T) {
	scene := orbitalScene()
	for _, metres := range []int{0, 1, 10, 1000000} {
		if err := scene.Check(map[string]any{"floor-hex-metres": metres}); err != nil {
			t.Fatal(err)
		}
	}
	for _, metres := range []any{-1, .5, 1000001, math.Inf(1)} {
		if err := scene.Check(map[string]any{"floor-hex-metres": metres}); err == nil {
			t.Fatalf("accepted unsafe cell width %v", metres)
		}
	}
}

// A physical-cell proof must retain the authored coastal city network. It
// must not replace dark ocean and rural regions with a generic land glow.
func TestOrbitalMetreFloorRetainsCitiesAndDarkOcean(t *testing.T) {
	world := thing.NewWorld()
	if err := (hoop{radius: orbitalRadius, sense: 1}).place(world, 0, &thing.Material{}, 24*time.Hour, 512, 10, io.Discard); err != nil {
		t.Fatal(err)
	}
	paint, ok := world.Things[0].Material.Texture.(orbitalMetreFloor)
	if !ok || paint.geography == nil {
		t.Fatal("physical cells lost the finite geography")
	}
	if paint.geography.Rivers == 0 || paint.geography.Cities == 0 {
		t.Fatal("physical floor lacks rivers or coastal cities")
	}
	glow := paint
	glow.glowing = true
	lit, darkWater := 0, 0
	for _, tile := range paint.macro.Tiles() {
		p := paint.macro.Centre(tile)
		city := paint.geography.Shine().Paint(tile).Luminance()
		got := texture.Sample(glow, p, 0).Luminance()
		if city > 0 {
			if got <= 0 {
				t.Fatal("city or lava light was lost")
			}
			lit++
		} else if paint.geography.Height(p) < floorSea {
			if got != 0 {
				t.Fatal("dark ocean became generic light")
			}
			darkWater++
		}
	}
	if lit == 0 || darkWater == 0 {
		t.Fatalf("fixture has %d lit cells and %d dark ocean cells", lit, darkWater)
	}
}
