package bodies_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/scene/world"
)

// A complaint carried up from a loss or a shell keeps its package
// prefix once, at the front.
func TestAWrappedComplaintCarriesThePrefixOnce(t *testing.T) {
	_, err := bodies.NewAir(bodies.Kilometres(8),
		bodies.PerKilometre(-1), maths.Light{})
	want := "bodies: the air's PerKilometre(-1) is no extinction"
	if err == nil || err.Error() != want {
		t.Errorf("NewAir with a negative loss: %v; want %q", err, want)
	}
	flatShell := bodies.NewShell(bodies.Kilometres(5), bodies.Kilometres(5))
	_, err = bodies.NewClouds(
		flatShell,
		maths.Share(0.5), bodies.PerKilometre(1), 1)
	front := "bodies: clouds: a shell from 5 to 5 km is no band"
	if err == nil || !strings.HasPrefix(err.Error(), front) ||
		strings.Count(err.Error(), "bodies:") != 1 {
		t.Errorf("NewClouds with a flat shell: %v; want it to start "+
			"%q with one prefix", err, front)
	}
}

var (
	nan     = math.NaN()
	inf     = math.Inf(1)
	badRGB  = maths.Light{R: math.NaN()}
	span    = bodies.Kilometres
	nothing = bodies.PerKilometre(0)
)

// A jet whose flow overflows is refused: the flow is the speed in base
// widths a second, and a base near the smallest float makes it endless.
func TestJetsWhoseFlowOverflowsAreRefused(t *testing.T) {
	_, err := bodies.NewJets(km(1000), km(5e-324), 0.1, km(0), nothing,
		measure.Radians(0), maths.Share(0.5), maths.Light{})
	if err == nil {
		t.Error("NewJets with a base of 5e-324 km passed")
	}
	_, err = bodies.NewJets(km(inf), km(1), 0.1, km(0), nothing,
		measure.Radians(0), maths.Share(0.5), maths.Light{})
	if err == nil {
		t.Error("NewJets with an infinite length passed")
	}
}

