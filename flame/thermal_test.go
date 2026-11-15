package flame

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// thermalNear is whether got is within tolerance of want; a NaN is
// never near.
func thermalNear(got, want, tolerance float64) bool {
	return math.Abs(got-want) <= tolerance
}

// aFireball is a fireball 10 m across in radius, its centre 30 m up, at
// 1,800 K in air at 290 K, with a stem.
func aFireball(t *testing.T) *Fireball {
	t.Helper()
	ball, err := NewFireball(Thermal{Radius: 10, Height: 30,
		Hottest: measure.Kelvin(1800), Air: measure.Kelvin(290), Soot: 0.5,
		Stem: 0.3, Seed: 11}, 1)
	if err != nil {
		t.Fatal(err)
	}
	return ball
}

// thermalAt is the fireball at seconds after it is let go.
func thermalAt(ball *Fireball, seconds float64) *Fireball {
	return ball.Tick(time.Duration(seconds * float64(time.Second))).(*Fireball)
}

// Scorer's laws, checked each against its own statement rather than
// the code's: the height over the virtual origin is n = 4 radii; the
// rise is C sqrt(g' r), with C = 1.2 and g' the gravity its lightness
// feels, g (1 - 290/1800) at first and falling as the cube of the
// radius. Worked with awk: 10.884132 m/s when let go, and a radius of
// 25.381225 m ten seconds on.
func TestAFireballRisesByScorersLaws(t *testing.T) {
	ball := aFireball(t)
	origin := 30 - 4*10.0
	lighter := 1 - 290.0/1800
	for _, seconds := range []float64{0, 1, 10, 60} {
		now := thermalAt(ball, seconds)
		r := now.Radius()
		if !thermalNear((now.Centre()-origin)/r, 4, 1e-12) {
			t.Errorf("at %v s the height is %v radii from its origin",
				seconds, (now.Centre()-origin)/r)
		}
		felt := 9.80665 * lighter * math.Pow(10/r, 3)
		if want := 1.2 * math.Sqrt(felt*r); !thermalNear(now.Rise(), want, 1e-9*want) {
			t.Errorf("at %v s it rises at %v m/s, Scorer says %v", seconds,
				now.Rise(), want)
		}
		step := 1e-4
		climbed := (thermalAt(ball, seconds+step).Centre() - now.Centre()) / step
		if !thermalNear(climbed, now.Rise(), 1e-3) {
			t.Errorf("at %v s it climbs %v m/s, says %v", seconds, climbed,
				now.Rise())
		}
	}
	if got := thermalAt(ball, 0).Rise(); !thermalNear(got, 10.884132, 1e-6) {
		t.Errorf("let go, it rises at %v m/s", got)
	}
	if got := thermalAt(ball, 10).Radius(); !thermalNear(got, 25.381225, 1e-6) {
		t.Errorf("ten seconds on its radius is %v m", got)
	}
}

// It grows as the root of time: twice as long, twice the growth in the
// square of its radius.
func TestAFireballGrowsAsTheRootOfTime(t *testing.T) {
	ball := aFireball(t)
	r0 := 10.0
	grew := func(seconds float64) float64 {
		r := thermalAt(ball, seconds).Radius()
		return r*r - r0*r0
	}
	for _, seconds := range []float64{0.5, 3, 40} {
		if !thermalNear(grew(2*seconds), 2*grew(seconds), 1e-9*grew(seconds)) {
			t.Errorf("from %v s to %v the square of its radius grew %v, "+
				"then %v", seconds, 2*seconds, grew(seconds), grew(2*seconds))
		}
	}
}

// Its excess heat times its volume holds as it takes in air: at its
// heart it is 1,510 K over the air when let go, and an eighth of that
// once it is twice the size.
func TestAFireballsHeatThinsWithItsVolume(t *testing.T) {
	ball := aFireball(t)
	for _, seconds := range []float64{0, 2, 30} {
		now := thermalAt(ball, seconds)
		r := now.Radius()
		if held := now.Excess() * r * r * r; !thermalNear(held, 1510*1000, 1e-9*1510*1000) {
			t.Errorf("at %v s its excess heat times its radius cubed is %v",
				seconds, held)
		}
		heart := now.Heat(maths.V(0, now.Centre(), 0)).InKelvin()
		if !thermalNear(heart, 290+now.Excess(), 1e-9) {
			t.Errorf("at %v s its heart is %v K", seconds, heart)
		}
		edge := now.Heat(maths.V(r, now.Centre(), 0)).InKelvin()
		if !thermalNear(edge, 290, 1e-9) {
			t.Errorf("at %v s its edge is %v K, not the air's", seconds, edge)
		}
	}
	// twice the radius, when r^2 = 400: (400 - 100) / k seconds on
	k := thermalAt(ball, 1).Radius()*thermalAt(ball, 1).Radius() - 100
	twice := thermalAt(ball, 300/k)
	if !thermalNear(twice.Excess(), 1510.0/8, 1e-6) {
		t.Errorf("at twice its size it is %v K over the air", twice.Excess())
	}
}

// It turns about its ring once for about every doubling: (3 x 4 /
// sqrt 2) ln 2 = 5.881549 radians (awk).
func TestAFireballTurnsAboutItsRingAsItGrows(t *testing.T) {
	ball := aFireball(t)
	k := thermalAt(ball, 1).Radius()*thermalAt(ball, 1).Radius() - 100
	if got := thermalAt(ball, 300/k).turned(); !thermalNear(got, 5.881549, 1e-6) {
		t.Errorf("at twice its size it has turned %v radians", got)
	}
	if got := thermalAt(ball, 0).turned(); !thermalNear(got, 0, 1e-12) {
		t.Errorf("let go, it has turned %v radians", got)
	}
}

