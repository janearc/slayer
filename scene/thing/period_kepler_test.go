package thing

import (
	"fmt"
	"math"
	"testing"
	"time"
)

// refusal is what Period panics with, and whether it did.
func refusal(mass, radius float64) (got string, did bool) {
	defer func() {
		if r := recover(); r != nil {
			got, did = fmt.Sprint(r), true
		}
	}()
	Period(mass, radius)
	return "", false
}

// massFor is the mass whose circular orbit at radius takes seconds,
// kepler's third law turned about: M = 4 pi^2 r^3 / (G T^2).
func massFor(radius, seconds float64) float64 {
	return 4 * math.Pi * math.Pi * radius * radius * radius / (G * seconds * seconds)
}

// T = 2 pi sqrt(r^3 / GM), worked by hand for a mass of 1/G, so that GM
// is one: at r = 1, T = 2 pi s = 6.283185307 s, and at r = 4,
// T = 2 pi sqrt(64) = 16 pi s = 50.26548245 s.
func TestPeriodIsKeplersLawForAUnitGM(t *testing.T) {
	for _, c := range []struct {
		radius float64
		want   time.Duration
	}{{1, 6283185307}, {4, 50265482457}} {
		if got := Period(1/G, c.radius); got != c.want {
			t.Errorf("at radius %v the period is %v ns, want %v", c.radius, int64(got), int64(c.want))
		}
	}
}

// a mass of one kilogram at a radius of one metre has a period, with
// G = 6.67430e-11 (codata): 2 pi / sqrt(G) is about 769,000 s, nine
// days. a mass or radius of one is a mass or radius all the same.
func TestPeriodOfOneKilogramAtOneMetre(t *testing.T) {
	want := 2 * math.Pi / math.Sqrt(6.67430e-11)
	if got := Period(1, 1).Seconds(); math.Abs(got-want) > 1e-9*want {
		t.Errorf("one kilogram at one metre takes %v s, want %v", got, want)
	}
	if got := Period(1, 1).Seconds(); got < 769000 || got > 769200 {
		t.Errorf("one kilogram at one metre takes %v s, want about 769,100", got)
	}
}

// no mass, a mass less than zero, no radius or a radius less than zero has no
// period: it is zero and does not panic. that holds for a mass so small that G
// times it is zero, where the law itself would be zero over zero.
func TestNoMassOrRadiusHasNoPeriod(t *testing.T) {
	for _, c := range []struct{ mass, radius float64 }{
		{0, 1}, {-1, 1}, {1, 0}, {1, -1}, {0, 0}, {1e10, 0}, {1e10, -1e10},
		{math.SmallestNonzeroFloat64, 0},
	} {
		if got := Period(c.mass, c.radius); got != 0 {
			t.Errorf("a mass of %v at a radius of %v has a period of %v", c.mass, c.radius, got)
		}
	}
}

// the most a Duration holds is 9223372036.85 s, and Period gives up to
// 9223372035 s, a little under. worked from the law, at a radius of one:
// the mass for half a second under that is let by, and the mass for
// half a second over is refused.
func TestPeriodRefusesPastTheMost(t *testing.T) {
	const most = 9223372035.0
	under, did := refusal(massFor(1, most-0.5), 1)
	if did {
		t.Errorf("an orbit of half a second under the most was refused: %q", under)
	}
	if got := Period(massFor(1, most-0.5), 1).Seconds(); math.Abs(got-(most-0.5)) > 1e-3 {
		t.Errorf("half a second under the most takes %v s", got)
	}
	if got, did := refusal(massFor(1, most+0.5), 1); !did {
		t.Error("an orbit of half a second over the most was let by")
	} else if got == "" {
		t.Error("the refusal said nothing")
	}
}

// the most itself is refused. found by search, not by the code's own
// answer: at radius 1 the mass 6.953037306082978e-09 kg makes the law's
// 2 pi sqrt(r^3 / GM) come to 9223372035 s to the last bit on this
// machine (no sum in it, so no fused multiply-add to move it).
func TestPeriodRefusesTheMostItself(t *testing.T) {
	const mass = 6.953037306082978e-09
	if seconds := 2 * math.Pi * math.Sqrt(1/(G*mass)); seconds != 9223372035 {
		t.Fatalf("this machine's law gives %v s, not 9223372035: find the mass again", seconds)
	}
	if _, did := refusal(mass, 1); !did {
		t.Error("an orbit of exactly the longest was let by")
	}
}

// the refusal says how many years, which are 365.25 days of 86400 s:
// worked by hand, a thousand years is 1000 * 365.25 * 86400 =
// 31,557,600,000 s.
func TestTheRefusalSaysHowManyYears(t *testing.T) {
	const years = 1000.0
	seconds := years * 365.25 * 86400
	msg, did := refusal(massFor(1, seconds), 1)
	if !did {
		t.Fatal("an orbit of a thousand years was let by")
	}
	var got float64
	if _, err := fmt.Sscanf(msg, "thing: an orbit of %g years is past the 292", &got); err != nil {
		t.Fatalf("the refusal %q: %v", msg, err)
	}
	if math.Abs(got-years) > 1e-9*years {
		t.Errorf("the refusal says %v years, want %v: %q", got, years, msg)
	}
}
