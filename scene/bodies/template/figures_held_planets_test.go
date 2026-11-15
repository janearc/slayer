package template

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/scene/world"
)

// the template's figures are held here, a body to a test, built through
// the doors with the numbers spelled at the call site. each figure is
// typed from the template, not from a source. a figure in hours is
// written in seconds, and one in degrees in degrees times radPerDeg, so
// that nothing here goes through hours or deg.
const radPerDeg = math.Pi / 180

// anyFeatures stands for a ground's Features where the figures are held:
// the test holds that one is there, and the features test holds what it
// does.
func anyFeatures(*world.World) ([]world.Feature, error) { return nil, nil }

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// the sun: a star of 696,000 km, one solar mass, turning in 609 h
// (2,192,400 s).
func TestSolFiguresAreHeld(t *testing.T) {
	holds(t, "sol", its(t, Sol),
		want(t, "sol", bodies.Star, 696000, bodies.Sun).
			Turning(2192400*time.Second, measure.Radians(0)))
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// mercury: 2439.7 km, 3.301e23 kg, 1407.6 h (5,067,360 s), 0.03
// degrees of tilt.
func TestMercuryFiguresAreHeld(t *testing.T) {
	holds(t, "mercury", its(t, Mercury),
		want(t, "mercury", bodies.Rock, 2439.7, 3.301e23,
			wantGround(t, world.Terrain{Seed: 1, Ocean: 0,
				Plates: 6, Cold: 0.3},
				world.Look{Lines: 0.3})).
			Turning(5067360*time.Second,
				measure.Radians(0.03*radPerDeg)))
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// venus: 6051.8 km, 4.867e24 kg, 5832.5 h (20,997,000 s), 177.4
// degrees of tilt; an air 15.9 km high of 0.6 per km that passes radio
// in full, and a ground that returns it at 0.9.
func TestVenusFiguresAreHeld(t *testing.T) {
	ground := wantGround(t, world.Terrain{Seed: 2, Ocean: 0, Plates: 8},
		world.Look{Lines: 0}).
		Featured(anyFeatures).
		Radiating(thing.Spectrum{Returns: map[radiation.Range]float64{
			radiation.Radio(): 0.9}}).
		RadarRough(world.Look{})
	air := wantAir(t, 15.9, 0.6).
		Radiating(thing.Spectrum{Passes: map[radiation.Range]float64{
			radiation.Radio(): 1}})
	holds(t, "venus", its(t, Venus),
		want(t, "venus", bodies.Rock, 6051.8, 4.867e24, ground, air).
			Turning(20997000*time.Second,
				measure.Radians(177.4*radPerDeg)))
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// venus's radar roughness, for each class from deep to ash: 0.1, 0.15,
// 0.5, 0.18, 0.15, 0.22, 0.3, 0.35, 0.4, 0.95, 0.8, 0.7 and 0.6, each
// a grey of that value.
func TestVenusRoughnessIsHeldClassByClass(t *testing.T) {
	want := [world.Biomes]float64{0.1, 0.15, 0.5, 0.18, 0.15, 0.22, 0.3,
		0.35, 0.4, 0.95, 0.8, 0.7, 0.6}
	got, worn := groundOf(t, its(t, Venus)).Rough()
	if !worn {
		t.Fatal("venus's ground wears no radar look")
	}
	for b, r := range want {
		if got.Paints[b] != maths.Grey(r) {
			t.Errorf("class %d is %v, want a grey of %v", b, got.Paints[b], r)
		}
	}
	if got.Lines != 0 {
		t.Errorf("the radar look has lines of %v", got.Lines)
	}
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// the earth: 6371 km, 5.972e24 kg, 23.93 h (86,148 s), 23.44 degrees
// of tilt, 71 per cent sea; an air 8.5 km high of 0.012 per km; clouds
// from 2 to 9 km, 55 per cent cover.
//
// cities three rungs down, 12 and 5 per cent; an aurora from 100 to
// 300 km at 67 degrees, 4 wide, 11 off the spin; and the moon on its
// 384,400 km.
func TestEarthFiguresAreHeld(t *testing.T) {
	clouds, err := bodies.NewClouds(
		bodies.NewShell(bodies.Kilometres(2), bodies.Kilometres(9)),
		maths.Share(0.55), bodies.PerKilometre(1), 5)
	if err != nil {
		t.Fatal(err)
	}
	cities, err := bodies.NewCities(3, maths.Share(0.12),
		maths.Share(0.05), maths.Light{}, 5)
	if err != nil {
		t.Fatal(err)
	}
	moon := want(t, "moon", bodies.Rock, 1737.4, 7.342e22,
		wantGround(t, world.Terrain{Seed: 6, Ocean: 0.3, Plates: 5,
			Cold: 0.2}, world.Look{Lines: 0.2}))
	holds(t, "earth", its(t, Earth),
		want(t, "earth", bodies.Rock, 6371, 5.972e24,
			wantGround(t, world.Terrain{Seed: 5, Ocean: 0.71,
				Plates: 12}, world.Look{Lines: 0.4}),
			wantAir(t, 8.5, 0.012),
			clouds,
			cities,
			wantAurora(t, 100, 300, 67*radPerDeg, 4*radPerDeg,
				11*radPerDeg, 0.006, 0.7, 5),
			wantMoon(t, moon, 384400, 0)).
			Turning(86148*time.Second,
				measure.Radians(23.44*radPerDeg)))
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// mars: 3389.5 km, 6.417e23 kg, 24.62 h (88,632 s), 25.19 degrees of
// tilt; an air 11.1 km high of 0.003 per km; phobos of 11.1 km and
// 1.07e16 kg on 9376 km, and deimos of 6.2 km and 1.5e15 kg on 23,463
// km, starting 2 radians round.
func TestMarsFiguresAreHeld(t *testing.T) {
	holds(t, "mars", its(t, Mars),
		want(t, "mars", bodies.Rock, 3389.5, 6.417e23,
			wantGround(t, world.Terrain{Seed: 4, Ocean: 0,
				Plates: 7, Cold: 0.25},
				world.Look{Lines: 0.3}).Featured(anyFeatures),
			wantAir(t, 11.1, 0.003),
			wantMoon(t, want(t, "phobos", bodies.Rock, 11.1,
				1.07e16), 9376, 0),
			wantMoon(t, want(t, "deimos", bodies.Rock, 6.2,
				1.5e15), 23463, 2)).
			Turning(88632*time.Second,
				measure.Radians(25.19*radPerDeg)))
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// pluto: 1188.3 km, 1.303e22 kg, 153.3 h (551,880 s), 122.5 degrees
// of tilt.
func TestPlutoFiguresAreHeld(t *testing.T) {
	holds(t, "pluto", its(t, Pluto),
		want(t, "pluto", bodies.Rock, 1188.3, 1.303e22,
			wantGround(t, world.Terrain{Seed: 18, Ocean: 0,
				Plates: 6, Cold: 0.4},
				world.Look{Lines: 0.2}).Featured(anyFeatures)).
			Turning(551880*time.Second,
				measure.Radians(122.5*radPerDeg)))
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// charon: 606 km, 1.586e21 kg, 153.3 h (551,880 s), no tilt.
func TestCharonFiguresAreHeld(t *testing.T) {
	holds(t, "charon", its(t, Charon),
		want(t, "charon", bodies.Rock, 606, 1.586e21,
			wantGround(t, world.Terrain{Seed: 19, Ocean: 0,
				Plates: 5, Cold: 0.3},
				world.Look{Lines: 0.2})).
			Turning(551880*time.Second, measure.Radians(0)))
}
