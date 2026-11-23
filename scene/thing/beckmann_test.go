package thing

import (
	"math"
	"testing"
)

// The numbers the Beckmann tests count in, each true by definition.
const beckmannHalf = 0.5 // defined: half, the middle of a cell

// The csv tables the Beckmann tests read.
const (
	beckmannRefusalsFile = "testdata/beckmann-refusals.csv"
	beckmannFacingsFile  = "testdata/beckmann-facings.csv"
	beckmannTailsFile    = "testdata/beckmann-tails.csv"
)

// beckmannSlopes names the mean-square slopes the normalizing test tries:
// narrow, Cox and Munk's sea at 7 m/s, and broad.
var beckmannSlopes = []string{"beckmann-narrow-slopes", "beckmann-sea-slopes",
	"beckmann-broad-slopes"}

// normalized is the integral over the hemisphere of a density of tiny
// faces times their cosine, by the midpoint rule in the tilt; a density
// of faces sums to one this way. The density is asked for a facing and
// the mean-square slope.
func normalized(t *testing.T, density func(facing, slopes float64) float64,
	slopes float64) float64 {
	t.Helper()
	rings := pickedCount(t, "beckmann-rings")
	var sum float64
	for i := range rings {
		tilt := (float64(i) + beckmannHalf) / float64(rings) * math.Pi / pair
		facing := math.Cos(tilt)
		sum += density(facing, slopes) * facing * math.Sin(tilt)
	}
	return sum * math.Pi / pair / float64(rings) * pair * math.Pi
}

// beckmannOneWay is a density of faces built on the variance one way
// rather than both ways together: the slip the normalizing anti-test
// holds the sum to.
func beckmannOneWay(facing, oneWay float64) float64 {
	square := facing * facing
	tangent := (unity - square) / square
	return math.Exp(-tangent/(pair*oneWay)) /
		(math.Pi * oneWay * square * square)
}

// Beckmann's density of faces, times their cosine, sums to one over the
// hemisphere, narrow or broad.
func TestBeckmannFacesNormalize(t *testing.T) {
	tolerance := picked(t, "beckmann-tolerance")
	for _, name := range beckmannSlopes {
		slopes := picked(t, name)
		got := normalized(t, beckmannFaces, slopes)
		if math.Abs(got-unity) > tolerance {
			t.Errorf("mean-square slope %v sums to %v, not one", slopes, got)
		}
	}
}

// Anti-test: the sum sees a density built on the variance one way
// rather than both ways together, the slip Cox and Munk's two
// variances invite: it sums to two, not one.
func TestBeckmannNormalizingSeesOneWayVariance(t *testing.T) {
	slopes := picked(t, "beckmann-sea-slopes")
	oneWay := slopes / pair
	got := normalized(t, beckmannOneWay, oneWay)
	if math.Abs(got-pair) > picked(t, "beckmann-tolerance") {
		t.Errorf("a one-way density sums to %v, want two", got)
	}
}

// Smith's share for Beckmann at a = 1, where Lambda is (exp(-1) /
// sqrt(pi) - erfc(1)) / 2 = 0.0251273, and all of it straight down.
func TestBeckmannSmithAtAKnownSlant(t *testing.T) {
	slopes := picked(t, "beckmann-sea-slopes")
	tolerance := picked(t, "beckmann-tolerance")
	// a = cot / sqrt(slopes) = 1: cot = sqrt(slopes)
	cos := math.Sqrt(slopes) / math.Sqrt(unity+slopes)
	denominator := picked(t, "beckmann-smith-denominator")
	got, want := beckmannSmith(cos, slopes), unity/denominator
	if math.Abs(got-want) > tolerance {
		t.Errorf("share at a = 1 is %v, want %v", got, want)
	}
	if got := beckmannSmith(unity, slopes); got != unity {
		t.Errorf("share straight down is %v, want one", got)
	}
	grazing := beckmannSmith(picked(t, "beckmann-grazing-cosine"), slopes)
	oblique := beckmannSmith(picked(t, "beckmann-oblique-cosine"), slopes)
	if !(grazing < oblique) {
		t.Error("a more slant way sees no fewer faces")
	}
}

// Anti-test: what cannot glint gives nothing, and a glint is never
// negative or not a number.
func TestBeckmannRefusesWhatCannotGlint(t *testing.T) {
	for _, each := range readSheet(t, beckmannRefusalsFile) {
		facing, lit := cell(t, each, "facing"), cell(t, each, "lit")
		viewed, slopes := cell(t, each, "viewed"), cell(t, each, "slopes")
		if got := Beckmann(facing, lit, viewed, slopes); got != nought {
			arguments := []float64{facing, lit, viewed, slopes}
			t.Errorf("Beckmann%v = %v, want nothing", arguments, got)
		}
	}
	beckmannNeverNegative(t)
}

// beckmannNeverNegative holds that the glint at each facing of a table is
// a number, and never below nought.
func beckmannNeverNegative(t *testing.T) {
	t.Helper()
	glancing := picked(t, "beckmann-glancing-cosine")
	slopes := picked(t, "beckmann-sea-slopes")
	for _, each := range readSheet(t, beckmannFacingsFile) {
		facing := cell(t, each, "facing")
		got := Beckmann(facing, glancing, glancing, slopes)
		if !(got >= nought) {
			t.Errorf("facing %v gives %v", facing, got)
		}
	}
}

// beckmannRatio is GGX over Beckmann, at equal peak density, at a count
// of widths of the tangent from the peak, for a mean-square slope.
func beckmannRatio(slopes, tangentWidths float64) float64 {
	alpha := math.Sqrt(slopes)
	tangent := tangentWidths * math.Sqrt(slopes)
	facing := unity / math.Sqrt(unity+tangent*tangent)
	// with lit and viewed one, both shadowings are one
	return Microfacet(facing, unity, unity, alpha) /
		Beckmann(facing, unity, unity, slopes)
}

// At equal peak density, GGX alpha the square root of the mean-square
// slope, GGX matches Beckmann at the peak, is narrower just past it and
// far heavier in the tails: the error the documented mapping carries.
// Each row of the table gives the ratio worked as exp(exponent) over
// divisor.
func TestGGXAtEqualPeakDensityDiffersInTheTails(t *testing.T) {
	slopes := picked(t, "beckmann-sea-slopes")
	tolerance := picked(t, "beckmann-tail-tolerance")
	for _, each := range readSheet(t, beckmannTailsFile) {
		widths := cell(t, each, "widths")
		exponent := cell(t, each, "exponent")
		want := math.Exp(exponent) / cell(t, each, "divisor")
		got := beckmannRatio(slopes, widths)
		if math.Abs(got/want-unity) > tolerance {
			t.Errorf("at %v widths GGX over Beckmann is %v, want %v",
				widths, got, want)
		}
	}
}
