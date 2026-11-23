package skylight

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"testing"
)

// The thin layer the sky is checked on.
const (
	checkedDepth  = 0.0025    // picked: multiple scattering under 3%
	thinPhotons   = 240000000 // picked: as many counts a bin as at 0.02
	thinZeniths   = 9         // picked: bands twenty degrees wide
	thinBand      = 1         // picked: the band from 20 to 40 degrees
	thinViews     = 10        // picked: view cosines a fifth wide
	thinAzimuths  = 4         // picked: azimuths 45 degrees wide
	lowestCosine  = 0.2       // picked: bins from here up, off the horizon
	quadrature    = 16        // picked: good to a thousandth, smooth sky
	threeQuarters = 0.75      // defined: Rayleigh's phase, 3/4 (1+cos^2)
	brighter      = 1.06      // picked: a sky six percent too bright
)

// The four depths the excess over single scattering is followed down.
// The sweep's bins of view cosine are a tenth wide, so the one from 0.9
// to 1 lies wholly within 30 degrees of the zenith; with twice the bins
// it takes twice the photons for as many counts in each.
const (
	sweepViews   = 20       // picked: cosines a tenth wide
	sweepPhotons = 60000000 // picked: at 0.02; scaled by 0.02 over depth
	sweepBase    = 0.02     // brief: test 5's depth as first written
	nearZenith   = 30       // coordinator: views within 30 degrees
	farZenith    = 60       // coordinator: views over 60 degrees out
	keptExcess   = 0.05     // picked: the excess test 5 saw at 0.02
)

// sweepDepths are the thin layer's depths, each half the one before.
var sweepDepths = []float64{0.02, 0.01, 0.005, 0.0025} // coordinator

// singly is the radiance at the floor of a plane layer of Rayleigh gas
// of optical depth depth over black ground, scattered once, for unit
// sunlight from a zenith whose cosine is sun, seen looking up at a
// cosine view from the vertical, round radians from the sun.
//
// Light reaches a depth t below the top as exp(-t/sun) and is sent on
// by p over four pi per unit depth along the eye's line. Going down that
// line, whose cosine from the downward vertical is view, from t to the
// floor, it keeps exp(-(depth-t)/view); a depth dt there is dt/view long.
//
// So L is p over four pi view, times the integral of exp(-t/sun)
// exp(-(depth-t)/view), over t from nought to depth: p over four pi,
// times sun over (sun less view), times exp(-depth/sun) less
// exp(-depth/view). The eye's cosine up is the light's cosine down.
//
// The light turns from the sun's way to its way down; that angle is
// the one between the sun and the view, whose cosine is sun times view
// plus their sines' product times the cosine of round.
func singly(depth, sun, view, round float64) float64 {
	turn := sun*view + math.Sqrt(one-sun*sun)*math.Sqrt(one-view*view)*
		math.Cos(round)
	phase := threeQuarters * (one + turn*turn)
	return phase / sphere * sun / (sun - view) *
		(math.Exp(-depth/sun) - math.Exp(-depth/view))
}

// multipleBound is the most that light scattered more than once can add
// to single scattering at the floor of a plane layer of Rayleigh gas of
// optical depth depth, as a share, with the sun no lower than a cosine
// of lowestSun. It is derived, not fitted.
//
// Write S for the light a depth sends on per unit depth, per steradian.
// Along a line from a depth t to the layer's edge at s/|mu|, light of
// order n is at most max S_n times one less exp(-s/|mu|), at most
// max S_n times the smaller of one and s/|mu|.
//
// Over a hemisphere that gives 2 pi max S_n times s(1 + ln(1/s)),
// which is concave in s, so both hemispheres together are at most
// 2 pi max S_n depth (1 + ln(2/depth)), at t half way: call it g.
//
// Turned on by a phase no more than 3/2, S_n+1 is then at most q max S_n
// with q = 3/4 g, so all orders after the first add at most max S_1
// q/(1 - q). And max S_1 is 3/2 over 4 pi, while S_1 is at least 3/4
// over 4 pi times exp(-depth/lowestSun).
//
// So at every depth the rest is at most 2 exp(depth/lowestSun) q/(1-q)
// of the first order, and the floor's radiance, a sum over depths with
// the same weights for every order, is too. Multiple scattering only
// adds, so the excess is not below nought.
func multipleBound(depth, lowestSun float64) float64 {
	grown := depth * (one + math.Log(two/depth))
	order := threeQuarters * grown
	return two * math.Exp(depth/lowestSun) * order / (one - order)
}

