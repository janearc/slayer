package main

import (
	"bytes"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/embers"
	"github.com/janearc/slayer/fermat"
	"github.com/janearc/slayer/firefront"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/fuelbed"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// boundaryField exercises the retained renderer's interface contract. Real
// flame evolution and illumination are tested separately on both backends.
type boundaryField struct {
	medium.Medium
	next    medium.Medium
	samples []flame.Source
}

func (field *boundaryField) Tick(time.Duration) medium.Medium {
	if field.next != nil {
		return field.next
	}
	return field
}
func (field *boundaryField) Sources(int) []flame.Source { return field.samples }

func boundaryFlame(t *testing.T) *flame.Field {
	t.Helper()
	field, err := flame.NewField([]flame.Ignition{{Fire: flame.Fire{Radius: .25, Height: 1, Absorption: 1, Hottest: measure.Kelvin(1600), Edge: measure.Kelvin(900)}}}, 1e-5)
	if err != nil {
		t.Fatal(err)
	}
	if len(field.Sources(6)) == 0 {
		t.Fatal("field fixture has no sampled emission")
	}
	return field
}

func TestRetainedFireRefusesUnsupportedTicksAndInvalidSourceSamples(t *testing.T) {
	for _, fault := range []string{"tick type", "lamp radius"} {
		t.Run(fault, func(t *testing.T) {
			field := &boundaryField{Medium: medium.Puff{Radius: 1}, samples: []flame.Source{{Across: -1, Strength: maths.Grey(1)}}}
			if fault == "tick type" {
				field.next = medium.Puff{Radius: 1}
			}
			fire := retainedFire{field: field, cells: 1}
			want := "radius"
			if fault == "tick type" {
				want = "unsupported timed fire"
			}
			if lamps, err := fire.lampsAt(time.Second); err == nil || lamps != nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("lamps %v error %v", lamps, err)
			}
			live := liveScene{fires: []retainedFire{fire}}
			if world, _, err := live.frameAt(time.Second); world != nil || err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("failed fire returned a frame: %v %v", world, err)
			}
		})
	}
}

func TestRetainedFrontAndBedFailuresReachTheFrameCaller(t *testing.T) {
	field := boundaryFlame(t)
	plan, err := firefront.New(firefront.Spec{Direction: maths.XAxis, Speed: 1, Cell: 1, Duration: time.Second}, []firefront.Target{{At: maths.Vec{}}})
	if err != nil {
		t.Fatal(err)
	}
	front := retainedFront{plan: plan, groups: []scheduledGroup{{start: 0, end: time.Second, fire: retainedFire{field: field, cells: 6, place: maths.Subspace{Scale: math.NaN()}}}}}
	if _, _, err := (&liveScene{fronts: []retainedFront{front}}).frameAt(time.Second / 2); err == nil || !strings.Contains(err.Error(), "finite") {
		t.Fatalf("invalid front source placement: %v", err)
	}
	spec := fuelbed.Grass(fuelbed.Spec{Width: 1, Depth: 1, Cell: .5, Step: .01, Ambient: 300, Initial: []fuelbed.Cell{{Excess: 500, Fuel: 1}}})
	for _, fault := range []string{"clock", "field count", "lamp radius"} {
		t.Run(fault, func(t *testing.T) {
			model, err := fuelbed.New(spec)
			if err != nil {
				t.Fatal(err)
			}
			bed := retainedBed{model: model, spec: spec, lightCells: 6, groups: []fuelGroup{{field: field, indices: []int{0}}}}
			at := time.Duration(0)
			want := ""
			switch fault {
			case "clock":
				at = -time.Second
				want = "time"
			case "field count":
				bed.groups[0].indices = nil
				want = "a fire that cannot be"
			case "lamp radius":
				bed.spec.Cell = -1
				want = "radius"
			}
			world, _, err := (&liveScene{beds: []retainedBed{bed}}).frameAt(at)
			if world != nil || err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("%s returned %v, %v", fault, world, err)
			}
		})
	}
}

