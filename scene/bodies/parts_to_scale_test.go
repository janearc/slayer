package bodies_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// km is a length in the unit bodies speak, for the tests' own hands.
func km(amount float64) bodies.Span { return bodies.Kilometres(amount) }

// rad is an angle in radians, for the tests' own hands.
func rad(amount float64) measure.Angle { return measure.Radians(amount) }

// a body of 8 km put in the world at half a unit a kilometre is 4 units
// across its radius, and each part it carries is that, by hand:
//
// a length in kilometres is half as many units, a length in shares of
// the radius is that many 4s, and a density per kilometre is twice as
// many per unit, since a unit is two kilometres. the numbers are halves
// and quarters, so a float holds each exactly.
func TestEachPartIsMadeToTheBodysScale(t *testing.T) {
	light := func(r, g, b float64) maths.Light { return maths.Light{R: r, G: g, B: b} }
	air, err := bodies.NewAir(km(6), bodies.PerKilometre(0.25),
		light(0.25, 0.5, 1))
	if err != nil {
		t.Fatal(err)
	}
	auroraShell := bodies.NewShell(km(128), km(384))
	auroraColours := bodies.NewAuroraColours(light(0.5, 1, 2), light(1, 2, 4),
		light(2, 4, 8))
	aurora, err := bodies.NewAurora(auroraShell,
		rad(1.25), rad(0.5), rad(math.Pi/2), maths.Share(0.25),
		maths.Share(0.75), 7,
		auroraColours)
	if err != nil {
		t.Fatal(err)
	}
	cloudShell := bodies.NewShell(km(64), km(128))
	clouds, err := bodies.NewClouds(cloudShell,
		maths.Share(0.75), bodies.PerKilometre(0.25), 9)
	if err != nil {
		t.Fatal(err)
	}
	rings, err := bodies.NewRings(1.5, 2.25, km(6),
		bodies.PerKilometre(0.375), light(0.5, 0.25, 0.125))
	if err != nil {
		t.Fatal(err)
	}
	rings = rings.Profiled([]float64{1, 0.5}).Grained(0.25, 12)
	coma, err := bodies.NewComa(km(12), bodies.PerKilometre(0.25),
		light(0.25, 0.25, 0.5), light(0, 0.5, 0))
	if err != nil {
		t.Fatal(err)
	}
	b := made(t, "p", bodies.Star, 8, 1, air, aurora, clouds, rings,
		coma).Seeded(5)
	w := thing.NewWorld()
	if err := b.Place(w, maths.V(10, 0, 0), 0.5, 2); err != nil {
		t.Fatal(err)
	}

	// 8 km x 0.5 = 4 units; the scale height 6 km is 3 units, and the
	// 0.25 a kilometre at the ground is 0.5 a unit.
	placedAir := volumeNamed(t, w, "p air")
	same(t, "the air", placedAir.Medium, medium.Air{Radius: 4, Height: 3, Thick: 0.5})
	same(t, "the air's colour", placedAir.Material.Colour, light(0.25, 0.5, 1))

	// the shell is 128 to 384 km, 64 to 192 units; it is 256 km deep, so a
	// kilometre of it is 1/256 and a unit of it twice that, 1/128. the
	// tilt of a quarter turn about x takes the spin's north, +y, to +z.
	// the colours are the strength, a quarter, of the ones given.
	placedAurora := volumeNamed(t, w, "p aurora")
	got := placedAurora.Medium.(medium.Aurora)
	nearVec(t, "the aurora's axis", got.Axis, maths.V(0, 0, 1), 1e-12)
	got.Axis = maths.V(0, 0, 1)
	same(t, "the aurora", got, medium.Aurora{Radius: 4, Low: 64, High: 192,
		Lat: 1.25, Width: 0.5, Axis: maths.V(0, 0, 1), Thick: 1.0 / 128,
		Curtains: 0.75, Seed: 7})
	same(t, "the aurora's colours", placedAurora.Material.Shine, texture.Aurora{Radius: 4,
		Low: 64, High: 192, Fringe: light(0.125, 0.25, 0.5),
		Bottom: light(0.25, 0.5, 1), Top: light(0.5, 1, 2)})

	// 64 to 128 km is 32 to 64 units, and 0.25 a kilometre is 0.5 a unit.
	placedClouds := volumeNamed(t, w, "p clouds")
	same(t, "the clouds", placedClouds.Medium, medium.Clouds{Radius: 4, Low: 32, High: 64,
		Cover: 0.75, Thick: 0.5, Seed: 9})
	same(t, "the clouds' colour", placedClouds.Material.Colour, maths.Grey(1))

	// the rings run from 1.5 to 2.25 radii, which are 6 to 9 units at a
	// radius of 4; 6 km thick is 3 units; 0.375 a kilometre is 0.75.
	placedRings := volumeNamed(t, w, "p rings")
	same(t, "the rings", placedRings.Medium, medium.Disc{Inner: 6, Outer: 9, Thick: 3,
		Dense: 0.75, Profile: []float64{1, 0.5}, Grain: 0.25, Ringlets: 12})
	same(t, "the rings' colour", placedRings.Material.Colour, light(0.5, 0.25, 0.125))

	// the coma is 12 km across its radius, 6 units, with a ragged scale of
	// a third of that, 2; 0.25 a kilometre is 0.5; its seed is the
	// body's, 5; the noise takes up to half.
	placedComa := volumeNamed(t, w, "p coma")
	same(t, "the coma", placedComa.Medium, medium.Wispy{
		Of:   medium.Puff{Radius: 6, Thick: 0.5},
		Seed: 5, Scale: 2, Amount: 0.5})
	same(t, "the coma's colour", placedComa.Material.Colour, light(0.25, 0.25, 0.5))
	same(t, "the coma's glow", placedComa.Material.Glow, light(0, 0.5, 0))
}