// a reference is a radiance worked out by hand, from the cosines of the
// sun's zenith and the view's, and the azimuth between them.
type reference func(sun, view, round float64) float64

// a span is the low and high ends of one bin's range.
type span struct{ low, high float64 }

// midpoint is the middle of the index'th of quadrature equal parts.
func (s span) midpoint(index int) float64 {
	return s.low + (float64(index)+half)*(s.high-s.low)/quadrature
}

// spanOf is the range of bin index of bins equal bins from low to high.
func spanOf(low, high float64, index, bins int) span {
	width := (high - low) / float64(bins)
	return span{low + float64(index)*width, low + float64(index+one)*width}
}

// binMean is the mean of a reference over a bin as the solver's
// estimator averages it: over the band's ground, which weighs the zenith
// by its sine, and evenly over the view's cosine and the azimuth.
func binMean(worked reference, zenith, view, round span) float64 {
	sum, weights := 0.0, 0.0
	for i := range quadrature {
		angle := zenith.midpoint(i)
		area := math.Sin(angle)
		for j := range quadrature {
			for k := range quadrature {
				sum += area * worked(math.Cos(angle), view.midpoint(j),
					round.midpoint(k))
				weights += area
			}
		}
	}
	return sum / weights
}

// singlyAt is single scattering through a layer of depth.
func singlyAt(depth float64) reference {
	return func(sun, view, round float64) float64 {
		return singly(depth, sun, view, round)
	}
}

// walkThin is the sky at the ground of a layer of Rayleigh gas of depth
// a kilometre thick on the large planet, over black ground, kept with
// views bins of view cosine.
func walkThin(depth float64, views, photons int) (*Field, error) {
	gas := Shell{GasScattering: depth / thinTop}
	grid := Grid{Heights: []float64{none}, Zeniths: thinZeniths,
		Views: views, Azimuths: thinAzimuths}
	field, _, err := Solve(uniformColumn(farRadius, thinTop, gas, none),
		grid, photons, seedOf, workersOf)
	return field, err
}

// thinField is test 5's sky, walked once for the test and its antis.
var thinField = sync.OnceValues(func() (*Field, error) {
	return walkThin(checkedDepth, thinViews, thinPhotons)
})

// sweeps are the sky at each of the sweep's depths, each walked once.
var sweeps = map[float64]func() (*Field, error){}

// sweepOf is the sweep's sky at depth, walked the first time it is
// asked, with photons scaled so each bin has as many counts.
func sweepOf(depth float64) (*Field, error) {
	if _, walked := sweeps[depth]; !walked {
		photons := int(math.Round(sweepPhotons * sweepBase / depth))
		sweeps[depth] = sync.OnceValues(func() (*Field, error) {
			return walkThin(depth, sweepViews, photons)
		})
	}
	return sweeps[depth]()
}

// a thinBin is one checked bin of the thin sky: its range of view
// cosine, its radiance, the reference's mean over it, and the crossings
// behind it.
type thinBin struct {
	view      span
	got, want float64
	counts    int64
}

// ratio is the bin's radiance over its reference.
func (b thinBin) ratio() float64 { return b.got / b.want }

// noise is the ratio's standard error, from the counts. Every weight is
// one, so a bin's spread is its counts' with the spread of one over the
// cosine inside it, which adds under five percent to the variance.
func (b thinBin) noise() float64 {
	return b.ratio() / math.Sqrt(float64(b.counts))
}

