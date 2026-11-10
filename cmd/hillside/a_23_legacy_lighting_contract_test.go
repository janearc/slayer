package main

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// These are authored choices for the legacy synthetic hillside, not measured
// sunlight or lunar spectra. Inspect the constructed scene's actual lighting,
// palette and surface separation rather than reading the same default vars.
func TestLegacyBareSceneKeepsItsChosenLightingAndPalette(t *testing.T) {
	checkColour := func(t *testing.T, got, want maths.Light) {
		t.Helper()
		if !(math.Abs(got.R-want.R) <= 1e-12 && math.Abs(got.G-want.G) <= 1e-12 && math.Abs(got.B-want.B) <= 1e-12) {
			t.Fatalf("actual legacy scene colour %v differs from authored %v", got, want)
		}
	}
	for _, night := range []bool{false, true} {
		for _, earth := range []bool{false, true} {
			t.Run(fmt.Sprintf("night%v-earth%v", night, earth), func(t *testing.T) {
				given := mustParse(t)
				given.night, given.earth = night, earth
				world, err := bare(given, 20)
				if err != nil {
					t.Fatal(err)
				}
				if len(world.Lamps) != 2 {
					t.Fatal("bare scene lost its primary light or its fill")
				}
				primary, fillLight := 1.5, .375
				horizon, zenith, ground := maths.RGB(.7, .8, .95), maths.RGB(.25, .45, .9), maths.RGB(.34, .29, .18)
				if night {
					primary, fillLight = .03, .0075
					horizon, zenith, ground = maths.RGB(.02, .03, .06), maths.RGB(.005, .008, .02), maths.Light{}
				}
				choices := []struct {
					way   maths.Vec
					light float64
				}{
					{maths.V(-.5, .8, .45).Unit(), primary},
					{maths.V(.6, .5, -.3).Unit(), fillLight},
				}
				seen := map[int]bool{}
				for _, lamp := range world.Lamps {
					direction, distance, light := lamp.Light(maths.Vec{}, maths.YAxis)
					matched := false
					for i, choice := range choices {
						if !(direction.Sub(choice.way).Len() <= 1e-12) {
							continue
						}
						if seen[i] {
							t.Fatal("bare scene duplicated one light direction")
						}
						seen[i], matched = true, true
						checkColour(t, light, maths.Grey(choice.light))
					}
					if !matched {
						t.Fatal("bare scene changed a chosen light direction")
					}
					if !earth && !(math.Abs(distance-4000) <= 1e-9) {
						t.Fatalf("flat hillside's documented 4 km source moved to %g m", distance)
					}
				}
				if len(seen) != 2 || world.Sky == nil {
					t.Fatal("painted legacy scene lost its lighting or sky")
				}
				checkColour(t, world.Sky.Light(maths.V(1, 0, 0)), horizon)
				checkColour(t, world.Sky.Light(maths.V(0, 1, 0)), zenith)
				checkColour(t, world.Sky.Light(maths.V(0, -1, 0)), ground)
				soil, globe := 0, 0
				for _, item := range world.Things {
					if item.Material == nil {
						continue
					}
					if item.Material.Name == "soil" {
						soil++
						checkColour(t, item.Material.Colour, maths.RGB(.2, .17, .1))
					}
					if item.Material.Name == "earth" {
						globe++
						top := item.At(0).Out(maths.V(0, item.Shape.Bounds().Max.Y, 0)).Y
						if !(math.Abs(top+.05) <= 1e-8) {
							t.Fatalf("earth surface lost its authored 5 cm separation below the ground: %g", top)
						}
					}
				}
				wantGlobe := 0
				if earth {
					wantGlobe = 1
				}
				if soil != 1 || globe != wantGlobe {
					t.Fatal("bare scene lost its ground or earth control")
				}
			})
		}
	}
}

// This finite spacing reaches the actual mesh consumer through the real
// command flags. It overflows a face cross product, not an allocation count.
// Keep the ordinary empty hillside as a positive construction control.
func TestLegacyHillsideForwardsAnActualFiniteGroundOverflow(t *testing.T) {
	normal := mustParse(t, only(map[string]string{})...)
	if _, _, err := hillside(normal); err != nil {
		t.Fatalf("ordinary flat ground should construct: %v", err)
	}
	arguments := append(only(map[string]string{}), "-apart", "1.8333333333333333e154")
	large := mustParse(t, arguments...)
	if _, _, err := hillside(large); !errors.Is(err, mesh.ErrMesh) {
		t.Fatalf("finite oversized ground lost the actual mesh refusal: %v", err)
	}
}
