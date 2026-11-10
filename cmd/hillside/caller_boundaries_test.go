package main

import (
	"github.com/janearc/slayer/set"
	"math"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestStagedRefusesAbsentFilmAndCameraAfterReadingTheSet(t *testing.T) {
	for _, missing := range []string{"film", "camera"} {
		t.Run(missing, func(t *testing.T) {
			given := mustParse(t)
			given.set = onFixture(t, "", "")
			raw, err := os.ReadFile(given.set)
			if err != nil {
				t.Fatal(err)
			}
			text := string(raw)
			start, end := strings.Index(text, "  film:"), strings.Index(text, "  cameras:")
			want := "no film"
			if missing == "camera" {
				start, end = end, strings.Index(text, "  set:")
				want = "no camera"
			}
			if start < 0 || end <= start {
				t.Fatal("fixture block changed")
			}
			text = text[:start] + text[end:]
			if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			if _, _, _, err := staged(given); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("missing %s: %v", missing, err)
			}
		})
	}
}

func TestStagingCannotSampleAFuelBedBeforeItsClockStarts(t *testing.T) {
	given := fuelOnFixture(t)
	given.at = -time.Second
	if _, _, _, err := staged(given); err == nil || !strings.Contains(err.Error(), "time") {
		t.Fatalf("negative bed clock lost: %v", err)
	}
}

func TestAValidRepresentationNameCanStillBeUnsupportedByTheChosenMaker(t *testing.T) {
	given := mustParse(t)
	given.set = onFixture(t, "kind: fern", "kind: can\n        complexity: hull")
	if _, _, _, err := staged(given); err == nil || !strings.Contains(err.Error(), "can has no hull representation") {
		t.Fatalf("stand representation refusal lost: %v", err)
	}
}

func TestLegacyStandPropagatesManufactureLampAndSmokeFailures(t *testing.T) {
	given := mustParse(t)
	if err := stand(thing.NewWorld(), &tally{}, given, stock{}, "missing kind", 0, maths.Subspace{}); err == nil || !strings.Contains(err.Error(), "missing kind") {
		t.Fatalf("maker refusal: %v", err)
	}
	given.alight, given.floor = 1, 1e9
	if err := stand(thing.NewWorld(), &tally{}, given, stock{}, "spruce", 0, maths.Subspace{}); err == nil {
		t.Fatal("smoke refusal lost")
	}
	given = mustParse(t)
	given.exposure = math.NaN()
	if err := stand(thing.NewWorld(), &tally{}, given, stock{}, "spotlight", 0, maths.Subspace{}); err == nil || !strings.Contains(err.Error(), "exposure") {
		t.Fatalf("lamp refusal lost: %v", err)
	}
}

func TestBareGroundRefusesANonfiniteSpan(t *testing.T) {
	if _, err := bare(mustParse(t), math.NaN()); err == nil || !strings.Contains(err.Error(), "forest: a ground") {
		t.Fatalf("nonfinite ground span: %v", err)
	}
}

func TestFullCircleOffersFillEveryQuadrantAndMissingOriginsReachCallers(t *testing.T) {
	stand := wedged(t, "{of: the first tree, from: 0, to: 360, near: 10, far: 50}")
	stage := staging{places: map[string]maths.Vec{"the first tree": {}}}
	offer, err := stage.offering(stand, rand.New(rand.NewPCG(5, 1)))
	if err != nil {
		t.Fatal(err)
	}
	var quadrants [4]int
	for range 1000 {
		x, north := offer()
		angle := math.Mod(math.Atan2(x, north)+2*math.Pi, 2*math.Pi)
		quadrants[min(3, int(angle/(math.Pi/2)))]++
		if radius := math.Hypot(x, north); radius < 10 || radius > 50 {
			t.Fatalf("full circle radius %g", radius)
		}
	}
	for i, count := range quadrants {
		if count < 100 {
			t.Fatalf("quadrant %d has only %d offers", i, count)
		}
	}
	delete(stage.places, "the first tree")
	if _, err := stage.inBand(stand, rand.New(rand.NewPCG(5, 1))); err == nil || !strings.Contains(err.Error(), "stands nowhere") {
		t.Fatalf("band origin refusal: %v", err)
	}
	if err := stage.scatter(stand); err == nil || !strings.Contains(err.Error(), "stands nowhere") {
		t.Fatalf("scatter origin refusal: %v", err)
	}
}

func TestGroveAssemblyRefusesAnInvalidCanopy(t *testing.T) {
	text := strings.Replace(fixtureScene, "  terrain:", "  weather:\n    mist:\n      kind: prescribed-mist\n      extinction: {value: 0.006, unit: m^-1}\n      depth: {value: 20, unit: m}\n      coverage: 1\n      seed: 23\n  terrain:", 1)
	text = strings.Replace(text, "        count: 12", "        count: 12\n        atmosphere: {weather: mist, extent: canopy-and-ground, fade-at-boundary: true}", 1)
	read, err := set.Read("bad-canopy.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	stage := staging{recipes: read, world: thing.NewWorld()}
	if err := stage.grove(read.Stands()[0], []groveTree{{bounds: maths.Box{}}}); err == nil || !strings.Contains(err.Error(), "canopy") {
		t.Fatalf("empty canopy cause: %v", err)
	}
}
