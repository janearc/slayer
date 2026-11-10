//go:build darwin

package gpu

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
)

// Rebase authored64-patch fields without changing their represented world.
// This holds the existing composite density/emission budget on representative
// grids; it does not claim a universal scene or whole-image error bound.
func TestFuelFieldRebasingKeepsCompositeAbsorptionAndEmissionBudget(t *testing.T) {
	const tolerance = 1e-4 // Existing composite field absorption/emission budget.
	for _, cell := range []float64{.25, .5, 2} {
		for _, strength := range []float64{.01, 1} {
			var patches []flame.Ignition
			for z := 0; z < 8; z++ {
				for x := 0; x < 8; x++ {
					fire := flame.Fire{Radius: .45, Height: 1.5 * math.Cbrt(strength),
						Absorption: strength, Hottest: measure.Kelvin(1750),
						Edge: measure.Kelvin(900), Seed: uint64(7 + z*8 + x)}
					patches = append(patches, flame.Ignition{
						At: maths.V((float64(x)+.57)*cell,
							1.25+.03*float64(x+z), (float64(z)+.39)*cell), Fire: fire})
				}
			}
			base, err := flame.NewField(patches, 1e-5)
			if err != nil {
				t.Fatal(err)
			}
			base = base.Tick(137 * time.Millisecond).(*flame.Field)
			original := boxOf(base, nil, base.Shine())
			productionCells := original.cells
			// Use a modest conformance grid, as the existing composite test
			// does. Also compare the untouched production dimension choices.
			original.cells = [3]int{37, 29, 31}
			density, glow, _, err := bakeFlame(original)
			if err != nil {
				t.Fatal(err)
			}
			shifts := []maths.Vec{
				maths.V(-48*cell, 0, 0), maths.V(24/cell-24*cell, 0, 0),
				maths.V(0, 0, -48*cell), maths.V(0, 0, 24/cell-24*cell),
				maths.V(0, 1, 0),
			}
			for _, shift := range shifts {
				moved := append([]flame.Ignition(nil), patches...)
				for i := range moved {
					moved[i].At = moved[i].At.Sub(shift)
				}
				field, err := flame.NewField(moved, 1e-5)
				if err != nil {
					t.Fatal(err)
				}
				field = field.Tick(137 * time.Millisecond).(*flame.Field)
				grid := boxOf(field, nil, field.Shine())
				if grid.cells != productionCells {
					t.Fatalf("rebasing changed production grid dimensions: cell%g strength%g shift%v: %v versus %v",
						cell, strength, shift, grid.cells, productionCells)
				}
				grid.cells = original.cells
				got, emitted, _, err := bakeFlame(grid)
				if err != nil {
					t.Fatal(err)
				}
				total, apart, light, lightApart := 0.0, 0.0, 0.0, 0.0
				for i, d := range density {
					total += math.Abs(float64(d))
					apart += math.Abs(float64(d) - float64(got[i]))
					for c := range 3 {
						want := float64(d) * float64(glow[3*i+c])
						value := float64(got[i]) * float64(emitted[3*i+c])
						light += math.Abs(want)
						lightApart += math.Abs(want - value)
					}
				}
				t.Logf("cell=%g strength=%g anchorDelta=%v densityRelative=%.12g emissionRelative=%.12g productionCells=%v",
					cell, strength, shift, apart/total, lightApart/light, productionCells)
				if !(total > 0 && light > 0 && apart/total <= tolerance && lightApart/light <= tolerance) {
					t.Fatal("rebased field exceeded the existing composite numeric budget")
				}
			}
		}
	}
}
