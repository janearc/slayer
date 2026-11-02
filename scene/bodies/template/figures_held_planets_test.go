package template

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/scene/world"
)

// the template's figures are held here by literal, a body to a test.
// each literal is typed from the template, not from a source. a figure
// in hours is written in seconds, and one in degrees in degrees times
// radPerDeg, so that nothing here goes through hours or deg.
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
	holds(t, "sol", Sol(), bodies.Body{Name: "sol", Kind: bodies.Star,
		Radius: 696000, Mass: bodies.Sun, Day: 2192400 * time.Second})
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// mercury: 2439.7 km, 3.301e23 kg, 1407.6 h (5,067,360 s), 0.03
// degrees of tilt.
func TestMercuryFiguresAreHeld(t *testing.T) {
	holds(t, "mercury", Mercury(), bodies.Body{Name: "mercury", Kind: bodies.Rock,
		Radius: 2439.7, Mass: 3.301e23, Day: 5067360 * time.Second,
		Tilt: 0.03 * radPerDeg,
		Ground: &bodies.Ground{
			Terrain: world.Terrain{Seed: 1, Ocean: 0, Plates: 6, Cold: 0.3},
			Look:    world.Look{Lines: 0.3}}})
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// venus: 6051.8 km, 4.867e24 kg, 5832.5 h (20,997,000 s), 177.4
// degrees of tilt; an air 15.9 km high of 0.6 per km that passes radio
// in full, and a ground that returns it at 0.9.
func TestVenusFiguresAreHeld(t *testing.T) {
	holds(t, "venus", Venus(), bodies.Body{Name: "venus", Kind: bodies.Rock,
		Radius: 6051.8, Mass: 4.867e24, Day: 20997000 * time.Second,
		Tilt: 177.4 * radPerDeg,
		Ground: &bodies.Ground{
			Terrain: world.Terrain{Seed: 2, Ocean: 0, Plates: 8},
			Look:    world.Look{Lines: 0}, Features: anyFeatures,
			Radiant: thing.Spectrum{Returns: map[radiation.Range]float64{radiation.Radio(): 0.9}},
			Rough:   &world.Look{}},
		Air: &bodies.Air{Height: 15.9, Dense: 0.6,
			Radiant: thing.Spectrum{Passes: map[radiation.Range]float64{radiation.Radio(): 1}}}})
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
	got := Venus().Ground.Rough
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
	holds(t, "earth", Earth(), bodies.Body{Name: "earth", Kind: bodies.Rock,
		Radius: 6371, Mass: 5.972e24, Day: 86148 * time.Second,
		Tilt: 23.44 * radPerDeg,
		Ground: &bodies.Ground{
			Terrain: world.Terrain{Seed: 5, Ocean: 0.71, Plates: 12},
			Look:    world.Look{Lines: 0.4}},
		Air:    &bodies.Air{Height: 8.5, Dense: 0.012},
		Clouds: &bodies.Clouds{Low: 2, High: 9, Cover: 0.55, Dense: 1, Seed: 5},
		Cities: &bodies.Cities{Rungs: 3, Share: 0.12, Metro: 0.05, Seed: 5},
		Aurora: &bodies.Aurora{Low: 100, High: 300, Lat: 67 * radPerDeg,
			Width: 4 * radPerDeg, Tilt: 11 * radPerDeg, Strength: 0.006,
			Curtains: 0.7, Seed: 5},
		Moons: []bodies.Moon{{Body: bodies.Body{Name: "moon", Kind: bodies.Rock,
			Radius: 1737.4, Mass: 7.342e22,
			Ground: &bodies.Ground{
				Terrain: world.Terrain{Seed: 6, Ocean: 0.3, Plates: 5, Cold: 0.2},
				Look:    world.Look{Lines: 0.2}}}, Orbit: 384400}}})
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// mars: 3389.5 km, 6.417e23 kg, 24.62 h (88,632 s), 25.19 degrees of
// tilt; an air 11.1 km high of 0.003 per km; phobos of 11.1 km and
// 1.07e16 kg on 9376 km, and deimos of 6.2 km and 1.5e15 kg on 23,463
// km, starting 2 radians round.
func TestMarsFiguresAreHeld(t *testing.T) {
	holds(t, "mars", Mars(), bodies.Body{Name: "mars", Kind: bodies.Rock,
		Radius: 3389.5, Mass: 6.417e23, Day: 88632 * time.Second,
		Tilt: 25.19 * radPerDeg,
		Ground: &bodies.Ground{
			Terrain:  world.Terrain{Seed: 4, Ocean: 0, Plates: 7, Cold: 0.25},
			Look:     world.Look{Lines: 0.3},
			Features: anyFeatures},
		Air: &bodies.Air{Height: 11.1, Dense: 0.003},
		Moons: []bodies.Moon{
			{Body: bodies.Body{Name: "phobos", Kind: bodies.Rock, Radius: 11.1, Mass: 1.07e16},
				Orbit: 9376},
			{Body: bodies.Body{Name: "deimos", Kind: bodies.Rock, Radius: 6.2, Mass: 1.5e15},
				Orbit: 23463, Phase: 2}}})
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// pluto: 1188.3 km, 1.303e22 kg, 153.3 h (551,880 s), 122.5 degrees
// of tilt.
func TestPlutoFiguresAreHeld(t *testing.T) {
	holds(t, "pluto", Pluto(), bodies.Body{Name: "pluto", Kind: bodies.Rock,
		Radius: 1188.3, Mass: 1.303e22, Day: 551880 * time.Second,
		Tilt: 122.5 * radPerDeg,
		Ground: &bodies.Ground{
			Terrain:  world.Terrain{Seed: 18, Ocean: 0, Plates: 6, Cold: 0.4},
			Look:     world.Look{Lines: 0.2},
			Features: anyFeatures}})
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// charon: 606 km, 1.586e21 kg, 153.3 h (551,880 s), no tilt.
func TestCharonFiguresAreHeld(t *testing.T) {
	holds(t, "charon", Charon(), bodies.Body{Name: "charon", Kind: bodies.Rock,
		Radius: 606, Mass: 1.586e21, Day: 551880 * time.Second,
		Ground: &bodies.Ground{
			Terrain: world.Terrain{Seed: 19, Ocean: 0, Plates: 5, Cold: 0.3},
			Look:    world.Look{Lines: 0.2}}})
}
