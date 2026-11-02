//ninja:vow dojo classify's edges, along and armsOf are biome's own and noise never lands on an edge, so they are held from inside
package biome

// the tests here are of what cannot be reached exactly from outside:
// classify's edges, which noise never lands on, the mixing of a sprawl's
// colours, and the ground a city's arms are measured on.

import (
	"encoding/csv"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"pgregory.net/rapid"
)

const (
	none        = 0    // defined: no amount
	one         = 1    // defined: one
	bits        = 64   // defined: the bits in the float64 a column is read as
	machineSize = 0    // defined: strconv's width for an int of the machine's size
	base        = 10   // defined: the base whole numbers are written in
	half        = 0.5  // defined: a half
	quarter     = 0.25 // defined: a quarter
	// turn is a whole turn in radians, defined
	turn = 2 * math.Pi
)

// insideNumbers is numbers.yaml, for the inside tests.
const insideNumbers = "testdata/numbers.yaml"

// asker is what a test and a property test both do.
type asker interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
}

// cases is a csv file of the inside tests, its rows by its header.
func cases(test asker, file string) []map[string]string {
	test.Helper()
	opened, err := os.Open(file)
	if err != nil {
		test.Fatal(err)
	}
	defer opened.Close()
	all, err := csv.NewReader(opened).ReadAll()
	if err != nil || len(all) == none {
		test.Fatalf("%s: %v, %d rows", file, err, len(all))
	}
	header, rows := all[none], all[one:]
	var named []map[string]string
	for _, row := range rows {
		each := map[string]string{}
		for column, name := range header {
			each[name] = row[column]
		}
		named = append(named, each)
	}
	return named
}

// number is a column of a row as a float64, or a stop.
func number(test asker, row map[string]string, column string) float64 {
	test.Helper()
	got, err := strconv.ParseFloat(row[column], bits)
	if err != nil {
		test.Fatalf("%s of %q: %v", column, row["name"], err)
	}
	return got
}

// chosenNumber is a number of numbers.yaml, by name.
func chosenNumber(test asker, name string) float64 {
	test.Helper()
	raw, err := os.ReadFile(insideNumbers)
	if err != nil {
		test.Fatal(err)
	}
	all, err := numbers.Read(insideNumbers, raw)
	if err != nil {
		test.Fatal(err)
	}
	got, err := all.Get(name)
	if err != nil {
		test.Fatal(err)
	}
	return got.Value
}

// classify holds each of its edges, from both sides: every row of
// classify.csv is painted as the row says.
func TestClassifyHoldsItsEdges(t *testing.T) {
	for _, row := range cases(t, "testdata/classify.csv") {
		want, err := strconv.ParseInt(row["paint"], base, machineSize)
		if err != nil {
			t.Fatalf("paint of %q: %v", row["name"], err)
		}
		height, wet := number(t, row, "height"), number(t, row, "wet")
		if got := classify(height, wet); got != int(want) {
			t.Errorf("%s: %v high and %v wet is paint %d, not %d",
				row["name"], height, wet, got, want)
		}
	}
}

// anti-tests: no height under the sea is land, and no height at or
// above it is sea, however wet.
func TestNoHeightIsOnTheWrongSideOfTheSea(t *testing.T) {
	rapid.Check(t, onTheRightSide)
}

func onTheRightSide(draw *rapid.T) {
	height := rapid.Float64Range(-one, one).Draw(draw, "height")
	wet := rapid.Float64Range(-one, one).Draw(draw, "wet")
	if (classify(height, wet) <= shallowSea) != (height < none) {
		draw.Fatalf("%v high and %v wet is paint %d", height, wet,
			classify(height, wet))
	}
}

// the three lights the mixing tests mix, each a primary
var (
	red   = maths.RGB(one, none, none)
	green = maths.RGB(none, one, none)
	blue  = maths.RGB(none, none, one)
)

// along runs from the first colour at zero to the last at one, through each
// colour at its own place, and halfway between two at the place halfway between
// theirs.
func TestAlongRunsFromTheFirstToTheLast(t *testing.T) {
	mixed := []maths.Light{red, green, blue}
	for share, want := range map[float64]maths.Light{
		none:    red,
		half:    green,
		one:     blue,
		quarter: maths.RGB(half, half, none),
	} {
		if got := along(mixed, share); got != want {
			t.Errorf("along %v is %v, not %v", share, got, want)
		}
	}
}

// anti-tests: along never mixes a colour outside its colours: each part
// stays between none and one, and the parts add to one.
func TestAlongNeverLeavesItsColours(t *testing.T) {
	rapid.Check(t, staysAmongItsColours)
}

func staysAmongItsColours(draw *rapid.T) {
	within := chosenNumber(draw, "within")
	share := rapid.Float64Range(none, one).Draw(draw, "share")
	got := along([]maths.Light{red, green, blue}, share)
	for _, part := range []float64{got.R, got.G, got.B} {
		if part < none || part > one {
			draw.Fatalf("along %v is %v", share, got)
		}
	}
	if math.Abs(got.R+got.G+got.B-one) > within {
		draw.Fatalf("along %v is %v, which adds to %v", share, got,
			got.R+got.G+got.B)
	}
}

