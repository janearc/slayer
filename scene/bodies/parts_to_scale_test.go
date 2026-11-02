package bodies_test

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// a body of 8 km put in the world at half a unit a kilometre is 4 units
// across its radius, and each part it carries is that, by hand:
//
// a length in kilometres is half as many units, a length in shares of
// the radius is that many 4s, and a density per kilometre is twice as
// many per unit, since a unit is two kilometres. the numbers are halves
// and quarters, so a float holds each exactly.
func TestEachPartIsMadeToTheBodysScale(t *testing.T) {
	light := func(r, g, b float64) maths.Light { return maths.Light{R: r, G: g, B: b} }
	b := bodies.Body{Name: "p", Kind: bodies.Star, Radius: 8, Seed: 5,
		Air: &bodies.Air{Height: 6, Dense: 0.25, Colour: light(0.25, 0.5, 1)},
		Aurora: &bodies.Aurora{Low: 128, High: 384, Lat: 1.25, Width: 0.5,
			Tilt: math.Pi / 2, Strength: 0.25, Curtains: 0.75, Seed: 7,
			Fringe: light(0.5, 1, 2), Bottom: light(1, 2, 4), Top: light(2, 4, 8)},
		Clouds: &bodies.Clouds{Low: 64, High: 128, Cover: 0.75, Dense: 0.25, Seed: 9},
		Rings: &bodies.Rings{Inner: 1.5, Outer: 2.25, Thick: 6, Dense: 0.375,
			Profile: []float64{1, 0.5}, Grain: 0.25, Ringlets: 12,
			Colour: light(0.5, 0.25, 0.125)},
		Coma: &bodies.Coma{Radius: 12, Dense: 0.25, Colour: light(0.25, 0.25, 0.5),
			Glow: light(0, 0.5, 0)},
	}
	w := thing.NewWorld()
	if err := b.Place(w, maths.V(10, 0, 0), 0.5, 2); err != nil {
		t.Fatal(err)
	}

	// 8 km x 0.5 = 4 units; the scale height 6 km is 3 units, and the
	// 0.25 a kilometre at the ground is 0.5 a unit.
	air := volumeNamed(t, w, "p air")
	same(t, "the air", air.Medium, medium.Air{Radius: 4, Height: 3, Thick: 0.5})
	same(t, "the air's colour", air.Material.Colour, light(0.25, 0.5, 1))

	// the shell is 128 to 384 km, 64 to 192 units; it is 256 km deep, so a
	// kilometre of it is 1/256 and a unit of it twice that, 1/128. the
	// tilt of a quarter turn about x takes the spin's north, +y, to +z.
	// the colours are the strength, a quarter, of the ones given.
	aurora := volumeNamed(t, w, "p aurora")
	got := aurora.Medium.(medium.Aurora)
	nearVec(t, "the aurora's axis", got.Axis, maths.V(0, 0, 1), 1e-12)
	got.Axis = maths.V(0, 0, 1)
	same(t, "the aurora", got, medium.Aurora{Radius: 4, Low: 64, High: 192,
		Lat: 1.25, Width: 0.5, Axis: maths.V(0, 0, 1), Thick: 1.0 / 128,
		Curtains: 0.75, Seed: 7})
	same(t, "the aurora's colours", aurora.Material.Shine, texture.Aurora{Radius: 4,
		Low: 64, High: 192, Fringe: light(0.125, 0.25, 0.5),
		Bottom: light(0.25, 0.5, 1), Top: light(0.5, 1, 2)})

	// 64 to 128 km is 32 to 64 units, and 0.25 a kilometre is 0.5 a unit.
	clouds := volumeNamed(t, w, "p clouds")
	same(t, "the clouds", clouds.Medium, medium.Clouds{Radius: 4, Low: 32, High: 64,
		Cover: 0.75, Thick: 0.5, Seed: 9})
	same(t, "the clouds' colour", clouds.Material.Colour, maths.Grey(1))

	// the rings run from 1.5 to 2.25 radii, which are 6 to 9 units at a
	// radius of 4; 6 km thick is 3 units; 0.375 a kilometre is 0.75.
	rings := volumeNamed(t, w, "p rings")
	same(t, "the rings", rings.Medium, medium.Disc{Inner: 6, Outer: 9, Thick: 3,
		Dense: 0.75, Profile: []float64{1, 0.5}, Grain: 0.25, Ringlets: 12})
	same(t, "the rings' colour", rings.Material.Colour, light(0.5, 0.25, 0.125))

	// the coma is 12 km across its radius, 6 units, with a ragged scale of
	// a third of that, 2; 0.25 a kilometre is 0.5; its seed is the
	// body's, 5; the noise takes up to half.
	coma := volumeNamed(t, w, "p coma")
	same(t, "the coma", coma.Medium, medium.Wispy{
		Of:   medium.Puff{Radius: 6, Thick: 0.5},
		Seed: 5, Scale: 2, Amount: 0.5})
	same(t, "the coma's colour", coma.Material.Colour, light(0.25, 0.25, 0.5))
	same(t, "the coma's glow", coma.Material.Glow, light(0, 0.5, 0))
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
	g := bodies.Body{Name: "g", Kind: bodies.Giant, Radius: 8,
		Gas: &bodies.Gas{Depth: 0.125, Opacity: 3, Bands: bands}}
	w := thing.NewWorld()
	if err := g.Place(w, maths.Vec{}, 0.5, 2); err != nil {
		t.Fatal(err)
	}
	gas := volumeNamed(t, w, "g")
	same(t, "the gas", gas.Medium, medium.Giant{Radius: 4, Depth: 0.5, Thick: 6})
	want := bands
	want.Axis = maths.Up
	same(t, "the bands", gas.Material.Texture, want)
}

