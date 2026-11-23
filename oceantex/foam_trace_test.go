package oceantex

import (
	"math"
	"testing"
)

// pulseField is an independently controlled patch of foam on an invented
// coast, with land at and east of x=10. The patch has a compact footprint so
// moving it to the wrong side is observable rather than an approximate match.
type pulseField struct {
	crest, strength float64
	dryBefore       float64
}

func (field pulseField) Wet(east, north float64) bool {
	return east > field.dryBefore && east < 10 && math.Abs(north) <= 10
}

func (field pulseField) Sample(east, north float64) (float64, float64, bool) {
	if !finite(east) || !finite(north) || math.Abs(east) > 20 || math.Abs(north) > 20 {
		return 0, 0, false
	}
	if math.Abs(east-field.crest) < .25 && math.Abs(north) < .25 {
		return 2, field.strength, true
	}
	return 2, 0, true
}

func TestFoamTraceMovesAndFadesAnEarlierCrest(t *testing.T) {
	current := pulseField{crest: -8, strength: 1, dryBefore: -20}
	prior := pulseField{crest: 0, strength: 1, dryBefore: -20}
	trace, err := NewFoamTrace(current, []FoamFrame{{Age: 1, Field: prior}}, 2, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	height, foam, ok := trace.Sample(2, 0)
	if !ok || height != 2 || math.Abs(foam-math.Exp(-.5)) > 1e-12 {
		t.Fatalf("advected crest: height %g foam %g ok %v", height, foam, ok)
	}
	if _, foam, _ := trace.Sample(-2, 0); foam != 0 {
		t.Fatal("foam moved against the chosen drift")
	}
	older, err := NewFoamTrace(current, []FoamFrame{{Age: 2, Field: prior}}, 2, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, faded, _ := older.Sample(2, 0); !(faded > 0 && faded < foam) {
		t.Fatalf("older foam did not fade: older %g, newer %g", faded, foam)
	}
}

func TestFoamTraceInstantControlNoFoamAndDryClipping(t *testing.T) {
	current := pulseField{crest: 1, strength: .4, dryBefore: -20}
	prior := pulseField{crest: 1, strength: 1, dryBefore: -20}
	instant, err := NewFoamTrace(current, []FoamFrame{{Age: 1, Field: prior}}, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, foam, ok := instant.Sample(1, 0); !ok || foam != .4 {
		t.Fatalf("zero lifetime must retain current foam: %g, %v", foam, ok)
	}
	traced, err := NewFoamTrace(current, []FoamFrame{{Age: 1, Field: prior}}, 2, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, foam, _ := traced.Sample(5, 0); foam != 0 {
		t.Fatal("foam appeared without a crest now or in history")
	}
	if _, foam, ok := traced.Sample(10, 0); !ok || foam != 0 {
		t.Fatal("foam appeared on dry target")
	}
	if _, _, ok := traced.Sample(21, 0); ok {
		t.Fatal("accepted outside the current field")
	}
	drySource := pulseField{crest: 0, strength: 1, dryBefore: 0}
	traced, _ = NewFoamTrace(current, []FoamFrame{{Age: 1, Field: drySource}}, 2, 1, 0)
	if _, foam, _ := traced.Sample(1, 0); foam != .4 {
		t.Fatalf("dry source contributed foam: %g", foam)
	}
}

func TestFoamTraceRejectsInvalidHistory(t *testing.T) {
	field := pulseField{dryBefore: -20}
	for _, test := range []struct {
		current               FoamField
		frames                []FoamFrame
		lifetime, east, north float64
	}{
		{nil, nil, 1, 0, 0},
		{field, nil, -1, 0, 0},
		{field, nil, math.NaN(), 0, 0},
		{field, nil, 1, math.Inf(1), 0},
		{field, []FoamFrame{{Age: 0, Field: field}}, 1, 0, 0},
		{field, []FoamFrame{{Age: 1}}, 1, 0, 0},
		{field, []FoamFrame{{Age: 2, Field: field}, {Age: 1, Field: field}}, 1, 0, 0},
	} {
		if _, err := NewFoamTrace(test.current, test.frames, test.lifetime, test.east, test.north); err == nil {
			t.Fatalf("accepted invalid history: %+v", test)
		}
	}
}
