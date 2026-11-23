package bodies_test

import (
	"image"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// xray is a spectrum that gives off x-rays by a share, for a part to
// carry and a reader to give back.
func xray(share float64) thing.Spectrum {
	return thing.Spectrum{Gives: map[radiation.Range]float64{
		radiation.XRay(): share}}
}

// sameRadiant fails the test when two radiants are not the same
// spectrum.
func sameRadiant(t *testing.T, what string, got, want thing.Radiant) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s is %v, want %v", what, got, want)
	}
}

// lit is a light of three different channels, so a reader that swapped
// two would show.
func lit(r, g, b float64) maths.Light { return maths.Light{R: r, G: g, B: b} }

// Each reader on a small part gives back the argument that went in at
// its own place.
func TestShellComaTailAndCloudsReadBackTheirArguments(t *testing.T) {
	shell := bodies.NewShell(km(2), km(11))
	if shell.Low() != 2 || shell.High() != 11 {
		t.Errorf("the shell is %v to %v, want 2 to 11", shell.Low(),
			shell.High())
	}
	coma, err := bodies.NewComa(km(120), bodies.PerKilometre(0.03),
		lit(0.1, 0.2, 0.3), lit(0.4, 0.5, 0.6))
	if err != nil {
		t.Fatal(err)
	}
	if coma.Radius() != 120 || coma.Dense() != 0.03 ||
		coma.Colour() != lit(0.1, 0.2, 0.3) ||
		coma.Glow() != lit(0.4, 0.5, 0.6) {
		t.Errorf("the coma reads %v %v %v %v, want 120 0.03 and its "+
			"two lights", coma.Radius(), coma.Dense(), coma.Colour(),
			coma.Glow())
	}
	tail, err := bodies.NewTail(km(900), km(30), 0.3, 0.7,
		bodies.PerKilometre(0.02), lit(0.7, 0.8, 0.9), lit(0.2, 0.1, 0))
	if err != nil {
		t.Fatal(err)
	}
	tail = tail.Radiating(xray(0.5))
	sameRadiant(t, "the tail's radiant", tail.Radiant(), xray(0.5))
	if tail.Length() != 900 || tail.Base() != 30 || tail.Spread() != 0.3 ||
		tail.Curve() != 0.7 || tail.Dense() != 0.02 {
		t.Errorf("the tail reads %v %v %v %v %v, want 900 30 0.3 0.7 "+
			"0.02", tail.Length(), tail.Base(), tail.Spread(),
			tail.Curve(), tail.Dense())
	}
	clouds, err := bodies.NewClouds(shell, maths.Share(0.4),
		bodies.PerKilometre(0.05), 13)
	if err != nil {
		t.Fatal(err)
	}
	if clouds.Low() != 2 || clouds.High() != 11 || clouds.Cover() != 0.4 ||
		clouds.Dense() != 0.05 || clouds.Seed() != 13 {
		t.Errorf("the clouds read %v %v %v %v %v, want 2 11 0.4 0.05 13",
			clouds.Low(), clouds.High(), clouds.Cover(), clouds.Dense(),
			clouds.Seed())
	}
}

