package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

// Each sampled cell becomes a finite sphere with half its cell diagonal as
// radius. Positions and integrated power follow the public placement rule.
func TestRetainedFireLampsKeepActualSourceCellsAndTheirFiniteRadius(t *testing.T) {
	for _, count := range []int{0, 3} {
		t.Run(fmt.Sprintf("patches%d", count), func(t *testing.T) {
			given := fireOnFixture(t)
			raw, err := os.ReadFile(given.set)
			if err != nil {
				t.Fatal(err)
			}
			if count > 0 {
				layout := "      layout:\n        pattern: cluster\n        count: 3\n        radius: {value: 0.55, unit: m}\n"
				text := strings.Replace(string(raw), "      flame:", layout+"      flame:", 1)
				if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			read := readBoundarySet(t, given.set)
			row, has := read.Recipe("wood fire")
			if !has {
				t.Fatal("fixture lost its actual fire recipe")
			}
			spec, has := row.Fire()
			if !has {
				t.Fatal("fixture no longer describes a volumetric fire")
			}
			burner, err := manufacture.Burn(spec)
			if err != nil {
				t.Fatal(err)
			}
			for _, scale := range []float64{1, 2} {
				place := maths.Subspace{Origin: maths.V(10, 2, 5), Scale: scale}
				volumes, _, err := burner.Place(place, 1e-5, 0)
				if err != nil {
					t.Fatal(err)
				}
				if len(volumes) == 0 {
					t.Fatal("actual public factory returned no fire medium")
				}
				field, ok := volumes[0].Medium.(emittingField)
				if !ok {
					t.Fatal("actual public factory returned an unsupported medium")
				}
				live := retainedFire{field: field, place: place, cells: spec.LightCells}
				for _, at := range []time.Duration{0, 137 * time.Millisecond} {
					var sources []flame.Source
					switch body := volumes[0].Medium.(type) {
					case *flame.Body:
						sources = body.Tick(at).(*flame.Body).Sources(spec.LightCells)
					case *flame.Field:
						sources = body.Tick(at).(*flame.Field).Sources(spec.LightCells)
					default:
						t.Fatal("fixture lost its body/field control")
					}
					lamps, err := live.lampsAt(at)
					if err != nil {
						t.Fatal(err)
					}
					if len(sources) == 0 || len(lamps) != len(sources) {
						t.Fatal("actual sampled cells and finite lamps no longer correspond")
					}
					power := 0.0
					for i, source := range sources {
						want := source.Placed(place)
						lamp, ok := lamps[i].(emit.Volume)
						if !ok {
							t.Fatal("cell did not become a finite source")
						}
						point := lamp.Point()
						if gap := point.At.Sub(want.At).Len(); !(gap <= 1e-12) {
							t.Fatalf("scale %g at %s: cell lamp moved %g m", scale, at, gap)
						}
						if gap := math.Abs(lamp.Radius().InMetres() - .5*want.Across); !(gap <= 1e-12*want.Across) {
							t.Fatalf("scale %g at %s: finite radius %g differs from half the sampled cell diagonal %g", scale, at, lamp.Radius().InMetres(), want.Across)
						}
						for _, pair := range [][2]float64{{point.Strength.R, want.Strength.R}, {point.Strength.G, want.Strength.G}, {point.Strength.B, want.Strength.B}} {
							if !(math.Abs(pair[0]-pair[1]) <= 1e-12*math.Max(1e-30, math.Abs(pair[1]))) {
								t.Fatal("finite cell lamp changed sampled emitted power")
							}
						}
						power += want.Strength.Luminance()
					}
					if !(power > 0) {
						t.Fatal("source controls emitted no actual light")
					}
				}
			}
		})
	}
}
