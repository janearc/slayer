package thing

import (
	"math"
	"testing"
)

// coxMunkAtSeven is Cox and Munk's clean-surface mean-square slope at
// 7 m/s, 0.003 + 0.00512 * 7, written out so this package's tests need
// no sea.
const coxMunkAtSeven = 0.03884

// normalized is the integral over the hemisphere of a density of tiny
// faces times their cosine, by the midpoint rule in the tilt; a density
// of faces sums to one this way.
func normalized(density func(facing float64) float64) float64 {
	const rings = 200000
	sum := 0.0
	for i := range rings {
		tilt := (float64(i) + 0.5) / rings * math.Pi / 2
		facing := math.Cos(tilt)
		sum += density(facing) * facing * math.Sin(tilt)
	}
	return sum * math.Pi / 2 / rings * 2 * math.Pi
}

// Beckmann's density of faces, times their cosine, sums to one over the
// hemisphere, narrow or broad.
func TestBeckmannFacesNormalize(t *testing.T) {
	for _, slopes := range []float64{0.0016, coxMunkAtSeven, 0.2} {
		got := normalized(func(facing float64) float64 {
			return beckmannFaces(facing, slopes)
		})
		if math.Abs(got-1) > 1e-6 {
			t.Errorf("mean-square slope %v sums to %v, not one", slopes, got)
		}
	}
}

// Anti-test: the sum sees a density built on the variance one way
// rather than both ways together, the slip Cox and Munk's two
// variances invite: it sums to two, not one.
func TestBeckmannNormalizingSeesOneWayVariance(t *testing.T) {
	slopes := coxMunkAtSeven
	oneWay := slopes / 2
	got := normalized(func(facing float64) float64 {
		square := facing * facing
		tangent := (1 - square) / square
		return math.Exp(-tangent/(2*oneWay)) /
			(math.Pi * oneWay * square * square)
	})
	if math.Abs(got-2) > 1e-6 {
		t.Errorf("a one-way density sums to %v, want two", got)
	}
}

// Smith's share for Beckmann at a = 1, where Lambda is (exp(-1) /
// sqrt(pi) - erfc(1)) / 2 = 0.0251273, and all of it straight down.
func TestBeckmannSmithAtAKnownSlant(t *testing.T) {
	slopes := coxMunkAtSeven
	// a = cot / sqrt(slopes) = 1: cot = sqrt(slopes)
	cos := math.Sqrt(slopes) / math.Sqrt(1+slopes)
	got, want := beckmannSmith(cos, slopes), 1/1.0251273
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("share at a = 1 is %v, want %v", got, want)
	}
	if got := beckmannSmith(1, slopes); got != 1 {
		t.Errorf("share straight down is %v, want one", got)
	}
	if !(beckmannSmith(0.1, slopes) < beckmannSmith(0.5, slopes)) {
		t.Error("a more slant way sees no fewer faces")
	}
}

// Anti-test: what cannot glint gives nothing, and a glint is never
// negative or not a number.
func TestBeckmannRefusesWhatCannotGlint(t *testing.T) {
	for _, each := range [][4]float64{
		{0, 0.8, 0.8, coxMunkAtSeven},
		{1, 0, 0.8, coxMunkAtSeven},
		{1, 0.8, 0, coxMunkAtSeven},
		{1, 0.8, 0.8, 0},
		{1, 0.8, 0.8, -1},
		{1, 0.8, 0.8, math.NaN()},
		{math.NaN(), 0.8, 0.8, coxMunkAtSeven},
	} {
		if got := Beckmann(each[0], each[1], each[2], each[3]); got != 0 {
			t.Errorf("Beckmann%v = %v, want nothing", each, got)
		}
	}
	for _, facing := range []float64{1, 0.99, 0.9, 0.5, 0.01} {
		got := Beckmann(facing, 0.3, 0.3, coxMunkAtSeven)
		if !(got >= 0) {
			t.Errorf("facing %v gives %v", facing, got)
		}
	}
}

// At equal peak density, GGX alpha the square root of the mean-square
// slope, GGX matches Beckmann at the peak, is narrower just past it and
// far heavier in the tails: the error the documented mapping carries.
func TestGGXAtEqualPeakDensityDiffersInTheTails(t *testing.T) {
	slopes := coxMunkAtSeven
	alpha := math.Sqrt(slopes)
	ratio := func(tangentWidths float64) float64 {
		tangent := tangentWidths * math.Sqrt(slopes)
		facing := 1 / math.Sqrt(1+tangent*tangent)
		// with lit and viewed one, both shadowings are one
		return Microfacet(facing, 1, 1, alpha) /
			Beckmann(facing, 1, 1, slopes)
	}
	for _, each := range []struct{ widths, want float64 }{
		{0, 1}, {1, math.E / 4}, {2, math.Exp(4) / 25}, {3, math.Exp(9) / 100},
	} {
		if got := ratio(each.widths); math.Abs(got/each.want-1) > 1e-9 {
			t.Errorf("at %v widths GGX over Beckmann is %v, want %v",
				each.widths, got, each.want)
		}
	}
}