// a giant's gas is a medium cut to the scale: a radius of 8 km at half a
// unit a kilometre is 4 units, and the depth, an eighth of the radius,
// is half a unit below the tops.
//
// an optical depth of 3 across half a unit is a density of 6 a unit.
// its bands are laid about the up axis, whichever they were given.
func TestAGiantsGasIsCutToTheScale(t *testing.T) {
	bands := texture.Banded{Paints: []maths.Light{maths.Grey(0.25), maths.Grey(0.5)},
		Axis: maths.V(1, 0, 0), Roil: 0.5}
	gas, err := bodies.NewGas(maths.Share(0.125), 3, bands)
	if err != nil {
		t.Fatal(err)
	}
	g := made(t, "g", bodies.Giant, 8, 1, gas)
	w := thing.NewWorld()
	if err := g.Place(w, maths.Vec{}, 0.5, 2); err != nil {
		t.Fatal(err)
	}
	placed := volumeNamed(t, w, "g")
	same(t, "the gas", placed.Medium, medium.Giant{Radius: 4, Depth: 0.5, Thick: 6})
	want := bands
	want.Axis = maths.Up
	same(t, "the bands", placed.Material.Texture, want)
}

// a giant with no gas is placed without a fault, and a star wearing a
// gas is placed without reading it: only a giant's gas is read.
func TestOnlyAGiantsGasIsRead(t *testing.T) {
	w := thing.NewWorld()
	if err := made(t, "g", bodies.Giant, 8, 1).Place(w, maths.Vec{}, 0.5, 2); err != nil {
		t.Errorf("a giant with no gas: %v", err)
	}
	if len(w.Volumes) != 0 {
		t.Errorf("a giant with no gas has %d volumes", len(w.Volumes))
	}
	gas, err := bodies.NewGas(maths.Share(0.5), 1, texture.Banded{})
	if err != nil {
		t.Fatal(err)
	}
	w = thing.NewWorld()
	if err := made(t, "s", bodies.Star, 8, 1, gas).Place(w, maths.Vec{}, 0.5, 2); err != nil {
		t.Errorf("a star with a gas: %v", err)
	}
	if len(w.Volumes) != 0 {
		t.Errorf("a star's gas was read: %d volumes", len(w.Volumes))
	}
}

// a flare is a unit puff born at its time and growing at its speed in
// kilometres a second: 24 km/s at half a unit a kilometre is 12 units
// a second, so a second after it is born it is 12 across.
//
// before it is born it is nothing. the optical depth across its radius
// is the one given.
func TestAFlareGrowsAtItsSpeedFromItsBirth(t *testing.T) {
	fireball, err := bodies.NewFlare(3*time.Second, 24, 3,
		maths.Light{R: 1, G: 0.5, B: 0.25})
	if err != nil {
		t.Fatal(err)
	}
	b := made(t, "f", bodies.Star, 8, 1, fireball)
	at := maths.V(10, 0, 0)
	w := thing.NewWorld()
	if err := b.Place(w, at, 0.5, 2); err != nil {
		t.Fatal(err)
	}
	flare := volumeNamed(t, w, "f flare")
	same(t, "the flare", flare.Medium, medium.Puff{Radius: 1, Thick: 3})
	same(t, "the flare's glow", flare.Material.Glow, maths.Light{R: 1, G: 0.5, B: 0.25})
	near(t, "its scale a second after", flare.At(4*time.Second).Scale, 12, 1e-12)
	near(t, "its scale three seconds after", flare.At(6*time.Second).Scale, 36, 1e-12)
	if s := flare.At(2 * time.Second).Scale; !(s <= 1e-9) {
		t.Errorf("a second before it is born its scale is %v", s)
	}
	nearVec(t, "where it grows", flare.At(4*time.Second).Origin, at, 1e-12)
}

