package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestCoveAddsARealScaleHumanAndFramesBoth(t *testing.T) {
	plan := defaultPlan(t)
	plan.Presentation = "cove"
	menu, err := readMenu()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := menu.selectEntries([]string{"spitzer"})
	if err != nil {
		t.Fatal(err)
	}
	scene, err := plan.construct(entries[0], "quick")
	if err != nil {
		t.Fatal(err)
	}
	var human, floor, cove bool
	for _, part := range scene.world.Things {
		switch part.Material.Name {
		case "two-metre scale human":
			human = true
		case "cove checkerboard":
			floor = true
			if part.Material.Texture == nil {
				t.Fatal("floor has no checkerboard")
			}
		case "white infinity cove":
			cove = true
		}
	}
	if !human || !floor || !cove {
		t.Fatalf("missing presentation objects: human %v, floor %v, cove %v", human, floor, cove)
	}
}

func TestCoveRefusesInvalidConstructionAndLostPrecision(t *testing.T) {
	box := maths.Box{Min: maths.V(-1, 0, -1), Max: maths.V(1, 2, 1)}
	if _, _, err := addCove(thing.NewWorld(), box, maths.ZAxis, "absent"); err == nil {
		t.Fatal("unsupported scale-figure detail was accepted")
	}
	box.Min.X = math.NaN()
	if _, _, err := addCove(thing.NewWorld(), box, maths.ZAxis, "quick"); err == nil {
		t.Fatal("non-finite scale-figure placement was accepted")
	}
	if _, err := coveSurfaces(thing.NewWorld(), maths.V(0, 0, 0), maths.XAxis, maths.ZAxis, 0); err == nil {
		t.Fatal("zero-area cove floor was accepted")
	}
	// At this offset a metre-scale sweep loses neighbouring vertices to
	// floating-point rounding even though the much larger floor survives.
	if _, err := coveSurfaces(thing.NewWorld(), maths.V(0, 1e16, 1e16), maths.XAxis, maths.ZAxis, 1); err == nil {
		t.Fatal("collapsed sweep was accepted")
	}
}

func TestSpacePresentationLoadsOnlyPinnedStarsAndKeepsItsSun(t *testing.T) {
	p := defaultPlan(t)
	p.Presentation = "space"
	file := filepath.Join(t.TempDir(), "gaia.csv")
	if err := os.WriteFile(file, []byte(gaiaFixture), 0600); err != nil {
		t.Fatal(err)
	}
	p.Gaia = &gaiaPlan{File: file, SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(gaiaFixture))), Radius: .1, Zero: 1}
	gaia, err := json.Marshal(p.Gaia)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(string(planRaw) + "\npresentation: space\ngaia-stars: " + string(gaia) + "\n")
	loaded, err := readPlan("space.yaml", raw)
	if err != nil || loaded.starField == nil || loaded.starField.Count() != 3 {
		t.Fatal("pinned sky not loaded", err)
	}
	entry := entry{Kind: "spitzer", Staging: &staging{Sources: []source{{Toward: maths.XAxis, Strength: maths.Grey(100)}}}}
	if staged := entry.staged(*loaded); staged.Sources[0] != loaded.Sources[0] {
		t.Fatal("entry replaced the presentation sunlight")
	}
	if _, err := loaded.construct(entry, "quick"); err != nil {
		t.Fatal(err)
	}
	loaded.starField = nil
	if _, err := loaded.construct(entry, "quick"); err == nil {
		t.Fatal("space scene rendered without a verified sky")
	}
	if err := os.WriteFile(file, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPlan("space.yaml", raw); err == nil {
		t.Fatal("changed source accepted by plan loading")
	}
	for _, invalid := range []plan{
		{Presentation: "space"},
		{Presentation: "cove", Gaia: p.Gaia},
		{Presentation: "cove", Interior: &interiorPlan{}},
	} {
		if err := invalid.check(); err == nil {
			t.Fatal("inconsistent presentation accepted", invalid.Presentation)
		}
	}
}

// anti-test of TestCoveAddsARealScaleHumanAndFramesBoth
func TestCoveGeometryHasAFloorAndTangentWhiteSweep(t *testing.T) {
	world := thing.NewWorld()
	bounds := maths.Box{Min: maths.V(-2, -3, -4), Max: maths.V(2, 5, 4)}
	shown, _, err := addCove(world, bounds, maths.V(0, 1, -1), "quick")
	if err != nil {
		t.Fatal(err)
	}
	if shown.Min.Y != -3 || shown.Max.Y != 5 || shown.Max.X <= bounds.Max.X {
		t.Fatalf("scale figure not beside the object on its floor: %v", shown)
	}
	for _, part := range world.Things {
		if part.Material.Name != "two-metre scale human" {
			continue
		}
		at := part.Motion.At(0)
		box := part.Shape.Bounds()
		bottom, top := at.Out(box.Min), at.Out(box.Max)
		if math.Abs(top.Y-bottom.Y-2) > 1e-12 || math.Abs(bottom.Y+3) > 1e-12 {
			t.Fatal("scale figure was resized or put on a different floor", bottom, top)
		}
	}
}
