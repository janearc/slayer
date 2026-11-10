package main

import (
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// A group's finite proxy preserves sampled power and its first spatial
// moment. It is an authored lighting approximation, not angular transport.
// Verify the actual lamp against public sources from the actual drawn field.
func TestFuelBedGroupLampsPreservePowerAndSourcePosition(t *testing.T) {
	given := fuelOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "      width: 8", "      width: 16", 1)
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	read := readBoundarySet(t, given.set)
	given.onGPU, given.exposure = false, 1e-5
	stage := staging{given: given, recipes: read, world: thing.NewWorld()}
	if err := stage.terrain(read); err != nil {
		t.Fatal(err)
	}
	if err := stage.prepareBeds(read); err != nil {
		t.Fatal(err)
	}
	defer (&liveScene{beds: stage.beds}).closeBeds()
	if len(stage.beds) != 1 || len(stage.beds[0].groups) != 2 {
		t.Fatal("fixture lost its two real fuel groups")
	}
	row, has := read.Recipe("wood fire")
	if !has {
		t.Fatal("fixture lost its checked fire recipe")
	}
	spec, has := row.Fire()
	if !has || spec.LightCells < 1 {
		t.Fatal("fixture lost its public source sampling count")
	}
	for _, at := range []time.Duration{0, 10 * time.Second} {
		world := thing.NewWorld()
		if err := stage.beds[0].addAt(world, &tally{}, at); err != nil {
			t.Fatal(err)
		}
		lamps := 0
		for _, volume := range world.Volumes {
			field, ok := volume.Medium.(*flame.Field)
			if !ok {
				t.Fatal("bed added a different participating medium")
			}
			power, moment := maths.Light{}, maths.Vec{}
			weight := 0.0
			low := maths.V(math.Inf(1), math.Inf(1), math.Inf(1))
			high := maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))
			for _, source := range field.Sources(spec.LightCells) {
				point := volume.At(at).Out(source.At)
				v := source.Strength.Luminance()
				power = power.Add(source.Strength)
				moment = moment.Add(point.Scale(v))
				weight += v
				low = maths.V(math.Min(low.X, point.X), math.Min(low.Y, point.Y), math.Min(low.Z, point.Z))
				high = maths.V(math.Max(high.X, point.X), math.Max(high.Y, point.Y), math.Max(high.Z, point.Z))
			}
			if weight == 0 {
				continue
			}
			if !(weight > 0) || lamps >= len(world.Lamps) {
				t.Fatal("positive sampled field power lost its actual lamp")
			}
			lamp, ok := world.Lamps[lamps].(emit.Volume)
			if !ok {
				t.Fatal("fuel group did not use its finite source proxy")
			}
			lamps++
			point := lamp.Point()
			if lamp.Radius().InMetres() != 2 {
				t.Fatalf("half-metre fuel cells lost their authored 2 m lamp radius: %g", lamp.Radius().InMetres())
			}
			for i, pair := range [][2]float64{{point.Strength.R, power.R}, {point.Strength.G, power.G}, {point.Strength.B, power.B}} {
				if !(math.Abs(pair[0]-pair[1]) <= 1e-12*math.Max(1e-30, math.Abs(pair[1]))) {
					t.Fatalf("at %s channel %d: lamp power %.17g differs from drawn field %.17g", at, i, pair[0], pair[1])
				}
			}
			if gap := point.At.Scale(weight).Sub(moment).Len(); !(gap <= 1e-10*math.Max(1, moment.Len())) {
				t.Fatalf("at %s: actual lamp moved the field's first spatial moment by %g", at, gap)
			}
			for _, bounds := range [][3]float64{{point.At.X, low.X, high.X}, {point.At.Y, low.Y, high.Y}, {point.At.Z, low.Z, high.Z}} {
				if !(bounds[0] >= bounds[1]-1e-10 && bounds[0] <= bounds[2]+1e-10) {
					t.Fatalf("at %s: positive source centroid escaped its actual source bounds", at)
				}
			}
		}
		if lamps == 0 || lamps != len(world.Lamps) {
			t.Fatalf("at %s: %d expected group lamps, %d actual", at, lamps, len(world.Lamps))
		}
	}
}

// A checked zero-radiance composition is still an absorbing medium. Its
// sampled geometry may be present without any RGB source power. The test
// line is authored input, not a claim about a measured emission spectrum.
func TestFuelBedWithZeroRadianceKeepsItsMediumWithoutALamp(t *testing.T) {
	given := fuelOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	composition := `        edge-temperature: {value: 900, unit: K}
        composition:
          continuum: 0
          lines:
            - species: authored test line
              wavelength: {value: 500, unit: nm}
              radiance: {value: 0, unit: W/m^2/sr}
              convention: vacuum
              source: authored zero-radiance test input
`
	text := strings.Replace(string(raw), "        edge-temperature: {value: 900, unit: K}\n", composition, 1)
	if text == string(raw) {
		t.Fatal("fixture lost its flame temperature setting")
	}
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	read := readBoundarySet(t, given.set)
	given.onGPU, given.exposure = false, 1
	stage := staging{given: given, recipes: read, world: thing.NewWorld()}
	if err := stage.terrain(read); err != nil {
		t.Fatal(err)
	}
	if err := stage.prepareBeds(read); err != nil {
		t.Fatal(err)
	}
	defer (&liveScene{beds: stage.beds}).closeBeds()
	if len(stage.beds) != 1 {
		t.Fatal("checked composition lost its real fuel model")
	}
	row, has := read.Recipe("wood fire")
	if !has {
		t.Fatal("fixture lost its checked fire recipe")
	}
	spec, has := row.Fire()
	if !has || spec.Flame.Composition == nil {
		t.Fatal("fixture did not retain its checked composition")
	}
	world, made := thing.NewWorld(), tally{}
	if err := stage.beds[0].addAt(world, &made, 0); err != nil {
		t.Fatalf("valid absorbing medium without RGB emission should not fail: %v", err)
	}
	if len(world.Volumes) != 1 || len(world.Lamps) != 0 {
		t.Fatalf("zero-radiance bed has %d media and %d lamps, want one medium and no lamp", len(world.Volumes), len(world.Lamps))
	}
	field, ok := world.Volumes[0].Medium.(*flame.Field)
	if !ok || len(field.Pieces()) == 0 || made.burning != len(field.Pieces()) {
		t.Fatal("turning off RGB emission also removed the actual absorbing patches")
	}
	sources := field.Sources(spec.LightCells)
	if len(sources) == 0 {
		t.Fatal("fixture did not sample actual occupied source cells")
	}
	for _, source := range sources {
		if source.Strength != (maths.Light{}) {
			t.Fatal("authored zero-radiance spectrum emitted nonzero RGB light")
		}
	}
}