// a plume stands on the ground where it points, and rises along the
// ground's up: toward +z of a body 4 units across its radius it is at
// z = 4 from the body, its y along +z and its x along +x.
//
// the height 40 km is 20 units, the width 24 km is 12, and 0.125 a
// kilometre is 0.25 a unit.
func TestAPlumeStandsOnTheGroundWhereItPoints(t *testing.T) {
	column, err := bodies.NewPlume(maths.V(0, 0, 2), km(40), km(24),
		bodies.PerKilometre(0.125), maths.Light{R: 0.5, G: 0.25, B: 0.125})
	if err != nil {
		t.Fatal(err)
	}
	b := made(t, "p", bodies.Star, 8, 1, column)
	w := thing.NewWorld()
	if err := b.Place(w, maths.V(10, 0, 0), 0.5, 2); err != nil {
		t.Fatal(err)
	}
	plume := volumeNamed(t, w, "p plume")
	same(t, "the plume", plume.Medium, medium.Plume{Height: 20, Width: 12, Thick: 0.25})
	same(t, "the plume's colour", plume.Material.Colour, maths.Light{R: 0.5, G: 0.25, B: 0.125})
	s := plume.At(0)
	nearVec(t, "where the plume stands", s.Origin, maths.V(10, 0, 4), 1e-12)
	nearVec(t, "the plume's rise", s.Orientation.Out(maths.Up), maths.V(0, 0, 1), 1e-12)
	nearVec(t, "the plume's x", s.Orientation.Out(maths.XAxis), maths.V(1, 0, 0), 1e-12)
}

// a body's jets are a cone each way: 64 km long is 32 units, 0.5 km at
// the base is 0.25, started 16 km out, 8 units; 0.0625 a kilometre is
// 0.125 a unit.
//
// the seed is the body's, 5, and 4 more, since the jets are not to grow
// from the disc's noise; the lean is the tilt about x.
//
// the knots ride out at the jet's speed, 0.5 c = 149,896.229 km/s, in
// base widths a second: over the base, 0.5 km, 299,792.458.
func TestJetsAreAConeCutToTheScale(t *testing.T) {
	jets, err := bodies.NewJets(km(64), km(0.5), 0.125, km(16),
		bodies.PerKilometre(0.0625), rad(math.Pi/3), maths.Share(0.5),
		maths.Light{R: 0.25, G: 0.5, B: 1})
	if err != nil {
		t.Fatal(err)
	}
	b := made(t, "j", bodies.Star, 8, 1, jets.Knotted(0.75)).Seeded(5)
	w := thing.NewWorld()
	if err := b.Place(w, maths.V(10, 0, 0), 0.5, 2); err != nil {
		t.Fatal(err)
	}
	v := volumeNamed(t, w, "j jets")
	got := v.Medium.(medium.Jet)
	near(t, "the knots' flow", got.Flow, 299792.458, 1e-12)
	got.Flow = 0
	same(t, "the jets", got, medium.Jet{Length: 32, Base: 0.25, Spread: 0.125,
		Thick: 0.125, From: 8, Knots: 0.75, Seed: 9})
	same(t, "the jets' glow", v.Material.Glow, maths.Light{R: 0.25, G: 0.5, B: 1})
	if v.Beam == nil || v.Beam.Speed != 0.5 || v.Beam.Axis != maths.Up || v.Beam.Round {
		t.Errorf("the jets' beam is %+v", v.Beam)
	}
	jetsAxis := v.At(0).Orientation.Out(maths.Up)
	sinSixty := math.Sqrt(3) / 2
	// a third of a turn up about x takes +y to (0, cos 60, sin 60)
	nearVec(t, "the jets' axis", jetsAxis,
		maths.V(0, 0.5, sinSixty), 1e-12)
	nearVec(t, "where the jets are", v.At(0).Origin, maths.V(10, 0, 0), 1e-12)
}

// jets with no base cannot be made: the knots would have no widths to
// ride out in, and their flow would be infinity. the door refuses them
// by name, so no jet ever flows endlessly.
func TestJetsWithNoBaseAreRefusedAtTheDoor(t *testing.T) {
	_, err := bodies.NewJets(km(64), km(0), 0.125, km(0),
		bodies.PerKilometre(0.0625), rad(0), maths.Share(0.5),
		maths.Light{})
	if err == nil || !strings.Contains(err.Error(), "base") {
		t.Errorf("jets with no base: %v", err)
	}
}
