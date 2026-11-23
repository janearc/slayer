package trace

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// biasWorths are what four lamps are worth to a point: eight in all, so
// that four picks are two apart and every number here is exact.
var biasWorths = []float64{1, 2, 3, 2}

// biasStarts are four starts spread evenly over a step: the first pick
// falls a quarter, three quarters, one and a quarter and one and three
// quarters into the worth laid end to end.
var biasStarts = []float64{0.125, 0.375, 0.625, 0.875}

// fourPicked is what picked takes of four lamps worth biasWorths, with
// four picks begun at start, as each lamp's count by its index.
func fourPicked(start float64) [4]float64 {
	lamps := make([]lit, len(biasWorths))
	for index := range lamps {
		lamps[index] = lit{index: index, counts: 1,
			light: maths.Grey(biasWorths[index])}
	}
	var counts [4]float64
	for _, one := range picked(nil, lamps, biasWorths, 8, start, 4) {
		counts[one.index] += one.counts
	}
	return counts
}

// meanCounts is each lamp's count averaged over biasStarts, for a way of
// picking.
func meanCounts(pick func(start float64) [4]float64) [4]float64 {
	var mean [4]float64
	for _, start := range biasStarts {
		for index, count := range pick(start) {
			mean[index] += count / float64(len(biasStarts))
		}
	}
	return mean
}

// everyLampOnce is whether every mean count is one, to twelve places:
// then the picks' light, counted as picked says, is on average every
// lamp's light, whatever each lamp sends.
func everyLampOnce(mean [4]float64) bool {
	for _, count := range mean {
		if !within(count, 1, 1e-12) {
			return false
		}
	}
	return true
}

// Four picks among lamps worth 1, 2, 3 and 2 fall two apart along the
// worths laid end to end, which the lamps hold from 0 to 1, 1 to 3, 3 to
// 6 and 6 to 8. Begun a quarter or three quarters in, the picks land at
// 0.25 or 0.75, then two, four and six further: one in each lamp. Begun
// at 1.25 or 1.75 they miss the first lamp and land twice in the third.
//
// A lamp picked n times counts n times the step of two over its worth:
// 2, 1, 2/3 and 1, and then 0, 1, 4/3 and 1. Over the four starts every
// lamp's count averages one, so the picks stand for every lamp's light
// and not only for the worth they were picked by.
func TestPickedCountsEveryLampOnceOnAverageOverItsStarts(t *testing.T) {
	early, late := [4]float64{2, 1, 2.0 / 3, 1}, [4]float64{0, 1, 4.0 / 3, 1}
	for at, want := range [][4]float64{early, early, late, late} {
		got := fourPicked(biasStarts[at])
		for index := range want {
			if !within(got[index], want[index], 1e-12) {
				t.Errorf("begun at %v lamp %d counts %v, want %v",
					biasStarts[at], index, got[index], want[index])
			}
		}
	}
	if mean := meanCounts(fourPicked); !everyLampOnce(mean) {
		t.Errorf("over the starts the lamps count %v, want one each", mean)
	}
}

// Anti: a picker that forgets its start is told. Begun at nought every
// time, the picks land at 0, 2, 4 and 6: the counts are 2, 1, 2/3 and 1
// at every start, worth eight in all as every lamp together is, and the
// first lamp is counted twice over and the third two thirds.
func TestAPickerThatForgetsItsStartIsSeenToFavourLamps(t *testing.T) {
	forgetful := func(float64) [4]float64 { return fourPicked(0) }
	mean := meanCounts(forgetful)
	worth := 0.0
	for index, count := range mean {
		worth += count * biasWorths[index]
	}
	if !within(worth, 8, 1e-12) {
		t.Fatalf("the forgetful picks are worth %v, want all eight", worth)
	}
	if everyLampOnce(mean) || !within(mean[0], 2, 1e-12) ||
		!within(mean[2], 2.0/3, 1e-12) {
		t.Errorf("a picker begun at nought every time counts the lamps "+
			"%v, and passed for fair", mean)
	}
}
