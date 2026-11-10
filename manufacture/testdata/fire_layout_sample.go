// This producer records authored source placement, phase and height choices.
package main

import (
	"fmt"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

func main() {
	for _, pattern := range []string{"cluster", "line"} {
		spec := manufacture.BurnSpec{
			Flame:      flame.Fire{Radius: .6, Height: 2.4, Absorption: 1, Hottest: measure.Kelvin(1603), Edge: measure.Kelvin(900), Seed: 5},
			LightCells: 3, Layout: manufacture.FireLayout{Pattern: pattern, Count: 3, Variation: .3},
		}
		if pattern == "cluster" {
			spec.Layout.Radius = measure.Metres(.8)
		} else {
			spec.Layout.Length = measure.Metres(4)
		}
		made, err := manufacture.Burn(spec)
		if err != nil {
			panic(err)
		}
		volumes, _, err := made.Place(maths.Subspace{}, 1, 0)
		if err != nil {
			panic(err)
		}
		field, ok := volumes[0].Medium.(*flame.Field)
		if !ok {
			panic("layout produced no source field")
		}
		for i, piece := range field.Pieces() {
			fire, _, _, _ := piece.Body.Samples()
			fmt.Printf("%s %d %d %.17g %.17g %.17g %.17g\n", pattern, i, fire.Seed, piece.At.X, piece.At.Y, piece.At.Z, fire.Height)
		}
	}
}