func TestSparkPreparationRefusesSourceAndPaletteFailures(t *testing.T) {
	stage := staging{given: asked{exposure: 1}}
	if _, err := stage.prepareSparks(manufacture.BurnSpec{Sparks: &embers.Spec{}}, maths.Subspace{}); err == nil || !strings.Contains(err.Error(), "embers:") {
		t.Fatalf("invalid source: %v", err)
	}
	spec := manufacture.BurnSpec{Sparks: &embers.Spec{Rate: 10, Life: time.Second, Drag: time.Second, Radius: measure.Metres(.02), Hot: measure.Kelvin(1700), Cool: measure.Kelvin(900)}}
	stage.given.exposure = math.NaN()
	if _, err := stage.prepareSparks(spec, maths.Subspace{}); err == nil || !strings.Contains(err.Error(), "a fire that cannot be") {
		t.Fatalf("invalid palette: %v", err)
	}
	stage.given.exposure = 1
	sparks, err := stage.prepareSparks(spec, maths.Subspace{Scale: 3})
	if err != nil {
		t.Fatal(err)
	}
	live := liveScene{fires: []retainedFire{{field: boundaryFlame(t), cells: 6, sparks: sparks}}}
	world, made, err := live.frameAt(time.Second / 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Things) != 5 || made.sparks != 5 || !strings.Contains(made.String(), "5 sparks") {
		t.Fatalf("particle count: %d, %v", len(world.Things), made)
	}
	for _, spark := range world.Things {
		if spark.At(0).Scale != .06 {
			t.Fatal("placement scale did not multiply particle radius")
		}
	}
}

func TestProofAndRawOutputFailuresPreserveTheFilesystemCause(t *testing.T) {
	dir := t.TempDir()
	blocked := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocked, []byte("keep me"), 0600); err != nil {
		t.Fatal(err)
	}
	picture := frame.New(2, 2)
	if err := keep(picture, filepath.Join(blocked, "raw.pfm")); err == nil || !strings.Contains(err.Error(), blocked) {
		t.Fatalf("raw parent failure: %v", err)
	}
	proof := fermat.Proof{Raw: filepath.Join(dir, "absent.pfm")}
	if err := sheet(proof, filepath.Join(blocked, "proof")); err == nil || !strings.Contains(err.Error(), blocked) {
		t.Fatalf("proof parent failure: %v", err)
	}
	if err := sheet(proof, filepath.Join(dir, "proof")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("raw read cause: %v", err)
	}
	path := filepath.Join(dir, "small.pfm")
	if err := raw.Write(path, picture); err != nil {
		t.Fatal(err)
	}
	given := mustParse(t)
	given.width, given.height, given.samples = 3, 2, 1
	given.prove, given.proof, given.many = path, filepath.Join(dir, "explicit-proof"), 1
	var wrong bytes.Buffer
	if code := proving(thing.NewWorld(), given, 1, nil, io.Discard, &wrong); code != 1 || !strings.Contains(wrong.String(), "2 by 2") {
		t.Fatalf("wrong proof size: %d %s", code, wrong.String())
	}
	if sheetFor(given) != given.proof {
		t.Fatal("explicit proof folder ignored")
	}
}

