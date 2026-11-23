package thing

import (
	"fmt"
	"math"
	"testing"
	"time"
)

// The files the period tests read their tables from.
const (
	periodKeplerUnitFile     = "testdata/period_kepler_unit_gm.csv"
	periodKeplerNoPeriodFile = "testdata/period_kepler_no_period.csv"
)

// What the period tests work with, each true by definition.
const (
	periodKeplerFour       = 4      // defined: the 4 in kepler's third law
	periodKeplerJulianYear = 365.25 // defined: days in a julian year
	periodKeplerDay        = 86400  // defined: seconds in a day
)

// periodKeplerRecord is deferred by refusal: it puts what a panic said into
// got and that there was one into did.
func periodKeplerRecord(got *string, did *bool) {
	if recovered := recover(); recovered != nil {
		*got, *did = fmt.Sprint(recovered), true
	}
}

// refusal is what Period panics with, and whether it did.
func refusal(mass, radius float64) (got string, did bool) {
	defer periodKeplerRecord(&got, &did)
	Period(mass, radius)
	return "", false
}

// massFor is the mass whose circular orbit at radius takes seconds,
// kepler's third law turned about: M = 4 pi^2 r^3 / (G T^2).
func massFor(radius, seconds float64) float64 {
	return periodKeplerFour * math.Pi * math.Pi * radius * radius * radius /
		(G * seconds * seconds)
}

// T = 2 pi sqrt(r^3 / GM), worked by hand for a mass of 1/G, so that GM
// is one: at r = 1, T = 2 pi s = 6.283185307 s, and at r = 4,
// T = 2 pi sqrt(64) = 16 pi s = 50.26548245 s.
func TestPeriodIsKeplersLawForAUnitGM(t *testing.T) {
	for _, each := range readSheet(t, periodKeplerUnitFile) {
		radius := cell(t, each, "radius")
		want := time.Duration(cellCount(t, each, "nanoseconds"))
		if got := Period(unity/G, radius); got != want {
			t.Errorf("at radius %v the period is %v ns, want %v",
				radius, int64(got), int64(want))
		}
	}
}

// a mass of one kilogram at a radius of one metre has a period, with
// G = 6.67430e-11 (codata): 2 pi / sqrt(G) is about 769,000 s, nine
// days. a mass or radius of one is a mass or radius all the same.
func TestPeriodOfOneKilogramAtOneMetre(t *testing.T) {
	gravity := picked(t, "period-kepler-codata-g")
	tolerance := picked(t, "period-kepler-relative-tolerance")
	least := picked(t, "period-kepler-one-kilogram-least")
	most := picked(t, "period-kepler-one-kilogram-most")
	want := pair * math.Pi / math.Sqrt(gravity)
	got := Period(unity, unity).Seconds()
	if !(math.Abs(got-want) <= tolerance*want) {
		t.Errorf("one kilogram at one metre takes %v s, want %v", got, want)
	}
	if got < least || got > most {
		t.Errorf("one kilogram at one metre takes %v s, want about 769,100",
			got)
	}
}

// periodKeplerNoPeriod fails the test when Period of mass and radius is not
// zero.
func periodKeplerNoPeriod(t *testing.T, mass, radius float64) {
	t.Helper()
	if got := Period(mass, radius); got != nought {
		t.Errorf("a mass of %v at a radius of %v has a period of %v",
			mass, radius, got)
	}
}

// no mass, a mass less than zero, no radius or a radius less than zero has no
// period: it is zero and does not panic. that holds for a mass so small that G
// times it is zero, where the law itself would be zero over zero.
func TestNoMassOrRadiusHasNoPeriod(t *testing.T) {
	for _, each := range readSheet(t, periodKeplerNoPeriodFile) {
		mass, radius := cell(t, each, "mass"), cell(t, each, "radius")
		periodKeplerNoPeriod(t, mass, radius)
	}
	periodKeplerNoPeriod(t, math.SmallestNonzeroFloat64, nought)
}

// the most a Duration holds is 9223372036.85 s, and Period gives up to
// 9223372035 s, a little under. worked from the law, at a radius of one:
// the mass for half a second under that is let by, and the mass for
// half a second over is refused.
func TestPeriodRefusesPastTheMost(t *testing.T) {
	most := picked(t, "period-kepler-most-seconds")
	half := picked(t, "period-kepler-half-second")
	tolerance := picked(t, "period-kepler-second-tolerance")
	lighter := massFor(unity, most-half)
	under, did := refusal(lighter, unity)
	if did {
		t.Errorf("half a second under the most was refused: %q", under)
	}
	got := Period(lighter, unity).Seconds()
	if !(math.Abs(got-(most-half)) <= tolerance) {
		t.Errorf("half a second under the most takes %v s", got)
	}
	if got, did := refusal(massFor(unity, most+half), unity); !did {
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
	most := picked(t, "period-kepler-most-seconds")
	// the mass is kept as a constant, so that G times it is worked exactly.
	const mass = 6.953037306082978e-09
	if seconds := pair * math.Pi * math.Sqrt(unity/(G*mass)); seconds != most {
		t.Fatalf("this machine's law gives %v s, not %v: find the mass again",
			seconds, most)
	}
	if _, did := refusal(mass, unity); !did {
		t.Error("an orbit of exactly the longest was let by")
	}
}

// the refusal says how many years, which are 365.25 days of 86400 s:
// worked by hand, a thousand years is 1000 * 365.25 * 86400 =
// 31,557,600,000 s.
func TestTheRefusalSaysHowManyYears(t *testing.T) {
	years := picked(t, "period-kepler-years")
	tolerance := picked(t, "period-kepler-relative-tolerance")
	seconds := years * periodKeplerJulianYear * periodKeplerDay
	msg, did := refusal(massFor(unity, seconds), unity)
	if !did {
		t.Fatal("an orbit of a thousand years was let by")
	}
	var got float64
	const format = "thing: an orbit of %g years is past the 292"
	if _, err := fmt.Sscanf(msg, format, &got); err != nil {
		t.Fatalf("the refusal %q: %v", msg, err)
	}
	if !(math.Abs(got-years) <= tolerance*years) {
		t.Errorf("the refusal says %v years, want %v: %q", got, years, msg)
	}
}
