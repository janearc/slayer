package world

import (
	"math"
	"testing"
)

// The spread test's numbers: the places asked, the levels, and how far
// a level's spread may be from the noise octave's, as a share.
const (
	spreadPlaces  = 3000
	spreadFrom    = 2
	spreadTo      = 10
	spreadAllowed = 0.15
	spreadWave    = 0.05
	spreadSeed    = 7
)

// A level of the tree spreads as far as an octave of gradient noise of
// the same gain does, at every level: the look the noise ground was set
// by eye for is the tree's look. At half the height, the land was flat.
func TestALevelSpreadsAsTheNoiseOctaveDoes(t *testing.T) {
	world := New(terran)
	grown := world.tree()
	octave := spreadOf(func(place int) float64 {
		return world.spectrum(spreadSeed, scatterOn(place, spreadPlaces),
			spreadWave, 1, 1)
	})
	seen := new(view)
	for level := spreadFrom; level <= spreadTo; level++ {
		height := amplitude(kinAttribute, level, 1) / valueScale
		got := spreadOf(func(place int) float64 {
			point := scatterOn(place, spreadPlaces)
			grown.read(level, &seen.levels[level], point)
			blended, _ := grown.blend(level, &seen.levels[level], point)
			return blended[kinAttribute] / height
		})
		if math.Abs(got/octave-1) > spreadAllowed {
			t.Errorf("level %d spreads %v for a gain of one, the octave %v",
				level, got, octave)
		}
	}
}

// spreadOf is how far a number spreads over the places asked: its
// standard deviation.
func spreadOf(at func(place int) float64) float64 {
	sum, square := 0.0, 0.0
	for place := range spreadPlaces {
		value := at(place)
		sum += value
		square += value * value
	}
	mean := sum / spreadPlaces
	return math.Sqrt(square/spreadPlaces - mean*mean)
}

// The shore test's numbers: a pixel's share is lone when it is this far
// from its two neighbours' mean, the most of a coast's pixels that may
// be, and the fewest coast pixels the walk must cross.
const (
	loneBy     = 0.5
	loneMost   = 0.01
	coastLeast = 50
	shareEdge  = 0.02
)

// Anti-test: a pixel on a shore is the share of land under it, not a
// pick. Walking a ring of the world a pixel at a time, no pixel at a
// coast stands alone, land among sea or sea among land, apart from its
// two neighbours: that is the stipple a pick at the pixel draws.
func TestAShorePixelIsAShareNotAPick(t *testing.T) {
	ground := grounded(New(terran))
	steps := int(math.Round(2 * math.Pi / middleFootprint))
	share := func(step int) float64 {
		turn := 2 * math.Pi * float64(step) / float64(steps)
		return ground.landShare(onRing(turn), middleFootprint)
	}
	coast, lone := 0, 0
	before, middle := share(0), share(1)
	for step := 2; step < steps; step++ {
		after := share(step)
		if middle > shareEdge && middle < 1-shareEdge {
			coast++
			if math.Abs(middle-(before+after)/2) > loneBy {
				lone++
			}
		}
		before, middle = middle, after
	}
	if coast < coastLeast {
		t.Fatalf("the walk crossed only %d coast pixels", coast)
	}
	if float64(lone) > loneMost*float64(coast) {
		t.Errorf("%d of %d coast pixels stand alone", lone, coast)
	}
}

// The grain test's numbers: how many pixels apart the two points are,
// and the least share of the far eye's grain the close eye sees; for the
// pixel test, the wider step, and the most a pixel's step may differ as a
// share of the wider one's.
const (
	grainApart = 4
	grainLeast = 0.6
	grainWide  = 8
	pixelMost  = 0.5
)

