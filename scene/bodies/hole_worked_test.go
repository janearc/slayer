package bodies_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// sunMass is 1.98847e30 kg, the mass at which G = 6.6743e-11 gives the
// iau's GM of 1.3271244e20 m^3/s^2 for the sun, whose schwarzschild
// radius r_s = 2GM/c^2 is then 2.95325 km.
const (
	sunMass = 1.98847e30
	sunRs   = 2.95325
)

// a hole is a field of its mass as a length, half its schwarzschild
// radius, reaching twenty radii, and spinning at its spin share of that
// length.
//
// at half a unit a kilometre the sun's r_s of 2.95325 km is 1.476625
// units, its mass as a length 0.7383125 (m), and its reach 29.5325. a
// spin of 0.6 is a = 0.6 m.
func TestAHoleIsAFieldOfItsMassAsALength(t *testing.T) {
	b := bodies.Body{Name: "h", Kind: bodies.Hole, Mass: sunMass, Spin: 0.6}
	w := thing.NewWorld()
	at := maths.V(10, 0, 0)
	if err := b.Place(w, at, 0.5, 2); err != nil {
		t.Fatal(err)
	}
	if len(w.Fields) != 1 {
		t.Fatalf("a hole has %d fields", len(w.Fields))
	}
	f := w.Fields[0]
	m := sunRs * 0.5 / 2
	near(t, "the hole's mass", f.Mass, m, 1e-4)
	near(t, "the hole's spin", f.Spin, 0.6*m, 1e-4)
	near(t, "the hole's reach", f.Reach, 20*sunRs*0.5, 1e-4)
	nearVec(t, "where the field is", f.At(0).Origin, at, 1e-12)
	if len(w.Things) != 0 || len(w.Volumes) != 0 {
		t.Errorf("a hole with no edge and no disc has %d things and %d volumes",
			len(w.Things), len(w.Volumes))
	}
}

// a hole that does not spin is a schwarzschild one, with no spin in its
// field.
func TestAHoleThatDoesNotSpinHasNoSpin(t *testing.T) {
	b := bodies.Body{Name: "h", Kind: bodies.Hole, Mass: sunMass}
	w := thing.NewWorld()
	if err := b.Place(w, maths.Vec{}, 0.5, 2); err != nil {
		t.Fatal(err)
	}
	if s := w.Fields[0].Spin; s != 0 {
		t.Errorf("the field spins at %v", s)
	}
}

// the ball the hexes are drawn on is a fiftieth outside the horizon at
// its widest. the kerr-schild horizon is r+ = m + sqrt(m^2 - a^2), at
// the poles; at the equator it reaches sqrt(r+^2 + a^2).
//
// with a = 0.6 m, sqrt(m^2 - a^2) is 0.8 m, r+ is 1.8 m, and the
// equator reaches m sqrt(3.24 + 0.36) = m sqrt(3.6). with a = 0 it is
// 2 m, the schwarzschild radius, all round. the ball is 1.02 times that.
func TestTheHexesAreDrawnOnABallJustOutsideTheHorizon(t *testing.T) {
	m := sunRs * 0.5 / 2
	for name, c := range map[string]struct{ spin, ball float64 }{
		"schwarzschild": {0, 1.02 * 2 * m},
		"kerr, a = 0.6": {0.6, 1.02 * math.Sqrt(3.6) * m},
		"kerr, a = 0.8": {0.8, 1.02 * math.Sqrt(2.56+0.64) * m},
	} {
		b := bodies.Body{Name: "h", Kind: bodies.Hole, Mass: sunMass, Spin: c.spin,
			Edge: bodies.Horizon{Hexes: 0.5}}
		w := thing.NewWorld()
		at := maths.V(10, 0, 0)
		if err := b.Place(w, at, 0.5, 2); err != nil {
			t.Fatal(err)
		}
		if len(w.Things) != 1 {
			t.Fatalf("%s: a hole with an edge has %d things", name, len(w.Things))
		}
		ball := w.Things[0]
		near(t, name+": the ball's radius", ball.Shape.(shape.Sphere).Radius, c.ball, 1e-4)
		nearVec(t, name+": where the ball is", ball.At(0).Origin, at, 1e-12)
	}
}

