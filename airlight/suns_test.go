package airlight

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/skylight"
)

// The suns the plan's span is checked with.
const (
	lowSunZenith = 73.04 // picked: a sun seventeen degrees up
	readFrom     = 70    // defined: the bins of 72.5 and 77.5 degrees
	readTo       = 80    // defined: their far edges
	spanSaid     = "for a sun 70 to 80 degrees from up"
	refusedSaid  = "walked for a sun"
	// quarterZeniths is four zenith bins of 45 degrees: a sun thirty
	// degrees from up is read between the two above the horizon alone.
	quarterZeniths = 4 // picked: see above
	// wantedCrossings is the crossings a plan for one sun wants.
	wantedCrossings = 12100 // worked: 1.32 over 0.012 is 110, squared
	// wantedShare is how near the plan's crossings must come to it.
	wantedShare = 1e-9 // picked: a few roundings of a division
	// heldCount, belowCount and otherCount are the crossings of the bins
	// above the horizon in the span, below it, and outside the span.
	heldCount  = 400 // picked: a root of twenty
	belowCount = 9   // picked: unlike the others
	otherCount = 1   // picked: unlike the others
	// heldNoise is the noise of heldCount crossings.
	heldNoise = 0.066 // worked: 1.32 over the root of 400, twenty
)

// a plan for a sun is walked for the two zenith bins the sun is read
// between, and says so.
func TestAPlanForASunWalksTheBinsItIsReadBetween(t *testing.T) {
	plan, err := NewPlan(thinKnown(t, thinDepth/thinTop, fewPhotons),
		thinGround)
	if err != nil {
		t.Fatal(err)
	}
	made := plan.For(wayAt(lowSunZenith*degree, none))
	want := skylight.Suns{From: readFrom * degree, To: readTo * degree}
	if math.Abs(made.suns.From-want.From) > handShare ||
		math.Abs(made.suns.To-want.To) > handShare {
		t.Errorf("the span is %+v, want %+v", made.suns, want)
	}
	if !strings.Contains(made.Said(), spanSaid) {
		t.Errorf("the plan says %q, not %q", made.Said(), spanSaid)
	}
}

// fields walked for some suns are read for a sun among them, and refuse
// one outside, by Holds and by Meter.
func TestFieldsRefuseASunTheyWereNotWalkedFor(t *testing.T) {
	fields := handFields(quarterZeniths, handViews, handAzimuths,
		even(handSky))
	fields.suns = skylight.Suns{From: none, To: math.Pi / two}
	if err := fields.Holds(wayAt(thirty*degree, none)); err != nil {
		t.Errorf("a sun thirty degrees from up, in the span: %v", err)
	}
	outside := wayAt(math.Pi-thirty*degree, none)
	if err := fields.Holds(outside); err == nil {
		t.Error("a sun below the span is read")
	}
	if _, err := Meter(fields, outside); err == nil ||
		errors.Is(err, errUnlit) {
		t.Errorf("a sun below the span meters: %v", err)
	}
	if _, err := NewSky(fields, outside, handExposure); err == nil ||
		!strings.Contains(err.Error(), refusedSaid) {
		t.Errorf("a sun below the span makes a sky: %v", err)
	}
	if _, err := NewSky(fields, wayAt(thirty*degree, none),
		handExposure); err != nil {
		t.Errorf("a sun in the span makes no sky: %v", err)
	}
}

// the anti-test: fields walked for every sun hold every sun, so the
// check would see a span that refused all.
func TestFieldsOfEverySunHoldEverySun(t *testing.T) {
	fields := handFields(handZeniths, handViews, handAzimuths,
		even(handSky))
	for _, zenith := range []float64{none, thirty * degree, math.Pi} {
		if err := fields.Holds(wayAt(zenith, none)); err != nil {
			t.Errorf("zenith %g: %v", zenith, err)
		}
	}
}