// thinBins are the field's bins in the band, from the lowest cosine up,
// each with the reference's mean over it.
func thinBins(field *Field, worked reference) []thinBin {
	zenith := spanOf(none, math.Pi, thinBand, thinZeniths)
	viewCount := float64(field.views)
	lowest := int(math.Round((lowestCosine + one) / two *
		viewCount))
	var bins []thinBin
	for v := lowest; v < field.views; v++ {
		view := spanOf(-one, one, v, field.views)
		for a := range field.azimuths {
			index := field.index(none, thinBand, v, a)
			want := binMean(worked, zenith, view,
				spanOf(none, math.Pi, a, field.azimuths))
			bins = append(bins, thinBin{view, field.radiance[index], want,
				field.counts[index]})
		}
	}
	return bins
}

// a summary is the mean ratio over some bins, its standard error, the
// ratio farthest from one, and how many bins.
type summary struct {
	mean, error, worst float64
	bins               int
}

// summarise is the summary of the bins keep takes.
func summarise(bins []thinBin, keep func(thinBin) bool) summary {
	made := summary{worst: one}
	variance := 0.0
	for _, each := range bins {
		if !keep(each) {
			continue
		}
		made.mean += each.ratio()
		variance += each.noise() * each.noise()
		if math.Abs(each.ratio()-one) > math.Abs(made.worst-one) {
			made.worst = each.ratio()
		}
		made.bins++
	}
	made.mean /= float64(made.bins)
	made.error = math.Sqrt(variance) / float64(made.bins)
	return made
}

// everyBin takes every bin.
func everyBin(thinBin) bool { return true }

// excessWithin is nil when a ratio is over one by no less than minus
// sigmas of its noise and no more than the bound and sigmas of it.
func excessWithin(ratio, noise, bound float64) error {
	excess := ratio - one
	if !(excess >= -sigmas*noise && excess <= bound+sigmas*noise) {
		return fmt.Errorf("ratio %.4f, allowed %.4f to %.4f", ratio,
			one-sigmas*noise, one+bound+sigmas*noise)
	}
	return nil
}

// thinAgrees is nil when each bin, and their mean, is single scattering
// with no more excess than the bound on multiple scattering allows.
func thinAgrees(bins []thinBin, bound float64) error {
	var failed []error
	for _, each := range bins {
		if err := excessWithin(each.ratio(), each.noise(),
			bound); err != nil {
			failed = append(failed, fmt.Errorf("view %.1f to %.1f: %w",
				each.view.low, each.view.high, err))
		}
	}
	all := summarise(bins, everyBin)
	if err := excessWithin(all.mean, all.error, bound); err != nil {
		failed = append(failed, fmt.Errorf("the mean: %w", err))
	}
	return errors.Join(failed...)
}

// lowestSun is the cosine of the band's lowest sun, its far edge.
func lowestSun() float64 {
	return math.Cos(spanOf(none, math.Pi, thinBand, thinZeniths).high)
}

// checkedBins are test 5's bins against single scattering at its depth,
// or against the reference given.
func checkedBins(t *testing.T, worked reference) []thinBin {
	t.Helper()
	field, err := thinField()
	if err != nil {
		t.Fatal(err)
	}
	return thinBins(field, worked)
}

// a thin layer of Rayleigh gas on a planet so large it is flat gives
// the sky of single scattering, worked out in singly, in the band of
// the sun near thirty degrees and away from the horizon. Bin by bin and
// on the mean, it is above it by no more than multipleBound allows.
func TestThinAirScattersOnce(t *testing.T) {
	bins := checkedBins(t, singlyAt(checkedDepth))
	bound := multipleBound(checkedDepth, lowestSun())
	for _, each := range bins {
		t.Logf("view %.1f to %.1f: %.5g, want %.5g (%d counts)",
			each.view.low, each.view.high, each.got, each.want,
			each.counts)
	}
	all := summarise(bins, everyBin)
	t.Logf("mean %.4f +- %.4f, worst %.4f, bound %.4f", all.mean,
		all.error, all.worst, bound)
	if err := thinAgrees(bins, bound); err != nil {
		t.Error(err)
	}
}

