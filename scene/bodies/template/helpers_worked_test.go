package template

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/world"
)

// helper, worked by hand: hours turns a number of hours into a Duration. an
// hour is 3600 s, so 1 is time.Hour, a half is thirty minutes, 24 is a day, a
// quarter is fifteen minutes, zero is nothing and a negative is a backward
// span.
//
// each is exact, since these are all dyadic fractions of an hour.
func TestHoursIsHoursAsADuration(t *testing.T) {
	for _, c := range []struct {
		in   float64
		want time.Duration
	}{
		{0, 0},
		{1, time.Hour},
		{0.5, 30 * time.Minute},
		{0.25, 15 * time.Minute},
		{24, 24 * time.Hour},
		{1.5, 90 * time.Minute},
		{-2, -2 * time.Hour},
	} {
		if got := hours(c.in); got != c.want {
			t.Errorf("hours(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

// helper, worked by hand: deg turns degrees into radians, a half turn
// being pi. 180 is pi, 90 pi/2, 60 pi/3, 1 is pi/180 and -45 is -pi/4.
func TestDegIsDegreesAsRadians(t *testing.T) {
	for _, c := range []struct{ in, want float64 }{
		{0, 0},
		{180, math.Pi},
		{90, math.Pi / 2},
		{60, math.Pi / 3},
		{1, math.Pi / 180},
		{-45, -math.Pi / 4},
		{360, 2 * math.Pi},
	} {
		if got := deg(c.in); math.Abs(got-c.want) > 1e-15 {
			t.Errorf("deg(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

// distinctPalette is a palette whose thirteen paints are told apart by
// their red: the k-th biome, from deep at 0, is k+1 twentieths of red
// and no green or blue. a paint put on the wrong biome shows.
func distinctPalette() swatches.Palette {
	p := func(k float64) swatch.Swatch { return srgb.FromLight((k+1)/20, 0, 0) }
	return swatches.Palette{
		Deep: p(0), Sea: p(1), Ice: p(2), Beach: p(3), Desert: p(4),
		Grass: p(5), Forest: p(6), Jungle: p(7), Tundra: p(8), Rock: p(9),
		Snow: p(10), Lava: p(11), Ash: p(12),
	}
}

// redOf is the light a biome of the distinct palette has: k+1 twentieths
// of red.
func redOf(b world.Biome) maths.Light { return maths.RGB((float64(b)+1)/20, 0, 0) }

// nearLights holds two lights to a tolerance, band by band.
func nearLights(a, b maths.Light) bool {
	return math.Abs(a.R-b.R) < 1e-9 && math.Abs(a.G-b.G) < 1e-9 &&
		math.Abs(a.B-b.B) < 1e-9
}

// helper, worked by hand: palette gives each biome its own paint as a
// light. the distinct palette's biome k is (k+1)/20 of red, so the
// light at biome k must be that, for all thirteen.
func TestPaletteGivesEachBiomeItsOwnPaint(t *testing.T) {
	got := palette(distinctPalette())
	for b := world.Biome(0); b < world.Biomes; b++ {
		if !nearLights(got[b], redOf(b)) {
			t.Errorf("biome %d is painted %v, want %v", b, got[b], redOf(b))
		}
	}
}

// helper, worked by hand: twoTone paints a ground in a pair's two
// lights, the pale in biomes 0, 2, 4, 6, 8, 10 and 12 (deep, ice,
// desert, forest, tundra, snow, ash) and the dark in the odd ones
// (sea, beach, grass, jungle, rock, lava).
//
// the pale is a twentieth of red and the dark a half, so each biome's
// light says which it took.
func TestTwoToneAlternatesPaleAndDarkDownTheBiomes(t *testing.T) {
	pale, dark := srgb.FromLight(0.05, 0, 0), srgb.FromLight(0.5, 0, 0)
	got := twoTone(swatches.Pair{Pale: pale, Dark: dark})
	for b := world.Biome(0); b < world.Biomes; b++ {
		want := maths.RGB(0.05, 0, 0)
		if b%2 == 1 {
			want = maths.RGB(0.5, 0, 0)
		}
		if !nearLights(got[b], want) {
			t.Errorf("biome %d is painted %v, want %v", b, got[b], want)
		}
	}
}

// helper, worked by hand: look keeps the paints it is given and the
// lines' darkness, 0.25 here, and a second look keeps its own.
func TestLookKeepsItsPaintsAndItsLines(t *testing.T) {
	paints := palette(distinctPalette())
	got := look(0.25, paints)
	if got.Lines != 0.25 {
		t.Errorf("lines are %v, want 0.25", got.Lines)
	}
	if got.Paints != paints {
		t.Errorf("paints are %v, want %v", got.Paints, paints)
	}
	if other := look(0, paints); other.Lines != 0 {
		t.Errorf("a look with no lines has lines of %v", other.Lines)
	}
}

// helper, worked by hand: roughness paints each class a grey of its
// roughness, so a class of 0.25 is the light (0.25, 0.25, 0.25), and
// each call is a look of its own to change.
func TestRoughnessPaintsEachClassAGreyOfItsRoughness(t *testing.T) {
	var by [world.Biomes]float64
	for b := range by {
		by[b] = float64(b) / 16
	}
	got := roughness(by)
	for b := range by {
		if want := maths.Grey(float64(b) / 16); got.Paints[b] != want {
			t.Errorf("class %d is %v, want %v", b, got.Paints[b], want)
		}
	}
	if got.Lines != 0 {
		t.Errorf("a radar look has lines of %v, want none", got.Lines)
	}
	got.Paints[world.Rock] = maths.Grey(9)
	if again := roughness(by); again.Paints[world.Rock] != maths.Grey(float64(world.Rock)/16) {
		t.Error("a change to one roughness look reached the next")
	}
	if blank := roughness([world.Biomes]float64{}); blank.Paints[world.Sea] != maths.Grey(0) {
		t.Errorf("a roughness of nought is %v, want black", blank.Paints[world.Sea])
	}
}

// helper, worked by hand: bands makes a banded texture of the lights it
// is given, in order, and the numbers it is given, and the detail the
// giants share: a shear of 7 and a grain of 0.08.
func TestBandsIsTheLightsTheNumbersAndTheSharedDetail(t *testing.T) {
	a, b := maths.RGB(1, 0, 0), maths.RGB(0, 0, 1)
	got := bands(0.1, 0.2, 0.3, 4, a, b)
	if len(got.Paints) != 2 || got.Paints[0] != a || got.Paints[1] != b {
		t.Errorf("paints are %v, want %v then %v", got.Paints, a, b)
	}
	for _, c := range []struct {
		what      string
		got, want float64
	}{
		{"roil", got.Roil, 0.1}, {"fine", got.Fine, 0.2},
		{"mottle", got.Mottle, 0.3}, {"shear", got.Shear, 7},
		{"grain", got.Grain, 0.08},
	} {
		if c.got != c.want {
			t.Errorf("%s is %v, want %v", c.what, c.got, c.want)
		}
	}
	if got.Jets != 4 {
		t.Errorf("jets are %d, want 4", got.Jets)
	}
	if got.Seed != 0 || got.Spots != nil || got.Streets != nil {
		t.Errorf("seed, spots and streets are %d, %v, %v; the caller sets those",
			got.Seed, got.Spots, got.Streets)
	}
	if none := bands(0, 0, 0, 0); len(none.Paints) != 0 || none.Shear != 7 {
		t.Errorf("bands with no lights has %d paints and a shear of %v",
			len(none.Paints), none.Shear)
	}
}

// helper, worked by hand: the earth's look is its terran palette with
// lines of 0.4, and no other.
func TestTerranIsTheEarthsPaintsWithLinesOfFourTenths(t *testing.T) {
	got := terran()
	if got.Lines != 0.4 {
		t.Errorf("lines are %v, want 0.4", got.Lines)
	}
	want := palette(swatches.Earth)
	if got.Paints != want {
		t.Errorf("paints are %v, want the earth palette's %v", got.Paints, want)
	}
}
