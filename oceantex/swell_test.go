package oceantex

import (
	"math"
	"testing"
)

// swellMetres, swellSeconds and swellToward are a swell of the kind a
// distant storm sends: two and a half metres, twelve seconds, northward.
const (
	swellMetres  = 2.5
	swellSeconds = 12.0
	swellToward  = math.Pi / 2
)

// heightWithin is how near, as a share, a swell's height read back by an
// independent sum must come to the height it was made with.
const heightWithin = 0.02

// exactWithin is how near two sums of one spectrum by one rule must be.
const exactWithin = 1e-9

// madeSwell is the test swell, or ends the test.
func madeSwell(t *testing.T, toward float64) *Swell {
	t.Helper()
	made, err := NewSwell(swellMetres, swellSeconds, toward)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// A sea with no swell is the wind sea alone, to the bit.
func TestSeaWithNoSwellIsTheWindSeaAlone(t *testing.T) {
	for _, k := range []float64{0.01, 0.1, 1} {
		got, want := north.wavenumber(k, 0.3*k), north.windWavenumber(k, 0.3*k)
		if got != want {
			t.Errorf("k %v: %v with no swell, the wind sea's %v", k, got, want)
		}
	}
	peak := north.Peak()
	want := heightTimes * math.Sqrt(
		logSum(sumLow*peak, sumHigh*peak, sumSteps, north.frequency))
	if north.Height() != want {
		t.Errorf("height %v with no swell, want %v", north.Height(), want)
	}
	budget, err := north.BudgetOf(0, 1)
	if err != nil || budget.Total != coxMunkSlope(north.Wind) {
		t.Errorf("total %v, %v, want Cox and Munk's alone", budget.Total, err)
	}
}

// A swell's height comes back as it was given, summed two ways: by the
// sea's own sum, and by a plain sum in frequency a different way.
func TestSwellHeightComesBack(t *testing.T) {
	sea := Sea{Swell: madeSwell(t, swellToward)}
	near(t, "the sea's height", sea.Height(), swellMetres,
		swellMetres*exactWithin)
	const steps = 200000
	peak := sea.Swell.peak()
	step := sumHigh * peak / steps
	var variance float64
	for i := 0; i < steps; i++ {
		variance += sea.Swell.frequency((float64(i)+0.5)*step) * step
	}
	near(t, "the plain sum's height", heightTimes*math.Sqrt(variance),
		swellMetres, swellMetres*heightWithin)
}

// A swell's spread sums to one over a full turn.
func TestSwellSpreadSumsToOne(t *testing.T) {
	const steps = 20000
	var sum float64
	for i := 0; i < steps; i++ {
		theta := -math.Pi + (float64(i)+0.5)*2*math.Pi/steps
		sum += swellSpread(theta) * 2 * math.Pi / steps
	}
	near(t, "the spread's sum", sum, 1, 1e-6)
}

// The wave-number spectrum keeps the swell's variance: summed over the
// plane it is the variance the height says.
func TestSwellWavenumberKeepsTheEnergy(t *testing.T) {
	sea := Sea{Swell: madeSwell(t, swellToward)}
	const turns, rings = 720, 1500
	low := math.Pow(sumLow*sea.Swell.peak(), 2) / gravity
	high := math.Pow(sumHigh*sea.Swell.peak(), 2) / gravity
	step := math.Log(high/low) / rings
	var sum float64
	for turn := 0; turn < turns; turn++ {
		sin, cos := math.Sincos((float64(turn) + 0.5) * 2 * math.Pi / turns)
		for ring := 0; ring < rings; ring++ {
			k := low * math.Exp((float64(ring)+0.5)*step)
			sum += sea.wavenumber(k*cos, k*sin) * k * k * step *
				2 * math.Pi / turns
		}
	}
	want := math.Pow(swellMetres/heightTimes, 2)
	near(t, "the variance by wave vector", sum, want, want*heightWithin)
}

// Anti-test: a swell travelling the other way puts its energy the other
// way. Along its direction there is energy and against it next to none.
func TestSwellTowardMovesItsEnergy(t *testing.T) {
	northward := Sea{Swell: madeSwell(t, swellToward)}
	southward := Sea{Swell: madeSwell(t, -swellToward)}
	k := math.Pow(northward.Swell.peak(), 2) / gravity
	with, against := northward.wavenumber(0, k), northward.wavenumber(0, -k)
	if !(with > 1e6*against) {
		t.Errorf("northward swell: %v along, %v against", with, against)
	}
	got := southward.wavenumber(0, -k)
	if math.Abs(got-with) > with*exactWithin {
		t.Errorf("southward swell: %v along, want the northward %v", got, with)
	}
	if got := southward.wavenumber(0, k); !(got < with/1e6) {
		t.Errorf("southward swell has %v northward, want next to none", got)
	}
}

// Anti-test: a swell is refused for a height or period of nought or less
// or not a number, and a direction that is not a number.
func TestSwellRefusesWhatIsNotASwell(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(1)
	for _, bad := range [][3]float64{
		{0, swellSeconds, 0}, {-1, swellSeconds, 0}, {swellMetres, 0, 0},
		{swellMetres, -3, 0}, {nan, swellSeconds, 0}, {swellMetres, inf, 0},
		{swellMetres, swellSeconds, nan},
	} {
		if _, err := NewSwell(bad[0], bad[1], bad[2]); err == nil {
			t.Errorf("NewSwell%v is made", bad)
		}
	}
}

// Shallow water carries less of the swell, as it does the wind sea.
func TestShallowWaterCarriesLessSwell(t *testing.T) {
	deep := Sea{Swell: madeSwell(t, 0)}
	shallow := Sea{Swell: madeSwell(t, 0), Depth: 3}
	if !(shallow.Height() < deep.Height()) {
		t.Errorf("height %v in shallows, %v in deep water",
			shallow.Height(), deep.Height())
	}
}

// The swell's slope is added to Cox and Munk's, and a short swell is
// steeper than a long one of the same height: the slope goes as the
// fourth power of the frequency, so half the period is sixteen times.
func TestSwellAddsItsOwnSlope(t *testing.T) {
	plain, err := north.BudgetOf(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	swelled := north
	swelled.Swell = madeSwell(t, 0)
	got, err := swelled.BudgetOf(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	near(t, "the total", got.Total, plain.Total+swelled.Swell.slope(), 1e-12)
	if !(swelled.Swell.slope() > 0) {
		t.Fatalf("the swell's slope is %v", swelled.Swell.slope())
	}
	short, err := NewSwell(swellMetres, swellSeconds/2, 0)
	if err != nil {
		t.Fatal(err)
	}
	const fourth = 16.0
	near(t, "the short swell's slope over the long one's",
		short.slope()/swelled.Swell.slope(), fourth, fourth*exactWithin)
}
