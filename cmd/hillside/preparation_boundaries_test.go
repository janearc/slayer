package main

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestFrontAndBedPreparationKeepTheInvalidPaletteCause(t *testing.T) {
	for _, kind := range []string{"front", "front sparks", "bed"} {
		t.Run(kind, func(t *testing.T) {
			given := frontOnFixture(t)
			if kind == "bed" {
				given = fuelOnFixture(t)
			}
			if kind == "front sparks" {
				study, err := os.ReadFile("../../doc/fire-night.yaml")
				if err != nil {
					t.Fatal(err)
				}
				start := strings.Index(string(study), "      sparks:\n")
				if start < 0 {
					t.Fatal("night scene lost its sparks")
				}
				end := strings.Index(string(study[start:]), "      light-cells:")
				if end < 0 {
					t.Fatal("night scene lost its light sampling setting")
				}
				raw, err := os.ReadFile(given.set)
				if err != nil {
					t.Fatal(err)
				}
				text := strings.Replace(string(raw), "      flame:", string(study[start:start+end])+"      flame:", 1)
				if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			read := readBoundarySet(t, given.set)
			given.exposure = math.NaN()
			stage := staging{given: given, world: thing.NewWorld(), recipes: read,
				places: map[string]maths.Vec{"the tree": {}},
				fuels:  map[string][]groveTree{"front fuel": {{at: maths.Vec{}}}}}
			var err error
			if kind == "bed" {
				if err := stage.terrain(read); err != nil {
					t.Fatal(err)
				}
				err = stage.prepareBeds(read)
			} else {
				err = stage.prepareFronts(read)
			}
			_, cause := flame.Colours(flame.ColourSpec{Exposure: math.NaN()})
			if cause == nil || err == nil || err.Error() != cause.Error() {
				t.Fatalf("%s lost palette cause: %v", kind, err)
			}
			if len(stage.fronts) != 0 || len(stage.beds) != 0 {
				t.Fatal("failed preparation retained a model")
			}
		})
	}
}
