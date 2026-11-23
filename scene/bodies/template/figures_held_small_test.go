package template

import (
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/thing"
)

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// hale-bopp: 30 km, 1e16 kg, a coma of radius 1e6 km and 6e-6 per km,
// and two tails: dust 4e7 km long and ion 6e7 km long.
func TestHaleBoppFiguresAreHeld(t *testing.T) {
	holds(t, "hale-bopp", its(t, HaleBopp),
		want(t, "hale-bopp", bodies.Rock, 30, 1e16,
			wantComa(t, 1e6, 6e-6),
			wantTail(t, 4e7, 6e5, 0.08, 0.35, 2e-7, 0.5, 97),
			wantTail(t, 6e7, 3e5, 0.03, 0.02, 2e-7, 0.7, 98)).
			Seeded(3))
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// hyakutake: 2 km, 1e13 kg, a coma of 5e5 km and 1e-5 per km, and two
// tails: dust 8e6 km long and ion 5e8 km.
func TestHyakutakeFiguresAreHeld(t *testing.T) {
	holds(t, "hyakutake", its(t, Hyakutake),
		want(t, "hyakutake", bodies.Rock, 2, 1e13,
			wantComa(t, 5e5, 1e-5),
			wantTail(t, 8e6, 4e5, 0.06, 0.3, 1.5e-7, 0.5, 96),
			wantTail(t, 5e8, 6e5, 0.01, 0.01, 3e-7, 0.7, 95)).
			Seeded(5))
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// the magnetar: 12 km, 1.4 solar masses, 7.54 s a turn; x-ray at 1 and
// gamma at 0.2; a flare at one second, 30,000 km/s, opacity 0.8,
// giving gamma 2 and x-ray 1 and passing gamma at 0.9.
func TestMagnetarFiguresAreHeld(t *testing.T) {
	flare, err := bodies.NewFlare(time.Second, 30000, 0.8, maths.Light{})
	if err != nil {
		t.Fatal(err)
	}
	flare = flare.Radiating(thing.Spectrum{
		Gives: map[radiation.Range]float64{radiation.Gamma(): 2,
			radiation.XRay(): 1},
		Passes: map[radiation.Range]float64{radiation.Gamma(): 0.9}})
	got := its(t, Magnetar)
	figure := want(t, "magnetar", bodies.Star, 12, 1.4*bodies.Sun, flare).
		Turning(7540*time.Millisecond, measure.Radians(0)).
		Glowing(maths.Light{}, thing.Spectrum{
			Gives: map[radiation.Range]float64{
				radiation.XRay():  1,
				radiation.Gamma(): 0.2}})
	holds(t, "magnetar", got, figure)
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// the pulsar: 10 km, 1.4 solar masses, 33 ms a turn, x-ray at 1; beams
// 1500 km long, 30 km at the base, 45 degrees off the spin, giving
// x-ray 0.8 and radio 1 and passing both at 0.97.
func TestPulsarFiguresAreHeld(t *testing.T) {
	beams := wantJets(t, 1500, 30, 0.12, 0, 0.004, 45*radPerDeg, 0).
		Radiating(thing.Spectrum{
			Gives: map[radiation.Range]float64{
				radiation.XRay(): 0.8, radiation.Radio(): 1},
			Passes: map[radiation.Range]float64{
				radiation.XRay(): 0.97, radiation.Radio(): 0.97}})
	got := its(t, Pulsar)
	figure := want(t, "pulsar", bodies.Star, 10, 1.4*bodies.Sun, beams).
		Seeded(8).
		Turning(33*time.Millisecond, measure.Radians(0)).
		Glowing(maths.Light{}, thing.Spectrum{
			Gives: map[radiation.Range]float64{
				radiation.XRay(): 1}})
	holds(t, "pulsar", got, figure)
}

// physics, worked by hand: a hole's radius is 2GM/c^2. with G =
// 6.6743e-11, c = 299,792,458 m/s and one sun of 1.989e30 kg that is
// 2.954 km, and the quasar's 6.6e10 suns make it 6.6e10 times that:
// 1.9497e11 km.
//
// held to a tenth of a per cent, since 2.954 is rounded.
func TestQuasarsRadiusIsTwoGMOverCSquaredOfSixPointSixTenBillionSuns(t *testing.T) {
	const perSun = 2.954
	q := its(t, Quasar)
	if want := 6.6e10 * bodies.Sun; q.Mass() != want {
		t.Errorf("mass is %v, want %v", q.Mass(), want)
	}
	if want := 6.6e10 * perSun; !(q.Radius() > want*0.999 && q.Radius() < want*1.001) {
		t.Errorf("radius is %v km, want %v within a tenth of a per cent", q.Radius(), want)
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
	jets := wantJets(t, 300*rs, rs, 0.03, 2*rs, 2/rs, 0, 0.6).
		Radiating(thing.Spectrum{
			Gives: map[radiation.Range]float64{
				radiation.XRay(): 1, radiation.Gamma(): 0.5},
			Passes: map[radiation.Range]float64{
				radiation.XRay(): 0.97, radiation.Gamma(): 0.98}})
	holds(t, "quasar", its(t, Quasar),
		want(t, "quasar", bodies.Hole, rs, mass,
			wantDisc(t, 3*rs, 30*rs, 0.3*rs, 10/rs, 12000, 60,
				0.41, 0, 1.5),
			jets).Seeded(7))
}