// The aurora, the cities and the jets give back what they were made
// with, and the colours give back their three.
func TestAuroraCitiesAndJetsReadBackTheirArguments(t *testing.T) {
	auroraShell := bodies.NewShell(km(1), km(5))
	auroraColours := bodies.NewAuroraColours(lit(1, 2, 3),
		lit(4, 5, 6), lit(7, 8, 9))
	aurora, err := bodies.NewAurora(auroraShell,
		rad(0.5), rad(0.25), rad(0.125), maths.Share(0.6),
		maths.Share(0.8), 21, auroraColours)
	if err != nil {
		t.Fatal(err)
	}
	aurora = aurora.Radiating(xray(0.25))
	sameRadiant(t, "the aurora's radiant", aurora.Radiant(), xray(0.25))
	colours := aurora.Colours()
	if colours.Fringe() != lit(1, 2, 3) || colours.Bottom() != lit(4, 5, 6) ||
		colours.Top() != lit(7, 8, 9) {
		t.Errorf("the colours read %v %v %v, want the three in order",
			colours.Fringe(), colours.Bottom(), colours.Top())
	}
	if aurora.Low() != 1 || aurora.High() != 5 || aurora.Lat() != 0.5 ||
		aurora.Width() != 0.25 || aurora.Tilt() != 0.125 ||
		aurora.Strength() != 0.6 || aurora.Curtains() != 0.8 ||
		aurora.Seed() != 21 {
		t.Errorf("the aurora reads wrong: %v %v %v %v %v %v %v %v",
			aurora.Low(), aurora.High(), aurora.Lat(), aurora.Width(),
			aurora.Tilt(), aurora.Strength(), aurora.Curtains(),
			aurora.Seed())
	}
	cities, err := bodies.NewCities(3, maths.Share(0.2), maths.Share(0.1),
		lit(0.3, 0.2, 0.1), 11)
	if err != nil {
		t.Fatal(err)
	}
	if cities.Rungs() != 3 || cities.Share() != 0.2 || cities.Metro() != 0.1 ||
		cities.Glow() != lit(0.3, 0.2, 0.1) || cities.Seed() != 11 {
		t.Errorf("the cities read %v %v %v %v %v, want 3 0.2 0.1 and "+
			"the glow and 11", cities.Rungs(), cities.Share(),
			cities.Metro(), cities.Glow(), cities.Seed())
	}
	jets, err := bodies.NewJets(km(5000), km(40), 0.25, km(70),
		bodies.PerKilometre(0.06), rad(0.1), maths.Share(0.3),
		lit(0.9, 0.8, 0.7))
	if err != nil {
		t.Fatal(err)
	}
	jets = jets.Knotted(0.45).Radiating(xray(0.75))
	sameRadiant(t, "the jets' radiant", jets.Radiant(), xray(0.75))
	if jets.Length() != 5000 || jets.Base() != 40 || jets.Spread() != 0.25 ||
		jets.From() != 70 || jets.Dense() != 0.06 || jets.Tilt() != 0.1 ||
		jets.Speed() != 0.3 || jets.Knots() != 0.45 ||
		jets.Glow() != lit(0.9, 0.8, 0.7) {
		t.Errorf("the jets read wrong: %v %v %v %v %v %v %v %v %v",
			jets.Length(), jets.Base(), jets.Spread(), jets.From(),
			jets.Dense(), jets.Tilt(), jets.Speed(), jets.Knots(),
			jets.Glow())
	}
}

