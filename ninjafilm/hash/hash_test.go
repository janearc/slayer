package hash_test

import (
	"math/bits"
	"os"
	"strconv"
	"testing"

	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/hash"
	"pgregory.net/rapid"
)

// what these tests count in, each true by definition
const (
	nothing      = 0         // defined: no amount at all
	unity        = 1         // defined: one, the top of the unit interval
	one          = uint64(1) // defined: the number one, as a word to shift
	wordBits     = 64        // defined: a uint64 is 64 bits wide
	fractionBits = 53        // defined: a float64 holds 53 bits of fraction
	decimal      = 10        // defined: the base ten numbers are written in
	fraction     = 1 << 53   // defined: 2 to the 53, how many numbers a hash's 53 bit fraction holds
)

// pinnedTable is the table of sites and what lamb's noise gives for them.
const pinnedTable = "testdata/pinned.csv"

// what these tests choose, each picked in picked-numbers.yml
var (
	stepLeast  = int64(data.Whole("hash-step-least"))
	stepMost   = int64(data.Whole("hash-step-most"))
	leastFlips = data.Picked("hash-avalanche-least")
	mostFlips  = data.Picked("hash-avalanche-most")
	bins       = data.Whole("hash-bins")
	side       = data.Whole("hash-side")
	chiLimit   = data.Picked("hash-chi-limit")
)

// drawn is a seed and a site of a frame, chosen by rapid.
func drawn(t *rapid.T) (seed uint64, x, y, frame int64) {
	seed = rapid.Uint64().Draw(t, "seed")
	x = rapid.Int64().Draw(t, "x")
	y = rapid.Int64().Draw(t, "y")
	frame = rapid.Int64().Draw(t, "frame")
	return seed, x, y, frame
}

// inUnit is whether a number is from zero up to one, zero in it and one out of
// it.
func inUnit(got float64) bool { return got >= nothing && got < unity }

// These are what lamb's noise gives for the sites in the table, and what
// the hash has always given. They are not right because they are these
// numbers. Changing a constant or a step of the mix cannot pass unseen.
func TestThreeHashesAsLambsNoiseDoes(t *testing.T) {
	for _, row := range pinned(t) {
		text, err := row.Text("seed")
		if err != nil {
			t.Fatal(err)
		}
		seed, err := strconv.ParseUint(text, decimal, wordBits)
		if err != nil {
			t.Fatal(err)
		}
		place := wholes(t, row, "x", "y", "frame")
		want, err := row.Float("want")
		if err != nil {
			t.Fatal(err)
		}
		got := hash.Three(seed, int64(place["x"]), int64(place["y"]),
			int64(place["frame"]))
		if got != want {
			t.Errorf("%d at %v hashed to %v, it always hashed to %v",
				seed, place, got, want)
		}
	}
}

// wholes are the whole numbers in the named columns of a row.
func wholes(t *testing.T, row data.Row, columns ...string) map[string]int {
	t.Helper()
	found := map[string]int{}
	for _, column := range columns {
		whole, err := row.Whole(column)
		if err != nil {
			t.Fatal(err)
		}
		found[column] = whole
	}
	return found
}

func TestThreeIsFromNoughtUpToOne(t *testing.T) {
	rapid.Check(t, hashIsInUnit)
}

func hashIsInUnit(t *rapid.T) {
	seed, x, y, frame := drawn(t)
	if got := hash.Three(seed, x, y, frame); !inUnit(got) {
		t.Fatalf("a hash of %v is outside nought up to one", got)
	}
}

func TestThreeIsTheSameForEver(t *testing.T) {
	rapid.Check(t, hashIsStable)
}

func hashIsStable(t *rapid.T) {
	seed, x, y, frame := drawn(t)
	first := hash.Three(seed, x, y, frame)
	if again := hash.Three(seed, x, y, frame); first != again {
		t.Fatalf("one site hashed to %v and then %v", first, again)
	}
}

// Every part of a site counts: move one and the hash is another. Two
// hashes agree by luck once in 2 to the 53, which a test cannot meet.
func TestEachPartOfASiteChangesTheHash(t *testing.T) {
	rapid.Check(t, hashFollowsEachPart)
}

func hashFollowsEachPart(t *rapid.T) {
	seed, x, y, frame := drawn(t)
	by := rapid.Int64Range(stepLeast, stepMost).Draw(t, "by")
	was := hash.Three(seed, x, y, frame)
	moved := map[string]float64{
		"seed":  hash.Three(seed+uint64(by), x, y, frame),
		"x":     hash.Three(seed, x+by, y, frame),
		"y":     hash.Three(seed, x, y+by, frame),
		"frame": hash.Three(seed, x, y, frame+by),
	}
	for part, got := range moved {
		if got == was {
			t.Fatalf("moving the %s by %d left the hash at %v", part,
				by, was)
		}
	}
}

