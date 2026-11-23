package bodies_test

import (
	"image"
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/scene/world"
)

// a method that cannot return an error holds the complaint, and the body
// that would wear the part refuses it, naming the input and its value.
func TestAMethodHoldsANonFiniteValueAndTheBodyRefusesThePart(t *testing.T) {
	bad := thing.Spectrum{Gives: map[radiation.Range]float64{
		radiation.XRay(): nan}}
	tail, err := bodies.NewTail(km(1), km(1), 0, 0, nothing, maths.Light{},
		maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	auroraShell := bodies.NewShell(km(1), km(2))
	aurora, err := bodies.NewAurora(auroraShell,
		measure.Radians(0), measure.Radians(0), measure.Radians(0),
		maths.Share(0.5), maths.Share(0.5), 1, bodies.AuroraColours{})
	if err != nil {
		t.Fatal(err)
	}
	jets, err := bodies.NewJets(km(1), km(1), 0, km(0), nothing,
		measure.Radians(0), maths.Share(0.5), maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	flare, err := bodies.NewFlare(time.Second, 1, 0.5, maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	ground, err := bodies.NewGround(world.Terrain{}, world.Look{})
	if err != nil {
		t.Fatal(err)
	}
	air, err := bodies.NewAir(km(1), nothing, maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	rings, err := bodies.NewRings(1, 2, km(0), nothing, maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	pixel := image.NewRGBA(image.Rect(0, 0, 1, 1))
	for name, c := range map[string]struct {
		part bodies.Part
		want string
	}{
		"tail":    {tail.Radiating(bad), "Tail.Radiating's radiant gives NaN"},
		"aurora":  {aurora.Radiating(bad), "Aurora.Radiating's radiant gives NaN"},
		"jets":    {jets.Radiating(bad), "Jets.Radiating's radiant gives NaN"},
		"flare":   {flare.Radiating(bad), "Flare.Radiating's radiant gives NaN"},
		"ground":  {ground.Radiating(bad), "Ground.Radiating's radiant gives NaN"},
		"air":     {air.Radiating(bad), "Air.Radiating's radiant gives NaN"},
		"turn":    {ground.Mapped(pixel, measure.Radians(nan)), "Ground.Mapped's turn is NaN"},
		"profile": {rings.Profiled([]float64{1, nan}), "Rings.Profiled's element 1 is NaN"},
	} {
		_, err := bodies.NewBody("a", bodies.Rock, km(1),
			measure.Kilograms(1), c.part)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v; want an error saying %q", name, err, c.want)
		}
	}
}

// a glow that is not a number is held, and Place refuses it by name.
func TestGlowingHoldsANonFiniteGlowAndPlaceRefusesIt(t *testing.T) {
	body := made(t, "a", bodies.Star, 1, 1).Glowing(badRGB, nil)
	err := body.Place(thing.NewWorld(), maths.Vec{}, 1, 2)
	if err == nil || !strings.Contains(err.Error(), "Glowing's glow") {
		t.Errorf("a NaN glow: %v", err)
	}
	bad := thing.Spectrum{Passes: map[radiation.Range]float64{
		radiation.XRay(): inf}}
	body = made(t, "a", bodies.Star, 1, 1).Glowing(maths.Light{}, bad)
	err = body.Place(thing.NewWorld(), maths.Vec{}, 1, 2)
	if err == nil || !strings.Contains(err.Error(), "Glowing's radiant") {
		t.Errorf("an infinite radiant: %v", err)
	}
}

// a Body{} was not made by NewBody and cannot be a moon.
func TestAMoonOfAnUnmadeBodyIsRefused(t *testing.T) {
	_, err := bodies.NewMoon(bodies.Body{}, km(1), measure.Radians(0))
	if err == nil || !strings.Contains(err.Error(), "NewBody") {
		t.Errorf("a moon of Body{}: %v", err)
	}
}

// a gas keeps its own copy of the bands it was given and gives out its own
// copy: a write to the caller's slice after NewGas, or through Bands, does
// not reach the part.
func TestAGasKeepsItsOwnBands(t *testing.T) {
	paints := []maths.Light{maths.Grey(0.1), maths.Grey(0.2)}
	spots := []texture.Spot{{Size: 0.1}}
	streets := []texture.Street{{Count: 3}}
	gas, err := bodies.NewGas(maths.Percent(8), 40, texture.Banded{
		Paints: paints, Spots: spots, Streets: streets})
	if err != nil {
		t.Fatal(err)
	}
	paints[0], spots[0].Size, streets[0].Count = maths.Grey(9), 9, 9
	got := gas.Bands()
	if got.Paints[0] != maths.Grey(0.1) || got.Spots[0].Size != 0.1 ||
		got.Streets[0].Count != 3 {
		t.Errorf("a write to the caller's slices reached the gas: %v %v %v",
			got.Paints, got.Spots, got.Streets)
	}
	got.Paints[0], got.Spots[0].Size, got.Streets[0].Count =
		maths.Grey(8), 8, 8
	again := gas.Bands()
	if again.Paints[0] != maths.Grey(0.1) || again.Spots[0].Size != 0.1 ||
		again.Streets[0].Count != 3 {
		t.Errorf("a write through Bands reached the gas: %v %v %v",
			again.Paints, again.Spots, again.Streets)
	}
}

// a gas refuses bands holding a number that is no number, by the input
// and its value.
func TestAGasRefusesBandsThatAreNoNumber(t *testing.T) {
	for name, c := range map[string]struct {
		bands texture.Banded
		want  string
	}{
		"roil":  {texture.Banded{Roil: nan}, "bands' roil is NaN"},
		"shear": {texture.Banded{Shear: inf}, "bands' shear is +Inf"},
		"grain": {texture.Banded{Grain: nan}, "bands' grain is NaN"},
		"axis":  {texture.Banded{Axis: maths.V(0, nan, 0)}, "bands' axis"},
		"paint": {texture.Banded{Paints: []maths.Light{{}, badRGB}},
			"bands' paint 1"},
	} {
		_, err := bodies.NewGas(maths.Percent(8), 40, c.bands)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v; want it to say %q", name, err, c.want)
		}
	}
}

// a radar look with a paint or lines that are no number, and a radiant
// that is nil, are held and refused when a body wears the ground.
func TestAGroundHoldsWhatItWasGivenThatIsNotUsable(t *testing.T) {
	ground, err := bodies.NewGround(world.Terrain{}, world.Look{})
	if err != nil {
		t.Fatal(err)
	}
	var paints [world.Biomes]maths.Light
	paints[2] = badRGB
	for name, c := range map[string]struct {
		part bodies.Part
		want string
	}{
		"a NaN paint": {ground.RadarRough(world.Look{Paints: paints}),
			"RadarRough's look paint 2"},
		"NaN lines": {ground.RadarRough(world.Look{Lines: nan}),
			"RadarRough's look lines is NaN"},
		"a nil radiant": {ground.Radiating(nil),
			"Ground.Radiating's radiant is nil"},
	} {
		_, err := bodies.NewBody("a", bodies.Rock, km(1),
			measure.Kilograms(1), c.part)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v; want it to say %q", name, err, c.want)
		}
	}
}

// Aimed holds a vector that is no number, and Place refuses it.
func TestAimedHoldsAVectorThatIsNoNumber(t *testing.T) {
	body := made(t, "a", bodies.Star, 1, 1)
	for name, aimed := range map[string]bodies.Body{
		"sun":   body.Aimed(maths.V(nan, 0, 0), maths.Vec{}),
		"trail": body.Aimed(maths.Vec{}, maths.V(0, inf, 0)),
	} {
		err := aimed.Place(thing.NewWorld(), maths.Vec{}, 1, 2)
		if err == nil || !strings.Contains(err.Error(), "Aimed's "+name) {
			t.Errorf("a %s that is no number: %v", name, err)
		}
	}
}

// Place and Binary refuse a point that is no number and a world that is
// nil, and Place refuses a Body{} by saying so.
func TestPlaceAndBinaryRefuseABadPointAndANilWorld(t *testing.T) {
	body := made(t, "a", bodies.Star, 1, 1)
	where := maths.V(nan, 0, 0)
	if err := body.Place(thing.NewWorld(), where, 1, 2); err == nil ||
		!strings.Contains(err.Error(), "NaN") {
		t.Errorf("Place at a NaN point: %v", err)
	}
	if err := bodies.Binary(thing.NewWorld(), body, body, where, km(10), 1,
		2); err == nil || !strings.Contains(err.Error(), "NaN") {
		t.Errorf("Binary at a NaN point: %v", err)
	}
	if err := body.Place(nil, maths.Vec{}, 1, 2); err == nil ||
		!strings.Contains(err.Error(), "nil world") {
		t.Errorf("Place in a nil world: %v", err)
	}
	if err := bodies.Binary(nil, body, body, maths.Vec{}, km(10), 1,
		2); err == nil || !strings.Contains(err.Error(), "nil world") {
		t.Errorf("Binary in a nil world: %v", err)
	}
	err := bodies.Body{}.Place(thing.NewWorld(), maths.Vec{}, 1, 2)
	if err == nil || !strings.Contains(err.Error(), "NewBody") {
		t.Errorf("Place of a Body{}: %v", err)
	}
}