// Every float a constructor takes is finite, or the constructor refuses.
func TestConstructorsRefuseWhatIsNotANumber(t *testing.T) {
	good := maths.Light{R: 1}
	shell := bodies.NewShell(km(1), km(2))
	var some = []struct {
		name string
		err  func() error
	}{
		{"shell nan low", func() error {
			nanLowShell := bodies.NewShell(km(nan), km(2))
			_, e := bodies.NewClouds(nanLowShell,
				maths.Share(0.5), nothing, 1)
			return e
		}},
		{"shell inf high", func() error {
			infHighShell := bodies.NewShell(km(1), km(inf))
			_, e := bodies.NewClouds(infHighShell,
				maths.Share(0.5), nothing, 1)
			return e
		}},
		{"coma infinite radius", func() error {
			_, e := bodies.NewComa(km(inf), nothing, good, good)
			return e
		}},
		{"tail colour", func() error {
			_, e := bodies.NewTail(km(1), km(1), 0, 0, nothing, badRGB, good)
			return e
		}},
		{"aurora fringe", func() error {
			_, e := bodies.NewAurora(shell, measure.Radians(1),
				measure.Radians(1), measure.Radians(1), maths.Share(0.5),
				maths.Share(0.5), 1, bodies.NewAuroraColours(badRGB, good, good))
			return e
		}},
		{"aurora top", func() error {
			_, e := bodies.NewAurora(shell, measure.Radians(1),
				measure.Radians(1), measure.Radians(1), maths.Share(0.5),
				maths.Share(0.5), 1, bodies.NewAuroraColours(good, good, badRGB))
			return e
		}},
		{"coma colour", func() error {
			_, e := bodies.NewComa(km(1), nothing, badRGB, good)
			return e
		}},
		{"coma glow", func() error {
			_, e := bodies.NewComa(km(1), nothing, good, badRGB)
			return e
		}},
		{"tail spread", func() error {
			_, e := bodies.NewTail(km(1), km(1), nan, 0, nothing, good, good)
			return e
		}},
		{"tail curve", func() error {
			_, e := bodies.NewTail(km(1), km(1), 0, nan, nothing, good, good)
			return e
		}},
		{"tail infinite length", func() error {
			_, e := bodies.NewTail(km(inf), km(1), 0, 0, nothing, good, good)
			return e
		}},
		{"tail glow", func() error {
			_, e := bodies.NewTail(km(1), km(1), 0, 0, nothing, good, badRGB)
			return e
		}},
		{"aurora lat", func() error {
			_, e := bodies.NewAurora(shell, measure.Radians(nan),
				measure.Radians(1), measure.Radians(1), maths.Share(0.5),
				maths.Share(0.5), 1, bodies.NewAuroraColours(good, good, good))
			return e
		}},
		{"aurora width", func() error {
			_, e := bodies.NewAurora(shell, measure.Radians(1),
				measure.Radians(inf), measure.Radians(1), maths.Share(0.5),
				maths.Share(0.5), 1, bodies.NewAuroraColours(good, good, good))
			return e
		}},
		{"aurora tilt", func() error {
			_, e := bodies.NewAurora(shell, measure.Radians(1),
				measure.Radians(1), measure.Radians(nan), maths.Share(0.5),
				maths.Share(0.5), 1, bodies.NewAuroraColours(good, good, good))
			return e
		}},
		{"aurora colour", func() error {
			_, e := bodies.NewAurora(shell, measure.Radians(1),
				measure.Radians(1), measure.Radians(1), maths.Share(0.5),
				maths.Share(0.5), 1, bodies.NewAuroraColours(good, badRGB, good))
			return e
		}},
		{"cities glow", func() error {
			_, e := bodies.NewCities(1, maths.Share(0.5), maths.Share(0.5), badRGB, 1)
			return e
		}},
		{"jets from nan", func() error {
			_, e := bodies.NewJets(km(1), km(1), 0, km(nan), nothing,
				measure.Radians(0), maths.Share(0.5), good)
			return e
		}},
		{"jets from negative", func() error {
			_, e := bodies.NewJets(km(1), km(1), 0, km(-1), nothing,
				measure.Radians(0), maths.Share(0.5), good)
			return e
		}},
		{"jets spread", func() error {
			_, e := bodies.NewJets(km(1), km(1), nan, km(0), nothing,
				measure.Radians(0), maths.Share(0.5), good)
			return e
		}},
		{"jets tilt", func() error {
			_, e := bodies.NewJets(km(1), km(1), 0, km(0), nothing,
				measure.Radians(inf), maths.Share(0.5), good)
			return e
		}},
		{"jets glow", func() error {
			_, e := bodies.NewJets(km(1), km(1), 0, km(0), nothing,
				measure.Radians(0), maths.Share(0.5), badRGB)
			return e
		}},
		{"flare opacity inf", func() error {
			_, e := bodies.NewFlare(0, 1, inf, good)
			return e
		}},
		{"flare glow", func() error {
			_, e := bodies.NewFlare(0, 1, 1, badRGB)
			return e
		}},
		{"plume toward nan", func() error {
			_, e := bodies.NewPlume(maths.V(nan, 0, 0), km(1), km(1), nothing, good)
			return e
		}},
		{"plume colour", func() error {
			_, e := bodies.NewPlume(maths.V(1, 0, 0), km(1), km(1), nothing, badRGB)
			return e
		}},
		{"plume infinite height", func() error {
			_, e := bodies.NewPlume(maths.V(1, 0, 0), km(inf), km(1), nothing, good)
			return e
		}},
		{"gas opacity inf", func() error {
			_, e := bodies.NewGas(maths.Share(0.5), inf, texture.Banded{})
			return e
		}},
		{"air colour", func() error {
			_, e := bodies.NewAir(km(1), nothing, badRGB)
			return e
		}},
		{"rings thick nan", func() error {
			_, e := bodies.NewRings(1, 2, km(nan), nothing, good)
			return e
		}},
		{"rings thick negative", func() error {
			_, e := bodies.NewRings(1, 2, km(-1), nothing, good)
			return e
		}},
		{"rings colour", func() error {
			_, e := bodies.NewRings(1, 2, km(1), nothing, badRGB)
			return e
		}},
		{"horizon light", func() error {
			_, e := bodies.NewHorizon(1, badRGB)
			return e
		}},
		{"disc infinite outer", func() error {
			_, e := bodies.NewDisc(km(1), km(inf), km(1), nothing, measure.Kelvin(1), 1)
			return e
		}},
		{"disc thick nan", func() error {
			_, e := bodies.NewDisc(km(1), km(2), km(nan), nothing, measure.Kelvin(1), 1)
			return e
		}},
		{"disc thick negative", func() error {
			_, e := bodies.NewDisc(km(1), km(2), km(-1), nothing, measure.Kelvin(1), 1)
			return e
		}},
		{"disc bright", func() error {
			_, e := bodies.NewDisc(km(1), km(2), km(1), nothing, measure.Kelvin(1), nan)
			return e
		}},
		{"disc hot inf", func() error {
			_, e := bodies.NewDisc(km(1), km(2), km(1), nothing, measure.Kelvin(inf), 1)
			return e
		}},
		{"moon phase", func() error {
			_, e := bodies.NewMoon(heavy(t), km(1), measure.Radians(nan))
			return e
		}},
	}
	for _, c := range some {
		if err := c.err(); err == nil {
			t.Errorf("%s: the constructor passed", c.name)
		}
	}
}

