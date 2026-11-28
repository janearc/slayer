package world

import (
	"math"
	"testing"
)

// A coast's gain is a share of the ground's slope over its wave, never
// under the floor nor over the ceiling, and away from the coast it is
// the plain gain.
func TestGainIsBetweenItsFloorAndItsCeiling(t *testing.T) {
	world := New(terran)
	for place := range pointsAsked {
		point := scatterOnSphere(place)
		got := world.coastGain(point)
		if math.Abs(world.lifted(point)) > coastReach {
			if got != heightGain {
				t.Fatalf("far from the coast the gain is %v", got)
			}
			continue
		}
		if got < slopeFloor || got > slopeCeil {
			t.Fatalf("the gain %v is outside %v to %v", got, slopeFloor,
				slopeCeil)
		}
	}
}

// Tangents are unit, at right angles to each other and to the point,
// whichever way the point is.
func TestTangentsAreAtRightAngles(t *testing.T) {
	for place := range pointsAsked {
		point := scatterOnSphere(place)
		first, second := tangents(point)
		for _, dot := range []float64{first.Dot(second), first.Dot(point),
			second.Dot(point)} {
			if math.Abs(dot) > 1e-9 {
				t.Fatalf("tangents at %v are not at right angles: %v", point,
					dot)
			}
		}
		if math.Abs(first.Len()-1) > 1e-9 || math.Abs(second.Len()-1) > 1e-9 {
			t.Fatalf("tangents at %v are not unit", point)
		}
	}
}

// Rock follows a range's crests on high ground: a crest on low ground is
// not rock, nor is high ground off a crest, and a crest on high ground is.
func TestRockNeedsACrestOnHighGround(t *testing.T) {
	look := rampLook()
	ground := &scrooged{w: New(terran), look: look, mix: 0.35}
	lowland := look.Paints[Grass]
	isRock := func(plain, crest float64) bool {
		got := ground.mountain(lowland, plain, crest, 0)
		return math.Abs(got.R-look.Paints[Rock].R) < paintHair
	}
	if isRock(rockHeight-2*crestPlain, 2*crestRock) {
		t.Error("a crest on low ground is rock")
	}
	if isRock(rockHeight, -crestRock) {
		t.Error("high ground off its crests is rock")
	}
	if !isRock(rockHeight, 2*crestRock) {
		t.Error("a crest on high ground is not rock")
	}
}

// Ridges are seen by a close pixel only: none for a pixel wider than
// their tiles, some seen closely.
func TestRidgesNeedAClosePixel(t *testing.T) {
	tree := New(terran).tree()
	seen := 0
	for place := range pointsAsked {
		point := scatterOnSphere(place)
		wide := tree.gather(point, distantFootprint)
		if got := wide.detail[ridgeAttribute]; got != 0 {
			t.Fatalf("a wide pixel sees ridges of %v", got)
		}
		if tree.gather(point, nearFootprint).detail[ridgeAttribute] != 0 {
			seen++
		}
	}
	if seen == 0 {
		t.Error("ground seen closely has no ridges")
	}
}

// The small patches are not seen by a wide pixel: from orbit there are
// none, and close up there are.
func TestTextureIsOnlyForAClosePixel(t *testing.T) {
	tree := New(terran).tree()
	seen := 0
	for place := range pointsAsked {
		point := scatterOnSphere(place)
		wide := tree.gather(point, distantFootprint)
		if got := wide.detail[patchAttribute]; got != 0 {
			t.Fatalf("a wide pixel sees patches of %v", got)
		}
		if tree.gather(point, nearFootprint).detail[patchAttribute] != 0 {
			seen++
		}
	}
	if seen == 0 {
		t.Error("a close pixel sees no patches")
	}
}

// onLand is a here on land at the sea's edge or inland, in a tile by the
// sea or not, for the beach tests, seen by a pixel so narrow that what
// it leaves out is a thousandth of the widest strip.
func onLand(height, wet float64) here {
	return here{smooth: Ground{Wet: wet}, height: height, plain: 0.2,
		rain: 0.4, chill: 0.2, shoreMix: 1, residue: shoreMost / 1000}
}

// A beach is a strip: land at the sea's edge, by the sea, is beach; land
// higher than the strip is not, nor is land far from the sea.
func TestABeachIsOnlyAStrip(t *testing.T) {
	look := rampLook()
	ground := &scrooged{w: New(terran), look: look, mix: 0.35}
	beach := look.Paints[Beach].R
	isBeach := func(at here) bool {
		paint, _ := ground.shared(at)
		return math.Abs(paint.R-beach) < paintHair
	}
	edge := shoreMost / 8
	if !isBeach(onLand(edge, 0)) {
		t.Error("land at the edge is not beach")
	}
	if isBeach(onLand(10*shoreMost, 0)) {
		t.Error("land a few strips from the sea is beach")
	}
	if isBeach(onLand(edge, 5)) {
		t.Error("land at the edge of a lake far from the sea is beach")
	}
}

// Anti-test: a pixel wider than the strip does not paint it whole. At a
// shore seen by a pixel that leaves out a height ten strips tall, the
// strip is a small share of the paint: it does not widen with the pixel.
func TestAWidePixelHasOnlyItsShareOfTheStrip(t *testing.T) {
	look := rampLook()
	ground := &scrooged{w: New(terran), look: look, mix: 0.35}
	wide := onLand(0, 0)
	wide.residue = 10 * shoreMost
	land, _ := ground.shared(onLand(10*shoreMost, 0))
	sea := ground.water(wide, 0)
	got, share := ground.shared(wide)
	if math.Abs(share-0.5) > 1e-9 {
		t.Fatalf("a pixel on the sea's level is %v land, not half", share)
	}
	plain := sea.Scale(0.5).Add(land.Scale(0.5))
	strip := math.Abs(look.Paints[Beach].R - land.R)
	if math.Abs(got.R-plain.R) > stripShareMost*strip {
		t.Errorf("a wide pixel at the shore is %v, the sea and land "+
			"halved are %v", got.R, plain.R)
	}
}

// stripShareMost is the most of a wide pixel at a shore the strip may
// be: the strip's height is a tenth of what the pixel leaves out, and
// its share is under a tenth.
const stripShareMost = 0.1

// paintHair is how near two paints are to be one, as the average of the
// soft mixes rounds.
const paintHair = 1e-9

// Anti-test: plains are not salt and pepper. Walking a ring of the world
// as closely as a pixel, the paint goes from rock or snow to something
// else, or back, only a few times for every thousand steps.
func TestRockAndSnowAreNotSpeckled(t *testing.T) {
	world := New(terran)
	look := rampLook()
	fine := grounded(world)
	high := look.Paints[Rock].R - 1e-9
	turns, last := 0, false
	for step := range pathPoints {
		turn := 2 * math.Pi * float64(step) / pathPoints
		paint := fine.Within(onRing(turn), middleFootprint).R
		now := paint >= high && paint <= look.Paints[Snow].R+1e-9
		if step > 0 && now != last {
			turns++
		}
		last = now
	}
	if turns > pathPoints/200 {
		t.Errorf("rock or snow comes and goes %d times in %d steps", turns,
			pathPoints)
	}
}