// The flare, the plume, the fineness and the gas give back their
// arguments.
func TestFlarePlumeFinenessAndGasReadBackTheirArguments(t *testing.T) {
	flare, err := bodies.NewFlare(90*time.Second, 12.5, 2.5,
		lit(0.5, 0.25, 0.125))
	if err != nil {
		t.Fatal(err)
	}
	flare = flare.Radiating(xray(0.5))
	sameRadiant(t, "the flare's radiant", flare.Radiant(), xray(0.5))
	if flare.At() != 90*time.Second || flare.Speed() != 12.5 ||
		flare.Opacity() != 2.5 || flare.Glow() != lit(0.5, 0.25, 0.125) {
		t.Errorf("the flare reads %v %v %v %v, want 90s 12.5 2.5 and the "+
			"glow", flare.At(), flare.Speed(), flare.Opacity(), flare.Glow())
	}
	plume, err := bodies.NewPlume(maths.V(0, 1, 0), km(300), km(500),
		bodies.PerKilometre(0.04), lit(0.6, 0.5, 0.4))
	if err != nil {
		t.Fatal(err)
	}
	if plume.Toward() != maths.V(0, 1, 0) || plume.Height() != 300 ||
		plume.Width() != 500 || plume.Dense() != 0.04 ||
		plume.Colour() != lit(0.6, 0.5, 0.4) {
		t.Errorf("the plume reads wrong: %v %v %v %v %v", plume.Toward(),
			plume.Height(), plume.Width(), plume.Dense(), plume.Colour())
	}
	fine, err := bodies.NewFineness(4, maths.Share(0.25), maths.Share(0.5))
	if err != nil {
		t.Fatal(err)
	}
	if fine.Rungs() != 4 || fine.Vary() != 0.25 || fine.Mix() != 0.5 {
		t.Errorf("the fineness reads %v %v %v, want 4 0.25 0.5",
			fine.Rungs(), fine.Vary(), fine.Mix())
	}
	gas, err := bodies.NewGas(maths.Share(0.3), 8,
		texture.Banded{Paints: []maths.Light{lit(1, 0, 0), lit(0, 1, 0)}})
	if err != nil {
		t.Fatal(err)
	}
	if gas.Depth() != 0.3 || gas.Opacity() != 8 {
		t.Errorf("the gas reads %v %v, want 0.3 and 8", gas.Depth(),
			gas.Opacity())
	}
}

// The air, the rings, the horizon, the disc and the moon give back their
// arguments.
func TestAirRingsHorizonDiscAndMoonReadBackTheirArguments(t *testing.T) {
	air, err := bodies.NewAir(km(7), bodies.PerKilometre(0.125),
		lit(0.1, 0.4, 0.9))
	if err != nil {
		t.Fatal(err)
	}
	air = air.Radiating(xray(0.125))
	sameRadiant(t, "the air's radiant", air.Radiant(), xray(0.125))
	if air.Height() != 7 || air.Dense() != 0.125 ||
		air.Colour() != lit(0.1, 0.4, 0.9) {
		t.Errorf("the air reads %v %v %v, want 7 0.125 and its colour",
			air.Height(), air.Dense(), air.Colour())
	}
	rings, err := bodies.NewRings(1.25, 2.5, km(0.5),
		bodies.PerKilometre(0.5), lit(0.8, 0.7, 0.6))
	if err != nil {
		t.Fatal(err)
	}
	rings = rings.Profiled([]float64{0.5, 1}).Grained(0.125, 40)
	if rings.Inner() != 1.25 || rings.Outer() != 2.5 || rings.Thick() != 0.5 ||
		rings.Dense() != 0.5 || rings.Grain() != 0.125 ||
		rings.Ringlets() != 40 || rings.Colour() != lit(0.8, 0.7, 0.6) ||
		!reflect.DeepEqual(rings.Profile(), []float64{0.5, 1}) {
		t.Errorf("the rings read wrong: %v %v %v %v %v %v %v %v",
			rings.Inner(), rings.Outer(), rings.Thick(), rings.Dense(),
			rings.Grain(), rings.Ringlets(), rings.Colour(),
			rings.Profile())
	}
	edge, err := bodies.NewHorizon(0.75, lit(0.2, 0.3, 0.4))
	if err != nil {
		t.Fatal(err)
	}
	if edge.Hexes() != 0.75 || edge.Light() != lit(0.2, 0.3, 0.4) {
		t.Errorf("the horizon reads %v %v, want 0.75 and its light",
			edge.Hexes(), edge.Light())
	}
	disc, err := bodies.NewDisc(km(30), km(300), km(2),
		bodies.PerKilometre(0.5), measure.Kelvin(6000), 3)
	if err != nil {
		t.Fatal(err)
	}
	disc, err = disc.Beamed(maths.Share(0.2))
	if err != nil {
		t.Fatal(err)
	}
	disc = disc.Dressed(0.5, 0.25)
	if disc.Inner() != 30 || disc.Outer() != 300 || disc.Thick() != 2 ||
		disc.Dense() != 0.5 || disc.Hot() != 6000 || disc.Bright() != 3 ||
		disc.Speed() != 0.2 || disc.Streaks() != 0.5 || disc.Shear() != 0.25 {
		t.Errorf("the disc reads wrong: %v %v %v %v %v %v %v %v %v",
			disc.Inner(), disc.Outer(), disc.Thick(), disc.Dense(),
			disc.Hot(), disc.Bright(), disc.Speed(), disc.Streaks(),
			disc.Shear())
	}
	moon, err := bodies.NewMoon(heavy(t), km(500), rad(0.5))
	if err != nil {
		t.Fatal(err)
	}
	if moon.Phase() != 0.5 || moon.Orbit() != 500 || moon.Body().Name() != "h" {
		t.Errorf("the moon reads %v %v %q, want 0.5 500 and h",
			moon.Phase(), moon.Orbit(), moon.Body().Name())
	}
}

