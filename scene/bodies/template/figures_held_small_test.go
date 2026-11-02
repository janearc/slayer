package template

import (
	"testing"
	"time"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/thing"
)

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// hale-bopp: 30 km, 1e16 kg, a coma of radius 1e6 km and 6e-6 per km,
// and two tails: dust 4e7 km long and ion 6e7 km long.
func TestHaleBoppFiguresAreHeld(t *testing.T) {
	holds(t, "hale-bopp", HaleBopp(), bodies.Body{Name: "hale-bopp", Kind: bodies.Rock,
		Radius: 30, Mass: 1e16, Seed: 3,
		Coma: &bodies.Coma{Radius: 1e6, Dense: 6e-6},
		Tails: []bodies.Tail{
			{Length: 4e7, Base: 6e5, Spread: 0.08, Curve: 0.35, Dense: 2e-7,
				Wisp: 0.5, Seed: 97},
			{Length: 6e7, Base: 3e5, Spread: 0.03, Curve: 0.02, Dense: 2e-7,
				Wisp: 0.7, Seed: 98}}})
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// hyakutake: 2 km, 1e13 kg, a coma of 5e5 km and 1e-5 per km, and two
// tails: dust 8e6 km long and ion 5e8 km.
func TestHyakutakeFiguresAreHeld(t *testing.T) {
	holds(t, "hyakutake", Hyakutake(), bodies.Body{Name: "hyakutake", Kind: bodies.Rock,
		Radius: 2, Mass: 1e13, Seed: 5,
		Coma: &bodies.Coma{Radius: 5e5, Dense: 1e-5},
		Tails: []bodies.Tail{
			{Length: 8e6, Base: 4e5, Spread: 0.06, Curve: 0.3, Dense: 1.5e-7,
				Wisp: 0.5, Seed: 96},
			{Length: 5e8, Base: 6e5, Spread: 0.01, Curve: 0.01, Dense: 3e-7,
				Wisp: 0.7, Seed: 95}}})
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// the magnetar: 12 km, 1.4 solar masses, 7.54 s a turn; x-ray at 1 and
// gamma at 0.2; a flare at one second, 30,000 km/s, opacity 0.8,
// giving gamma 2 and x-ray 1 and passing gamma at 0.9.
func TestMagnetarFiguresAreHeld(t *testing.T) {
	holds(t, "magnetar", Magnetar(), bodies.Body{Name: "magnetar", Kind: bodies.Star,
		Radius: 12, Mass: 1.4 * bodies.Sun, Day: 7540 * time.Millisecond,
		Radiant: thing.Spectrum{Gives: map[radiation.Range]float64{
			radiation.XRay(): 1, radiation.Gamma(): 0.2}},
		Flare: &bodies.Flare{At: time.Second, Speed: 30000, Opacity: 0.8,
			Radiant: thing.Spectrum{
				Gives:  map[radiation.Range]float64{radiation.Gamma(): 2, radiation.XRay(): 1},
				Passes: map[radiation.Range]float64{radiation.Gamma(): 0.9}}}})
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// the pulsar: 10 km, 1.4 solar masses, 33 ms a turn, x-ray at 1; beams
// 1500 km long, 30 km at the base, 45 degrees off the spin, giving
// x-ray 0.8 and radio 1 and passing both at 0.97.
func TestPulsarFiguresAreHeld(t *testing.T) {
	holds(t, "pulsar", Pulsar(), bodies.Body{Name: "pulsar", Kind: bodies.Star,
		Radius: 10, Mass: 1.4 * bodies.Sun, Seed: 8, Day: 33 * time.Millisecond,
		Radiant: thing.Spectrum{Gives: map[radiation.Range]float64{radiation.XRay(): 1}},
		Jets: &bodies.Jets{Length: 1500, Base: 30, Spread: 0.12, Dense: 0.004,
			Tilt: 45 * radPerDeg,
			Radiant: thing.Spectrum{
				Gives:  map[radiation.Range]float64{radiation.XRay(): 0.8, radiation.Radio(): 1},
				Passes: map[radiation.Range]float64{radiation.XRay(): 0.97, radiation.Radio(): 0.97}}}})
}

// physics, worked by hand: a hole's radius is 2GM/c^2. with G =
// 6.6743e-11, c = 299,792,458 m/s and one sun of 1.989e30 kg that is
// 2.954 km, and the quasar's 6.6e10 suns make it 6.6e10 times that:
// 1.9497e11 km.
//
// held to a tenth of a per cent, since 2.954 is rounded.
func TestQuasarsRadiusIsTwoGMOverCSquaredOfSixPointSixTenBillionSuns(t *testing.T) {
	const perSun = 2.954
	q := Quasar()
	if want := 6.6e10 * bodies.Sun; q.Mass != want {
		t.Errorf("mass is %v, want %v", q.Mass, want)
	}
	if want := 6.6e10 * perSun; !(q.Radius > want*0.999 && q.Radius < want*1.001) {
		t.Errorf("radius is %v km, want %v within a tenth of a per cent", q.Radius, want)
	}
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// the quasar: ton 618's 6.6e10 solar masses and its schwarzschild
// radius, rs; a disc from 3 rs to 30 rs, 0.3 rs thick, 10 per rs
// dense, 12,000 K, bright 60, at 0.41 of the speed of light, sheared
// 1.5.
//
// jets from 2 rs out to 300 rs, 1 rs at the base, spread 0.03, 2 per
// rs dense, at 0.6 of the speed of light, giving x-ray 1 and gamma 0.5
// and passing them at 0.97 and 0.98.
func TestQuasarFiguresAreHeld(t *testing.T) {
	const g, c = 6.6743e-11, 299792458.0
	mass := 6.6e10 * bodies.Sun
	rs := 2 * g * mass / (c * c) / 1000
	holds(t, "quasar", Quasar(), bodies.Body{Name: "quasar", Kind: bodies.Hole,
		Radius: rs, Mass: mass, Seed: 7,
		Disc: &bodies.Disc{Inner: 3 * rs, Outer: 30 * rs, Thick: 0.3 * rs,
			Dense: 10 / rs, Hot: 12000, Bright: 60, Speed: 0.41, Shear: 1.5},
		Jets: &bodies.Jets{Length: 300 * rs, Base: rs, Spread: 0.03, From: 2 * rs,
			Dense: 2 / rs, Speed: 0.6,
			Radiant: thing.Spectrum{
				Gives:  map[radiation.Range]float64{radiation.XRay(): 1, radiation.Gamma(): 0.5},
				Passes: map[radiation.Range]float64{radiation.XRay(): 0.97, radiation.Gamma(): 0.98}}}})
}
