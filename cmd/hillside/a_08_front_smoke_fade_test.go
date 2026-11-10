package main

import (
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/scene/thing"
)

// At the midpoint of a symmetric fade its share is exactly one half.
// Two overlapping burns therefore give explicit quarter/half/full column
// strengths and centres without repeating the implementation's polynomial.
func TestFrontSmokeFadesAtIgnitionAndFollowsWeightedFuelCentres(t *testing.T) {
	type snapshot struct {
		at       time.Duration
		share, x float64
	}
	for _, study := range []struct {
		name, fade string
		samples    []snapshot
	}{
		{"no fade", "0", []snapshot{
			{2 * time.Second, .5, 10},
			{2500 * time.Millisecond, .5, 10},
			{3 * time.Second, 1, 10.5},
			{3500 * time.Millisecond, 1, 10.5},
			{4 * time.Second, .5, 11},
			{4500 * time.Millisecond, .5, 11},
			{5 * time.Second, 0, 0},
		}},
		{"one second", "1", []snapshot{
			{2 * time.Second, 0, 0},
			{2500 * time.Millisecond, .25, 10},
			{3 * time.Second, .5, 10},
			{3500 * time.Millisecond, .5, 10.5},
			{4 * time.Second, .5, 11},
			{4500 * time.Millisecond, .25, 11},
			{5 * time.Second, 0, 0},
		}},
		{"one nanosecond", "0.000000001", []snapshot{
			{2 * time.Second, 0, 0},
			{2*time.Second + time.Nanosecond, .5, 10},
			{3 * time.Second, .5, 10},
			{3*time.Second + time.Nanosecond, 1, 10.5},
			{4 * time.Second, .5, 11},
			{5 * time.Second, 0, 0},
		}},
	} {
		t.Run(study.name, func(t *testing.T) {
			given := frontOnFixture(t)
			raw, err := os.ReadFile(given.set)
			if err != nil {
				t.Fatal(err)
			}
			text := strings.Replace(string(raw), "kind: fern, count: 4", "kind: fern, count: 2", 1)
			text = strings.Replace(text, "speed: {value: 1000, unit: m/s}", "speed: {value: 1, unit: m/s}", 1)
			text = strings.Replace(text, "fade-time: {value: 0.2, unit: s}", "fade-time: {value: "+study.fade+", unit: s}", 1)
			if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			read := readBoundarySet(t, given.set)
			given.exposure = 1
			stage := staging{given: given, recipes: read, world: thing.NewWorld(),
				places: map[string]maths.Vec{"the tree": maths.V(10, 0, 5)},
				fuels:  map[string][]groveTree{"front fuel": {{at: maths.V(10, 0, 5)}, {at: maths.V(11, 0, 5)}}}}
			if err := stage.prepareFronts(read); err != nil {
				t.Fatal(err)
			}
			if len(stage.fronts) != 1 {
				t.Fatal("fixture lost front")
			}
			front := stage.fronts[0]
			row, has := read.Recipe("wood fire")
			if !has {
				t.Fatal("fixture lost recipe")
			}
			spec, has := row.Fire()
			if !has || spec.Smoke == nil {
				t.Fatal("fixture lost smoke")
			}
			smoke := *spec.Smoke
			smoke.Base = 16
			reference, err := plume.New(smoke)
			if err != nil {
				t.Fatal(err)
			}
			point := maths.V(0, 5, 0)
			density := reference.Density(point)
			if !(density > 0) {
				t.Fatal("reference missed plume")
			}
			for _, sample := range study.samples {
				world := thing.NewWorld()
				if err := front.addAt(world, &tally{}, sample.at); err != nil {
					t.Fatal(err)
				}
				count := 0
				for _, volume := range world.Volumes {
					if volume.Material.Name != "front smoke" {
						continue
					}
					count++
					if sample.share == 0 {
						t.Fatalf("zero faded occupancy emitted smoke at%s", sample.at)
					}
					want := density * sample.share
					if got := volume.Medium.Density(point); !(math.Abs(got-want) <= want*1e-12) {
						t.Fatalf("at%s: smoke density%.17g, want%.17g (share%g)", sample.at, got, want, sample.share)
					}
					expected := maths.V(sample.x, 0, 5)
					if gap := volume.At(sample.at).Origin.Sub(expected).Len(); !(gap < 1e-12) {
						t.Fatalf("at%s: smoke centre%v, want%v", sample.at, volume.At(sample.at).Origin, expected)
					}
				}
				if sample.share > 0 && count != 1 {
					t.Fatalf("at%s: got%d shared smoke columns, want1", sample.at, count)
				}
			}
		})
	}
}
