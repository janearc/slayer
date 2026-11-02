package colour_test

import (
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/colour"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/frame"
	"pgregory.net/rapid"
)

// the tables the bloom tests are run over
const (
	weightsFile = "testdata/bloom-weights.csv"
	starsFile   = "testdata/bloom-stars.csv"
	widthsFile  = "testdata/bloom-widths.csv"
)

// what the bloom tests choose, each picked in picked-numbers.yml
var (
	baseLight   = data.Picked("colour-base")
	rowsInATest = data.Whole("colour-rows")
	bloomRadius = data.Picked("colour-bloom-radius-most")
	bloomSize   = data.Whole("colour-bloom-picture")
	dimStep     = data.Picked("colour-step")
	dimLight    = data.Picked("colour-dim")
	starLight   = data.Picked("colour-star")
)

// weight is the weight of a gaussian of sigma, a number of steps from its
// middle. The table holds the weights worked by hand. Past its reach the weight
// is zero.
func weight(t *testing.T, sigma float64, steps int) float64 {
	t.Helper()
	for _, row := range rows(t, weightsFile) {
		if amount(t, row, "sigma") == sigma &&
			int(amount(t, row, "offset")) == max(steps, -steps) {
			return amount(t, row, "weight")
		}
	}
	return none
}

// sigmas are the two widths the table of weights holds.
func sigmas(t *testing.T) []float64 {
	t.Helper()
	var found []float64
	for _, row := range rows(t, weightsFile) {
		sigma := amount(t, row, "sigma")
		if !slicesContains(found, sigma) {
			found = append(found, sigma)
		}
	}
	return found
}

// slicesContains is whether a list holds a number.
func slicesContains(list []float64, number float64) bool {
	for _, each := range list {
		if each == number {
			return true
		}
	}
	return false
}

// Blooms are a third of the light beyond white spread by the gaussian
// across and again down. They are added back at the scale of the
// exposure. In a row of one pixel's height the pass down finds only the
// middle. So what a pixel steps away gets is the spill times the weight
// of the steps times the weight of none.
//
// The stars sit at both ends. The first and last columns are both a
// source of light and a place it falls. Each spill is worked in the
// exposed amount of the light. That is a third of what is past white of
// grey. It is divided by the exposure to be back in the light given.
func TestABloomIsAThirdOfWhatIsBeyondWhiteSpreadByAGaussian(t *testing.T) {
	for _, star := range rows(t, starsFile) {
		for _, sigma := range sigmas(t) {
			lights := make([]maths.Light, rowsInATest)
			for place := range lights {
				lights[place] = maths.Grey(baseLight)
			}
			last := rowsInATest - whole
			lights[none], lights[last] = maths.Grey(amount(t, star, "star")),
				maths.Grey(amount(t, star, "star"))
			for _, shape := range []string{"row", "column"} {
				across, down := rowsInATest, whole
				if shape == "column" {
					across, down = whole, rowsInATest
				}
				got := colour.Bloom(lights, across, down, amount(t, star,
					"exposure"), sigma)
				for place := range got {
					spill := amount(t, star, "spill") * (weight(t, sigma,
						place) + weight(t, sigma, last-place)) *
						weight(t, sigma, none)
					want := lights[place].Add(maths.Grey(spill))
					if !near(got[place], want) {
						t.Errorf("%s, sigma %v, %s, light %d: %v, want %v",
							text(t, star, "label"), sigma, shape, place,
							got[place], want)
					}
				}
			}
		}
	}
}

// Stars away from the edges of a picture wider and taller than their
// reach spill by the weight across times the weight down. That goes to
// every pixel in reach. What falls off the edge is lost and not given
// back. Every pixel starts different. So a light read from the wrong
// place shows.
func TestABloomSpillsAcrossAndDown(t *testing.T) {
	across, down := data.Whole("colour-spill-across"), data.Whole("colour-spill-down")
	column, row := data.Whole("colour-star-column"), data.Whole("colour-star-row")
	lights := make([]maths.Light, across*down)
	for place := range lights {
		lights[place] = maths.Grey(dimStep * float64(whole+place))
	}
	lights[row*across+column] = maths.Grey(starLight)
	got := colour.Bloom(lights, across, down, whole, whole)
	for y := range down {
		for x := range across {
			spill := weight(t, whole, x-column) * weight(t, whole, y-row)
			want := lights[y*across+x].Add(maths.Grey(spill))
			if !near(got[y*across+x], want) {
				t.Errorf("(%d, %d) is %v, want %v", x, y, got[y*across+x],
					want)
			}
		}
	}
}

// bloomOfAStar is a bloom of a single star of a radius, for a call that
// may panic.
func bloomOfAStar(radius float64) {
	star := []maths.Light{maths.Grey(starLight)}
	colour.Bloom(star, whole, whole, whole, radius)
}

// anti-tests: blooms wider than any picture are refused by name. So are
// endless ones. The widest allowed is allowed.
func TestABloomWiderThanAnyPictureIsRefused(t *testing.T) {
	widest := data.Picked("bloom-widest")
	if said := refusal(bloomOfAStar, widest); said != "" {
		t.Errorf("a bloom of %v pixels was refused: %q", widest, said)
	}
	for _, radius := range []float64{math.Inf(whole), math.NaN(),
		widest * pair} {
		if said := refusal(bloomOfAStar, radius); !strings.Contains(said,
			"bloom") {
			t.Errorf("a bloom of %v: %q", radius, said)
		}
	}
}

