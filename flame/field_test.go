package flame

import (
	"math"
	"testing"
	"testing/quick"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

func patchFixture() []Ignition {
	f := Fire{Radius: 0.2, Height: 1.4, Absorption: 1, Hottest: measure.Kelvin(1750), Edge: measure.Kelvin(900), Seed: 5}
	return []Ignition{{At: maths.V(-0.7, 0, 0), Fire: f}, {At: maths.V(0.7, 0, 0), Fire: f}}
}

func TestFuelPatchesKeepGapsAndTheirPhysicalBounds(t *testing.T) {
	patches := patchFixture()
	field, err := NewField(patches, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, patch := range patches {
		if field.Density(patch.At.Add(maths.V(0, 0.2, 0))) <= 0 {
			t.Fatal("source patch is empty")
		}
	}
	if field.Density(maths.V(0, 0.2, 0)) != 0 {
		t.Fatal("fuel gap filled in")
	}
	if field.Bounds().Min.X >= -0.7 || field.Bounds().Max.X <= 0.7 || field.Finest() <= 0 || field.Tint() != Tilt() {
		t.Fatal("field lost extent, detail or tint")
	}
}

func TestOverlappingPatchesAddAbsorptionAndEmission(t *testing.T) {
	f := patchFixture()[0].Fire
	cooler := f
	cooler.Hottest = measure.Kelvin(1300)
	field, err := NewField([]Ignition{{Fire: f}, {Fire: cooler}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	hot, cold := body(t, f), body(t, cooler)
	p := maths.V(0, 0.2, 0)
	a, b := hot.Density(p), cold.Density(p)
	want := hot.Shine().At(p).Scale(a).Add(cold.Shine().At(p).Scale(b))
	got := field.Shine().At(p).Scale(field.Density(p))
	if field.Density(p) != a+b || math.Abs(got.R-want.R)+math.Abs(got.G-want.G)+math.Abs(got.B-want.B) > 1e-12*max(1, want.R+want.G+want.B) {
		t.Fatal("overlap lost or invented emitted light")
	}
}

func TestFuelFieldDoesNotRetainMutablePlacementSlices(t *testing.T) {
	cells := patchFixture()
	field, err := NewField(cells, 1)
	if err != nil {
		t.Fatal(err)
	}
	p := cells[0].At.Add(maths.V(0, 0.2, 0))
	before := field.Density(p)
	cells[0].At = maths.V(99, 0, 0)
	cells[0].Fire.Absorption = 0
	copy := field.Pieces()
	if len(copy) != 2 {
		t.Fatal("missing source pieces")
	}
	copy[0].At = maths.V(99, 0, 0)
	copy[0].Body = nil
	if field.Density(p) != before || before <= 0 {
		t.Fatal("caller changed stored fuel")
	}
}

func TestFuelFieldRejectsInvalidOrUnboundedSources(t *testing.T) {
	for _, cells := range [][]Ignition{
		nil, make([]Ignition, 65),
		{{At: maths.V(math.NaN(), 0, 0), Fire: patchFixture()[0].Fire}},
		{{At: maths.V(math.Inf(1), 0, 0), Fire: patchFixture()[0].Fire}},
		{{Fire: Fire{}}},
	} {
		if _, err := NewField(cells, 1); err == nil {
			t.Fatal("invalid fuel field accepted")
		}
	}
	if _, err := NewField(patchFixture(), 0); err == nil {
		t.Fatal("zero exposure accepted")
	}
}

func TestFuelFieldTimeAndSourceLightStayTogether(t *testing.T) {
	field, err := NewField(patchFixture(), 1)
	if err != nil {
		t.Fatal(err)
	}
	later := field.Tick(137 * time.Millisecond).(*Field)
	shine := field.Shine().Tick(137 * time.Millisecond).(fieldShine)
	changed := false
	for _, patch := range patchFixture() {
		for i := range 20 {
			p := patch.At.Add(maths.V(0, 0.04+float64(i)*0.05, 0))
			changed = changed || field.Density(p) != later.Density(p)
			if shine.At(p) != later.Shine().At(p) {
				t.Fatal("emission and density clocks separated")
			}
		}
	}
	if !changed {
		t.Fatal("source field is frozen")
	}
	if len(later.Sources(3)) == 0 || later.Sources(0) != nil {
		t.Fatal("invalid source quadrature")
	}
	if later.Pieces()[0].Body == field.Pieces()[0].Body {
		t.Fatal("tick reused mutable time state")
	}
}

func TestFuelFieldPreservesAdditiveTransportAcrossSeededSamples(t *testing.T) {
	property := func(seed uint32, u, v uint8) bool {
		f := patchFixture()[0].Fire
		f.Seed = uint64(seed)
		other := f
		other.Absorption *= .4
		other.Hottest = measure.Kelvin(1300)
		field, err := NewField([]Ignition{{Fire: f}, {Fire: other}}, 1)
		if err != nil {
			return false
		}
		at := time.Duration(u) * time.Millisecond
		field = field.Tick(at).(*Field)
		p := maths.V(0, .01+float64(v)/256*f.Height, 0)
		a, b := field.Pieces()[0].Body, field.Pieces()[1].Body
		want := a.Shine().At(p).Scale(a.Density(p)).Add(b.Shine().At(p).Scale(b.Density(p)))
		got := field.Shine().At(p).Scale(field.Density(p))
		return math.Abs(got.R-want.R)+math.Abs(got.G-want.G)+math.Abs(got.B-want.B) <=
			1e-12*max(1, want.R+want.G+want.B)
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 50}); err != nil {
		t.Fatal(err)
	}
}

func TestFuelFieldRefusesOverflowInTranslatedBounds(t *testing.T) {
	fire := patchFixture()[0].Fire
	fire.Height = 1e308
	if _, err := NewField([]Ignition{{At: maths.V(0, 1e308, 0), Fire: fire}}, 1); err == nil {
		t.Fatal("overflowing translated extent accepted")
	}
}

// Distinct ignition windows must survive repeated time jumps. The field keeps
// its complete source plan; ticking a late frame cannot erase an early source.
func TestAFieldKeepsIndependentIgnitionAgesAndLifetimes(t *testing.T) {
	fire := Fire{Radius: 1, Height: 3, Absorption: 1, Hottest: measure.Kelvin(1800), Edge: measure.Kelvin(900), Seed: 5}
	field, err := NewField([]Ignition{
		{At: maths.V(0, 0, 0), Fire: fire, Start: time.Second, Duration: 2 * time.Second},
		{At: maths.V(5, 0, 0), Fire: fire, Start: 2 * time.Second, Duration: 3 * time.Second},
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	before := field.Tick(0).(*Field)
	if len(before.Pieces()) != 0 || before.Density(maths.V(0, 1, 0)) != 0 || len(before.Sources(3)) != 0 {
		t.Fatal("unlit fuel emitted")
	}
	both := field.Tick(2500 * time.Millisecond).(*Field)
	pieces := both.Pieces()
	if len(pieces) != 2 {
		t.Fatal("missing overlapping ignition windows")
	}
	_, first, _, _ := pieces[0].Body.Samples()
	_, second, _, _ := pieces[1].Body.Samples()
	if first != 1500*time.Millisecond || second != 500*time.Millisecond {
		t.Fatal("patches lost independent ages")
	}
	after := both.Tick(4 * time.Second).(*Field)
	if len(after.Pieces()) != 1 || after.Pieces()[0].At.X != 5 {
		t.Fatal("burned patch survived")
	}
	replay := after.Tick(1500 * time.Millisecond).(*Field)
	if len(replay.Pieces()) != 1 || replay.Pieces()[0].At.X != 0 {
		t.Fatal("backward jump lost planned source")
	}
	end := field.Tick(5 * time.Second).(*Field)
	if len(end.Pieces()) != 0 || len(end.Sources(3)) != 0 {
		t.Fatal("expired field still emitted")
	}
	fireAt := maths.V(.1, 1, 0)
	truth, _ := NewBody(fire, 1)
	truth = truth.Tick(500 * time.Millisecond).(*Body)
	if replay.Density(fireAt) != truth.Density(fireAt) || replay.Shine().At(fireAt) != truth.Shine().At(fireAt) {
		t.Fatal("ignition age disagrees with independent body")
	}
}

func TestIgnitionFadesOpacityAndEmissionTogether(t *testing.T) {
	fire := patchFixture()[0].Fire
	field, err := NewField([]Ignition{{Fire: fire, Start: time.Second, Duration: 4 * time.Second, Fade: time.Second}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range []time.Duration{1500 * time.Millisecond, 4500 * time.Millisecond} {
		age := at - time.Second
		truth, _ := NewBody(fire, 1)
		truth = truth.Tick(age).(*Body)
		tick := field.Tick(at).(*Field)
		p := maths.V(0, .1, 0)
		if math.Abs(tick.Density(p)/truth.Density(p)-.5) > 1e-12 {
			t.Fatal("fade did not halve opacity at midpoint")
		}
		got := tick.Shine().At(p).Scale(tick.Density(p))
		want := truth.Shine().At(p).Scale(truth.Density(p) * .5)
		if math.Abs(got.R-want.R)+math.Abs(got.G-want.G)+math.Abs(got.B-want.B) > 1e-12 {
			t.Fatal("fade lost matching emitted coefficient")
		}
	}
	for _, fade := range []time.Duration{-time.Second, 3 * time.Second} {
		if _, err := NewField([]Ignition{{Fire: fire, Duration: 4 * time.Second, Fade: fade}}, 1); err == nil {
			t.Fatal("invalid fade accepted")
		}
	}
}
