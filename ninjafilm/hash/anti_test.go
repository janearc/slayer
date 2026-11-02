package hash_test

import (
	"strconv"
	"testing"

	"github.com/janearc/slayer/ninjafilm/hash"
	"pgregory.net/rapid"
)

const (
	pair    = 2 // defined: a count of two
	lowBits = 8 // defined: the bits of a byte, the low ones of a hash
)

// anti-tests: no neighbour of a pinned site hashes to the number pinned
// for it. The table is of sites, not of numbers that any site may give.
func TestNoNeighbourOfAPinnedSiteHashesAsItDoes(t *testing.T) {
	for _, row := range pinned(t) {
		text, _ := row.Text("seed")
		seed, err := strconv.ParseUint(text, decimal, wordBits)
		if err != nil {
			t.Fatal(err)
		}
		place := wholes(t, row, "x", "y", "frame")
		want, _ := row.Float("want")
		next := int64(place["x"]) + int64(unity)
		if got := hash.Three(seed, next, int64(place["y"]),
			int64(place["frame"])); got == want {
			t.Errorf("%d at %v and its neighbour both hash to %v", seed,
				place, got)
		}
	}
}

// anti-tests: a hash is not one number for every site. Across a sample of sites
// it takes more than one value.
func TestSitesDoNotAllHashAlike(t *testing.T) {
	rapid.Check(t, sitesDoNotAllHashAlike)
}

func sitesDoNotAllHashAlike(t *rapid.T) {
	seed, x, y, frame := drawn(t)
	first := hash.Three(seed, x, y, frame)
	differed := false
	for step := range bins {
		now := hash.Three(seed, x+int64(step)+int64(unity), y, frame)
		differed = differed || now != first
	}
	if !differed {
		t.Fatalf("every site near %d,%d hashed to %v", x, y, first)
	}
}

// anti-tests: another seed is not the same hash. Two seeds that differ
// hash one site to two numbers.
func TestAnotherSeedIsNotTheSameHash(t *testing.T) {
	rapid.Check(t, anotherSeedHashesAnother)
}

func anotherSeedHashesAnother(t *rapid.T) {
	seed, x, y, frame := drawn(t)
	by := rapid.Int64Range(stepLeast, stepMost).Draw(t, "by")
	if hash.Three(seed, x, y, frame) == hash.Three(seed+uint64(by), x, y,
		frame) {
		t.Fatalf("seeds %d and %d hashed %d,%d alike", seed,
			seed+uint64(by), x, y)
	}
}

// anti-tests: a part moved by nothing is not moved. The hash stays where
// it was, whichever part it is.
func TestAPartMovedByNothingLeavesTheHash(t *testing.T) {
	rapid.Check(t, hashStaysWhenNothingMoves)
}

func hashStaysWhenNothingMoves(t *rapid.T) {
	seed, x, y, frame := drawn(t)
	was := hash.Three(seed, x, y, frame)
	still := []float64{
		hash.Three(seed+uint64(nothing), x, y, frame),
		hash.Three(seed, x+nothing, y, frame),
		hash.Three(seed, x, y+nothing, frame),
		hash.Three(seed, x, y, frame+nothing),
	}
	for part, got := range still {
		if got != was {
			t.Fatalf("part %d moved by nothing changed %v to %v", part, was,
				got)
		}
	}
}

// anti-tests: no other order of a site hashes as it does. Swapping the
// column with the frame is another site. So is swapping the row.
func TestNoOtherOrderOfASiteHashesAsItDoes(t *testing.T) {
	rapid.Check(t, noOtherOrderHashesAlike)
}

func noOtherOrderHashesAlike(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	x := rapid.Int64().Draw(t, "x")
	y := rapid.Int64().Draw(t, "y")
	frame := rapid.Int64().Filter(differsFrom(x)).Filter(differsFrom(y)).Draw(
		t, "frame")
	was := hash.Three(seed, x, y, frame)
	if was == hash.Three(seed, frame, y, x) || was == hash.Three(seed, x,
		frame, y) {
		t.Fatalf("%d,%d,%d hashed as another order of itself", x, y, frame)
	}
}

// anti-tests: a flip of no bit changes no bit. The avalanche is made by
// the flipped bit and by nothing else.
func TestNoBitFlippedChangesNoBit(t *testing.T) {
	rapid.Check(t, noFlipChangesNothing)
}

func noFlipChangesNothing(t *rapid.T) {
	seed, x, y, frame := drawn(t)
	was := hash.Three(seed, x, y, frame)
	for _, same := range flippedBy(uint64(nothing), seed, x, y, frame) {
		if same != was {
			t.Fatalf("a flip of no bit changed %v to %v", was, same)
		}
	}
}

// anti-tests: no bin of an even spread holds every site. Hashes that
// piled their range into one share would fail here before the chi
// squared.
func TestNoBinOfTheSpreadHoldsEverySite(t *testing.T) {
	rapid.Check(t, noBinHoldsEverySite)
}

func noBinHoldsEverySite(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	frame := rapid.Int64().Draw(t, "frame")
	filled := map[int]bool{}
	for y := range side {
		for x := range side {
			got := hash.Three(seed, int64(x), int64(y), frame)
			filled[int(got*float64(bins))] = true
		}
	}
	if len(filled) < pair {
		t.Fatalf("every site fell in one of %d bins", bins)
	}
}

// anti-tests: no neighbour of a site has the same whole hash. Moving the
// column, the row or the frame by one moves every part of the bits.
func TestNoNeighbourOfASiteHasTheSameBits(t *testing.T) {
	rapid.Check(t, neighboursHaveOtherBits)
}

func neighboursHaveOtherBits(t *rapid.T) {
	seed, x, y, frame := drawn(t)
	was := hash.Bits(seed, x, y, frame)
	next := int64(unity)
	for part, got := range []uint64{hash.Bits(seed, x+next, y, frame),
		hash.Bits(seed, x, y+next, frame), hash.Bits(seed, x, y,
			frame+next)} {
		if got == was {
			t.Fatalf("part %d moved by one left the bits at %d", part, was)
		}
	}
}

// anti-tests: the low bits are not left alone. Moving a site changes the
// bits that gradient noise takes its gradient from, not only the top.
func TestMovingASiteChangesTheLowBitsToo(t *testing.T) {
	rapid.Check(t, lowBitsMove)
}

func lowBitsMove(t *rapid.T) {
	seed, x, y, frame := drawn(t)
	by := rapid.Int64Range(stepLeast, stepMost).Draw(t, "by")
	changed := false
	for step := range wordBits / pair {
		now := hash.Bits(seed, x+by+int64(step), y, frame)
		changed = changed || now%(one<<lowBits) !=
			hash.Bits(seed, x, y, frame)%(one<<lowBits)
	}
	if !changed {
		t.Fatalf("the low bits stayed as they were across %d sites",
			wordBits/pair)
	}
}