// A ground gives back its wrapped pictures, its turn, its night glow,
// its finer paint, its radar look and its radiant.
func TestAGroundReadsBackWhatItWears(t *testing.T) {
	day := image.NewGray(image.Rect(0, 0, 2, 1))
	lights := image.NewGray(image.Rect(0, 0, 4, 2))
	rough := look()
	rough.Lines = 0.5
	fine, err := bodies.NewFineness(2, maths.Share(0.5), maths.Share(0.25))
	if err != nil {
		t.Fatal(err)
	}
	g := ground(t)
	if _, ok := g.Fine(); ok {
		t.Error("a plain ground reports a finer paint")
	}
	if _, ok := g.Rough(); ok {
		t.Error("a plain ground reports a radar look")
	}
	g = g.Mapped(day, rad(0.75)).Fined(fine).RadarRough(rough).
		Radiating(xray(0.5))
	g, err = g.NightLit(lights, maths.Share(0.125))
	if err != nil {
		t.Fatal(err)
	}
	if g.Map() != image.Image(day) || g.Lights() != image.Image(lights) {
		t.Error("the ground does not give back the pictures it wears")
	}
	if g.Turn() != 0.75 || g.NightGlow() != 0.125 {
		t.Errorf("the ground reads turn %v glow %v, want 0.75 and 0.125",
			g.Turn(), g.NightGlow())
	}
	if got, ok := g.Fine(); !ok || got.Rungs() != 2 || got.Vary() != 0.5 {
		t.Errorf("the finer paint is %v %v, want 2 rungs varying 0.5", got, ok)
	}
	if got, ok := g.Rough(); !ok || got.Lines != 0.5 {
		t.Errorf("the radar look has lines %v %v, want 0.5", got.Lines, ok)
	}
	sameRadiant(t, "the ground's radiant", g.Radiant(), xray(0.5))
	if !reflect.DeepEqual(g.Terrain(), terrain()) ||
		!reflect.DeepEqual(g.Look(), look()) {
		t.Error("the ground does not give back its terrain and look")
	}
	if g.Features() != nil {
		t.Error("a ground never featured reports features")
	}
}