// heavy is a door-made body the tests hang on others.
func heavy(t *testing.T) bodies.Body {
	t.Helper()
	b, err := bodies.NewBody("h", bodies.Rock, km(1), measure.Kilograms(1))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Orbit says so when the body has no moon at the place, and Place
// refuses a body turning in a negative or no-number day.
func TestVerbsThatCannotAnswerSayWhy(t *testing.T) {
	b := heavy(t)
	if _, err := b.Orbit(0, maths.Vec{}, 1); err == nil {
		t.Error("Orbit(0) of a body with no moons passed")
	}
	if _, err := b.Orbit(-1, maths.Vec{}, 1); err == nil {
		t.Error("Orbit(-1) passed")
	}
	tilted := b.Turning(time.Hour, measure.Radians(nan))
	if err := tilted.Place(thing.NewWorld(), maths.Vec{}, 1, 2); err == nil {
		t.Error("a body tilted by NaN was placed")
	}
}

// A ground made from a terrain or look holding NaN is refused.
func TestAGroundWithNoNumberInItIsRefused(t *testing.T) {
	if _, err := bodies.NewGround(world.Terrain{Ocean: nan},
		world.Look{}); err == nil {
		t.Error("NewGround with an ocean of NaN passed")
	}
	if _, err := bodies.NewGround(world.Terrain{},
		world.Look{Lines: inf}); err == nil {
		t.Error("NewGround with infinite lines passed")
	}
	look := world.Look{}
	look.Paints[0] = badRGB
	if _, err := bodies.NewGround(world.Terrain{}, look); err == nil {
		t.Error("NewGround with a NaN paint passed")
	}
	if _, err := bodies.NewGround(world.Terrain{}, world.Look{}); err != nil {
		t.Errorf("NewGround with plain zeros refused: %v", err)
	}
}

// A part that did not come from its constructor is refused by NewBody
// and by With, by name, and never reaches Place.
func TestAPartNotMadeByItsConstructorIsRefused(t *testing.T) {
	unmade := []struct {
		name string
		part bodies.Part
	}{
		{"Coma", bodies.Coma{}},
		{"Tail", bodies.Tail{}},
		{"Clouds", bodies.Clouds{}},
		{"Aurora", bodies.Aurora{}},
		{"Cities", bodies.Cities{}},
		{"Gas", bodies.Gas{}},
		{"Jets", bodies.Jets{}},
		{"Flare", bodies.Flare{}},
		{"Plume", bodies.Plume{}},
		{"Moon", bodies.Moon{}},
		{"Horizon", bodies.Horizon{}},
		{"Ground", bodies.Ground{}},
		{"Air", bodies.Air{}},
		{"Rings", bodies.Rings{}},
		{"Disc", bodies.Disc{}},
		{"nil", nil},
	}
	for _, c := range unmade {
		_, err := bodies.NewBody("b", bodies.Giant, span(1),
			measure.Kilograms(1), c.part)
		if err == nil {
			t.Errorf("NewBody wearing %s passed", c.name)
		} else if c.name != "nil" && !strings.Contains(err.Error(),
			"bodies."+c.name) {
			t.Errorf("NewBody wearing %s: %v; want the type named",
				c.name, err)
		}
		if err != nil && c.name != "nil" &&
			!strings.Contains(err.Error(), "was not made by New"+c.name) {
			t.Errorf("NewBody wearing %s: %v; want the constructor named",
				c.name, err)
		}
		if _, err := heavy(t).With(c.part); err == nil {
			t.Errorf("With wearing %s passed", c.name)
		}
	}
}

// The review's probe: an empty gas hung on a giant was placed, and a
// volume built from a depth of zero.
func TestAnEmptyGasIsNotPlaced(t *testing.T) {
	giant, err := bodies.NewBody("g", bodies.Giant, span(1000),
		measure.Kilograms(1e27), bodies.Gas{})
	if err == nil {
		w := thing.NewWorld()
		perr := giant.Place(w, maths.Vec{}, 1, 2)
		t.Fatalf("NewBody wore an empty gas; Place: %v, volumes: %d",
			perr, len(w.Volumes))
	}
}

// A bad argument to a fluent method is held on the part, and the part
// is refused when a body is asked to wear it.
func TestAPartHoldingABadArgumentIsRefused(t *testing.T) {
	good := maths.Light{R: 1}
	jets, err := bodies.NewJets(span(10), span(1), 0.1, span(0), nothing,
		measure.Radians(0), maths.Share(0.5), good)
	if err != nil {
		t.Fatal(err)
	}
	rings, err := bodies.NewRings(1, 2, span(1), nothing, good)
	if err != nil {
		t.Fatal(err)
	}
	disc, err := bodies.NewDisc(span(1), span(2), span(1), nothing,
		measure.Kelvin(1), 1)
	if err != nil {
		t.Fatal(err)
	}
	held := map[string]bodies.Part{
		"Knotted": jets.Knotted(nan),
		"Grained": rings.Grained(1, inf),
		"Dressed": disc.Dressed(nan, 0),
	}
	for name, part := range held {
		_, err := bodies.NewBody("b", bodies.Hole, span(1),
			measure.Kilograms(1), part)
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("NewBody wearing a part held by %s: %v", name, err)
		}
	}
	if _, err := bodies.NewBody("b", bodies.Hole, span(1),
		measure.Kilograms(1), jets.Knotted(1), rings.Grained(1, 1)); err != nil {
		t.Errorf("good fluent arguments refused: %v", err)
	}
}