// the hexes are a wire on a sphere of six splits, in the colour given
// at the brightness given: a light of (0.25, 0.5, 1) at a half is
// (0.125, 0.25, 0.5), and a hexes of white when no light is given, at a
// half, is a grey of a half. a hole whose edge has no hexes has no ball.
func TestTheHexesAreAWireInTheColourAtTheBrightness(t *testing.T) {
	for name, c := range map[string]struct {
		light, want maths.Light
	}{
		"given":  {maths.Light{R: 0.25, G: 0.5, B: 1}, maths.Light{R: 0.125, G: 0.25, B: 0.5}},
		"unset":  {maths.Light{}, maths.Grey(0.5)},
		"bright": {maths.Grey(1), maths.Grey(0.5)},
	} {
		b := bodies.Body{Name: "h", Kind: bodies.Hole, Mass: sunMass,
			Edge: bodies.Horizon{Hexes: 0.5, Light: c.light}}
		w := thing.NewWorld()
		if err := b.Place(w, maths.Vec{}, 0.5, 2); err != nil {
			t.Fatal(err)
		}
		ball := solidNamed(t, w, "h horizon")
		same(t, name+": the hexes", ball.Material.Shine,
			texture.Wire{Tiling: hexes.Sphere{N: 6}, Light: c.want})
	}
	b := bodies.Body{Name: "h", Kind: bodies.Hole, Mass: sunMass}
	w := thing.NewWorld()
	if err := b.Place(w, maths.Vec{}, 0.5, 2); err != nil {
		t.Fatal(err)
	}
	if len(w.Things) != 0 {
		t.Errorf("a hole with no hexes has %d things", len(w.Things))
	}
}

// a hole's disc is cut to the scale: 8 to 24 km is 4 to 12 units, 2 km
// thick is 1, and 0.25 a kilometre is 0.5 a unit.
//
// its colour is the black body's, from the temperature and brightness
// given, the seed the body's, and the streaks wound at the orbital rate
// at one unit out, which is 2 km: omega = 2 pi / T, T = 2 pi sqrt(r^3/GM).
func TestAHolesDiscIsCutToTheScale(t *testing.T) {
	b := bodies.Body{Name: "h", Kind: bodies.Hole, Mass: sunMass, Seed: 3,
		Disc: &bodies.Disc{Inner: 8, Outer: 24, Thick: 2, Dense: 0.25, Hot: 6000,
			Bright: 4, Streaks: 0.5, Shear: 2}}
	w := thing.NewWorld()
	at := maths.V(10, 0, 0)
	if err := b.Place(w, at, 0.5, 2); err != nil {
		t.Fatal(err)
	}
	disc := volumeNamed(t, w, "h disc")
	same(t, "the disc", disc.Medium, medium.Disc{Inner: 4, Outer: 12, Thick: 1, Dense: 0.5})
	thin := disc.Material.Shine.(texture.Thin)
	near(t, "the disc's omega", thin.Omega, omega(sunMass, 2), 1e-12)
	thin.Omega = 0
	same(t, "the disc's colour", thin, texture.Thin{Inner: 4, Outer: 12, Hot: 6000,
		Bright: 4, Streaks: 0.5, Shear: 2, Seed: 3})
	nearVec(t, "where the disc is", disc.At(0).Origin, at, 1e-12)
	if disc.Beam != nil {
		t.Errorf("a disc of no speed is beamed: %+v", disc.Beam)
	}
}

// a disc with a speed is a flow round its axis at that share of light's
// speed at its inner edge, 4 units, prograde.
func TestADiscWithASpeedIsABeamRoundTheAxis(t *testing.T) {
	b := bodies.Body{Name: "h", Kind: bodies.Hole, Mass: sunMass,
		Disc: &bodies.Disc{Inner: 8, Outer: 24, Thick: 2, Dense: 0.25, Speed: 0.25}}
	w := thing.NewWorld()
	if err := b.Place(w, maths.Vec{}, 0.5, 2); err != nil {
		t.Fatal(err)
	}
	beam := volumeNamed(t, w, "h disc").Beam
	if beam == nil {
		t.Fatal("a disc with a speed has no beam")
	}
	same(t, "the beam", *beam, thing.Beam{Axis: maths.Up, Speed: 0.25, Round: true, At: 4})
	// prograde, from +x toward -z
	dir, speed := beam.Flow(maths.V(4, 0, 0))
	nearVec(t, "the flow at +x", dir, maths.V(0, 0, -1), 1e-12)
	near(t, "the flow's speed at the inner edge", speed, 0.25, 1e-12)
}