// a giant with no gas, and a body that is not a giant with a gas of no
// depth, are placed without a fault: only a giant's gas is read.
func TestOnlyAGiantsGasIsRead(t *testing.T) {
	w := thing.NewWorld()
	if err := (bodies.Body{Name: "g", Kind: bodies.Giant, Radius: 8}).Place(w, maths.Vec{}, 0.5, 2); err != nil {
		t.Errorf("a giant with no gas: %v", err)
	}
	if len(w.Volumes) != 0 {
		t.Errorf("a giant with no gas has %d volumes", len(w.Volumes))
	}
	star := bodies.Body{Name: "s", Kind: bodies.Star, Radius: 8, Gas: &bodies.Gas{}}
	if err := star.Place(thing.NewWorld(), maths.Vec{}, 0.5, 2); err != nil {
		t.Errorf("a star with a gas of no depth: %v", err)
	}
}

// a flare is a unit puff born at its time and growing at its speed in
// kilometres a second: 24 km/s at half a unit a kilometre is 12 units
// a second, so a second after it is born it is 12 across.
//
// before it is born it is nothing. the optical depth across its radius
// is the one given.
func TestAFlareGrowsAtItsSpeedFromItsBirth(t *testing.T) {
	b := bodies.Body{Name: "f", Kind: bodies.Star, Radius: 8,
		Flare: &bodies.Flare{At: 3 * time.Second, Speed: 24, Opacity: 3,
			Glow: maths.Light{R: 1, G: 0.5, B: 0.25}}}
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
	if s := flare.At(2 * time.Second).Scale; s > 1e-9 {
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
	b := bodies.Body{Name: "p", Kind: bodies.Star, Radius: 8,
		Plumes: []bodies.Plume{{Toward: maths.V(0, 0, 2), Height: 40, Width: 24,
			Dense: 0.125, Colour: maths.Light{R: 0.5, G: 0.25, B: 0.125}}}}
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
	jets := &bodies.Jets{Length: 64, Base: 0.5, Spread: 0.125, From: 16,
		Dense: 0.0625, Tilt: math.Pi / 3, Speed: 0.5, Knots: 0.75,
		Glow: maths.Light{R: 0.25, G: 0.5, B: 1}}
	b := bodies.Body{Name: "j", Kind: bodies.Star, Radius: 8, Seed: 5, Jets: jets}
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
	// a third of a turn up about x takes +y to (0, cos 60, sin 60)
	nearVec(t, "the jets' axis", v.At(0).Orientation.Out(maths.Up),
		maths.V(0, 0.5, math.Sqrt(3)/2), 1e-12)
	nearVec(t, "where the jets are", v.At(0).Origin, maths.V(10, 0, 0), 1e-12)
}

// jets with no base have no widths to ride out in, and their knots stand still,
// with a flow of zero and not of infinity. a base of under a kilometre has one:
// the 0.5 above.
func TestJetsWithNoBaseHaveNoFlow(t *testing.T) {
	b := bodies.Body{Name: "j", Kind: bodies.Star, Radius: 8,
		Jets: &bodies.Jets{Length: 64, Spread: 0.125, Dense: 0.0625, Speed: 0.5}}
	w := thing.NewWorld()
	if err := b.Place(w, maths.Vec{}, 0.5, 2); err != nil {
		t.Fatal(err)
	}
	if flow := volumeNamed(t, w, "j jets").Medium.(medium.Jet).Flow; flow != 0 {
		t.Errorf("a jet with no base flows at %v", flow)
	}
}