func TestTheOrderOfASitesPlacesCounts(t *testing.T) {
	rapid.Check(t, hashFollowsOrder)
}

func hashFollowsOrder(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	x := rapid.Int64().Draw(t, "x")
	y := rapid.Int64().Filter(differsFrom(x)).Draw(t, "y")
	frame := rapid.Int64().Draw(t, "frame")
	if hash.Three(seed, x, y, frame) == hash.Three(seed, y, x, frame) {
		t.Fatalf("%d,%d and %d,%d hashed alike", x, y, y, x)
	}
}

// mirrored is a place that its mirror must differ from.
type mirrored int64

// differs is whether a number is another than the place.
func (each mirrored) differs(other int64) bool { return other != int64(each) }

// differsFrom is the test that a number is not the one given.
func differsFrom(x int64) func(int64) bool { return mirrored(x).differs }

// onto is the hash as the whole number it is a share of: the top 53 bits
// of a mixed 64.
func onto(got float64) uint64 { return uint64(got * fraction) }

// One bit of a site flipped changes about half the bits of its hash,
// whichever part of the site it is and whichever bit.
func TestOneBitFlippedChangesHalfTheHash(t *testing.T) {
	rapid.Check(t, hashAvalanches)
}

func hashAvalanches(t *rapid.T) {
	seed, x, y, frame := drawn(t)
	was := onto(hash.Three(seed, x, y, frame))
	var changed, flips float64
	for bit := range wordBits {
		mask := one << bit
		for _, flipped := range flippedBy(mask, seed, x, y, frame) {
			now := onto(flipped)
			changed += float64(bits.OnesCount64(was ^ now))
			flips++
		}
	}
	if mean := changed / flips; mean < leastFlips || mean > mostFlips {
		t.Fatalf("a flipped bit changed %v of 53 bits, want 26.5", mean)
	}
}

// flippedBy is the hashes of a site with a bit of each part flipped in
// turn.
func flippedBy(mask, seed uint64, x, y, frame int64) []float64 {
	signed := int64(mask)
	return []float64{
		hash.Three(seed^mask, x, y, frame),
		hash.Three(seed, x^signed, y, frame),
		hash.Three(seed, x, y^signed, frame),
		hash.Three(seed, x, y, frame^signed),
	}
}

// Sites in a square hash evenly: equal bins hold about an equal share
// each. Pearson's chi squared for seven degrees of freedom passes 40
// about once in ten million.
func TestHashesFallEvenly(t *testing.T) {
	rapid.Check(t, hashesAreEven)
}

func hashesAreEven(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	frame := rapid.Int64().Draw(t, "frame")
	counts := make([]float64, bins)
	for y := range side {
		for x := range side {
			got := hash.Three(seed, int64(x), int64(y), frame)
			counts[int(got*float64(bins))]++
		}
	}
	expected, chi := float64(side*side)/float64(bins), float64(nothing)
	for _, count := range counts {
		chi += (count - expected) * (count - expected) / expected
	}
	if chi > chiLimit {
		t.Fatalf("the bins hold %v: chi squared %v", counts, chi)
	}
}

// pinned is the rows of the table of sites and what they hash to.
func pinned(t *testing.T) []data.Row {
	t.Helper()
	text, err := os.ReadFile(pinnedTable)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := data.Parse(string(text))
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// Three is Bits with its top 53 bits made a fraction, and Bits holds all
// 64 bits for a site.
func TestThreeIsTheTopOfBits(t *testing.T) {
	rapid.Check(t, threeIsTheTopOfBits)
}

func threeIsTheTopOfBits(t *rapid.T) {
	seed, x, y, frame := drawn(t)
	whole := hash.Bits(seed, x, y, frame)
	want := float64(whole>>(wordBits-fractionBits)) / fraction
	if got := hash.Three(seed, x, y, frame); got != want {
		t.Fatalf("Three is %v and the top of Bits is %v", got, want)
	}
}

// Bits move with every part of a site. That holds for the low bits as
// well as the high. Gradients are taken from the low ones.
func TestEachPartOfASiteChangesTheBits(t *testing.T) {
	rapid.Check(t, bitsFollowEachPart)
}

func bitsFollowEachPart(t *rapid.T) {
	seed, x, y, frame := drawn(t)
	by := rapid.Int64Range(stepLeast, stepMost).Draw(t, "by")
	was := hash.Bits(seed, x, y, frame)
	moved := map[string]uint64{
		"seed":  hash.Bits(seed+uint64(by), x, y, frame),
		"x":     hash.Bits(seed, x+by, y, frame),
		"y":     hash.Bits(seed, x, y+by, frame),
		"frame": hash.Bits(seed, x, y, frame+by),
	}
	for part, got := range moved {
		if got == was {
			t.Fatalf("moving the %s by %d left the bits at %v", part, by,
				was)
		}
	}
}
