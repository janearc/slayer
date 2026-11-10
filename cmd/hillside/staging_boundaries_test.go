package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/lens"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

func readBoundarySet(t *testing.T, path string) set.Set {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	return read
}

func TestStagedRefusesUnwritablePlacesAndInvalidAssetReferences(t *testing.T) {
	for _, fault := range []string{"places", "camera", "focus"} {
		t.Run(fault, func(t *testing.T) {
			given := mustParse(t)
			given.set = onFixtureRigged(t, "0.5", false)
			switch fault {
			case "places":
				given.placed = t.TempDir()
			case "camera":
				raw, err := os.ReadFile(given.set)
				if err != nil {
					t.Fatal(err)
				}
				text := strings.Replace(string(raw), "object: the stone", "object: missing stone", 1)
				if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			case "focus":
				raw, err := os.ReadFile(given.set)
				if err != nil {
					t.Fatal(err)
				}
				text := strings.Replace(string(raw), "focus: the tree", "focus: missing tree", 1)
				if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			_, _, _, err := staged(given)
			want := map[string]string{"places": given.placed, "camera": "missing stone", "focus": "missing tree"}[fault]
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("%s: %v", fault, err)
			}
		})
	}
}

func TestFilmAndRigConstructionKeepTheirRefusals(t *testing.T) {
	given := mustParse(t)
	if _, err := filmed(set.Set{}, given); err == nil || !strings.Contains(err.Error(), "no film") {
		t.Fatalf("missing film: %v", err)
	}
	if _, err := rigged(set.Rig{}, given); err == nil {
		t.Fatal("empty rig frame accepted")
	}
	read := readBoundarySet(t, onFixtureRigged(t, "0.5", false))
	film, has := read.Film()
	if !has {
		t.Fatal("fixture has no film")
	}
	rig, has := film.Rig()
	if !has {
		t.Fatal("fixture has no rig")
	}
	f, err := lens.FrameOf(70.41, 51.61)
	if err != nil {
		t.Fatal(err)
	}
	given.frame = &f
	stage := staging{given: given, world: thing.NewWorld(), places: map[string]maths.Vec{"the tree": {}}}
	_, _, focus := rig.Lens()
	if err := stage.focused(rig, maths.Subspace{Origin: maths.V(0, 0, 1)}); err != nil {
		t.Fatal(err)
	}
	delete(stage.places, focus)
	if err := stage.focused(rig, maths.Subspace{}); err == nil || !strings.Contains(err.Error(), "not on the set") {
		t.Fatalf("missing placed focus: %v", err)
	}
	stage.places[focus] = maths.Vec{}

	// A focus coincident with the eye has zero distance.
	if err := stage.focused(rig, maths.Subspace{}); err == nil {
		t.Fatal("zero focus distance accepted")
	}
	path := onFixture(t, "    width: 24", "    wide: 35\n    width: 24")
	read = readBoundarySet(t, path)
	got, err := filmed(read, given)
	if err != nil || got.wide != 35 {
		t.Fatalf("film field of view: %g %v", got.wide, err)
	}
}

func TestTerrainAndPlacedObjectsRefuseUnavailableGround(t *testing.T) {
	for _, fault := range []string{"missing file", "window outside", "thing outside"} {
		t.Run(fault, func(t *testing.T) {
			was, now := "", ""
			switch fault {
			case "missing file":
				was, now = "file: fixture-plain.tif", "file: absent.tif"
			case "window outside":
				was, now = "across: 150", "across: 1500"
			case "thing outside":
				was, now = "east: 1050", "east: 100000"
			}
			given := mustParse(t)
			given.set = onFixture(t, was, now)
			_, _, _, err := staged(given)
			if err == nil {
				t.Fatal("unavailable ground staged")
			}
			if fault == "missing file" && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("terrain read cause: %v", err)
			}
			if fault == "thing outside" && (!strings.Contains(err.Error(), "the tree") || !strings.Contains(err.Error(), "off the ground")) {
				t.Fatalf("placed object cause: %v", err)
			}
		})
	}
	stage := staging{given: mustParse(t)}
	if err := stage.terrain(set.Set{}); err == nil || !strings.Contains(err.Error(), "no terrain") {
		t.Fatalf("missing terrain: %v", err)
	}
}

