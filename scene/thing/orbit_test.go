package thing

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// Kepler: a moon's period comes from its parent's mass and its radius.
// Io round Jupiter takes about a day and three quarters; the earth round
// the sun about a year; a bigger radius takes longer by the three halves
// power.
func TestPeriodIsKeplers(t *testing.T) {
	const jupiter, sun = 1.898e27, 1.989e30
	io := Period(jupiter, 421.7e6)
	if d := io.Hours() / 24; math.Abs(d-1.77) > 0.02 {
		t.Errorf("io goes round in %.3f days", d)
	}
	year := Period(sun, 1.496e11)
	if d := year.Hours() / 24; math.Abs(d-365.25) > 1 {
		t.Errorf("the earth goes round in %.1f days", d)
	}
	a, b := Period(jupiter, 1e8), Period(jupiter, 4e8)
	if r := float64(b) / float64(a); math.Abs(r-8) > 1e-6 {
		t.Errorf("four times the radius takes %v times as long", r)
	}
	if Period(0, 1) != 0 || Period(1, 0) != 0 {
		t.Error("no mass or no radius has a period")
	}
}

// An orbit goes round its centre at its radius, once a period, in the
// plane square to its axis, and keeps the thing's own facing and scale.
func TestOrbitGoesRound(t *testing.T) {
	o := Orbit{Centre: maths.V(1, 2, 3), Radius: 5, Period: 10 * time.Second,
		Phase: 0, Scale: 0.5}
	for _, c := range []struct {
		t    time.Duration
		want maths.Vec
	}{
		{0, maths.V(6, 2, 3)},
		{2500 * time.Millisecond, maths.V(1, 2, -2)},
		{5 * time.Second, maths.V(-4, 2, 3)},
		{10 * time.Second, maths.V(6, 2, 3)},
	} {
		s := o.At(c.t)
		if !s.Origin.Near(c.want, 1e-9) || s.Scale != 0.5 {
			t.Errorf("at %v: %v, want %v", c.t, s.Origin, c.want)
		}
		if d := s.Origin.Sub(o.Centre).Len(); math.Abs(d-5) > 1e-9 {
			t.Errorf("at %v the radius is %v", c.t, d)
		}
	}
	tipped := Orbit{Radius: 1, Axis: maths.XAxis, Period: time.Second}
	for _, tt := range []time.Duration{0, 300 * time.Millisecond, 700 * time.Millisecond} {
		if x := tipped.At(tt).Origin.X; math.Abs(x) > 1e-9 {
			t.Errorf("tipped about x, at %v it left the plane: x %v", tt, x)
		}
	}
	still := Orbit{Radius: 2, Phase: math.Pi / 2}
	if a, b := still.At(0), still.At(time.Hour); a != b {
		t.Error("an orbit with no period moved")
	}
}

// A locked orbit keeps one face to the centre all the way round; an
// unlocked one keeps its facing as it was.
func TestALockedOrbitKeepsOneFaceToTheCentre(t *testing.T) {
	o := Orbit{Radius: 2, Period: 4 * time.Second, Locked: true}
	for k := range 4 {
		s := o.At(time.Duration(k) * time.Second)
		face := s.Orientation.Out(maths.XAxis.Neg())
		toCentre := s.Origin.Neg().Unit()
		if face.Sub(toCentre).Len() > 1e-9 {
			t.Fatalf("at %d s the face points %v, the centre is %v", k, face, toCentre)
		}
	}
	u := Orbit{Radius: 2, Period: 4 * time.Second}
	if face := u.At(time.Second).Orientation.Out(maths.XAxis.Neg()); face.Sub(maths.XAxis.Neg()).Len() > 1e-12 {
		t.Fatalf("an unlocked orbit turned its thing: %v", face)
	}
}

// A nested subspace goes round with its parent: a point on the far side
// of a locked moon stays on the far side.
func TestNestedGoesRoundWithItsParent(t *testing.T) {
	o := Orbit{Radius: 2, Period: 4 * time.Second, Locked: true}
	n := Nested{Of: o, In: maths.Subspace{Origin: maths.V(1, 0, 0)}}
	if at := n.At(0).Origin; at.Sub(maths.V(3, 0, 0)).Len() > 1e-9 {
		t.Fatalf("at nought the far side is at %v", at)
	}
	if at := n.At(time.Second).Origin; at.Sub(maths.V(0, 0, -3)).Len() > 1e-9 {
		t.Fatalf("a quarter round the far side is at %v", at)
	}
}