// Stars under the widest bloom keep their own light. They gain a sliver
// of their spill. Kernels hundreds of thousands of pixels across leave
// that at their middle.
func TestAStarUnderTheWidestBloomGainsASliver(t *testing.T) {
	star := []maths.Light{maths.Grey(starLight)}
	got := colour.Bloom(star, whole, whole, whole, data.Picked(
		"bloom-widest"))
	if lit := got[none].R; !(lit > starLight && lit < starLight+dimStep) {
		t.Errorf("a star under the widest bloom is %v", got[none])
	}
}

// anti-tests: pictures with nothing beyond white are returned as they
// were, before the radius is looked at. White itself is not beyond white, and an
// endless radius is no business of a picture that does not bloom.
func TestAPictureWithNothingBeyondWhiteIsLeftWhateverTheRadius(t *testing.T) {
	lights := []maths.Light{maths.Grey(whole), maths.Grey(half),
		maths.Grey(none)}
	got := colour.Bloom(lights, len(lights), whole, whole, math.Inf(whole))
	for place := range lights {
		if got[place] != lights[place] {
			t.Errorf("light %d is %v, was %v", place, got[place],
				lights[place])
		}
	}
}

// Blooms so fine that their radius squared is zero leave a star as it was. The
// middle is weighed one. All of the spill is kept at it.
func TestTheFinestBloomKeepsAllTheSpillAtTheMiddle(t *testing.T) {
	lights := []maths.Light{maths.Grey(none), maths.Grey(starLight),
		maths.Grey(none)}
	got := colour.Bloom(lights, len(lights), whole, whole, math.SmallestNonzeroFloat64)
	spill := (starLight - whole) / data.Picked("bloom-spill-divisor")
	if want := maths.Grey(starLight + spill); !near(got[whole], want) {
		t.Errorf("a star under the finest bloom is %v", got)
	}
	if got[none] != (maths.Light{}) {
		t.Errorf("the pixel beside a star under the finest bloom is %v", got)
	}
}

// Blooms only add light. They never add more than what spilled. Light
// that falls off an edge is lost. Stars whose reach is inside the
// picture give back all of it.
func TestABloomAddsOnlyWhatSpilled(t *testing.T) {
	rapid.Check(t, bloomAddsOnlyWhatSpilled)
}

func bloomAddsOnlyWhatSpilled(t *rapid.T) {
	radius := rapid.Float64Range(radiusLeast, bloomRadius).Draw(t, "radius")
	exposure := rapid.Float64Range(half, exposureMost).Draw(t, "exposure")
	light := rapid.Float64Range(whole+whole, lightMost).Draw(t, "star")
	size, middle := bloomSize, bloomSize/pair
	lights := make([]maths.Light, size*size)
	lights[middle*size+middle] = maths.Grey(light)
	got := colour.Bloom(lights, size, size, exposure, radius)
	var added float64
	for place := range got {
		added += got[place].R - lights[place].R
	}
	exposed := light * exposure
	spilled := (exposed - whole) / data.Picked("bloom-spill-divisor") / exposure
	if math.Abs(added-spilled) > within*lightMost {
		t.Fatalf("a star of %v at exposure %v added %v, spilled %v", light,
			exposure, added, spilled)
	}
}

// bloomThrough is a picture of a row of three pixels through a tone. It
// has a star between two dim ones.
func bloomThrough(tone colour.Tone) (star, beside color.RGBA) {
	made := frame.New(pair+whole, whole)
	for x, light := range []float64{dimLight, starLight, dimLight} {
		made.Set(x, none, greyPixel(light))
	}
	img := tone.Develop(made)
	return img.RGBAAt(whole, none), img.RGBAAt(none, none)
}

// anti-tests: blooms of zero or less are no bloom at all. Stars far past white
// through a knee are as bright as the knee makes them. Spilling onto themselves
// makes them no brighter.
func TestABloomOfNoughtOrLessAddsNothing(t *testing.T) {
	rolled := colour.NewTone(none, half, none, none).Squeeze(maths.Grey(
		starLight)).R
	for _, bloom := range []float64{none, -whole} {
		star, beside := bloomThrough(colour.NewTone(none, half, none, bloom))
		want := colour.Encode(rolled)
		dim := colour.Encode(dimLight)
		if star.R != want || beside.R != dim {
			t.Errorf("a bloom of %v: the star is %v and its neighbour %v, want %d and %d",
				bloom, star, beside, want, dim)
		}
	}
}

// Blooms of half a pixel are blooms. Stars at four times white light the
// pixel beside them. That is by the weight of one step times the weight
// of none of that width. It lies on top of the light they had.
func TestASmallBloomStillBlooms(t *testing.T) {
	_, beside := bloomThrough(colour.NewTone(none, none, none, half))
	spill := weight(t, half, whole) * weight(t, half, none)
	lit := colour.Encode(dimLight + spill)
	if beside.R != lit || lit == colour.Encode(dimLight) {
		t.Errorf("the pixel beside a star under a bloom of half is %v, want %d",
			beside, lit)
	}
}
