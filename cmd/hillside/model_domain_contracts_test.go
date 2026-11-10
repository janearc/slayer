package main

import (
	"errors"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/firefront"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/lens"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/scene/thing"
)

func TestStagingPropagatesTheLensMinimumFocusDistance(t *testing.T) {
	given := mustParse(t)
	given.set = onFixtureRigged(t, "0.5", false)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "focus: the tree", "focus: the stone", 1)
	text = strings.Replace(text, "standoff_z: 2", "standoff_z: 0.01", 1)
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	frame, err := lens.FrameOf(70.41, 51.61)
	if err != nil {
		t.Fatal(err)
	}
	_, cause := lens.NewThin(frame, 672, 8, .01)
	if cause == nil {
		t.Fatal("focus control is not inside focal length")
	}
	if _, _, _, err := staged(given); err == nil || !strings.Contains(err.Error(), cause.Error()) {
		t.Fatalf("staging lost lens refusal: %v", err)
	}
}

func TestPlanetAssemblyRefusesARingInsideTheBody(t *testing.T) {
	extra := "    rings:\n      - {inner: 1e6, outer: 2e6, depth: 0.1, albedo: 0.6, lean: 0, made_of: {ice: 1}}\n"
	given := mustParse(t)
	given.set = inSpace(t, "    zones:", extra+"    zones:")
	read := readBoundarySet(t, given.set)
	made, has := read.Planet()
	if !has {
		t.Fatal("missing planet")
	}
	stage := staging{given: given, world: thing.NewWorld()}
	if _, err := stage.body(made); err == nil || !strings.Contains(err.Error(), "inside") {
		t.Fatalf("body admitted an interior ring: %v", err)
	}
}

func TestPlanetLampConstructionPropagatesNonfiniteExposure(t *testing.T) {
	given := mustParse(t)
	given.set = inSpace(t, "", "")
	read := readBoundarySet(t, given.set)
	made, has := read.Planet()
	if !has {
		t.Fatal("missing planet")
	}
	given.exposure = math.NaN()
	stage := staging{given: given, world: thing.NewWorld(), places: map[string]maths.Vec{made.Name(): {}}}
	if err := stage.planetLit(read, made, maths.ZAxis.Scale(1e9)); err == nil || !strings.Contains(err.Error(), "distant") {
		t.Fatalf("planet lost distant-light refusal: %v", err)
	}
}

func TestRingCollectionPreservesTheMoonletMassRefusal(t *testing.T) {
	extra := "    rings:\n      - inner: 1e8\n        outer: 2e8\n        depth: 0.1\n        albedo: 0.6\n        lean: 0\n        made_of: {ice: 1}\n        moonlets: {seed: 3, count: 1, smallest: 1, largest: 2, slope: 2, opens: 1}\n"
	made := weatherBoundaryPlanet(t, extra)
	if _, err := ringsOf(made, 1e27); err != nil {
		t.Fatal(err)
	}
	_, cause := ringOf(made, made.Rings()[0], 0)
	if cause == nil {
		t.Fatal("zero mass control accepted")
	}
	if _, err := ringsOf(made, 0); err == nil || err.Error() != cause.Error() {
		t.Fatalf("ring collection lost mass refusal: %v", err)
	}
}

// This is a retained-model unit boundary, not a scene-reader refusal.
func TestFrontSmokeRebuildPreservesAnInvalidSourceCause(t *testing.T) {
	plan, err := firefront.New(firefront.Spec{Direction: maths.XAxis, Speed: 1, Cell: 1, Duration: time.Second}, []firefront.Target{{}})
	if err != nil {
		t.Fatal(err)
	}
	invalid := plume.Fire{}
	_, cause := plume.New(invalid)
	if cause == nil {
		t.Fatal("invalid plume control accepted")
	}
	front := retainedFront{plan: plan, peakFuel: 1, smoke: &thing.Volume{}, smokeSpec: &invalid}
	if err := front.addAt(thing.NewWorld(), &tally{}, 0); !errors.Is(err, cause) {
		t.Fatalf("front lost plume cause: %v", err)
	}
}