// the anti-test: the same sky against single scattering with the sun's
// and the view's cosines swapped does not agree, so the check would
// catch the one mistake the formula invites.
func TestSwappedCosinesAreCaught(t *testing.T) {
	swapped := func(sun, view, round float64) float64 {
		return singly(checkedDepth, view, sun, round)
	}
	bins := checkedBins(t, swapped)
	lowSun := lowestSun()
	if thinAgrees(bins, multipleBound(checkedDepth, lowSun)) == nil {
		t.Error("the swapped cosines agree")
	}
}

// the anti-test: the same sky six percent brighter is refused on its
// mean, so the check would catch an estimator off by that much.
func TestASkyTooBrightIsCaught(t *testing.T) {
	bins := checkedBins(t, singlyAt(checkedDepth))
	for index := range bins {
		bins[index].got *= brighter
	}
	lowSun := lowestSun()
	if thinAgrees(bins, multipleBound(checkedDepth, lowSun)) == nil {
		t.Error("a sky six percent too bright agrees")
	}
}

// nearTheZenith takes the bins wholly within 30 degrees of the zenith.
func nearTheZenith(each thinBin) bool {
	return each.view.low >= math.Cos(nearZenith*degree)
}

// farFromTheZenith takes the bins wholly more than 60 degrees out.
func farFromTheZenith(each thinBin) bool {
	return each.view.high <= math.Cos(farZenith*degree)
}

// the excess over single scattering falls with the layer's depth as
// light scattered twice does, about in proportion, near the zenith and
// far from it alike: halving the depth halves it, less a slow log. At
// each depth the mean keeps within multipleBound.
func TestThinExcessFallsWithDepth(t *testing.T) {
	if testing.Short() {
		t.Skip("the four depths walk nine hundred million photons")
	}
	for _, depth := range sweepDepths {
		field, err := sweepOf(depth)
		if err != nil {
			t.Fatal(err)
		}
		bins := thinBins(field, singlyAt(depth))
		all := summarise(bins, everyBin)
		near := summarise(bins, nearTheZenith)
		far := summarise(bins, farFromTheZenith)
		bound := multipleBound(depth, lowestSun())
		t.Logf("depth %g: all %.4f +- %.4f worst %.4f (%d bins); "+
			"near %.4f +- %.4f (%d); far %.4f +- %.4f (%d); excess over "+
			"depth %.3g; bound %.4f", depth, all.mean, all.error,
			all.worst, all.bins, near.mean, near.error, near.bins,
			far.mean, far.error, far.bins, (all.mean-one)/depth, bound)
		if err := excessWithin(all.mean, all.error, bound); err != nil {
			t.Errorf("depth %g: %v", depth, err)
		}
	}
}

// the anti-test: the excess seen at 0.02 carried down unchanged to the
// thinnest depth is refused there, so the check would catch an excess
// from the estimator, which would not fall with the depth.
func TestAnExcessThatStaysIsCaught(t *testing.T) {
	if testing.Short() {
		t.Skip("the thinnest depth walks four hundred million photons")
	}
	thinnest := sweepDepths[len(sweepDepths)-one]
	field, err := sweepOf(thinnest)
	if err != nil {
		t.Fatal(err)
	}
	singlyAtThinnest := singlyAt(thinnest)
	all := summarise(thinBins(field, singlyAtThinnest), everyBin)
	stayed := all.mean + keptExcess
	lowSun := lowestSun()
	if excessWithin(stayed, all.error,
		multipleBound(thinnest, lowSun)) == nil {
		t.Errorf("an excess of %g that stayed passes", keptExcess)
	}
}