// Anti-test: no grain finer than a few pixels. Between two points a
// pixel apart the grain differs much less than between two eight apart:
// a pixel is the average of what is under it, not one dithered sample.
func TestNoGrainIsFinerThanAFewPixels(t *testing.T) {
	grown := New(terran).tree()
	step := func(apart float64) float64 {
		return spreadOf(func(place int) float64 {
			point := scatterOn(place, spreadPlaces)
			first, _ := tangents(point)
			beside := point.Add(first.Scale(apart * nearFootprint)).Unit()
			return grown.gather(point, nearFootprint).detail[grainAttribute] -
				grown.gather(beside, nearFootprint).detail[grainAttribute]
		})
	}
	pixel, wide := step(1), step(grainWide)
	if pixel > pixelMost*wide {
		t.Errorf("the grain differs by %v a pixel apart and %v at %d", pixel,
			wide, grainWide)
	}
}

// Anti-test: a range's rock is decided on its crests, not on finer
// specks. Once a pixel reads every level of the ridges, a narrower one
// reads the same plain relief and the same crest, so rock does not come
// and go as the eye comes down.
func TestARangesCrestIsTheSameOnceItsRidgesAreRead(t *testing.T) {
	ground := grounded(New(terran))
	whole := ground.tree.spacing(ridgeLast) / (2 * peakSpan)
	for place := range pointsAsked {
		point := scatterOnSphere(place)
		far, _ := ground.here(point, whole)
		near, _ := ground.here(point, nearFootprint)
		if far.height < 0 || near.height < 0 {
			// the sea is not asked its relief
			continue
		}
		if math.Abs(far.plain-near.plain) > reliefHair ||
			math.Abs(far.crest-near.crest) > reliefHair {
			t.Fatalf("at %v the plain and crest are %v, %v read whole and "+
				"%v, %v near", point, far.plain, far.crest, near.plain,
				near.crest)
		}
	}
}

// reliefHair is how near two reliefs are to be one, as sums of the same
// levels in the same order round.
const reliefHair = 1e-9

// The crest test's widths, as shares of the deepest ridge level's
// spacing: one where every level is read, and one where only the first
// few are; and how far the crest's spread may be from one.
const (
	crestWhole   = 1.0 / (2 * peakSpan)
	crestFew     = 1.0
	crestAllowed = 0.15
)

// A crest is in spreads of the ridges read, at any distance: over many
// places it spreads by about one whether the pixel reads every level of
// the ridges or only the coarsest, so ranges are as much of the land
// from far as from near.
func TestACrestSpreadsByOneAtAnyDistance(t *testing.T) {
	grown := New(terran).tree()
	for _, share := range []float64{crestWhole, crestFew} {
		angle := share * grown.spacing(ridgeLast)
		spread := grown.ridgeSpread(angle)
		got := spreadOf(func(place int) float64 {
			noise := grown.gather(scatterOn(place, spreadPlaces),
				angle).detail[ridgeAttribute]
			return crestOf(noise, spread)
		})
		if math.Abs(got-1) > crestAllowed {
			t.Errorf("at %v of the finest ridge's spacing the crest spreads "+
				"by %v", share, got)
		}
	}
}

// Anti-test: the close ground is not flat. Between two points a few
// pixels apart, the light's grain differs about as much for a close eye
// as for a far one: every scale has its grain, and the finest levels a
// close eye sees are not faded to nothing.
func TestTheGrainIsAsStrongCloseAsFar(t *testing.T) {
	world := New(terran)
	grown := world.tree()
	step := func(footprint float64) float64 {
		return spreadOf(func(place int) float64 {
			point := scatterOn(place, spreadPlaces)
			first, _ := tangents(point)
			beside := point.Add(first.Scale(grainApart * footprint)).Unit()
			return grown.gather(point, footprint).detail[grainAttribute] -
				grown.gather(beside, footprint).detail[grainAttribute]
		})
	}
	far, near := step(middleFootprint), step(nearFootprint)
	if near < grainLeast*far {
		t.Errorf("the grain a few pixels apart differs by %v close and %v "+
			"far", near, far)
	}
}