// A body wearing every part gives each back from its own reader, and the
// body's own numbers from theirs.
func TestABodyGivesBackEachPartItWears(t *testing.T) {
	gas, err := bodies.NewGas(maths.Share(0.3), 8, texture.Banded{})
	if err != nil {
		t.Fatal(err)
	}
	air, err := bodies.NewAir(km(7), nothing, lit(0.1, 0.4, 0.9))
	if err != nil {
		t.Fatal(err)
	}
	cloudShell := bodies.NewShell(km(2), km(11))
	clouds, err := bodies.NewClouds(cloudShell,
		maths.Share(0.4), nothing, 13)
	if err != nil {
		t.Fatal(err)
	}
	cities, err := bodies.NewCities(3, maths.Share(0.2), maths.Share(0.1),
		lit(0.3, 0.2, 0.1), 11)
	if err != nil {
		t.Fatal(err)
	}
	auroraShell := bodies.NewShell(km(1), km(5))
	aurora, err := bodies.NewAurora(auroraShell,
		rad(0.5), rad(0.25), rad(0.125), maths.Share(0.6),
		maths.Share(0.8), 21, bodies.AuroraColours{})
	if err != nil {
		t.Fatal(err)
	}
	rings, err := bodies.NewRings(1.25, 2.5, km(0.5), nothing, lit(1, 1, 1))
	if err != nil {
		t.Fatal(err)
	}
	disc, err := bodies.NewDisc(km(30), km(300), km(2), nothing,
		measure.Kelvin(6000), 3)
	if err != nil {
		t.Fatal(err)
	}
	jets, err := bodies.NewJets(km(5000), km(40), 0.25, km(70), nothing,
		rad(0.1), maths.Share(0.3), maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	flare, err := bodies.NewFlare(time.Second, 12.5, 2.5, maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	coma, err := bodies.NewComa(km(120), nothing, maths.Light{}, maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	edge, err := bodies.NewHorizon(0.75, lit(0.2, 0.3, 0.4))
	if err != nil {
		t.Fatal(err)
	}
	body := made(t, "all", bodies.Star, 700, 2e30, ground(t), gas, air,
		clouds, cities, aurora, rings, disc, jets, flare, coma, edge)
	body = body.Spinning(0.5).Seeded(77).Glowing(lit(2, 3, 4), xray(0.25))
	if body.Spin() != 0.5 || body.Seed() != 77 || body.Glow() != lit(2, 3, 4) {
		t.Errorf("the body reads spin %v seed %v glow %v, want 0.5 77 "+
			"and its glow", body.Spin(), body.Seed(), body.Glow())
	}
	sameRadiant(t, "the body's radiant", body.Radiant(), xray(0.25))
	if got, ok := body.Ground(); !ok || !reflect.DeepEqual(got.Terrain(), terrain()) {
		t.Error("the body does not give back its ground")
	}
	if got, ok := body.Gas(); !ok || got.Depth() != 0.3 {
		t.Errorf("the body's gas is %v %v, want depth 0.3", got.Depth(), ok)
	}
	if got, ok := body.Air(); !ok || got.Height() != 7 {
		t.Errorf("the body's air is %v %v, want height 7", got.Height(), ok)
	}
	if got, ok := body.Clouds(); !ok || got.Seed() != 13 {
		t.Errorf("the body's clouds are %v %v, want seed 13", got.Seed(), ok)
	}
	if got, ok := body.Cities(); !ok || got.Seed() != 11 {
		t.Errorf("the body's cities are %v %v, want seed 11", got.Seed(), ok)
	}
	if got, ok := body.Aurora(); !ok || got.Seed() != 21 {
		t.Errorf("the body's aurora is %v %v, want seed 21", got.Seed(), ok)
	}
	if got, ok := body.Rings(); !ok || got.Inner() != 1.25 {
		t.Errorf("the body's rings are %v %v, want inner 1.25", got.Inner(), ok)
	}
	if got, ok := body.Disc(); !ok || got.Inner() != 30 {
		t.Errorf("the body's disc is %v %v, want inner 30", got.Inner(), ok)
	}
	if got, ok := body.Jets(); !ok || got.Length() != 5000 {
		t.Errorf("the body's jets are %v %v, want length 5000", got.Length(), ok)
	}
	if got, ok := body.Flare(); !ok || got.Speed() != 12.5 {
		t.Errorf("the body's flare is %v %v, want speed 12.5", got.Speed(), ok)
	}
	if got, ok := body.Coma(); !ok || got.Radius() != 120 {
		t.Errorf("the body's coma is %v %v, want radius 120", got.Radius(), ok)
	}
	if body.Edge().Hexes() != 0.75 {
		t.Errorf("the body's edge has hexes %v, want 0.75", body.Edge().Hexes())
	}

	// each Without verb takes off its own part and no other
	bare := body.WithoutAurora().WithoutClouds().WithoutCities().
		WithoutJets().WithoutFlare()
	if _, ok := bare.Aurora(); ok {
		t.Error("WithoutAurora left the aurora")
	}
	if _, ok := bare.Clouds(); ok {
		t.Error("WithoutClouds left the clouds")
	}
	if _, ok := bare.Cities(); ok {
		t.Error("WithoutCities left the cities")
	}
	if _, ok := bare.Jets(); ok {
		t.Error("WithoutJets left the jets")
	}
	if _, ok := bare.Flare(); ok {
		t.Error("WithoutFlare left the flare")
	}
	if _, ok := bare.Gas(); !ok {
		t.Error("the Without verbs took off the gas as well")
	}
	if _, ok := body.Aurora(); !ok {
		t.Error("a Without verb changed the body it was called on")
	}
}

// A part given a bad argument is refused with a message naming what was
// wrong.
func TestPartConstructorsSayWhatIsWrong(t *testing.T) {
	bad := bodies.PerKilometre(-1)
	shell := bodies.NewShell(km(1), km(2))
	colours := bodies.AuroraColours{}
	cases := []struct {
		name string
		err  func() error
		want string
	}{
		{"coma loss", func() error {
			_, e := bodies.NewComa(km(1), bad, maths.Light{}, maths.Light{})
			return e
		}, "a coma's"},
		{"tail loss", func() error {
			_, e := bodies.NewTail(km(1), km(1), 0, 0, bad, maths.Light{},
				maths.Light{})
			return e
		}, "a tail's"},
		{"clouds cover", func() error {
			_, e := bodies.NewClouds(shell, maths.Share(2), nothing, 1)
			return e
		}, "clouds' cover"},
		{"clouds loss", func() error {
			_, e := bodies.NewClouds(shell, maths.Share(0.5), bad, 1)
			return e
		}, "clouds'"},
		{"aurora strength", func() error {
			_, e := bodies.NewAurora(shell, rad(0), rad(0), rad(0),
				maths.Share(2), maths.Share(0.5), 1, colours)
			return e
		}, "strength"},
		{"aurora curtains", func() error {
			_, e := bodies.NewAurora(shell, rad(0), rad(0), rad(0),
				maths.Share(0.5), maths.Share(2), 1, colours)
			return e
		}, "curtains"},
		{"cities towns", func() error {
			_, e := bodies.NewCities(1, maths.Share(2), maths.Share(0.5),
				maths.Light{}, 1)
			return e
		}, "towns"},
		{"cities metros", func() error {
			_, e := bodies.NewCities(1, maths.Share(0.5), maths.Share(2),
				maths.Light{}, 1)
			return e
		}, "metros"},
		{"jets loss", func() error {
			_, e := bodies.NewJets(km(1), km(1), 0, km(0), bad,
				rad(0), maths.Share(0.5), maths.Light{})
			return e
		}, "the jets'"},
		{"jets speed", func() error {
			_, e := bodies.NewJets(km(1), km(1), 0, km(0), nothing,
				rad(0), maths.Share(2), maths.Light{})
			return e
		}, "speed"},
		{"flare speed", func() error {
			_, e := bodies.NewFlare(0, 0, 1, maths.Light{})
			return e
		}, "never shows"},
		{"plume loss", func() error {
			_, e := bodies.NewPlume(maths.V(0, 1, 0), km(1), km(1), bad,
				maths.Light{})
			return e
		}, "a plume's"},
		{"fineness vary", func() error {
			_, e := bodies.NewFineness(1, maths.Share(2), maths.Share(0.5))
			return e
		}, "fineness vary"},
		{"fineness mix", func() error {
			_, e := bodies.NewFineness(1, maths.Share(0.5), maths.Share(2))
			return e
		}, "fineness mix"},
		{"night lights glow", func() error {
			dark := image.NewGray(image.Rect(0, 0, 1, 1))
			_, e := ground(t).NightLit(dark,
				maths.Share(2))
			return e
		}, "night lights"},
		{"gas depth", func() error {
			_, e := bodies.NewGas(maths.Share(2), 1, texture.Banded{})
			return e
		}, "the gas's depth"},
		{"air height", func() error {
			_, e := bodies.NewAir(km(0), nothing, maths.Light{})
			return e
		}, "no atmosphere"},
		{"rings edges", func() error {
			_, e := bodies.NewRings(2, 1, km(1), nothing, maths.Light{})
			return e
		}, "no disc"},
		{"rings loss", func() error {
			_, e := bodies.NewRings(1, 2, km(1), bad, maths.Light{})
			return e
		}, "the rings'"},
		{"horizon hexes", func() error {
			_, e := bodies.NewHorizon(-1, maths.Light{})
			return e
		}, "no dressing"},
		{"disc loss", func() error {
			_, e := bodies.NewDisc(km(1), km(2), km(1), bad,
				measure.Kelvin(1), 1)
			return e
		}, "the disc's"},
		{"disc beamed", func() error {
			disc, err := bodies.NewDisc(km(1), km(2), km(1), nothing,
				measure.Kelvin(1), 1)
			if err != nil {
				t.Fatal(err)
			}
			_, e := disc.Beamed(maths.Share(2))
			return e
		}, "the disc's speed"},
		{"tail wisp", func() error {
			tail, err := bodies.NewTail(km(1), km(1), 0, 0, nothing,
				maths.Light{}, maths.Light{})
			if err != nil {
				t.Fatal(err)
			}
			_, e := tail.Wisped(maths.Share(2), 1)
			return e
		}, "a tail's wisp"},
		{"moon of Body{}", func() error {
			_, e := bodies.NewMoon(bodies.Body{}, km(1), rad(0))
			return e
		}, "needs a body made"},
		{"moon orbit", func() error {
			_, e := bodies.NewMoon(heavy(t), km(-1), rad(0))
			return e
		}, "goes nowhere"},
		{"moon holding a complaint", func() error {
			notANumber := rad(nan)
			_, e := bodies.NewMoon(heavy(t).Turning(time.Hour, notANumber),
				km(1), rad(0))
			return e
		}, "as a moon"},
	}
	for _, c := range cases {
		err := c.err()
		if err == nil {
			t.Errorf("%s: the constructor passed", c.name)
			continue
		}
		if !strings.HasPrefix(err.Error(), "bodies: ") ||
			!strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: the error is %q, want it to start with "+
				"\"bodies: \" and contain %q", c.name, err, c.want)
		}
	}
}

// A moon whose orbit is far past what a duration can time is refused
// when its body is placed.
func TestAMoonOnAnOrbitTooWideToTimeIsRefusedInPlace(t *testing.T) {
	moon, err := bodies.NewMoon(heavy(t), km(1e12), rad(0))
	if err != nil {
		t.Fatal(err)
	}
	host := made(t, "host", bodies.Rock, 1, 1, moon)
	err = host.Place(thing.NewWorld(), maths.Vec{}, 1, 2)
	if err == nil || !strings.Contains(err.Error(), "years") {
		t.Errorf("Place of a moon on a 1e12 km orbit round 1 kg gave %v, "+
			"want a refusal naming the years", err)
	}
}

// plainRadiant is a radiant that is no spectrum, answering one share to every
// band.
type plainRadiant struct{}

// Through is the share of a band that passes.
func (plainRadiant) Through(maths.Band) float64 { return 1 }

// Emits is the share of a band given off.
func (plainRadiant) Emits(maths.Band) float64 { return 0 }

// Scatters is the share of a band sent back.
func (plainRadiant) Scatters(maths.Band) float64 { return 0 }

// A radiant given as a pointer to a spectrum is checked as the spectrum
// is, a nil pointer is refused, and a radiant of another kind is taken.
func TestARadiantGivenByPointerOrOfAnotherKindIsChecked(t *testing.T) {
	wear := func(radiant thing.Radiant) error {
		_, err := bodies.NewBody("r", bodies.Rock, km(1), measure.Kilograms(1),
			func() bodies.Air {
				air, err := bodies.NewAir(km(1), nothing, maths.Light{})
				if err != nil {
					t.Fatal(err)
				}
				return air.Radiating(radiant)
			}())
		return err
	}
	good := xray(0.5)
	if err := wear(&good); err != nil {
		t.Errorf("a pointer to a good spectrum was refused: %v", err)
	}
	if err := wear(plainRadiant{}); err != nil {
		t.Errorf("a radiant that is no spectrum was refused: %v", err)
	}
	badSpectrum := xray(nan)
	if err := wear(&badSpectrum); err == nil ||
		!strings.Contains(err.Error(), "gives") {
		t.Errorf("a pointer to a spectrum giving NaN gave %v, want a "+
			"refusal naming gives", err)
	}
	var none *thing.Spectrum
	if err := wear(none); err == nil ||
		!strings.Contains(err.Error(), "nil *thing.Spectrum") {
		t.Errorf("a nil spectrum pointer gave %v, want a refusal naming "+
			"it", err)
	}
	if err := wear(nil); err == nil || !strings.Contains(err.Error(), "is nil") {
		t.Errorf("a nil radiant gave %v, want a refusal saying nil", err)
	}
}

// NewBody refuses a missing name, a kind that is none, and a radius or
// a mass that is zero, negative, endless or no number.
func TestNewBodyRefusesACoreThatIsNoBody(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		kind   bodies.Kind
		radius float64
		mass   float64
		want   string
	}{
		{"no name", "", bodies.Rock, 1, 1, "needs a name"},
		{"kind below the first", "k", bodies.Rock - 1, 1, 1, "no kind"},
		{"kind past the last", "k", bodies.Star + 1, 1, 1, "no kind"},
		{"zero radius", "r", bodies.Rock, 0, 1, "radius"},
		{"endless radius", "r", bodies.Rock, inf, 1, "radius"},
		{"no-number radius", "r", bodies.Rock, nan, 1, "radius"},
		{"zero mass", "m", bodies.Rock, 1, 0, "mass"},
		{"endless mass", "m", bodies.Rock, 1, inf, "mass"},
		{"negative mass", "m", bodies.Rock, 1, -1, "mass"},
	}
	for _, c := range cases {
		_, err := bodies.NewBody(c.body, c.kind, km(c.radius),
			measure.Kilograms(c.mass))
		if err == nil || !strings.Contains(err.Error(), c.want) ||
			!strings.HasPrefix(err.Error(), "bodies: ") {
			t.Errorf("%s: NewBody gave %v, want a refusal containing %q",
				c.name, err, c.want)
		}
	}
}

// A body that wears no part says so from every part reader.
func TestABareBodyReportsNoParts(t *testing.T) {
	body := heavy(t)
	if _, ok := body.Ground(); ok {
		t.Error("a bare body reports a ground")
	}
	if _, ok := body.Gas(); ok {
		t.Error("a bare body reports a gas")
	}
	if _, ok := body.Air(); ok {
		t.Error("a bare body reports an air")
	}
	if _, ok := body.Rings(); ok {
		t.Error("a bare body reports rings")
	}
	if _, ok := body.Disc(); ok {
		t.Error("a bare body reports a disc")
	}
	if _, ok := body.Coma(); ok {
		t.Error("a bare body reports a coma")
	}
	if _, ok := body.Clouds(); ok {
		t.Error("a bare body reports clouds")
	}
	if _, ok := body.Jets(); ok {
		t.Error("a bare body reports jets")
	}
}
