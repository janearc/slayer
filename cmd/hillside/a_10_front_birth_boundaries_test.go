package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/embers"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Each fixture uses a checked recipe, real source and valid uint64 CLI seed.
func particleBoundaryFront(t *testing.T, seed string) (retainedFront, embers.Spec) {
	t.Helper()
	fixture := frontOnFixture(t)
	raw, err := os.ReadFile(fixture.set)
	if err != nil {
		t.Fatal(err)
	}
	night, err := os.ReadFile("../../doc/fire-night.yaml")
	if err != nil {
		t.Fatal(err)
	}
	first := strings.Index(string(night), "      sparks:\n")
	if first < 0 {
		t.Fatal("night scene lost sparks")
	}
	next := strings.Index(string(night[first:]), "      light-cells:")
	if next < 0 {
		t.Fatal("night scene lost following light-cells")
	}
	text := strings.Replace(string(raw), "      flame:", string(night[first:first+next])+"      flame:", 1)
	text = strings.Replace(text, "kind: fern, count: 4", "kind: fern, count: 1", 1)
	if err := os.WriteFile(fixture.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	read := readBoundarySet(t, fixture.set)
	given := mustParse(t, "-seed", seed)
	given.set, given.exposure = fixture.set, 1
	stage := staging{given: given, recipes: read, world: thing.NewWorld(),
		places: map[string]maths.Vec{"the tree": {}},
		fuels:  map[string][]groveTree{"front fuel": {{at: maths.Vec{}}}}}
	if err := stage.prepareFronts(read); err != nil {
		t.Fatal(err)
	}
	if len(stage.fronts) != 1 || len(stage.fronts[0].sparks) != 1 {
		t.Fatal("fixture lost target")
	}
	front := stage.fronts[0]
	groups := front.plan.Groups()
	if len(groups) != 1 || len(groups[0]) != 1 || groups[0][0].Start != 2*time.Second {
		t.Fatal("fixture lost exact ignition")
	}
	row, has := read.Recipe("wood fire")
	if !has {
		t.Fatal("fixture lost fire")
	}
	spec, has := row.Fire()
	if !has || spec.Sparks == nil {
		t.Fatal("fixture lost sparks")
	}
	return front, *spec.Sparks
}

// The independent C producer verifies that this CLI seed gives target seed0.
// Birth0 is then exactly at ignition, rather than almost surely between frames.
func TestFrontRetainsAnActualParticleBornExactlyAtIgnition(t *testing.T) {
	front, spec := particleBoundaryFront(t, "4354685564936845354")
	if front.plan.Groups()[0][0].Seed != 0 {
		t.Fatal("fixture lost the independently specified zero seed")
	}
	spec.Seed = 0
	reference, err := embers.New(spec)
	if err != nil {
		t.Fatal(err)
	}
	born := reference.At(0)
	if len(born) != 1 || born[0].ID() != 0 || born[0].Age() != 0 {
		t.Fatal("public source did not produce the exact birth0 control")
	}
	for _, sample := range []struct {
		at    time.Duration
		count int
	}{
		{2*time.Second - time.Nanosecond, 0}, {2 * time.Second, 1},
	} {
		world, made := thing.NewWorld(), tally{}
		if err := front.addAt(world, &made, sample.at); err != nil {
			t.Fatal(err)
		}
		if len(world.Things) != sample.count || made.sparks != sample.count {
			t.Fatalf("at%s: got%d particles/tally%d, want%d", sample.at, len(world.Things), made.sparks, sample.count)
		}
		if sample.count == 1 {
			actual := world.Things[0].At(0).Origin
			if gap := actual.Sub(born[0].Position()).Len(); !(gap < 1e-12) {
				t.Fatalf("first birth moved: %v", actual)
			}
		}
	}
}

// The C producer inverts the mixer to put birth320 exactly at2s. At160/s,
// the [0,2s) source admits IDs0..319 and excludes that deadline birth.
func TestFrontStopsNewBirthsExactlyAtBurnout(t *testing.T) {
	front, spec := particleBoundaryFront(t, "8268544622336750318")
	const targetSeed uint64 = 4226584411067704768
	if front.plan.Groups()[0][0].Seed != targetSeed || spec.Rate != 160 || spec.Life != 4*time.Second {
		t.Fatal("fixture lost the independently specified deadline source")
	}
	spec.Seed = targetSeed
	reference, err := embers.New(spec)
	if err != nil {
		t.Fatal(err)
	}
	born := reference.At(2 * time.Second)
	if len(born) != 321 || born[320].ID() != 320 || born[320].Age() != 0 {
		t.Fatal("reference did not expose the exact deadline birth")
	}
	world, made := thing.NewWorld(), tally{}
	if err := front.addAt(world, &made, 4*time.Second); err != nil {
		t.Fatal(err)
	}
	if len(world.Things) != 320 || made.sparks != 320 {
		t.Fatalf("deadline birth admitted: got%d particles/tally%d, want320", len(world.Things), made.sparks)
	}
	for i, particle := range born[:320] {
		if gap := world.Things[i].At(0).Origin.Sub(particle.Position()).Len(); !(gap < 1e-12) {
			t.Fatalf("airborne birth%d was lost or replaced at burnout", particle.ID())
		}
	}
}