// Its soot is nowhere before it is let go, nowhere outside its box,
// never below zero or past its first thickness, and its eddies leave
// between 0.55 and all of it.
func TestAFireballsSootStaysInItsBox(t *testing.T) {
	ball := aFireball(t)
	if got := thermalAt(ball, -1).Density(maths.V(0, 30, 0)); got != 0 {
		t.Errorf("before it is let go its heart holds %v", got)
	}
	for _, seconds := range []float64{0, 3, 20} {
		now := thermalAt(ball, seconds)
		box := now.Bounds()
		size := box.Max.Sub(box.Min)
		for i := range 2000 {
			// a lattice over a box half again as big as the bounds
			u := float64(i%13) / 12
			v := float64(i/13%13) / 12
			w := float64(i/169) / 11
			p := box.Min.Sub(size.Scale(0.25)).Add(maths.V(u*size.X*1.5,
				v*size.Y*1.5, w*size.Z*1.5))
			d := now.Density(p)
			inside := p.X >= box.Min.X && p.X <= box.Max.X &&
				p.Y >= box.Min.Y && p.Y <= box.Max.Y &&
				p.Z >= box.Min.Z && p.Z <= box.Max.Z
			if !inside && d != 0 {
				t.Fatalf("at %v s, %v outside its box holds %v", seconds, p, d)
			}
			if !(d >= 0 && d <= 0.5*(1+0.3)) {
				t.Fatalf("at %v s, %v holds %v", seconds, p, d)
			}
			q := p.Sub(maths.V(0, now.Centre(), 0)).Scale(1 / now.Radius())
			if c := now.clumps(q); !(c >= 0.55-1e-12 && c <= 1) {
				t.Fatalf("at %v s the eddies leave %v of the soot", seconds, c)
			}
		}
	}
}

// Its glow is brightest when it is hottest: its heart when let go
// outshines its heart half a minute on, and its stem, at the air's
// heat, gives next to nothing.
func TestAFireballGlowsByItsHeat(t *testing.T) {
	ball := aFireball(t)
	first := thermalAt(ball, 0)
	later := thermalAt(ball, 30)
	hot := first.Shine().At(maths.V(0, first.Centre(), 0))
	cooled := later.Shine().At(maths.V(0, later.Centre(), 0))
	stem := later.Shine().At(maths.V(0, 1, 0))
	if !(hot.Luminance() > cooled.Luminance() && cooled.Luminance() >= stem.Luminance()) {
		t.Errorf("its glow is %v let go, %v later, %v in its stem",
			hot.Luminance(), cooled.Luminance(), stem.Luminance())
	}
}

// Start moves the whole fireball along the picture's clock and changes
// nothing else: let go at 5 s and seen at 7 s, it is the one let go at
// once and seen at 2 s, and before 5 s it is not there.
func TestAFireballsStartOnlyMovesItsClock(t *testing.T) {
	early := aFireball(t)
	thermal := early.thermal
	thermal.Start = 5 * time.Second
	late, err := NewFireball(thermal, 1)
	if err != nil {
		t.Fatal(err)
	}
	seen, ahead := thermalAt(late, 7), thermalAt(early, 2)
	p := maths.V(3, 70, -2)
	if seen.Radius() != ahead.Radius() || seen.Density(p) != ahead.Density(p) ||
		seen.Heat(p) != ahead.Heat(p) {
		t.Errorf("let go at 5 s and seen at 7 s it is %v m, %v, %v; let go "+
			"at once and seen at 2 s, %v m, %v, %v", seen.Radius(),
			seen.Density(p), seen.Heat(p).InKelvin(), ahead.Radius(),
			ahead.Density(p), ahead.Heat(p).InKelvin())
	}
	if got := thermalAt(late, 4).Density(maths.V(0, 30, 0)); got != 0 {
		t.Errorf("before its start its heart holds %v", got)
	}
}

// A thermal that cannot rise is refused, by each of its numbers.
func TestAThermalThatCannotRiseIsRefused(t *testing.T) {
	sound := Thermal{Radius: 10, Height: 30, Hottest: measure.Kelvin(1800),
		Air: measure.Kelvin(290), Soot: 0.5, Seed: 1}
	broken := []func(*Thermal){
		func(th *Thermal) { th.Radius = 0 },
		func(th *Thermal) { th.Radius = math.NaN() },
		func(th *Thermal) { th.Soot = math.Inf(1) },
		func(th *Thermal) { th.Hottest = measure.Kelvin(290) },
		func(th *Thermal) { th.Air = measure.Kelvin(0) },
		func(th *Thermal) { th.Height = -1 },
		func(th *Thermal) { th.Spread = 0.5 },
		func(th *Thermal) { th.Spread = math.NaN() },
		func(th *Thermal) { th.Stem = 1.5 },
		func(th *Thermal) { th.Stem = math.NaN() },
	}
	for i, breaks := range broken {
		thermal := sound
		breaks(&thermal)
		if _, err := NewFireball(thermal, 1); err == nil {
			t.Errorf("broken thermal %d was let go: %+v", i, thermal)
		}
	}
	if _, err := NewFireball(sound, 1); err != nil {
		t.Errorf("a sound thermal was refused: %v", err)
	}
}
