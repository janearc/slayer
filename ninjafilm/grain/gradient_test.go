package grain_test

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/grain"
	"pgregory.net/rapid"
)

// what these tests count in, each defined
const (
	none     = 0  // defined: no amount at all
	whole    = 1  // defined: one
	decimal  = 10 // defined: the base ten numbers are written in
	sixteen  = 16 // defined: the base hexadecimal is written in
	wordBits = 64 // defined: a uint64 is 64 bits wide
)

// the tables the tests are run over
const (
	pinnedFile = "testdata/pinned.csv"
	digestFile = "testdata/digest.csv"
)

// what these tests choose, each picked in picked-numbers.yml
var (
	bound     = data.Picked("grain-bound")
	slopeMost = data.Picked("grain-slope-most")
	step      = data.Picked("grain-step")
	farthest  = data.Picked("grain-coordinate-most")
	sample    = data.Whole("grain-sample")
)

// rows are the rows of a table in testdata, or the test stops.
func rows(t *testing.T, name string) []data.Row {
	t.Helper()
	text, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	found, err := data.Parse(string(text))
	if err != nil {
		t.Fatal(err)
	}
	return found
}

// amount is the number in a column of a row, or the test stops.
func amount(t *testing.T, row data.Row, column string) float64 {
	t.Helper()
	found, err := row.Float(column)
	if err != nil {
		t.Fatal(err)
	}
	return found
}

// seedOf is the seed in the named column of a row, or the test stops.
func seedOf(t *testing.T, row data.Row, column string) uint64 {
	t.Helper()
	text, err := row.Text(column)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := strconv.ParseUint(text, decimal, wordBits)
	if err != nil {
		t.Fatal(err)
	}
	return seed
}

// These are what the audited noise gives for the points in the table.
// They are not right because they are these numbers. Changing a step of
// the noise or one of the twelve gradients cannot pass unseen.
func TestGradientIsTheNoiseItAlwaysWas(t *testing.T) {
	for _, row := range rows(t, pinnedFile) {
		seed := seedOf(t, row, "seed")
		x, y, z := amount(t, row, "x"), amount(t, row, "y"),
			amount(t, row, "z")
		want := amount(t, row, "want")
		if got := grain.Gradient(seed, x, y, z); got != want {
			t.Errorf("seed %d at %v, %v, %v is %v, it always was %v", seed,
				x, y, z, got, want)
		}
	}
}

// The field over a grid of points is the one the audited noise made.
// The table holds the digest of its values, taken as bits.
func TestGradientOverAGridIsTheFieldItAlwaysWas(t *testing.T) {
	for _, row := range rows(t, digestFile) {
		sum := fnv.New64a()
		seeds, across, down := whole*int(amount(t, row, "seeds")),
			int(amount(t, row, "across")), int(amount(t, row, "down"))
		for seed := range seeds {
			for i := range across {
				for column := range down {
					x := float64(i)*amount(t, row, "x_step") +
						amount(t, row, "x_start")
					y := float64(column)*amount(t, row, "y_step") +
						amount(t, row, "y_start")
					z := float64(i^column) * amount(t, row, "z_step")
					got := grain.Gradient(uint64(seed), x, y, z)
					bits := binary.LittleEndian.AppendUint64(nil,
						math.Float64bits(got))
					sum.Write(bits)
				}
			}
		}
		text, _ := row.Text("digest")
		want, err := strconv.ParseUint(text, sixteen, wordBits)
		if err != nil {
			t.Fatal(err)
		}
		if got := sum.Sum64(); got != want {
			t.Errorf("the field's digest is %x, it always was %x", got, want)
		}
	}
}

// point draws a point, each coordinate within farthest cells of zero.
func point(t *rapid.T) (x, y, z float64) {
	far := rapid.Float64Range(-farthest, farthest)
	return far.Draw(t, "x"), far.Draw(t, "y"), far.Draw(t, "z")
}

// Gradient noise is zero on every integer lattice point.
func TestGradientIsNoughtOnTheLattice(t *testing.T) {
	rapid.Check(t, gradientIsNoughtOnTheLattice)
}

func gradientIsNoughtOnTheLattice(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	lattice := rapid.Int64Range(-int64(farthest), int64(farthest))
	x, y, z := lattice.Draw(t, "x"), lattice.Draw(t, "y"),
		lattice.Draw(t, "z")
	if got := grain.Gradient(seed, float64(x), float64(y),
		float64(z)); got != none {
		t.Fatalf("the noise at %d, %d, %d is %v", x, y, z, got)
	}
}

// Gradient noise is seldom past one, and never as far past as the bound.
func TestGradientIsSeldomPastOne(t *testing.T) {
	rapid.Check(t, gradientStaysInBounds)
}

func gradientStaysInBounds(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	x, y, z := point(t)
	if got := grain.Gradient(seed, x, y, z); math.Abs(got) > bound {
		t.Fatalf("the noise at %v, %v, %v is %v", x, y, z, got)
	}
}

// Gradient noise is smooth: a small step moves it by a small amount.
func TestGradientIsSmooth(t *testing.T) {
	rapid.Check(t, gradientIsSmooth)
}

func gradientIsSmooth(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	x, y, z := point(t)
	here := grain.Gradient(seed, x, y, z)
	over := grain.Gradient(seed, x+step, y+step, z+step)
	if moved := math.Abs(over - here); moved > slopeMost*step {
		t.Fatalf("the noise moved %v for a step of %v at %v, %v, %v", moved,
			step, x, y, z)
	}
}

// The same seed and point give the same noise. Another seed gives another
// field. Two seeds may agree at a point, as twelve gradients are few.
// They do not agree across a sample.
func TestGradientIsTheSameForASeedAndAnotherForAnother(t *testing.T) {
	rapid.Check(t, gradientFollowsItsSeed)
}

func gradientFollowsItsSeed(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	by := rapid.Uint64Range(whole, math.MaxUint64).Draw(t, "by")
	differed := false
	for range sample {
		x, y, z := point(t)
		first := grain.Gradient(seed, x, y, z)
		if again := grain.Gradient(seed, x, y, z); again != first {
			t.Fatalf("one point gave %v and then %v", first, again)
		}
		other := grain.Gradient(seed+by, x, y, z)
		differed = differed || other != first
	}
	if !differed {
		t.Fatalf("seeds %d and %d gave one field", seed, seed+by)
	}
}

// anti-tests: a point with no number in it has no noise.
func TestGradientOfNoNumberIsNoNumber(t *testing.T) {
	nan := math.NaN()
	if got := grain.Gradient(none, nan, whole, whole); !math.IsNaN(got) {
		t.Errorf("the noise at no number is %v", got)
	}
}