func TestStockAndLegacyStagingRefuseInvalidInputs(t *testing.T) {
	given := mustParse(t)
	if _, _, err := (stock{}).made(given, "absent kind", 0); err == nil || !strings.Contains(err.Error(), "absent kind") {
		t.Fatalf("stock kind refusal: %v", err)
	}
	if _, err := manufacturedRecipe(given, "can", 0, "demo", nil); err == nil || !strings.Contains(err.Error(), "no demo") {
		t.Fatalf("stock representation refusal: %v", err)
	}
	optical := manufacture.Optical{}
	if _, err := manufacturedRecipe(given, "optical-observatory", 0, "quick", &optical); err == nil {
		t.Fatal("invalid observatory accepted")
	}
	one, err := manufactured(given, "spotlight", 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := one.lamps(maths.Subspace{Scale: math.NaN()}); err == nil || !strings.Contains(err.Error(), "invalid light placement") {
		t.Fatalf("lamp placement: %v", err)
	}
	given.exposure = math.NaN()
	one, err = manufactured(given, "spotlight", 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := one.lamps(maths.Subspace{}); err == nil || !strings.Contains(err.Error(), "exposure") {
		t.Fatalf("lamp exposure: %v", err)
	}
	if got := facesIn(&shape.Mesh{}); got != 0 {
		t.Fatalf("empty legacy face count: %d", got)
	}
	if facesIn(shape.Sphere{}) != 0 {
		t.Fatal("non-mesh counted as a face")
	}
	given = mustParse(t)
	given.apart = -1
	if _, _, err := hillside(given); err == nil {
		t.Fatal("invalid placement spacing accepted")
	}
	given = mustParse(t)
	given.above = true
	if eyeFor(given, 10) != aboveAt(10) {
		t.Fatal("above view dispatch changed")
	}
}

func TestLegacyFireRefusalsAndImplicitUnitScale(t *testing.T) {
	given := mustParse(t)
	one, err := manufactured(given, "spruce", 0)
	if err != nil {
		t.Fatal(err)
	}
	unitR, unitH := reach(one.things, 1)
	if r, h := reach(one.things, 0); r != unitR || h != unitH {
		t.Fatal("implicit unit scale changed reach")
	}
	if _, err := alight(thing.NewWorld(), kept{}, maths.Subspace{}, 1, given); err == nil {
		t.Fatal("empty burning geometry accepted")
	}
	given.floor = 1e9
	if _, err := alight(thing.NewWorld(), one, maths.Subspace{}, 1, given); err == nil {
		t.Fatal("out-of-atmosphere fire accepted")
	}
	if err := smoke(thing.NewWorld(), maths.Subspace{}, -1, 0, 1); err == nil {
		t.Fatal("negative smoke radius accepted")
	}
	if err := lamps(thing.NewWorld(), asked{earth: true}, -1); err == nil {
		t.Fatal("negative distant light accepted")
	}
}

func TestNamedFireRecipeAndDaylightRefusals(t *testing.T) {
	stage := staging{given: mustParse(t), world: thing.NewWorld()}
	if _, err := stage.putFireRecipe("absent", 1, 0, maths.Vec{}, ""); err == nil || !strings.Contains(err.Error(), "named recipe") {
		t.Fatalf("missing recipe: %v", err)
	}
	read, err := set.Read("nonfire.yaml", []byte(strings.Replace(fixtureScene, "  terrain:", "  recipes:\n    solid: {kind: can}\n  terrain:", 1)))
	if err != nil {
		t.Fatal(err)
	}
	stage.recipes = read
	if _, err := stage.putFireRecipe("solid", 1, 0, maths.Vec{}, ""); err == nil || !strings.Contains(err.Error(), "not a fire recipe") {
		t.Fatalf("solid recipe: %v", err)
	}
	for _, altitude := range []string{"0", "35"} {
		read, err = set.Read("sun.yaml", []byte(strings.Replace(fixtureScene, "light: moon", "light: sun\n  sun: {azimuth: 180, altitude: "+altitude+"}", 1)))
		if err != nil {
			t.Fatal(err)
		}
		if read.Light() != "sun" {
			t.Fatal("sun fixture replacement failed")
		}
		stage.world = thing.NewWorld()
		err = stage.light(read)
		if altitude == "0" {
			if err == nil || !strings.Contains(err.Error(), "above the horizon") {
				t.Fatalf("sun horizon: %v", err)
			}
		} else if err != nil || len(stage.world.Lamps) != 2 {
			t.Fatalf("sun lamps %d: %v", len(stage.world.Lamps), err)
		}
	}
}