// moonNamed is a moon of a body of that name, an orbit out.
func moonNamed(t *testing.T, name string) bodies.Moon {
	t.Helper()
	b, err := bodies.NewBody(name, bodies.Rock, span(1), measure.Kilograms(1))
	if err != nil {
		t.Fatal(err)
	}
	m, err := bodies.NewMoon(b, span(10), measure.Radians(0))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// Two bodies made from one base by With do not share their moons: the
// fourth moon of each is its own.
func TestBodiesFromOneBaseKeepTheirOwnMoons(t *testing.T) {
	base, err := bodies.NewBody("base", bodies.Rock, span(1),
		measure.Kilograms(1), moonNamed(t, "a"), moonNamed(t, "b"),
		moonNamed(t, "c"))
	if err != nil {
		t.Fatal(err)
	}
	one, err := base.With(moonNamed(t, "ONE"))
	if err != nil {
		t.Fatal(err)
	}
	two, err := base.With(moonNamed(t, "TWO"))
	if err != nil {
		t.Fatal(err)
	}
	if got := one.Moons()[3].Body().Name(); got != "ONE" {
		t.Errorf("the first body's fourth moon is %s; want ONE", got)
	}
	if got := two.Moons()[3].Body().Name(); got != "TWO" {
		t.Errorf("the second body's fourth moon is %s; want TWO", got)
	}
	if n := len(base.Moons()); n != 3 {
		t.Errorf("the base has %d moons; want 3", n)
	}
}

// Aiming a body's copy does not re-aim the body it was copied from.
func TestAimingACopyLeavesTheOriginalAlone(t *testing.T) {
	tail, err := bodies.NewTail(span(10), span(1), 0, 0, nothing,
		maths.Light{}, maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	comet, err := bodies.NewBody("comet", bodies.Rock, span(1),
		measure.Kilograms(1), tail)
	if err != nil {
		t.Fatal(err)
	}
	copyOf := comet.Aimed(maths.V(1, 0, 0), maths.V(0, 1, 0))
	if got := comet.Tails()[0].Sun(); got != (maths.Vec{}) {
		t.Errorf("aiming the copy moved the original's sun to %v", got)
	}
	if got := copyOf.Tails()[0].Sun(); got != maths.V(1, 0, 0) {
		t.Errorf("the copy's sun is %v; want (1, 0, 0)", got)
	}
}

// Rings keep their own profile: the caller's slice going in and the
// slice read out are copies.
func TestRingsKeepTheirOwnProfile(t *testing.T) {
	rings, err := bodies.NewRings(1, 2, span(1), nothing, maths.Light{R: 1})
	if err != nil {
		t.Fatal(err)
	}
	given := []float64{1, 2, 3}
	profiled := rings.Profiled(given)
	given[0] = 99
	if got := profiled.Profile()[0]; got != 1 {
		t.Errorf("editing the slice given changed the profile to %v", got)
	}
	profiled.Profile()[1] = 99
	if got := profiled.Profile()[1]; got != 2 {
		t.Errorf("editing the slice read changed the profile to %v", got)
	}
}

// dense is a door-made body massive enough that a pair of them 10 km
// apart go round in minutes.
func dense(t *testing.T) bodies.Body {
	t.Helper()
	b, err := bodies.NewBody("d", bodies.Rock, span(1),
		measure.Kilograms(1e24))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Binary refuses what Place refuses: a scale that is no size, a body
// resized to no size, and it refuses a distance that is no distance.
func TestBinaryRefusesWhatPlaceRefuses(t *testing.T) {
	one := dense(t)
	for _, scale := range []float64{-1, 0, nan, inf} {
		if err := bodies.Binary(thing.NewWorld(), one, one, maths.Vec{},
			span(10), scale, 2); err == nil {
			t.Errorf("Binary at a scale of %v passed", scale)
		}
		if err := one.Place(thing.NewWorld(), maths.Vec{}, scale,
			2); err == nil {
			t.Errorf("Place at a scale of %v passed", scale)
		}
	}
	for _, apart := range []float64{-1, nan, inf} {
		if err := bodies.Binary(thing.NewWorld(), one, one, maths.Vec{},
			span(apart), 1, 2); err == nil {
			t.Errorf("Binary %v km apart passed", apart)
		}
	}
	if err := bodies.Binary(thing.NewWorld(), one, one, maths.Vec{},
		span(10), 1, 2); err != nil {
		t.Errorf("a plain pair was refused: %v", err)
	}
}

// A body resized to no size is refused at Place and at Binary, naming
// the radius.
func TestAResizedBodyOfNoSizeIsRefused(t *testing.T) {
	for _, radius := range []float64{-1, 0, nan, inf} {
		sized := dense(t).Resized(span(radius))
		err := sized.Place(thing.NewWorld(), maths.Vec{}, 1, 2)
		if err == nil || !strings.Contains(err.Error(), "resized") {
			t.Errorf("Place resized to %v km: %v", radius, err)
		}
		err = bodies.Binary(thing.NewWorld(), dense(t), sized,
			maths.Vec{}, span(10), 1, 2)
		if err == nil || !strings.Contains(err.Error(), "resized") {
			t.Errorf("Binary resized to %v km: %v", radius, err)
		}
	}
	ok := dense(t).Resized(span(2))
	if err := ok.Place(thing.NewWorld(), maths.Vec{}, 1, 2); err != nil {
		t.Errorf("a body resized to 2 km was refused: %v", err)
	}
}

// A hole spinning backward faster than its mass allows has no horizon,
// as one spinning forward that fast has none: Place and Binary refuse
// the magnitude.
func TestAHoleSpinningBackwardPastItsMassIsRefused(t *testing.T) {
	hole, err := bodies.NewBody("h", bodies.Hole, span(1),
		measure.Kilograms(1e24))
	if err != nil {
		t.Fatal(err)
	}
	for _, spin := range []float64{-2, -1, 1, 2} {
		spun := hole.Spinning(spin)
		err := spun.Place(thing.NewWorld(), maths.Vec{}, 1, 2)
		if err == nil || !strings.Contains(err.Error(), "horizon") {
			t.Errorf("Place of a hole spinning at %v: %v", spin, err)
		}
		err = bodies.Binary(thing.NewWorld(), hole, spun, maths.Vec{},
			span(10), 1, 2)
		if err == nil || !strings.Contains(err.Error(), "horizon") {
			t.Errorf("Binary with a hole spinning at %v: %v", spin, err)
		}
	}
	for _, spin := range []float64{-0.5, 0, 0.5} {
		spun := hole.Spinning(spin)
		if err := spun.Place(thing.NewWorld(), maths.Vec{}, 1, 2); err != nil {
			t.Errorf("Place of a hole spinning at %v: %v", spin, err)
		}
	}
}

// tailOf is a door-made tail of a length, for telling tails apart.
func tailOf(t *testing.T, length float64) bodies.Tail {
	t.Helper()
	tail, err := bodies.NewTail(span(length), span(1), 0, 0, nothing,
		maths.Light{}, maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	return tail
}

// plumeOf is a door-made plume of a height, for telling plumes apart.
func plumeOf(t *testing.T, height float64) bodies.Plume {
	t.Helper()
	plume, err := bodies.NewPlume(maths.V(0, 1, 0), span(height), span(1),
		nothing, maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	return plume
}

// Two bodies made from one base by With do not share their tails: three
// tails leave room for a fourth in the base's own array, so a With that
// appended without copying would give both bodies the same fourth tail.
func TestBodiesFromOneBaseKeepTheirOwnTails(t *testing.T) {
	base, err := bodies.NewBody("base", bodies.Rock, span(1),
		measure.Kilograms(1), tailOf(t, 1), tailOf(t, 2), tailOf(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	one, err := base.With(tailOf(t, 101))
	if err != nil {
		t.Fatal(err)
	}
	two, err := base.With(tailOf(t, 202))
	if err != nil {
		t.Fatal(err)
	}
	if got := one.Tails()[3].Length(); got != 101 {
		t.Errorf("the first body's fourth tail is %v km; want 101", got)
	}
	if got := two.Tails()[3].Length(); got != 202 {
		t.Errorf("the second body's fourth tail is %v km; want 202", got)
	}
	if n := len(base.Tails()); n != 3 {
		t.Errorf("the base has %d tails; want 3", n)
	}
}

// Two bodies made from one base by With do not share their plumes.
func TestBodiesFromOneBaseKeepTheirOwnPlumes(t *testing.T) {
	base, err := bodies.NewBody("base", bodies.Rock, span(1),
		measure.Kilograms(1), plumeOf(t, 1), plumeOf(t, 2), plumeOf(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	one, err := base.With(plumeOf(t, 101))
	if err != nil {
		t.Fatal(err)
	}
	two, err := base.With(plumeOf(t, 202))
	if err != nil {
		t.Fatal(err)
	}
	if got := one.Plumes()[3].Height(); got != 101 {
		t.Errorf("the first body's fourth plume is %v km; want 101", got)
	}
	if got := two.Plumes()[3].Height(); got != 202 {
		t.Errorf("the second body's fourth plume is %v km; want 202", got)
	}
	if n := len(base.Plumes()); n != 3 {
		t.Errorf("the base has %d plumes; want 3", n)
	}
}

// Aiming a body made by With re-aims neither its base nor a sibling made
// from the same base.
func TestAimingABodyMadeByWithLeavesItsBaseAndSiblingAlone(t *testing.T) {
	base, err := bodies.NewBody("base", bodies.Rock, span(1),
		measure.Kilograms(1), tailOf(t, 1), tailOf(t, 2), tailOf(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	one, err := base.With(tailOf(t, 101))
	if err != nil {
		t.Fatal(err)
	}
	two, err := base.With(tailOf(t, 202))
	if err != nil {
		t.Fatal(err)
	}
	sun := maths.V(1, 0, 0)
	aimed := one.Aimed(sun, maths.V(0, 1, 0))
	for place, tail := range aimed.Tails() {
		if tail.Sun() != sun {
			t.Errorf("the aimed body's tail %d points from %v; want %v",
				place, tail.Sun(), sun)
		}
	}
	for name, body := range map[string]bodies.Body{
		"the base": base, "the body aimed": one, "the sibling": two} {
		for place, tail := range body.Tails() {
			if tail.Sun() != (maths.Vec{}) {
				t.Errorf("aiming moved %s's tail %d to %v", name, place,
					tail.Sun())
			}
		}
	}
}

// A typed nil pointer to a part is a nil part, refused by name, and is
// never called: a pointer to a part satisfies Part, and calling a method
// through it would crash.
func TestATypedNilPartIsRefusedByName(t *testing.T) {
	for name, part := range map[string]bodies.Part{
		"Gas": (*bodies.Gas)(nil), "Tail": (*bodies.Tail)(nil),
		"Moon": (*bodies.Moon)(nil), "Ground": (*bodies.Ground)(nil),
	} {
		_, err := bodies.NewBody("b", bodies.Giant, span(1),
			measure.Kilograms(1), part)
		if err == nil || !strings.Contains(err.Error(), "nil *bodies."+name) {
			t.Errorf("NewBody wearing a nil *%s: %v", name, err)
		}
	}
}

// A shell that was not made by NewShell is no band, and the parts that
// take one refuse it.
func TestAZeroShellIsRefused(t *testing.T) {
	_, err := bodies.NewClouds(bodies.Shell{}, maths.Share(0.5), nothing, 1)
	if err == nil || !strings.Contains(err.Error(), "NewShell") {
		t.Errorf("NewClouds with a zero Shell: %v", err)
	}
	_, err = bodies.NewAurora(bodies.Shell{}, measure.Radians(0),
		measure.Radians(0), measure.Radians(0), maths.Share(0.5),
		maths.Share(0.5), 1, bodies.AuroraColours{})
	if err == nil || !strings.Contains(err.Error(), "NewShell") {
		t.Errorf("NewAurora with a zero Shell: %v", err)
	}
}

// a caller's type that embeds a nil part satisfies Part, and wearing it
// would crash: it is refused with an error instead.
type holeInGas struct{ *bodies.Gas }
type holeInPart struct{ bodies.Part }

func TestAnEmbeddedNilPartIsRefusedNotPanicked(t *testing.T) {
	for name, part := range map[string]bodies.Part{
		"an embedded nil *Gas": holeInGas{}, "an embedded nil Part": holeInPart{},
	} {
		if _, err := bodies.NewBody("b", bodies.Giant, span(1),
			measure.Kilograms(1), part); err == nil {
			t.Errorf("NewBody wearing %s passed", name)
		}
		if _, err := heavy(t).With(part); err == nil {
			t.Errorf("With wearing %s passed", name)
		}
	}
}

// A refusal names the input and gives its value, at least once for each
// constructor, so a caller can tell what was wrong from the message.
func TestARefusalNamesTheInputAndItsValue(t *testing.T) {
	good := maths.Light{R: 1}
	shell := bodies.NewShell(km(1), km(2))
	colours := bodies.NewAuroraColours(good, good, good)
	for name, c := range map[string]struct {
		err  func() error
		want []string
	}{
		"NewComa": {func() error {
			_, e := bodies.NewComa(km(inf), nothing, good, good)
			return e
		}, []string{"bodies:", "+Inf"}},
		"NewTail": {func() error {
			_, e := bodies.NewTail(km(1), km(1), nan, 0, nothing, good, good)
			return e
		}, []string{"a tail's spread", "NaN"}},
		"NewClouds": {func() error {
			nanLowShell := bodies.NewShell(km(nan), km(2))
			_, e := bodies.NewClouds(nanLowShell,
				maths.Share(0.5), nothing, 1)
			return e
		}, []string{"a shell from NaN to 2 km"}},
		"NewAurora": {func() error {
			_, e := bodies.NewAurora(shell, measure.Radians(nan),
				measure.Radians(1), measure.Radians(1), maths.Share(0.5),
				maths.Share(0.5), 1, colours)
			return e
		}, []string{"an aurora's latitude", "NaN"}},
		"NewCities": {func() error {
			_, e := bodies.NewCities(1, maths.Share(0.5), maths.Share(0.5),
				badRGB, 1)
			return e
		}, []string{"the cities' glow", "NaN"}},
		"NewJets": {func() error {
			_, e := bodies.NewJets(km(1), km(1), nan, km(0), nothing,
				measure.Radians(0), maths.Share(0.5), good)
			return e
		}, []string{"spread", "NaN"}},
		"NewFlare": {func() error {
			_, e := bodies.NewFlare(0, 1, inf, good)
			return e
		}, []string{"+Inf"}},
		"NewPlume": {func() error {
			_, e := bodies.NewPlume(maths.V(1, 0, 0), km(inf), km(1),
				nothing, good)
			return e
		}, []string{"+Inf"}},
		"NewGas": {func() error {
			_, e := bodies.NewGas(maths.Share(0.5), inf, texture.Banded{})
			return e
		}, []string{"a gas of opacity +Inf"}},
		"NewAir": {func() error {
			_, e := bodies.NewAir(km(1), nothing, badRGB)
			return e
		}, []string{"the air's colour", "NaN"}},
		"NewRings": {func() error {
			_, e := bodies.NewRings(1, 2, km(-1), nothing, good)
			return e
		}, []string{"rings -1 km thick"}},
		"NewHorizon": {func() error {
			_, e := bodies.NewHorizon(1, badRGB)
			return e
		}, []string{"NaN"}},
		"NewDisc": {func() error {
			_, e := bodies.NewDisc(km(1), km(2), km(-1), nothing,
				measure.Kelvin(1), 1)
			return e
		}, []string{"a disc -1 km thick"}},
		"NewMoon": {func() error {
			_, e := bodies.NewMoon(heavy(t), km(1), measure.Radians(nan))
			return e
		}, []string{"a moon's phase", "NaN"}},
		"NewGround": {func() error {
			_, e := bodies.NewGround(world.Terrain{Ocean: nan}, world.Look{})
			return e
		}, []string{"the terrain's ocean", "NaN"}},
	} {
		err := c.err()
		if err == nil {
			t.Errorf("%s: the constructor passed", name)
			continue
		}
		for _, want := range c.want {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s: %q does not say %q", name, err, want)
			}
		}
	}
}

// A body that is not a hole never reads its spin, but Spin() hands it
// back, so a spin that is not a finite number is refused at Place.
func TestABodyThatIsNoHoleRefusesASpinThatIsNoNumber(t *testing.T) {
	rock, err := bodies.NewBody("r", bodies.Rock, span(1000),
		measure.Kilograms(1e24))
	if err != nil {
		t.Fatal(err)
	}
	for _, spin := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		err := rock.Spinning(spin).Place(thing.NewWorld(), maths.Vec{}, 1, 2)
		if err == nil || !strings.Contains(err.Error(), "finite") {
			t.Errorf("Place of a rock spinning at %v: %v", spin, err)
		}
	}
	if err := rock.Spinning(-3).Place(thing.NewWorld(), maths.Vec{}, 1,
		2); err != nil {
		t.Errorf("Place of a rock spinning at -3: %v", err)
	}
}