func TestPinnedAssetRefusalsAndCorruptMoonUseTheCheckedBytes(t *testing.T) {
	given := mustParse(t)
	given.set = onFixture(t, "", "")
	near := filepath.Dir(given.set)
	root := filepath.Dir(near)
	catalogue := filepath.Join(root, "sources.yaml")
	original, err := os.ReadFile(catalogue)
	if err != nil {
		t.Fatal(err)
	}
	stage := staging{given: given}
	for _, fault := range []string{"missing entry", "bad catalogue"} {
		t.Run(fault, func(t *testing.T) {
			if fault == "bad catalogue" {
				if err := os.WriteFile(catalogue, []byte("sources: ["), 0600); err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := os.WriteFile(catalogue, original, 0600); err != nil {
						t.Error(err)
					}
				}()
			}
			key := "fixture"
			if fault == "missing entry" {
				key = "absent"
			}
			if _, err := stage.shelved(key); err == nil || !strings.Contains(err.Error(), map[string]string{"missing entry": "not in sources.yaml", "bad catalogue": "sources.yaml"}[fault]) {
				t.Fatalf("%s: %v", fault, err)
			}
		})
	}
	// Pin deliberately non-image bytes. Hash validation must succeed before
	// the real lunar image decoder supplies its format refusal.
	corrupt := []byte("a pinned document, not an image")
	sum := sha256.Sum256(corrupt)
	shelf := os.Getenv("SLAYER_SHELF")
	if err := os.WriteFile(filepath.Join(shelf, "corrupt-moon.tif"), corrupt, 0600); err != nil {
		t.Fatal(err)
	}
	moonRow := "  - key: corrupt-moon\n    from: https://example.test/corrupt.tif\n    licence: test fixture\n    sha256: " + hex.EncodeToString(sum[:]) + "\n"
	if err := os.WriteFile(catalogue, append(bytes.Clone(original), []byte(moonRow)...), 0600); err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(fixtureScene, "light: moon", "light: moon\n  moon: {azimuth: 180, altitude: 35, face: corrupt-moon}", 1)
	read, err := set.Read("moon.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	moon, has := read.Moonlit()
	if !has {
		t.Fatal("fixture has no moon")
	}
	if _, err := stage.faced(moon, maths.YAxis, maths.Grey(1)); err == nil || !strings.Contains(err.Error(), "format") {
		t.Fatalf("corrupt moon decoder: %v", err)
	}
	t.Setenv("SLAYER_SHELF", "")
	t.Setenv("HOME", "")
	if _, _, err := pinnedSource("fixture", near); err == nil {
		t.Fatal("missing shelf home accepted")
	}
	if _, err := weatherCache(); err == nil {
		t.Fatal("missing cache home accepted")
	}
	t.Setenv("SLAYER_WEATHER", t.TempDir())
	if got, err := weatherCache(); err != nil || got != os.Getenv("SLAYER_WEATHER") {
		t.Fatalf("explicit weather cache: %s %v", got, err)
	}
}

func TestStageRecipePlacementAndLightsKeepTheirCauses(t *testing.T) {
	path := onFixture(t, "  terrain:", "  recipes:\n    scope:\n      kind: optical-observatory\n      enclosure:\n        kind: dome\n        diameter: {value: 20, unit: m}\n        opening: {value: 30, unit: deg}\n      telescope:\n        aperture: {value: 8, unit: m}\n        mount: alt-azimuth\n        primary: segmented\n        pointing: {azimuth: 0, altitude: 45}\n  terrain:")
	read := readBoundarySet(t, path)
	stage := staging{given: mustParse(t), world: thing.NewWorld(), have: stock{}, radii: map[string]float64{}, recipes: read}
	if _, err := stage.putRecipe("optical-observatory", "scope", "hull", 1, 0, maths.Vec{}, false, "scope"); err != nil {
		t.Fatal(err)
	}
	if stage.made.things != 1 || stage.radii["scope"] <= 0 {
		t.Fatal("optical recipe was not placed")
	}
	if _, err := stage.putRecipe("absent", "", "quick", 1, 0, maths.Vec{}, false, ""); err == nil || !strings.Contains(err.Error(), "absent") {
		t.Fatalf("unknown stage recipe: %v", err)
	}
	stage.given.faces = 0
	if err := stage.add(kept{}, "can", maths.Subspace{}, 1, false); err == nil || !errors.Is(err, errTooMany) {
		t.Fatalf("kept face limit: %v", err)
	}
	stage.given = mustParse(t)
	lamp, err := manufactured(stage.given, "spotlight", 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := stage.add(lamp, "spotlight", maths.Subspace{}, 1, false); err != nil || len(stage.world.Lamps) == 0 {
		t.Fatalf("carried stage lamp: %v", err)
	}
	if err := stage.add(lamp, "spotlight", maths.Subspace{Scale: -1}, 1, false); err == nil || !strings.Contains(err.Error(), "invalid light placement") {
		t.Fatalf("carried lamp placement: %v", err)
	}
}
