package arid

import (
	"math"
	"testing"

	"github.com/janearc/slayer/oceantex"
)

// The representative peak is held to each band's actual synthesis range.
func TestBendUsesCascadeCutForBandRanges(t *testing.T) {
	bands := []*oceantex.Field{{N: 1024, Extent: 4096}, {N: 1024, Extent: 512}, {N: 1024, Extent: 64}}
	first := oceantex.CascadeCut() * math.Pi * 1024 / 4096
	second := oceantex.CascadeCut() * math.Pi * 1024 / 512
	third := oceantex.CascadeCut() * math.Pi * 1024 / 64
	cases := []struct {
		peak float64
		want []float64
	}{
		{first / 2, []float64{first / 2, first, second}},
		{(first + second) / 2, []float64{first, (first + second) / 2, second}},
		{third * 2, []float64{first, second, third}},
	}
	for _, trial := range cases {
		got := bandNumbers(bands, trial.peak)
		for index := range trial.want {
			if math.Abs(got[index]-trial.want[index]) > 1e-12 {
				t.Errorf("peak %g band %d got %g want %g", trial.peak, index, got[index], trial.want[index])
			}
		}
	}
}

// The four configured cascades carry wavelengths down to 32, 4, 0.5 and
// 0.0625 metres. Test both sides of their boundaries using the fields that
// synthesis actually constructs, including the finest cascade.
func TestBendConfiguredCascadeTransitions(t *testing.T) {
	cascades, err := oceantex.WaveCascades(oceantex.Sea{Wind: 6, Fetch: 10000}, 0, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(cascades) != 4 {
		t.Fatalf("got %d cascades, want four", len(cascades))
	}
	fields := make([]*oceantex.Field, len(cascades))
	for index, cascade := range cascades {
		fields[index] = cascade.Field
	}
	bounds := []float64{math.Pi / 16, math.Pi / 2, 4 * math.Pi, 32 * math.Pi}
	for edge := 0; edge < len(bounds)-1; edge++ {
		for _, peak := range []float64{math.Nextafter(bounds[edge], 0), bounds[edge], math.Nextafter(bounds[edge], math.Inf(1))} {
			got := bandNumbers(fields, peak)
			for index := range fields {
				want := peak
				if index > 0 && want < bounds[index-1] {
					want = bounds[index-1]
				}
				if want > bounds[index] {
					want = bounds[index]
				}
				if math.Abs(got[index]-want) > 1e-12 {
					t.Errorf("edge %d peak %.17g band %d: got %.17g want %.17g", edge, peak, index, got[index], want)
				}
			}
		}
	}
}

// The choice of band number must affect the lag on a constant-depth coast.
// The expected shallow-water wave number is found independently by bisection
// of k*tanh(k*depth)=k_deep, rather than the surf solver's Newton method.
func TestBendSelectedBandReachesShoreLag(t *testing.T) {
	const count = 1024
	fields := []*oceantex.Field{{N: count, Extent: 4096}, {N: count, Extent: 512}, {N: count, Extent: 64}, {N: count, Extent: 8}}
	shore := testCoast(t, shaped(40, 3, 0.25, evenHeight(0.25)))
	waves := &surf{shore: shore, toward: [2]float64{1, 0}}
	sea := &Sea{bands: fields}
	waves.bend(sea, arrival{peak: math.Sqrt(gravity * 0.67), rms: 0.01})
	numbers := []float64{math.Pi / 16, 0.67, math.Pi / 2, 4 * math.Pi}
	if len(waves.lags) != len(numbers) {
		t.Fatalf("got %d lag grids", len(waves.lags))
	}
	for index, number := range numbers {
		low, high := number, number+10
		for range 100 {
			middle := (low + high) / 2
			if middle*math.Tanh(middle*0.25) < number {
				low = middle
			} else {
				high = middle
			}
		}
		want := (low+high)/(2*number) - 1
		got := (waves.lags[index][40+30] - waves.lags[index][40+10]) / (20 * 0.25)
		if math.Abs(got-want) > 1e-8 {
			t.Errorf("band %d lag slope %.12g want %.12g", index, got, want)
		}
	}
}
