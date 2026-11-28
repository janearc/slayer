package template

import (
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/world"
)

// jupiterBands is the held shape of jupiter's banding: eleven bands
// held by how many, not by colour, the storm and the street of ovals.
func jupiterBands() texture.Banded {
	return texture.Banded{
		Paints: make([]maths.Light, 11),
		Roil:   0.06, Shear: 7, Grain: 0.08, Fine: 0.05, Jets: 10,
		Mottle: 0.5,
		Spots: []texture.Spot{{Lat: -0.38, Lon: -1.3, Size: 0.16,
			Stretch: 1.7, Turn: 2.8}},
		Streets: []texture.Street{{Lat: -0.55, Count: 7, Size: 0.045,
			Turn: 1.4, Seed: 14}},
		Seed: 14}
}

// jupiterFigures is the figures held for jupiter, which the hot
// jupiter starts from.
func jupiterFigures(t *testing.T) bodies.Body {
	t.Helper()
	io := wantGround(t, world.Terrain{Seed: 7, Ocean: 0, Plates: 9},
		world.Look{Lines: 0.2}).Featured(anyFeatures)
	pele := wantPlume(t, maths.V(-0.24, -0.32, -0.92), 300, 1100, 0.006)
	europa := wantGround(t, world.Terrain{Seed: 8, Ocean: 0, Plates: 4},
		world.Look{Lines: 0.1})
	ganymede := wantGround(t, world.Terrain{Seed: 9, Ocean: 0.4, Plates: 8},
		world.Look{Lines: 0.2})
	callisto := wantGround(t, world.Terrain{Seed: 10, Ocean: 0, Plates: 6},
		world.Look{Lines: 0.2})
	gasPart := wantGas(t, 0.08, 40, jupiterBands())
	auroraPart := wantAurora(t, 300, 1500, 74*radPerDeg, 2*radPerDeg,
		10*radPerDeg, 0.02, 0.5, 14)
	ioPart := wantMoon(t, want(t, "io", bodies.Rock, 1821.6, 8.93e22, io,
		pele), 421700, 0)
	europaPart := wantMoon(t, want(t, "europa", bodies.Rock, 1560.8, 4.8e22,
		europa), 671034, 1.2)
	ganymedePart := wantMoon(t, want(t, "ganymede", bodies.Rock, 2634.1, 1.48e23,
		ganymede), 1070412, 2.5)
	callistoPart := wantMoon(t, want(t, "callisto", bodies.Rock, 2410.3, 1.08e23,
		callisto), 1882709, 4)
	return want(t, "jupiter", bodies.Giant, 69911, 1.898e27,
		gasPart, auroraPart, ioPart, europaPart, ganymedePart,
		callistoPart).
		Turning(35748*time.Second, measure.Radians(3.13*radPerDeg))
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
	holds(t, "jupiter", its(t, Jupiter), jupiterFigures(t))
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// the hot jupiter: jupiter with no moons and no aurora, a day of 18 h
// (64,800 s), and a tail 30 radii long, 2 wide at the base, bending
// 0.25, 0.4 per radius dense, ragged 0.6.
func TestHotJupiterFiguresAreHeld(t *testing.T) {
	// the body's radius, from scene/generate (hotJupiterRadiusKm), and
	// the unit its tail is said in
	const radius = 111962
	got := its(t, HotJupiter)
	gasPart := wantGas(t, 0.08, 40, jupiterBands())
	tailPart := wantTail(t, 30*radius, 2*radius, 0.06, 0.25,
		0.4/radius, 0.6, 21)
	figure := want(t, "a hot jupiter", bodies.Giant, radius, 1.898e27,
		gasPart, tailPart).
		Turning(64800*time.Second,
			measure.Radians(3.13*radPerDeg))
	holds(t, "a hot jupiter", got, figure)
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
	titan := want(t, "titan", bodies.Rock, 2574.7, 1.345e23,
		wantGround(t, world.Terrain{Seed: 11, Ocean: 0, Plates: 5},
			world.Look{Lines: 0}),
		wantAir(t, 40, 0.02))
	got := its(t, Saturn)
	gasPart := wantGas(t, 0.08, 40, texture.Banded{
		Paints: make([]maths.Light, 9),
		Roil:   0.04, Shear: 7, Grain: 0.08, Fine: 0.03,
		Jets: 8, Mottle: 0.35, Seed: 15})
	auroraPart := wantAurora(t, 800, 2000, 75*radPerDeg, 2*radPerDeg,
		0, 0.02, 0.5, 15)
	ringPart := wantRings(t, 1.24, 2.27, 1, 1.2,
		[]float64{0.2, 0.25, 0.3, 0.5, 0.95, 1, 1,
			0.95, 0.08, 0.7, 0.65, 0.5}, 0.6, 60)
	titanPart := wantMoon(t, titan, 1221870, 0)
	rheaPart := wantMoon(t, want(t, "rhea", bodies.Rock, 763.8,
		2.31e21), 527108, 2)
	enceladusPart := wantMoon(t, want(t, "enceladus", bodies.Rock, 252.1,
		1.08e20), 237948, 4)
	figure := want(t, "saturn", bodies.Giant, 58232, 5.683e26,
		gasPart, auroraPart, ringPart, titanPart, rheaPart,
		enceladusPart).
		Turning(38520*time.Second,
			measure.Radians(26.73*radPerDeg))
	holds(t, "saturn", got, figure)
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// uranus: 25,362 km, 8.681e25 kg, 17.24 h (62,064 s), 97.77 degrees of
// tilt; five bands, six jets; rings from 1.65 to 2.02 radii in nine
// samples; titania, oberon and miranda on 435,910, 583,520 and 129,390
// km, started 0, 2 and 1 radians round.
func TestUranusFiguresAreHeld(t *testing.T) {
	got := its(t, Uranus)
	gasPart := wantGas(t, 0.08, 40, texture.Banded{
		Paints: make([]maths.Light, 5),
		Roil:   0.02, Shear: 7, Grain: 0.08, Fine: 0.01,
		Jets: 6, Mottle: 0.15, Seed: 16})
	ringPart := wantRings(t, 1.65, 2.02, 0.5, 0.5,
		[]float64{0.2, 0.05, 0.4, 0.05, 0.3, 0.05, 1,
			0.05, 0.5}, 0.4, 40)
	titaniaPart := wantMoon(t, want(t, "titania", bodies.Rock, 788.4,
		3.4e21), 435910, 0)
	oberonPart := wantMoon(t, want(t, "oberon", bodies.Rock, 761.4,
		3.08e21), 583520, 2)
	mirandaPart := wantMoon(t, want(t, "miranda", bodies.Rock, 235.8,
		6.6e19), 129390, 1)
	figure := want(t, "uranus", bodies.Giant, 25362, 8.681e25,
		gasPart, ringPart, titaniaPart, oberonPart, mirandaPart).
		Turning(62064*time.Second,
			measure.Radians(97.77*radPerDeg))
	holds(t, "uranus", got, figure)
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// neptune: 24,622 km, 1.024e26 kg, 16.11 h (57,996 s), 28.32 degrees
// of tilt; seven bands, eight jets, a dark storm and a street of four
// clouds; and triton on 354,759 km.
func TestNeptuneFiguresAreHeld(t *testing.T) {
	got := its(t, Neptune)
	gasPart := wantGas(t, 0.08, 40, texture.Banded{
		Paints: make([]maths.Light, 7),
		Roil:   0.05, Shear: 7, Grain: 0.08, Fine: 0.03,
		Jets: 8, Mottle: 0.3,
		Spots: []texture.Spot{{Lat: -0.35, Lon: -1.2,
			Size: 0.12, Stretch: 1.5, Turn: 2}},
		Streets: []texture.Street{{Lat: -0.5, Count: 4,
			Size: 0.05, Turn: 0.5, Seed: 17}},
		Seed: 17})
	tritonPart := wantMoon(t, want(t, "triton", bodies.Rock, 1353.4,
		2.14e22), 354759, 0)
	figure := want(t, "neptune", bodies.Giant, 24622, 1.024e26,
		gasPart, tritonPart).
		Turning(57996*time.Second,
			measure.Radians(28.32*radPerDeg))
	holds(t, "neptune", got, figure)
}
