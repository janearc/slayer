package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/planet"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

func boundaryMoon(t *testing.T, extra string) (set.Set, set.Moon) {
	t.Helper()
	read, err := set.Read("moon-boundary.yaml", []byte(strings.Replace(fixtureScene, "light: moon", "light: moon\n  moon: {azimuth: 180, altitude: 35"+extra+"}", 1)))
	if err != nil {
		t.Fatal(err)
	}
	moon, ok := read.Moonlit()
	if !ok {
		t.Fatal("fixture has no moon")
	}
	return read, moon
}

func TestMoonAndSkyBoundariesRejectInvalidAtmosphericHeightAndLight(t *testing.T) {
	for _, disc := range []string{", disc: false", ", disc: true"} {
		read, moon := boundaryMoon(t, disc)
		stage := staging{given: mustParse(t), world: thing.NewWorld(), lowest: 1e9}
		if _, err := stage.moonlit(moon); err == nil {
			t.Fatal("moon transmission accepted out-of-atmosphere height")
		}
		if err := stage.skies(read, maths.YAxis); err == nil {
			t.Fatal("sky scattering accepted out-of-atmosphere height")
		}
	}
	_, moon := boundaryMoon(t, ", disc: false")
	stage := staging{given: asked{exposure: math.NaN()}, world: thing.NewWorld()}
	if _, err := stage.moonlit(moon); err == nil || !strings.Contains(err.Error(), "distant") {
		t.Fatalf("invalid distant moonlight: %v", err)
	}
	_, moon = boundaryMoon(t, ", face: absent-mosaic")
	stage.given = mustParse(t)
	stage.given.set = onFixture(t, "", "")
	if _, err := stage.moonlit(moon); err == nil || !strings.Contains(err.Error(), "absent-mosaic") {
		t.Fatalf("unavailable lunar mosaic: %v", err)
	}
	if _, err := stage.faced(moon, maths.YAxis, maths.Grey(1)); err == nil || !strings.Contains(err.Error(), "absent-mosaic") {
		t.Fatalf("mosaic pin lookup: %v", err)
	}
}

func TestAuthoredFillDirectionRefusalPropagatesThroughSceneAssembly(t *testing.T) {
	previous := fillWay
	t.Cleanup(func() { fillWay = previous })
	fillWay = maths.V(math.NaN(), 0, 0)
	given := mustParse(t)
	given.earth = true
	if _, _, err := hillside(given); err == nil || !strings.Contains(err.Error(), "distant") {
		t.Fatalf("hillside lost lamp refusal: %v", err)
	}
	read, err := set.Read("day.yaml", []byte(strings.Replace(fixtureScene, "light: moon", "light: sun\n  sun: {azimuth: 180, altitude: 35}", 1)))
	if err != nil {
		t.Fatal(err)
	}
	stage := staging{given: given, world: thing.NewWorld()}
	if err := stage.light(read); err == nil || !strings.Contains(err.Error(), "distant") {
		t.Fatalf("daylight lost fill refusal: %v", err)
	}
}

func TestFrontPreparationRefusesMissingOriginAndUnplacedTargets(t *testing.T) {
	given := frontOnFixture(t)
	read := readBoundarySet(t, given.set)
	stage := staging{given: given, world: thing.NewWorld(), recipes: read, places: map[string]maths.Vec{}}
	if err := stage.prepareFronts(read); err == nil || !strings.Contains(err.Error(), "no placed origin") {
		t.Fatalf("origin refusal: %v", err)
	}
	stage.places["the tree"] = maths.Vec{}
	if err := stage.prepareFronts(read); err == nil || !strings.Contains(err.Error(), "invalid prescribed ignition plan") {
		t.Fatalf("empty target plan refusal: %v", err)
	}
}

func TestFrontRetainsAGroupGapWithoutDrawingItsExpiredPatch(t *testing.T) {
	given := frontOnFixture(t)
	_, _, ready, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	front := ready.live.fronts[0]
	group := front.groups[0]
	// Keep the actual immutable field, while widening its scheduling group.
	// At this time no patch is active inside the conservative group window.
	group.end += 10 * time.Second
	front.groups = []scheduledGroup{group}
	front.smoke = nil
	front.sparks = nil
	world := thing.NewWorld()
	if err := front.addAt(world, &tally{}, group.end-time.Second); err != nil {
		t.Fatal(err)
	}
	if len(world.Volumes) != 0 || len(world.Lamps) != 0 {
		t.Fatal("expired field drawn inside a wider group window")
	}
}

func TestFuelBedBudgetsIncludeConcurrentFrontVolumes(t *testing.T) {
	given := fuelOnFixture(t)
	read := readBoundarySet(t, given.set)
	stage := staging{given: given, world: thing.NewWorld(), recipes: read}
	if err := stage.terrain(read); err != nil {
		t.Fatal(err)
	}
	stage.fronts = []retainedFront{{peakVolumes: 1000}}
	if err := stage.prepareBeds(read); err == nil || !strings.Contains(err.Error(), "volume budget") {
		t.Fatalf("aggregate bed/front budget: %v", err)
	}
}

func TestPlanetAssemblyBoundaryRefusalsAndUnmarkedStorms(t *testing.T) {
	stage := staging{given: mustParse(t), world: thing.NewWorld(), places: map[string]maths.Vec{}}
	if err := stage.dusk(set.Set{}, set.Planet{}); err == nil || !strings.Contains(err.Error(), "needs a sun") {
		t.Fatalf("unlit planet: %v", err)
	}
	if (darkness{}).Light(maths.YAxis) != (maths.Light{}) {
		t.Fatal("space background emits light")
	}
	if _, err := ringOf(set.Planet{}, set.Ring{}, 1); err == nil {
		t.Fatal("empty ring accepted")
	}
	if _, err := columnOf([]set.Layer{{}}, nil); err == nil {
		t.Fatal("empty pressure layer accepted")
	}
	if _, err := marked(planet.Body{}, set.Planet{}, set.Storms{}, nil); err != nil {
		t.Fatalf("unmarked storms need no extra columns: %v", err)
	}
	given := mustParse(t)
	given.set = inSpace(t, "", "")
	read := readBoundarySet(t, given.set)
	made, ok := read.Planet()
	if !ok {
		t.Fatal("fixture has no planet")
	}
	stage = staging{given: given, world: thing.NewWorld(), places: map[string]maths.Vec{made.Name(): {}}}
	if err := stage.planetLit(read, made, maths.V(math.NaN(), 0, 0)); err == nil || !strings.Contains(err.Error(), "rings seen") {
		t.Fatalf("invalid observer: %v", err)
	}
	catalogue := filepath.Join(filepath.Dir(filepath.Dir(given.set)), "sources.yaml")
	if err := os.WriteFile(catalogue, []byte("sources: ["), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := stage.gases(); err == nil || !strings.Contains(err.Error(), "sources.yaml") {
		t.Fatalf("gas catalogue: %v", err)
	}
	if _, err := stage.body(made); err == nil || !strings.Contains(err.Error(), "sources.yaml") {
		t.Fatalf("body gas refusal: %v", err)
	}
}

func TestFirePlacementRefusesAnUncheckedNonfiniteOrigin(t *testing.T) {
	given := fireOnFixture(t)
	read := readBoundarySet(t, given.set)
	stage := staging{given: given, world: thing.NewWorld(), recipes: read}
	if _, err := stage.putFireRecipe("wood fire", 1, 0, maths.V(math.NaN(), 0, 0), ""); err == nil || !strings.Contains(err.Error(), "invalid fire placement") {
		t.Fatalf("nonfinite source origin: %v", err)
	}
}
