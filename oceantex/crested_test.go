package oceantex

import (
	"math"
	"testing"
)

// oneWave is a field of one wave along x, turns of it across the side:
// height a cos(kx), its slope, the push that makes it Gerstner's, -a
// sin(kx), and that push's stretch. scale is the share of the push kept:
// nought leaves the water uncarried.
func oneWave(n int, extent, amplitude float64, turns int,
	scale float64) *Field {
	cells := n * n
	wave := &Field{N: n, Extent: extent, Height: make([]float64, cells),
		SlopeX: make([]float64, cells), SlopeY: make([]float64, cells),
		PushX: make([]float64, cells), PushY: make([]float64, cells),
		StretchXX: make([]float64, cells),
		StretchXY: make([]float64, cells),
		StretchYY: make([]float64, cells)}
	k := 2 * math.Pi * float64(turns) / extent
	for row := range n {
		for column := range n {
			sin, cos := math.Sincos(k * float64(column) * extent / float64(n))
			at := row*n + column
			wave.Height[at] = amplitude * cos
			wave.SlopeX[at] = -amplitude * k * sin
			wave.PushX[at] = -scale * amplitude * sin
			wave.StretchXX[at] = -scale * amplitude * k * cos
		}
	}
	return wave
}

// One wave read where its water is has Gerstner's shape: the crest
// stands half k a squared higher over the mean than the wave's own
// amplitude, the trough as much less under it, and more of the surface
// lies under the mean than over it.
func TestCrestedSingleWaveIsGerstners(t *testing.T) {
	const n, extent, amplitude, turns = 256, 64.0, 0.5, 2
	crested := oneWave(n, extent, amplitude, turns, 1).Crested()
	k := 2 * math.Pi * turns / extent
	lift := k * amplitude * amplitude / 2
	highest, lowest, under := math.Inf(-1), math.Inf(1), 0
	for _, height := range crested.Height {
		highest, lowest = math.Max(highest, height), math.Min(lowest, height)
		if height < 0 {
			under++
		}
	}
	if math.Abs(highest-(amplitude+lift)) > 0.02*amplitude ||
		math.Abs(lowest-(lift-amplitude)) > 0.02*amplitude {
		t.Errorf("crest %g and trough %g, want %g and %g", highest, lowest,
			amplitude+lift, lift-amplitude)
	}
	if 2*under <= len(crested.Height) {
		t.Errorf("%d of %d samples under the mean: troughs are not longer "+
			"than crests", under, len(crested.Height))
	}
}

// The crested field's slope is its own heights' slope, to the sampling.
func TestCrestedSlopeIsTheCarriedSurfacesOwn(t *testing.T) {
	const n, extent = 256, 64.0
	crested := oneWave(n, extent, 0.5, 2, 1).Crested()
	texel := extent / n
	var steepest, worst float64
	for column := 1; column < n-1; column++ {
		across := (crested.Height[column+1] - crested.Height[column-1]) /
			(2 * texel)
		steepest = math.Max(steepest, math.Abs(across))
		worst = math.Max(worst, math.Abs(across-crested.SlopeX[column]))
	}
	if !(steepest > 0) || worst > 0.05*steepest {
		t.Errorf("slope differs from the heights' by %g of a steepest %g",
			worst, steepest)
	}
}

// Anti: water that is not carried is read where it is. A field with no
// push is its own crested field, less its mean.
func TestCrestedOfUncarriedWaterIsItself(t *testing.T) {
	wave := oneWave(64, 64, 0.5, 2, 0)
	crested := wave.Crested()
	for index, height := range wave.Height {
		if math.Abs(crested.Height[index]-height) > 1e-9 ||
			math.Abs(crested.SlopeX[index]-wave.SlopeX[index]) > 1e-9 {
			t.Fatalf("sample %d: height %g for %g, slope %g for %g", index,
				crested.Height[index], height, crested.SlopeX[index],
				wave.SlopeX[index])
		}
	}
}

// The smooth read passes through every sample.
func TestSmoothPassesThroughItsSamples(t *testing.T) {
	wave := oneWave(16, 32, 1, 3, 1)
	texel := wave.Extent / float64(wave.N)
	for row := range wave.N {
		for column := range wave.N {
			got := wave.Smooth(wave.Height, float64(column)*texel,
				float64(row)*texel)
			if want := wave.Height[row*wave.N+column]; math.Abs(got-want) >
				1e-12 {
				t.Fatalf("at %d, %d: %g, want %g", column, row, got, want)
			}
		}
	}
}

// A wave's variance, half its amplitude squared, is in the ring of its
// own wave number and in no other.
func TestRingedHoldsAWaveInItsOwnRing(t *testing.T) {
	const turns, amplitude = 5, 0.5
	rings := Ringed(oneWave(64, 64, amplitude, turns, 0).Height, 64)
	for ring, held := range rings {
		want := 0.0
		if ring == turns {
			want = amplitude * amplitude / 2
		}
		if math.Abs(held-want) > 1e-12 {
			t.Errorf("ring %d holds %g, want %g", ring, held, want)
		}
	}
}

// The peak wave by wave number is near the wave of the peak frequency,
// and a little longer. Anti: a calm sea has no peak.
func TestPeakWavelengthIsNearThePeakFrequencysWave(t *testing.T) {
	sea := Sea{Wind: 7, Fetch: 500000}
	byFrequency := 2 * math.Pi * gravity / (sea.Peak() * sea.Peak())
	got := sea.PeakWavelength()
	if !(got >= byFrequency) || got > 1.2*byFrequency {
		t.Errorf("peak wave %g m, by the peak frequency %g m", got,
			byFrequency)
	}
	if calm := (Sea{Fetch: 500000}).PeakWavelength(); calm != 0 {
		t.Errorf("a calm sea's peak wave is %g m", calm)
	}
}

// Every band stops at its cut: no wave shorter than two samples over the
// cut is in it. Anti: the band is not empty under the cut.
func TestCascadesHoldNothingPastTheirCut(t *testing.T) {
	cascades, err := WaveCascades(Sea{Wind: 7, Fetch: 500000}, 0, 7)
	if err != nil {
		t.Fatal(err)
	}
	for index, cascade := range cascades {
		rings := Ringed(cascade.Field.Height, cascade.Field.N)
		size := float64(cascade.Field.N)
		last := int(math.Ceil(cascadeCut * size / 2))
		var under, past float64
		for ring, held := range rings {
			if ring > last {
				past += held
			} else {
				under += held
			}
		}
		if !(under > 0) || past > 1e-12*under {
			t.Errorf("band %d holds %g under its cut and %g past it", index,
				under, past)
		}
	}
}

// A band fades from a pixel by its shortest wave, not by its sample: it
// is whole while the pixel covers half that wave or less and gone once
// the pixel covers all of it. Anti: between the two it is neither.
func TestPixelShareFadesByTheShortestWave(t *testing.T) {
	const texel = 0.5
	shortest := 2 * texel / cascadeCut
	if whole := PixelShare(shortest/2, texel); whole != 1 {
		t.Errorf("a pixel of half the shortest wave shows %g", whole)
	}
	if none := PixelShare(shortest, texel); none != 0 {
		t.Errorf("a pixel of the whole shortest wave shows %g", none)
	}
	if some := PixelShare(0.75*shortest, texel); !(some > 0 && some < 1) {
		t.Errorf("a pixel of three quarters of the wave shows %g", some)
	}
	keep, drop := PixelShareSpan()
	if keep*texel != shortest/2 || drop*texel != shortest {
		t.Errorf("the span is %g to %g samples", keep, drop)
	}
}
