//go:build draw

package session

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/embers"
	"github.com/janearc/slayer/firefront"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/fuelbed"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
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
	if _, err := stage.prepareSparks(manufacture.BurnSpec{Sparks: &embers.Spec{}}, maths.Local()); err == nil || !strings.Contains(err.Error(), "embers:") {
		t.Fatalf("invalid source: %v", err)
	}
	spec := manufacture.BurnSpec{Sparks: &embers.Spec{Rate: 10, Life: time.Second, Drag: time.Second, Radius: measure.Metres(.02), Hot: measure.Kelvin(1700), Cool: measure.Kelvin(900)}}
	stage.given.exposure = math.NaN()
	if _, err := stage.prepareSparks(spec, maths.Local()); err == nil || !strings.Contains(err.Error(), "a fire that cannot be") {
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

func TestNamedFireRecipeAndDaylightRefusals(t *testing.T) {
	stage := staging{given: mustParse(t), world: thing.NewWorld()}
	if _, err := stage.putFireRecipe("absent", 1, 0, maths.Vec{}, ""); err == nil || !strings.Contains(err.Error(), "named recipe") {
		t.Fatalf("missing recipe: %v", err)
	}
	read, err := set.Read("nonfire.yaml", []byte(strings.Replace(fixtureScene, "  recipes:\n", "  recipes:\n    solid: {kind: can}\n", 1)))
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

func (field *boundaryField) Tick(time.Duration) medium.Medium {
	if field.next != nil {
		return field.next
	}
	return field
}

func (field *boundaryField) Sources(int) []flame.Source { return field.samples }