// pilotMost is the most photons the tests' plans for one sun walk.
const pilotMost = 4 * fewPhotons // picked: a test's moment

// a plan for one sun wants the crossings of one 8-bit step of colour,
// and walks between the air's photons and its most to have them.
func TestAPlanForASunWalksEnoughPhotons(t *testing.T) {
	plan, err := NewPlan(thinKnown(t, thinDepth/thinTop, fewPhotons),
		thinGround)
	if err != nil {
		t.Fatal(err)
	}
	made := plan.For(wayAt(thirty*degree, none))
	if math.Abs(made.wanted-wantedCrossings) > wantedShare ||
		made.most != mostPhotons {
		t.Errorf("the plan wants %g crossings to %d photons, not %d",
			made.wanted, made.most, wantedCrossings)
	}
	made.most = pilotMost
	photons, err := made.enough()
	if err != nil {
		t.Fatal(err)
	}
	// a thin air walked with a few thousand photons holds far fewer
	// crossings than are wanted, so the count must rise above the air's.
	if photons <= fewPhotons || photons > pilotMost {
		t.Errorf("%d photons, not above %d and at most %d", photons,
			fewPhotons, pilotMost)
	}
}

// the anti-test: a plan of every sun wants no crossings and walks the
// air's photons as they are.
func TestAPlanOfEverySunWalksItsPhotons(t *testing.T) {
	plan, err := NewPlan(thinKnown(t, thinDepth/thinTop, fewPhotons),
		thinGround)
	if err != nil {
		t.Fatal(err)
	}
	if photons, err := plan.enough(); err != nil || photons != fewPhotons {
		t.Errorf("%d photons, %v; want %d", photons, err, fewPhotons)
	}
}

// knownCounts is hand fields of four zenith bins whose crossings are
// heldCount above the horizon in the two a sun thirty degrees from up
// is read between, belowCount below the horizon there, and otherCount
// in the other two.
func knownCounts() *Fields {
	fields := handFields(quarterZeniths, handViews, handAzimuths,
		even(handSky))
	for zenith := range fields.zeniths {
		for view := range fields.views {
			count := float64(otherCount)
			if zenith < quarterZeniths/two {
				count = countFor(view, fields.views)
			}
			for round := range fields.azimuths {
				fields.counts[none][fields.index(zenith, view, round)] = count
			}
		}
	}
	return fields
}

// countFor is a read bin's count: heldCount above the horizon,
// belowCount below.
func countFor(view, views int) float64 {
	if view >= views/two {
		return heldCount
	}
	return belowCount
}

// fields say the crossings behind the bins above the horizon a sun is
// read from, and the colour noise they leave: worked by hand, 1.32 over
// the root of 400.
func TestFieldsSayTheNoiseTheyKeep(t *testing.T) {
	crossings, noise := knownCounts().Noise(wayAt(thirty*degree, none))
	if crossings != heldCount || math.Abs(noise-heldNoise) > handShare {
		t.Errorf("%g crossings, a noise of %g; want %d and %g", crossings,
			noise, heldCount, heldNoise)
	}
}

// the anti-test: fields with no crossings say an endless noise, not
// none.
func TestFieldsWithNoCrossingsSayEndlessNoise(t *testing.T) {
	fields := handFields(handZeniths, handViews, handAzimuths,
		even(handSky))
	if _, noise := fields.Noise(wayAt(thirty*degree, none)); !math.IsInf(
		noise, one) {
		t.Errorf("no crossings, a noise of %g", noise)
	}
}

// bytes whose span of suns runs down, or past pi, are refused.
func TestBytesOfABrokenSpanAreRefused(t *testing.T) {
	for _, suns := range []skylight.Suns{{From: one, To: none},
		{From: none, To: two * math.Pi}} {
		fields := countedFields()
		fields.suns = suns
		if _, err := FieldsFromBytes(fields.Bytes()); err == nil {
			t.Errorf("a span of %+v read as fields", suns)
		}
	}
}
