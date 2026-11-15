package lights

import (
	"math"
	"reflect"
	"slices"
	"testing"
)

func TestFixturePlanFixedOrderAndOriginalIntensities(t *testing.T) {
	config := FixtureConfig{FloodPasses: [2]string{"back-left", "back-right"}, FloodGain: 400, SpotGain: 150}
	for _, aim := range []float64{-1, 0, 0.25, 1, 1.75, 2, 3} {
		cue := FixtureCues{FloodLeft: 0.2, FloodRight: 0.3, Spots: 0.4, SpotAim: aim, HasColour: true, Colour: [3]float64{0.2, 0.4, 0.8}}
		clamped := math.Max(0, math.Min(2, aim))
		near := math.Min(1, math.Floor(clamped))
		f := clamped - near
		a, b := int(near), int(near)+1
		names := [3]string{"0", "1", "2"}
		tint := [3]float64{0.5 + 0.5*cue.Colour[0], 0.5 + 0.5*cue.Colour[1], 0.5 + 0.5*cue.Colour[2]}
		all := []FixtureLayer{
			{"back-left", 400 * cue.FloodLeft, [3]float64{1, 1, 1}},
			{"back-right", 400 * cue.FloodRight, [3]float64{1, 0.25, 0.85}},
			{"spot-left-" + names[a], 150 * cue.Spots * (1 - f), tint},
			{"spot-left-" + names[b], 150 * cue.Spots * f, tint},
			{"spot-right-" + names[2-a], 150 * cue.Spots * (1 - f), tint},
			{"spot-right-" + names[2-b], 150 * cue.Spots * f, tint},
		}
		var want []FixtureLayer
		for _, layer := range all {
			if layer.Amount >= 1e-3 {
				want = append(want, layer)
			}
		}
		for range 16 {
			got, err := PlanFixtures(config, cue)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("aim %v: got %+v want %+v", aim, got, want)
			}
		}
	}
}

func TestFixtureThresholdAndUntintedSpots(t *testing.T) {
	config := FixtureConfig{FloodPasses: [2]string{"left", "right"}, FloodGain: 1, SpotGain: 1}
	layers, err := PlanFixtures(config, FixtureCues{FloodLeft: 0.001, FloodRight: 0.000999, Spots: -1})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(layers, []FixtureLayer{{"left", 0.001, [3]float64{1, 1, 1}}}) {
		t.Fatalf("threshold: %+v", layers)
	}
	layers, err = PlanFixtures(config, FixtureCues{Spots: 1, SpotAim: 0.5, Colour: [3]float64{9, 8, 7}})
	if err != nil {
		t.Fatal(err)
	}
	for _, layer := range layers {
		if layer.Colour != [3]float64{1, 1, 1} {
			t.Fatalf("unexpected tint: %+v", layer)
		}
	}
}

func TestFixtureExplicitPassesExactOperationOrderAndNoInputMutation(t *testing.T) {
	t.Setenv("PASSES", "/missing")
	t.Setenv("FLOOD_GAIN", "invalid")
	config := FixtureConfig{FloodPasses: [2]string{"left", "right"}, FloodGain: 0.13, SpotGain: 0.27}
	cue := FixtureCues{FloodLeft: 0.31, FloodRight: 0.41, Spots: 0.73, SpotAim: 0.37, HasColour: true, Colour: [3]float64{0.12, 0.34, 0.56}}
	layers, err := PlanFixtures(config, cue)
	if err != nil {
		t.Fatal(err)
	}
	passes := make(map[string]Image)
	originals := make(map[string][]float64)
	for k, layer := range layers {
		values := []float64{float64(k+1) * 0.17, float64(k+2) * 0.23, float64(k+3) * 0.31}
		passes[layer.Name] = Image{1, 1, values}
		originals[layer.Name] = slices.Clone(values)
	}
	before := slices.Clone(layers)
	sky := Image{1, 1, []float64{0.11, 0.22, 0.33}}
	want := slices.Clone(sky.Light)
	for _, layer := range layers {
		for c := range 3 {
			want[c] += layer.Amount * layer.Colour[c] * passes[layer.Name].Light[c]
		}
	}
	if err := AddFixtures(sky, layers, passes); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sky.Light, want) {
		t.Fatalf("got %v want %v", sky.Light, want)
	}
	if !reflect.DeepEqual(layers, before) {
		t.Fatal("layers mutated")
	}
	for name, pass := range passes {
		if !slices.Equal(pass.Light, originals[name]) {
			t.Fatal("pass mutated")
		}
	}
	ordered := []FixtureLayer{{"huge", 1, [3]float64{1, 1, 1}}, {"small", 1, [3]float64{1, 1, 1}}, {"small", 1, [3]float64{1, 1, 1}}}
	clear(sky.Light)
	if err := AddFixtures(sky, ordered, map[string]Image{"huge": {1, 1, []float64{1e16, 1e16, 1e16}}, "small": {1, 1, []float64{1, 1, 1}}}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sky.Light, []float64{1e16, 1e16, 1e16}) {
		t.Fatal("layers reordered or combined")
	}
}