// A spin turns in place: a quarter of a period is a quarter turn about
// the axis, and the origin stays put.
func TestSpinTurnsInPlace(t *testing.T) {
	s := Spin{In: maths.Subspace{Origin: maths.V(1, 2, 3)}, Period: 4 * time.Second}
	at := s.At(time.Second)
	if at.Origin != maths.V(1, 2, 3) {
		t.Fatalf("the origin moved to %v", at.Origin)
	}
	if x := at.Orientation.Out(maths.XAxis); x.Sub(maths.V(0, 0, -1)).Len() > 1e-9 {
		t.Fatalf("a quarter turn about up took x to %v", x)
	}
	if x := s.At(0).Orientation.Out(maths.XAxis); x.Sub(maths.XAxis).Len() > 1e-12 {
		t.Fatalf("at nought x is %v", x)
	}
}

// A growing subspace is as good as nothing before it is born, then grows
// at its rate, carried by its motion.
func TestGrowingGrowsFromItsBirth(t *testing.T) {
	o := Orbit{Radius: 2, Period: 4 * time.Second}
	g := Growing{Of: o, From: time.Second, Rate: 3}
	if s := g.At(0); s.Scale > 1e-6 {
		t.Fatalf("before its birth the scale is %v", s.Scale)
	}
	s := g.At(3 * time.Second)
	if s.Scale != 6 {
		t.Fatalf("two seconds on the scale is %v", s.Scale)
	}
	if s.Origin.Sub(o.At(3*time.Second).Origin).Len() > 1e-12 {
		t.Fatalf("it is at %v, not with its parent", s.Origin)
	}
}

// A beam boosts what comes toward the eye and dims what goes away, the
// same on both sides of the origin, and does nothing at rest.
func TestBeamBoostsTowardTheEye(t *testing.T) {
	b := Beam{Speed: 0.9}
	coming := b.Boost(maths.Up, maths.V(0, 1, 0))
	going := b.Boost(maths.Up.Neg(), maths.V(0, 1, 0))
	side := b.Boost(maths.XAxis, maths.V(0, 1, 0))
	if !(coming > 50 && going < 0.05 && side < 1) {
		t.Fatalf("coming %v, going %v, side %v", coming, going, side)
	}
	if other := b.Boost(maths.Up.Neg(), maths.V(0, -1, 0)); other != coming {
		t.Fatalf("the lower jet toward the eye boosts %v, the upper %v", other, coming)
	}
	if rest := (Beam{}).Boost(maths.Up, maths.V(0, 1, 0)); rest != 1 {
		t.Fatalf("at rest the boost is %v", rest)
	}
}

// A round beam boosts the side of a disc coming toward the eye and
// dims the side going away, mirrored across the axis, and less further
// out where the orbit is slower.
func TestARoundBeamBoostsTheSideComing(t *testing.T) {
	b := Beam{Speed: 0.5, Round: true, At: 1}
	// at plus x the flow is up cross x: minus z
	coming := b.Boost(maths.V(0, 0, -1), maths.V(1, 0, 0))
	going := b.Boost(maths.V(0, 0, 1), maths.V(1, 0, 0))
	if !(coming > 2 && going < 0.5) {
		t.Fatalf("coming %v, going %v", coming, going)
	}
	if other := b.Boost(maths.V(0, 0, 1), maths.V(-1, 0, 0)); math.Abs(other-coming) > 1e-9 {
		t.Fatalf("across the axis the boost is %v, not %v", other, coming)
	}
	if far := b.Boost(maths.V(0, 0, -1), maths.V(4, 0, 0)); !(far < coming && far > 1) {
		t.Fatalf("four radii out the boost is %v", far)
	}
}

// A beam's flow: a round one goes round prograde, from +X toward -Z,
// slowing outside At by the square root; one along its axis flows away
// from its origin on either side.
func TestABeamFlows(t *testing.T) {
	round := Beam{Speed: 0.4, Round: true, At: 1}
	dir, speed := round.Flow(maths.V(1, 0, 0))
	if dir.Sub(maths.V(0, 0, -1)).Len() > 1e-12 || speed != 0.4 {
		t.Errorf("a round beam at (1,0,0) flows %v at %v", dir, speed)
	}
	if _, speed := round.Flow(maths.V(4, 0, 0)); math.Abs(speed-0.2) > 1e-12 {
		t.Errorf("at four times At it flows at %v, not half", speed)
	}
	jet := Beam{Speed: 0.6}
	if dir, _ := jet.Flow(maths.V(0, -2, 0)); dir != maths.Up.Neg() {
		t.Errorf("below its origin a jet flows %v", dir)
	}
}