// glowOf is what a city tile gives off when its sprawl lights it so
// brightly, with no lights given: the one tile of a flat world of one.
func glowOf(level float64) maths.Light {
	//ninja:vow stranger hexes makes no tiling; a Flat is a plain value
	tiling := hexes.Flat{Step: one, Across: one, Down: one}
	tile := tiling.Nearest(maths.V(none, none, none))
	world := &World{Fields: Fields{Tiling: tiling},
		lit: map[hexes.Tile]float64{tile: level}}
	return world.Shine().Paint(tile)
}

// a city's light is no brighter than its brightest: lit past it, it
// gives off what it does at it.
func TestACityLightIsCappedAtItsBrightest(t *testing.T) {
	brightest := chosenNumber(t, "brightest")
	if glowOf(brightest+one) != glowOf(brightest) {
		t.Errorf("lit %v it gives off %v, not %v as at %v", brightest+one,
			glowOf(brightest+one), glowOf(brightest), brightest)
	}
	if glowOf(brightest-quarter) == glowOf(brightest) {
		t.Errorf("lit %v it gives off what it does at %v",
			brightest-quarter, brightest)
	}
}

// anti-tests: a light too faint is dark: at the faintest it is seen, and
// at the float below it, not.
func TestALightTooFaintIsDark(t *testing.T) {
	faintest := chosenNumber(t, "faintest")
	unlit := maths.RGB(none, none, none)
	if glowOf(faintest) == unlit {
		t.Errorf("lit %v it is dark", faintest)
	}
	if below := math.Nextafter(faintest, none); glowOf(below) != unlit {
		t.Errorf("lit %v it gives off %v", below, glowOf(below))
	}
}

// sphere is a world on a sphere of tiles, for its arms.
func sphere(test asker) *World {
	test.Helper()
	//ninja:vow stranger hexes makes no tiling; a Sphere is a plain value
	tiling := hexes.Sphere{N: int(chosenNumber(test, "sphere-split"))}
	return &World{Fields: Fields{Tiling: tiling,
		Seed:   uint64(chosenNumber(test, "seed")),
		Sprawl: chosenNumber(test, "sprawl")}}
}

// level is whether a city's arms are measured on its ground: across and
// along each of length one, at right angles to each other and to the
// way out from the middle of the world.
func level(within float64, shape arms) bool {
	out := shape.heart.Unit()
	return math.Abs(shape.across.Len()-one) <= within &&
		math.Abs(shape.along.Len()-one) <= within &&
		math.Abs(shape.across.Dot(out)) <= within &&
		math.Abs(shape.along.Dot(out)) <= within &&
		math.Abs(shape.across.Dot(shape.along)) <= within
}

// a city at a pole, where up is the way out, has its arms measured on
// its ground as a city anywhere else does.
func TestArmsStandOnTheGroundAtThePoles(t *testing.T) {
	world, within := sphere(t), chosenNumber(t, "within")
	for _, way := range []maths.Vec{maths.V(none, one, none),
		maths.V(none, -one, none)} {
		shape := world.armsOf(world.Tiling.Nearest(way))
		if !level(within, shape) {
			t.Errorf("the arms of a city at %v are not on its ground: %+v",
				way, shape)
		}
	}
}

// anti-tests: off the poles too, where up is not the way out: at the
// ends of the other two axes, and between them.
func TestArmsStandOnTheGroundOffThePoles(t *testing.T) {
	world, within := sphere(t), chosenNumber(t, "within")
	for _, way := range []maths.Vec{maths.V(one, none, none),
		maths.V(none, none, one), maths.V(-one, none, -one)} {
		shape := world.armsOf(world.Tiling.Nearest(way))
		if !level(within, shape) {
			t.Errorf("the arms of a city at %v are not on its ground: %+v",
				way, shape)
		}
	}
}

// every city has its fewest to its most arms, a whole number of them,
// turned somewhere in a whole turn, spreading as far as the world's
// sprawl, from its tile's middle.
func TestEveryCityHasItsFewestToItsMostArms(t *testing.T) {
	world := sphere(t)
	fewest, most := chosenNumber(t, "fewest-arms"), chosenNumber(t, "most-arms")
	for _, tile := range world.Tiling.Tiles() {
		shape := world.armsOf(tile)
		if shape.count < fewest || shape.count > most ||
			shape.count != math.Trunc(shape.count) || shape.turn < none ||
			shape.turn >= turn || shape.spread != world.Sprawl ||
			shape.heart != world.Tiling.Centre(tile) {
			t.Errorf("a city at %v has arms %+v", tile, shape)
		}
	}
}

// anti-tests: cities are not all one shape: over the sphere's tiles their
// arms come in more than one count and more than one turn.
func TestCitiesAreNotAllOneShape(t *testing.T) {
	world := sphere(t)
	counts, turns := map[float64]bool{}, map[float64]bool{}
	for _, tile := range world.Tiling.Tiles() {
		shape := world.armsOf(tile)
		counts[shape.count], turns[shape.turn] = true, true
	}
	if len(counts) <= one || len(turns) <= one {
		t.Errorf("cities' arms come in %d counts and %d turns", len(counts),
			len(turns))
	}
}