func TestFixtureInvalidInputsNeverMutateOutput(t *testing.T) {
	valid := Image{1, 1, []float64{1, 2, 3}}
	layers := []FixtureLayer{{"first", 1, [3]float64{1, 1, 1}}, {"last", 1, [3]float64{1, 1, 1}}}
	for _, pass := range []Image{{}, {2, 1, make([]float64, 6)}, {1, 1, []float64{1}}} {
		sky := Image{1, 1, slices.Clone(valid.Light)}
		if err := AddFixtures(sky, layers, map[string]Image{"first": valid, "last": pass}); err == nil {
			t.Fatal("accepted invalid last pass")
		}
		if !slices.Equal(sky.Light, valid.Light) {
			t.Fatal("output mutated on invalid last pass")
		}
	}
	sky := Image{1, 1, slices.Clone(valid.Light)}
	if err := AddFixtures(sky, layers, map[string]Image{"first": valid}); err == nil {
		t.Fatal("accepted missing last pass")
	}
	if !slices.Equal(sky.Light, valid.Light) {
		t.Fatal("output mutated on missing last pass")
	}
	if err := AddFixtures(Image{}, nil, nil); err == nil {
		t.Fatal("accepted invalid output")
	}
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := PlanFixtures(FixtureConfig{}, FixtureCues{SpotAim: value}); err == nil {
			t.Fatal("accepted nonfinite aim")
		}
		for _, layer := range []FixtureLayer{{"first", value, [3]float64{1, 1, 1}}, {"first", 1, [3]float64{value, 1, 1}}} {
			if err := AddFixtures(sky, []FixtureLayer{layers[0], layer}, map[string]Image{"first": valid}); err == nil {
				t.Fatal("accepted nonfinite intensity")
			}
			if !slices.Equal(sky.Light, valid.Light) {
				t.Fatal("output mutated on invalid intensity")
			}
		}
	}
	if err := AddFixtures(sky, []FixtureLayer{{"unused", 0.0009, [3]float64{1, 1, 1}}}, nil); err != nil {
		t.Fatalf("inactive pass required: %v", err)
	}
}

func TestFixtureConcurrentIndependentInputs(t *testing.T) {
	for _, level := range []float64{0.25, 0.75} {
		t.Run("independent", func(t *testing.T) {
			t.Parallel()
			for range 8 {
				sky := Image{1, 1, make([]float64, 3)}
				layers, err := PlanFixtures(FixtureConfig{FloodPasses: [2]string{"left", "right"}, FloodGain: 2}, FixtureCues{FloodLeft: level})
				if err != nil {
					t.Fatal(err)
				}
				if err := AddFixtures(sky, layers, map[string]Image{"left": {1, 1, []float64{1, 2, 3}}}); err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(sky.Light, []float64{2 * level, 4 * level, 6 * level}) {
					t.Fatal("instance inputs leaked")
				}
			}
		})
	}
}
