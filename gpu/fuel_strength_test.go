//go:build darwin

package gpu

import (
	"github.com/janearc/slayer/flame"
	"math"
	"testing"
)

func TestFuelStrengthsBakeTheirActualAbsorptionAndHeight(t *testing.T) {
	grid := compositeFireGrid(t, 0)
	field, err := grid.med.(*flame.Field).WithStrengths([]float64{.125, 0, .8})
	if err != nil {
		t.Fatal(err)
	}
	grid.med = field
	grid.shine = field.Shine()
	density, shine, _, err := bakeFlame(grid)
	if err != nil {
		t.Fatal(err)
	}
	truth, glow, _ := grid.asked()
	total, apart := 0.0, 0.0
	for i, d := range truth {
		for c := range 3 {
			want := float64(d) * float64(glow[3*i+c])
			got := float64(density[i]) * float64(shine[3*i+c])
			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Fatal("non-finite GPU emission")
			}
			total += math.Abs(want)
			apart += math.Abs(want - got)
		}
	}
	if total <= 0 || apart/total > 1e-4 {
		t.Fatal("fuel field CPU/GPU mismatch", apart/total)
	}
	dead, err := field.WithStrengths([]float64{0, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	grid.med = dead
	grid.shine = dead.Shine()
	if _, _, _, err = bakeFlame(grid); err == nil {
		t.Fatal("empty field should be omitted by staging")
	}
}
