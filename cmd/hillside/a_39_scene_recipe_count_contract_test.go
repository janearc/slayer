package main

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestSceneRecipeAcceptsItsExactPublicStoredFaceBudget(t *testing.T) {
	for _, name := range []string{"can", "coast-redwood"} {
		t.Run(name, func(t *testing.T) {
			given := mustParse(t)
			kind, err := manufacture.KindNamed(name)
			if err != nil {
				t.Fatal(err)
			}
			object, err := manufacture.MakeWithComplexity(kind,
				manufacture.Sow(stir(given.seed, 7)), "quick")
			if err != nil {
				t.Fatal(err)
			}
			parts, err := object.Things(maths.Subspace{})
			if err != nil {
				t.Fatal(err)
			}
			stored, logical := publicGeometryCounts(t, parts)
			if !(stored > 1 && logical >= stored) {
				t.Fatal("reference lost its real stored and placed faces")
			}
			if name == "coast-redwood" && logical <= stored {
				t.Fatal("reference lost its actual repeated prototype savings")
			}
			for _, limit := range []int{stored, stored - 1} {
				given.faces = limit
				stage := staging{given: given, world: thing.NewWorld(), have: stock{}, radii: map[string]float64{}}
				_, err := stage.putRecipe(name, "", "quick", 7, 0, maths.Vec{}, false, name)
				if limit < stored {
					if !errors.Is(err, errTooMany) || len(stage.world.Things) != 0 || stage.made.things != 0 {
						t.Fatalf("one-face-short actual recipe lost its pre-placement refusal: %v", err)
					}
					continue
				}
				if err != nil || stage.made.things != 1 || stage.made.kept != stored || stage.made.faces != logical ||
					len(stage.world.Things) != len(parts) {
					t.Fatalf("exact actual geometry budget failed or lost counts: %v, %+v; public stored%d/logical%d", err, stage.made, stored, logical)
				}
			}
		})
	}
}

func TestSceneRecipeReportsItsActualCarriedAndFireLights(t *testing.T) {
	for _, name := range []string{"spotlight", "spruce"} {
		t.Run(name, func(t *testing.T) {
			given := mustParse(t)
			stage := staging{given: given, world: thing.NewWorld(), have: stock{}, radii: map[string]float64{}}
			burning := name == "spruce"
			_, err := stage.putRecipe(name, "", "quick", 7, 0, maths.Vec{}, burning, name)
			if err != nil {
				t.Fatal(err)
			}
			if len(stage.world.Lamps) == 0 || stage.made.lamps != len(stage.world.Lamps) {
				t.Fatalf("%s reported %d lamps for %d actual sources", name, stage.made.lamps, len(stage.world.Lamps))
			}
			if burning && (stage.made.burning != 1 || len(stage.world.Volumes) != 2) {
				t.Fatal("single requested burning specimen lost its actual flame or smoke")
			}
			if !burning && (stage.made.burning != 0 || len(stage.world.Volumes) != 0) {
				t.Fatal("carried-light control ignited a specimen")
			}
		})
	}
}
