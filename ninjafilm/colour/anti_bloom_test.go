package colour_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/colour"
	"github.com/janearc/slayer/ninjafilm/data"
	"pgregory.net/rapid"
)

// starIn is a row of lights of one dim grey with a star of a light in
// its middle.
func starIn(across int, star float64) []maths.Light {
	row := make([]maths.Light, across)
	for place := range row {
		row[place] = maths.Grey(baseLight)
	}
	row[across/pair] = maths.Grey(star)
	return row
}

// anti-tests: blooms do not spill all the light beyond white. What is
// added is a third of it, and so less than the whole.
func TestABloomDoesNotSpillAllTheLightBeyondWhite(t *testing.T) {
	rapid.Check(t, bloomSpillsOnlyAThird)
}

func bloomSpillsOnlyAThird(t *rapid.T) {
	radius := rapid.Float64Range(radiusLeast, bloomRadius).Draw(t, "radius")
	star := rapid.Float64Range(pair, lightMost).Draw(t, "star")
	across := rowsInATest * pair
	lights := starIn(across, star)
	got := colour.Bloom(lights, across, whole, whole, radius)
	var added float64
	for place := range got {
		added += got[place].R - lights[place].R
	}
	if added >= (star-whole)-within || added <= none {
		t.Fatalf("a star of %v added %v, beyond white it had %v", star, added,
			star-whole)
	}
}

// anti-tests: blooms do not reach past their reach. Pixels further from the
// star than three radii are as they were, bit for bit.
func TestABloomDoesNotReachPastThreeRadii(t *testing.T) {
	across := rowsInATest * pair * pair
	lights := starIn(across, starLight)
	radius := data.Picked("bloom-reach-sigmas") / (pair + whole)
	got := colour.Bloom(lights, across, whole, whole, radius)
	reach := math.Ceil(data.Picked("bloom-reach-sigmas") * radius)
	far := across/pair + int(reach) + whole
	if got[far] != lights[far] {
		t.Errorf("a pixel %d from the star is %v, it was %v", far-across/pair,
			got[far], lights[far])
	}
}

// anti-tests: the widest bloom adds far less to its star than a bloom of
// one pixel does. The spill is spread thin and not kept at the star.
func TestTheWidestBloomAddsLessToItsStarThanANarrowOne(t *testing.T) {
	lights := starIn(rowsInATest, starLight)
	narrow := colour.Bloom(lights, rowsInATest, whole, whole, whole)
	wide := colour.Bloom(lights, rowsInATest, whole, whole,
		data.Picked("bloom-widest"))
	middle := rowsInATest / pair
	if !(wide[middle].R < narrow[middle].R) {
		t.Errorf("the star has %v under the widest bloom, %v under one",
			wide[middle].R, narrow[middle].R)
	}
}

// anti-tests: the finest bloom does not spill onto a neighbour. The pixel
// beside a star is as it was, bit for bit.
func TestTheFinestBloomDoesNotSpillOntoANeighbour(t *testing.T) {
	lights := starIn(rowsInATest, starLight)
	got := colour.Bloom(lights, rowsInATest, whole, whole,
		math.SmallestNonzeroFloat64)
	beside := rowsInATest/pair + whole
	if got[beside] != lights[beside] {
		t.Errorf("the pixel beside the star is %v, it was %v", got[beside],
			lights[beside])
	}
}

// anti-tests: a bloom never takes light away. Every pixel is as bright
// after it as before, in each lamp.
func TestABloomNeverTakesLightAway(t *testing.T) {
	rapid.Check(t, bloomNeverDims)
}

func bloomNeverDims(t *rapid.T) {
	radius := rapid.Float64Range(radiusLeast, bloomRadius).Draw(t, "radius")
	star := rapid.Float64Range(none, lightMost).Draw(t, "star")
	exposure := rapid.Float64Range(half, exposureMost).Draw(t, "exposure")
	lights := starIn(rowsInATest, star)
	got := colour.Bloom(lights, rowsInATest, whole, exposure, radius)
	for place := range got {
		if got[place].R < lights[place].R || got[place].G < lights[place].G ||
			got[place].B < lights[place].B {
			t.Fatalf("pixel %d went from %v to %v", place, lights[place],
				got[place])
		}
	}
}

// anti-tests: small blooms do not reach as far as big ones. Pixels four
// from the star are lit by the big and not by the small.
func TestASmallBloomDoesNotReachAsFarAsABigOne(t *testing.T) {
	lights := starIn(rowsInATest*pair, starLight)
	far := rowsInATest + pair*pair
	small := colour.Bloom(lights, rowsInATest*pair, whole, whole, half)
	big := colour.Bloom(lights, rowsInATest*pair, whole, whole, pair)
	if small[far] != lights[far] || !(big[far].R > lights[far].R) {
		t.Errorf("four from the star: small %v, big %v, was %v", small[far],
			big[far], lights[far])
	}
}

// anti-tests: a picture with something beyond white is not returned as
// it was. The star's own pixel gains light.
func TestAPictureWithAStarIsNotReturnedAsItWas(t *testing.T) {
	lights := starIn(rowsInATest, starLight)
	got := colour.Bloom(lights, rowsInATest, whole, whole, whole)
	if got[rowsInATest/pair] == lights[rowsInATest/pair] {
		t.Errorf("a star was returned as it was")
	}
}
