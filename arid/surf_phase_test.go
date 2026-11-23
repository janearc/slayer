package arid

import (
	"math"
	"testing"

	"github.com/janearc/slayer/oceantex"
)

// testCoast is a coast over a ground, its first sample at the sea's middle.
func testCoast(t *testing.T, ground Ground) *coast {
	t.Helper()
	shore, err := newCoast(ground, 0, 0, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	return shore
}

// evenHeight is ground a depth under the sea, the same everywhere.
func evenHeight(depth float64) func(int, int) float64 {
	return func(int, int) float64 { return -depth }
}

// northEast is the way north-east, as an east and a south.
var northEast = [2]float64{math.Sqrt2 / 2, -math.Sqrt2 / 2}

func TestBendOverDeepWaterLagsNothing(t *testing.T) {
	ground := shaped(30, 30, 5, evenHeight(400))
	shore := testCoast(t, ground)
	lags, rounds := shore.lag(0.1, 0, 1, northEast)
	for at, lag := range lags {
		if math.Abs(lag) > 1e-9 {
			t.Fatalf("sample %d lags %g m over deep water", at, lag)
		}
	}
	if rounds > sweepRounds {
		t.Fatalf("%d rounds, more than %d", rounds, sweepRounds)
	}
}

func TestBendShortensTheWavesOverShallowWater(t *testing.T) {
	const number, depth = 0.1, 2
	ground := shaped(40, 3, 5, evenHeight(depth))
	shore := testCoast(t, ground)
	lags, _ := shore.lag(number, 0, 0.5, [2]float64{1, 0})
	index := waveNumber(frequency(number, 0), depth) / number
	got := (lags[1*40+30] - lags[1*40+10]) / (20 * 5)
	if math.Abs(got-(index-1)) > 1e-9 {
		t.Fatalf("the lag grows %g a metre, want %g", got, index-1)
	}
}

func TestBendTurnsTheCrestsTowardTheShallows(t *testing.T) {
	// the ground rises eastward, from 10 m deep to the shore
	shore := testCoast(t, shaped(41, 41, 5, func(column, _ int) float64 {
		return math.Min(-10+0.24*float64(column), -0.1)
	}))
	field := shore.sweepFor(0.1, 0, 0.5)
	shore.seed(field, northEast)
	field.solve()
	turned := func(column int) float64 {
		at := 20*41 + column
		east := (field.time[at+1] - field.time[at-1]) / (2 * 5)
		south := (field.time[at+41] - field.time[at-41]) / (2 * 5)
		return math.Atan2(-south, east) // from east toward north
	}
	if deep, shallow := turned(3), turned(37); !(shallow < deep) {
		t.Fatalf("the waves run %g rad from east in shallow water and %g "+
			"in deep; they did not turn toward the shore", shallow, deep)
	}
}

func TestBendWithNoWayInLagsNothing(t *testing.T) {
	// the waves come from the west, and the west edge is all land
	shore := testCoast(t, shaped(10, 10, 5, func(column, _ int) float64 {
		if column == 0 {
			return 3
		}
		return -5
	}))
	lags, _ := shore.lag(0.1, 0, 0.5, [2]float64{1, 0})
	for at, lag := range lags {
		if lag != 0 {
			t.Fatalf("sample %d lags %g with no water the waves came in by",
				at, lag)
		}
	}
}

func TestBendFillsTheLandBetweenTheWater(t *testing.T) {
	shore := testCoast(t, shaped(41, 3, 5, rampHeight))
	lags, _ := shore.lag(0.1, 0, 0.5, [2]float64{1, 0})
	for at, lag := range lags {
		if math.IsNaN(lag) || math.IsInf(lag, 0) {
			t.Fatalf("sample %d lags %g", at, lag)
		}
	}
}

func TestBandNumbersHoldThePeakToEachBand(t *testing.T) {
	bands := []*oceantex.Field{{N: 256, Extent: 4096}, {N: 256, Extent: 512},
		{N: 256, Extent: 64}}
	got := bandNumbers(bands, 0.05)
	want := []float64{0.05, math.Pi * 256 / 4096, math.Pi * 256 / 512}
	for index := range want {
		if math.Abs(got[index]-want[index]) > 1e-12 {
			t.Errorf("band %d: %g, want %g", index, got[index], want[index])
		}
	}
}
