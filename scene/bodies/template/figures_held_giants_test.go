package template

import (
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/world"
)

// jupiterFigures is the figures held for jupiter, which the hot
// jupiter starts from. the gas's eleven bands are held by how many,
// not by colour.
func jupiterFigures() bodies.Body {
	return bodies.Body{Name: "jupiter", Kind: bodies.Giant,
		Radius: 69911, Mass: 1.898e27, Day: 35748 * time.Second,
		Tilt: 3.13 * radPerDeg,
		Gas: &bodies.Gas{Depth: 0.08, Opacity: 40, Bands: texture.Banded{
			Paints: make([]maths.Light, 11),
			Roil:   0.06, Shear: 7, Grain: 0.08, Fine: 0.05, Jets: 10, Mottle: 0.5,
			Spots: []texture.Spot{{Lat: -0.38, Lon: -1.3, Size: 0.16,
				Stretch: 1.7, Turn: 2.8}},
			Streets: []texture.Street{{Lat: -0.55, Count: 7, Size: 0.045,
				Turn: 1.4, Seed: 14}},
			Seed: 14}},
		Aurora: &bodies.Aurora{Low: 300, High: 1500, Lat: 74 * radPerDeg,
			Width: 2 * radPerDeg, Tilt: 10 * radPerDeg, Strength: 0.02,
			Curtains: 0.5, Seed: 14},
		Moons: []bodies.Moon{
			{Body: bodies.Body{Name: "io", Kind: bodies.Rock, Radius: 1821.6, Mass: 8.93e22,
				Ground: &bodies.Ground{
					Terrain:  world.Terrain{Seed: 7, Ocean: 0, Plates: 9},
					Look:     world.Look{Lines: 0.2},
					Features: anyFeatures},
				Plumes: []bodies.Plume{{Toward: maths.V(-0.24, -0.32, -0.92),
					Height: 300, Width: 1100, Dense: 0.006}}},
				Orbit: 421700},
			{Body: bodies.Body{Name: "europa", Kind: bodies.Rock, Radius: 1560.8, Mass: 4.8e22,
				Ground: &bodies.Ground{
					Terrain: world.Terrain{Seed: 8, Ocean: 0, Plates: 4},
					Look:    world.Look{Lines: 0.1}}},
				Orbit: 671034, Phase: 1.2},
			{Body: bodies.Body{Name: "ganymede", Kind: bodies.Rock, Radius: 2634.1, Mass: 1.48e23,
				Ground: &bodies.Ground{
					Terrain: world.Terrain{Seed: 9, Ocean: 0.4, Plates: 8},
					Look:    world.Look{Lines: 0.2}}},
				Orbit: 1070412, Phase: 2.5},
			{Body: bodies.Body{Name: "callisto", Kind: bodies.Rock, Radius: 2410.3, Mass: 1.08e23,
				Ground: &bodies.Ground{
					Terrain: world.Terrain{Seed: 10, Ocean: 0, Plates: 6},
					Look:    world.Look{Lines: 0.2}}},
				Orbit: 1882709, Phase: 4},
		}}
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// jupiter: 69,911 km, 1.898e27 kg, 9.93 h (35,748 s), 3.13 degrees of
// tilt; a gas 0.08 deep of 40 optical depth, ten jets, eleven bands, a
// storm and a street of seven ovals; an aurora at 74 degrees.
//
// io, europa, ganymede and callisto on 421,700, 671,034, 1,070,412
// and 1,882,709 km, started 0, 1.2, 2.5 and 4 radians round.
func TestJupiterFiguresAreHeld(t *testing.T) {
	holds(t, "jupiter", Jupiter(), jupiterFigures())
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// the hot jupiter: jupiter with no moons and no aurora, a day of 18 h
// (64,800 s), and a tail 30 radii long, 2 wide at the base, bending
// 0.25, 0.4 per radius dense, ragged 0.6.
func TestHotJupiterFiguresAreHeld(t *testing.T) {
	want := jupiterFigures()
	want.Name, want.Moons, want.Aurora = "a hot jupiter", nil, nil
	want.Day = 64800 * time.Second
	const radius = 69911
	want.Tails = []bodies.Tail{{Length: 30 * radius, Base: 2 * radius,
		Spread: 0.06, Curve: 0.25, Dense: 0.4 / radius, Wisp: 0.6, Seed: 21}}
	holds(t, "a hot jupiter", HotJupiter(), want)
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// saturn: 58,232 km, 5.683e26 kg, 10.7 h (38,520 s), 26.73 degrees of
// tilt; nine bands, eight jets; an aurora at 75 degrees on the spin;
// rings from 1.24 to 2.27 radii in twelve samples.
//
// titan, rhea and enceladus on 1,221,870, 527,108 and 237,948 km,
// started 0, 2 and 4 radians round.
func TestSaturnFiguresAreHeld(t *testing.T) {
	holds(t, "saturn", Saturn(), bodies.Body{Name: "saturn", Kind: bodies.Giant,
		Radius: 58232, Mass: 5.683e26, Day: 38520 * time.Second,
		Tilt: 26.73 * radPerDeg,
		Gas: &bodies.Gas{Depth: 0.08, Opacity: 40, Bands: texture.Banded{
			Paints: make([]maths.Light, 9),
			Roil:   0.04, Shear: 7, Grain: 0.08, Fine: 0.03, Jets: 8, Mottle: 0.35,
			Seed: 15}},
		Aurora: &bodies.Aurora{Low: 800, High: 2000, Lat: 75 * radPerDeg,
			Width: 2 * radPerDeg, Strength: 0.02, Curtains: 0.5, Seed: 15},
		Rings: &bodies.Rings{Inner: 1.24, Outer: 2.27, Thick: 1, Dense: 1.2,
			Profile: []float64{0.2, 0.25, 0.3, 0.5, 0.95, 1, 1, 0.95, 0.08,
				0.7, 0.65, 0.5},
			Grain: 0.6, Ringlets: 60},
		Moons: []bodies.Moon{
			{Body: bodies.Body{Name: "titan", Kind: bodies.Rock, Radius: 2574.7, Mass: 1.345e23,
				Ground: &bodies.Ground{
					Terrain: world.Terrain{Seed: 11, Ocean: 0, Plates: 5},
					Look:    world.Look{Lines: 0}},
				Air: &bodies.Air{Height: 40, Dense: 0.02}},
				Orbit: 1221870},
			{Body: bodies.Body{Name: "rhea", Kind: bodies.Rock, Radius: 763.8, Mass: 2.31e21},
				Orbit: 527108, Phase: 2},
			{Body: bodies.Body{Name: "enceladus", Kind: bodies.Rock, Radius: 252.1, Mass: 1.08e20},
				Orbit: 237948, Phase: 4}}})
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// uranus: 25,362 km, 8.681e25 kg, 17.24 h (62,064 s), 97.77 degrees of
// tilt; five bands, six jets; rings from 1.65 to 2.02 radii in nine
// samples; titania, oberon and miranda on 435,910, 583,520 and 129,390
// km, started 0, 2 and 1 radians round.
func TestUranusFiguresAreHeld(t *testing.T) {
	holds(t, "uranus", Uranus(), bodies.Body{Name: "uranus", Kind: bodies.Giant,
		Radius: 25362, Mass: 8.681e25, Day: 62064 * time.Second,
		Tilt: 97.77 * radPerDeg,
		Gas: &bodies.Gas{Depth: 0.08, Opacity: 40, Bands: texture.Banded{
			Paints: make([]maths.Light, 5),
			Roil:   0.02, Shear: 7, Grain: 0.08, Fine: 0.01, Jets: 6, Mottle: 0.15,
			Seed: 16}},
		Rings: &bodies.Rings{Inner: 1.65, Outer: 2.02, Thick: 0.5, Dense: 0.5,
			Profile: []float64{0.2, 0.05, 0.4, 0.05, 0.3, 0.05, 1, 0.05, 0.5},
			Grain:   0.4, Ringlets: 40},
		Moons: []bodies.Moon{
			{Body: bodies.Body{Name: "titania", Kind: bodies.Rock, Radius: 788.4, Mass: 3.4e21},
				Orbit: 435910},
			{Body: bodies.Body{Name: "oberon", Kind: bodies.Rock, Radius: 761.4, Mass: 3.08e21},
				Orbit: 583520, Phase: 2},
			{Body: bodies.Body{Name: "miranda", Kind: bodies.Rock, Radius: 235.8, Mass: 6.6e19},
				Orbit: 129390, Phase: 1}}})
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// neptune: 24,622 km, 1.024e26 kg, 16.11 h (57,996 s), 28.32 degrees
// of tilt; seven bands, eight jets, a dark storm and a street of four
// clouds; and triton on 354,759 km.
func TestNeptuneFiguresAreHeld(t *testing.T) {
	holds(t, "neptune", Neptune(), bodies.Body{Name: "neptune", Kind: bodies.Giant,
		Radius: 24622, Mass: 1.024e26, Day: 57996 * time.Second,
		Tilt: 28.32 * radPerDeg,
		Gas: &bodies.Gas{Depth: 0.08, Opacity: 40, Bands: texture.Banded{
			Paints: make([]maths.Light, 7),
			Roil:   0.05, Shear: 7, Grain: 0.08, Fine: 0.03, Jets: 8, Mottle: 0.3,
			Spots: []texture.Spot{{Lat: -0.35, Lon: -1.2, Size: 0.12,
				Stretch: 1.5, Turn: 2}},
			Streets: []texture.Street{{Lat: -0.5, Count: 4, Size: 0.05,
				Turn: 0.5, Seed: 17}},
			Seed: 17}},
		Moons: []bodies.Moon{{Body: bodies.Body{Name: "triton", Kind: bodies.Rock,
			Radius: 1353.4, Mass: 2.14e22}, Orbit: 354759}}})
}
