package maths_test

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestPercentSaysPercents(test *testing.T) {
	for _, said := range []struct {
		amount, share float64
	}{{50, 0.5}, {1, 0.01}, {0, 0}, {100, 1}} {
		got, err := maths.Percent(said.amount).Share()
		if err != nil || got != said.share {
			test.Fatalf("Percent(%v) said %v, %v; want %v",
				said.amount, got, err, said.share)
		}
	}
}

func TestPercentTakesFractionsOfAPercentFromOneUp(test *testing.T) {
	got, err := maths.Percent(99.9).Share()
	if err != nil || !(math.Abs(got-0.999) <= 1e-12) {
		test.Fatalf("Percent(99.9) said %v, %v; want 0.999", got, err)
	}
}

func TestPercentNamesTheFixForAFraction(test *testing.T) {
	_, err := maths.Percent(0.5).Share()
	if err == nil {
		test.Fatal("Percent(0.5) passed")
	}
	for _, fix := range []string{"Percent(50)", "Share(0.5)"} {
		if !strings.Contains(err.Error(), fix) {
			test.Errorf("the complaint %q does not name %s", err, fix)
		}
	}
}

func TestPercentRefusesWhatReadsAsAFraction(test *testing.T) {
	for _, amount := range []float64{0.5, 0.006, 0.999} {
		if _, err := maths.Percent(amount).Share(); err == nil {
			test.Fatalf("Percent(%v) passed; it reads as a "+
				"fraction, not a percent", amount)
		}
	}
}

func TestPercentRefusesWhatNoPortionIs(test *testing.T) {
	for _, amount := range []float64{-1, 101, math.NaN(),
		math.Inf(1), math.Inf(-1)} {
		if _, err := maths.Percent(amount).Share(); err == nil {
			test.Fatalf("Percent(%v) passed; no portion is", amount)
		}
	}
}

func TestShareSaysFractions(test *testing.T) {
	for _, said := range []float64{0.5, 0.006, 0, 1} {
		got, err := maths.Share(said).Share()
		if err != nil || got != said {
			test.Fatalf("Share(%v) said %v, %v; want itself",
				said, got, err)
		}
	}
}

func TestShareRefusesOutsideTheWhole(test *testing.T) {
	for _, amount := range []float64{-0.1, 1.1, math.NaN(),
		math.Inf(1)} {
		if _, err := maths.Share(amount).Share(); err == nil {
			test.Fatalf("Share(%v) passed; a share is of [0, 1]",
				amount)
		}
	}
}
