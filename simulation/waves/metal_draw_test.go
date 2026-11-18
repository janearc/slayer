//go:build draw && darwin

package waves

import (
	"fmt"
	"math"
	"math/cmplx"
	"slices"
	"testing"

	"github.com/janearc/slayer/simulation/spectral"
	"pgregory.net/rapid"
)

func openWaveMetal(t testing.TB, field *Field) *MetalField {
	t.Helper()
	plan, err := field.OpenMetal()
	if err != nil || plan == nil {
		t.Fatalf("OpenMetal: %v, %v", plan, err)
	}
	t.Cleanup(plan.Close)
	return plan
}

func waveMetalFixture(shape spectral.Shape) (State, []float64) {
	cells := shape.Width * shape.Height
	state := State{Value: make([]complex128, cells), Rate: make([]complex128, cells)}
	frequency := make([]float64, cells)
	for i := range cells {
		state.Value[i] = complex(float64(i%17)/7-1, float64(i%13)/11-0.5)
		state.Rate[i] = complex(float64(i%11)/9-0.25, float64(i%19)/13-0.75)
		frequency[i] = float64(i % 7)
	}
	return state, frequency
}

// waveMetalNorm forms a finite complex L2 norm without squaring overflow.
func waveMetalNorm(values []complex128) float64 {
	var norm float64
	for _, value := range values {
		norm = math.Hypot(norm, cmplx.Abs(value))
	}
	return norm
}

// requireWaveMetal uses the Cauchy input scale and an independent frequency
// bound. It preserves tiny physical amplitudes and allows bounded error when
// the contributing fields cancel. All acceptance limits are fixed here.
func requireWaveMetal(t testing.TB, got, want, initial State, frequency []float64, seconds float64) {
	t.Helper()
	var largestFrequency float64
	for _, omega := range frequency {
		largestFrequency = math.Max(largestFrequency, omega)
	}
	q, v := waveMetalNorm(initial.Value), waveMetalNorm(initial.Rate)
	scales := [2]float64{q + math.Abs(seconds)*v, largestFrequency*q + v}
	for component, pair := range [][2][]complex128{{got.Value, want.Value}, {got.Rate, want.Rate}} {
		if len(pair[0]) != len(pair[1]) || len(pair[0]) == 0 {
			t.Fatalf("invalid component lengths %d/%d", len(pair[0]), len(pair[1]))
		}
		var errorNorm float64
		for i, value := range pair[0] {
			if math.IsNaN(real(value)) || math.IsInf(real(value), 0) || math.IsNaN(imag(value)) || math.IsInf(imag(value), 0) {
				t.Fatal("non-finite Metal wave state")
			}
			errorNorm = math.Hypot(errorNorm, cmplx.Abs(value-pair[1][i]))
		}
		limit := 3e-5 * math.Max(scales[component], waveMetalNorm(pair[1]))
		if math.IsNaN(errorNorm) || math.IsInf(errorNorm, 0) || errorNorm > limit {
			t.Fatalf("component %d error %g exceeds fixed scale bound %g", component, errorNorm, limit)
		}
	}
}

// Both Cauchy fields agree with float64 through different grid parities.
func TestWaveMetalMatchesBothCauchyComponents(t *testing.T) {
	for _, dimensions := range [][2]int{{8, 4}, {64, 32}, {256, 256}} {
		t.Run(fmt.Sprint(dimensions), func(t *testing.T) {
			shape := spectral.Shape{Width: dimensions[0], Height: dimensions[1], MaxCells: dimensions[0] * dimensions[1]}
			initial, frequency := waveMetalFixture(shape)
			before := State{Value: slices.Clone(initial.Value), Rate: slices.Clone(initial.Rate)}
			field, err := New(shape, frequency, initial)
			if err != nil {
				t.Fatal(err)
			}
			plan := openWaveMetal(t, field)
			for _, seconds := range []float64{0, 0.125, -0.375, 0.125} {
				want, err := field.At(seconds)
				if err != nil {
					t.Fatal(err)
				}
				got, err := plan.At(seconds)
				if err != nil {
					t.Fatal(err)
				}
				requireWaveMetal(t, got, want, initial, frequency, seconds)
				if seconds == 0 && (!slices.Equal(got.Value, initial.Value) || !slices.Equal(got.Rate, initial.Rate)) {
					t.Fatal("zero time lost exact initial data")
				}
				got.Value[0] = 123
				got.Rate[0] = 456
			}
			if !slices.Equal(initial.Value, before.Value) || !slices.Equal(initial.Rate, before.Rate) {
				t.Fatal("Metal altered Cauchy input")
			}
		})
	}
}

// anti-test of TestWaveMetalMatchesBothCauchyComponents
// A purely drifting field must retain its independent rate, without a
// zero-frequency division or an invented restoring force.
func TestWaveMetalZeroModeDrift(t *testing.T) {
	initial, frequency := waveState(), make([]float64, 32)
	field := openWave(t, frequency, initial)
	plan := openWaveMetal(t, field)
	want := State{Value: make([]complex128, 32), Rate: slices.Clone(initial.Rate)}
	for i := range want.Value {
		want.Value[i] = initial.Value[i] + 0.75*initial.Rate[i]
	}
	got, err := plan.At(0.75)
	if err != nil {
		t.Fatal(err)
	}
	requireWaveMetal(t, got, want, initial, frequency, 0.75)
}

// A traveling analytic mode resolves small real strain without an absolute
// comparison floor; both fields carry nonzero independent complex parts.
func TestWaveMetalResolvesSmallAmplitudeAndAnalyticPhase(t *testing.T) {
	shape := waveShape()
	initial := waveState()
	frequency := make([]float64, 32)
	for y := range shape.Height {
		for x := range shape.Width {
			i := y*shape.Width + x
			phase := 2 * math.Pi * (float64(x)/float64(shape.Width) + float64(y)/float64(shape.Height))
			initial.Value[i] = complex(0.75e-21, -0.25e-21) * cmplx.Exp(complex(0, phase))
			initial.Rate[i] = complex(0, -3) * initial.Value[i]
			frequency[i] = 3
		}
	}
	field := openWave(t, frequency, initial)
	plan := openWaveMetal(t, field)
	for _, seconds := range []float64{0.125, -0.25} {
		want := State{Value: make([]complex128, 32), Rate: make([]complex128, 32)}
		for i := range want.Value {
			want.Value[i] = initial.Value[i] * cmplx.Exp(complex(0, -3*seconds))
			want.Rate[i] = complex(0, -3) * want.Value[i]
		}
		got, err := plan.At(seconds)
		if err != nil {
			t.Fatal(err)
		}
		requireWaveMetal(t, got, want, initial, frequency, seconds)
	}
}

// anti-test of TestWaveMetalResolvesSmallAmplitudeAndAnalyticPhase
// A large finite phase needs accurate reduction before coefficient packing.
func TestWaveMetalKeepsLargeAbsoluteTimePhase(t *testing.T) {
	initial := State{Value: make([]complex128, 32), Rate: make([]complex128, 32)}
	frequency := make([]float64, 32)
	for i := range frequency {
		frequency[i] = 1
		initial.Value[i] = complex(0.75, -0.25)
	}
	field := openWave(t, frequency, initial)
	plan := openWaveMetal(t, field)
	got, err := plan.At(1e16)
	if err != nil {
		t.Fatal(err)
	}
	sine, cosine := math.Sincos(1e16)
	want := State{Value: make([]complex128, 32), Rate: make([]complex128, 32)}
	for i, value := range initial.Value {
		want.Value[i] = complex(cosine, 0) * value
		want.Rate[i] = complex(-sine, 0) * value
	}
	requireWaveMetal(t, got, want, initial, frequency, 1e16)
}

// Invalid query coefficients must leave the immutable plan usable.
func TestWaveMetalRefusedTimesDoNotChangePlan(t *testing.T) {
	initial, frequency := waveState(), waveFrequencies()
	field := openWave(t, frequency, initial)
	plan := openWaveMetal(t, field)
	for _, seconds := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), math.MaxFloat64} {
		got, err := plan.At(seconds)
		if err == nil || got.Value != nil || got.Rate != nil {
			t.Fatal("invalid time returned partial state")
		}
	}
	got, err := plan.At(0.25)
	if err != nil {
		t.Fatal(err)
	}
	want, err := field.At(0.25)
	if err != nil {
		t.Fatal(err)
	}
	requireWaveMetal(t, got, want, initial, frequency, 0.25)
}

// anti-test of TestWaveMetalRefusedTimesDoNotChangePlan
// Generated intervals either match the reference or obey the documented
// coefficient-range refusal, without poisoning a later ordinary query.
func TestWaveMetalGeneratedQueriesMatchReference(t *testing.T) {
	initial, frequency := waveState(), waveFrequencies()
	field := openWave(t, frequency, initial)
	plan := openWaveMetal(t, field)
	rapid.Check(t, func(t *rapid.T) {
		seconds := rapid.Float64Range(-0.5, 0.5).Draw(t, "seconds")
		got, err := plan.At(seconds)
		// For this fixture's integer frequencies 0..4, the zero mode's
		// drift t is the smallest nonzero coefficient near t=0. The
		// smallest normal float32 is 2^-126. A frequency below one would
		// move this edge through its acceleration coefficient w*w*t.
		if seconds != 0 && math.Abs(seconds) < 0x1p-126 {
			if err == nil || got.Value != nil || got.Rate != nil {
				t.Fatal("subnormal coefficient returned a partial or accepted state")
			}
			seconds = 0.25
			got, err = plan.At(seconds)
		}
		if err != nil {
			t.Fatal(err)
		}
		want, err := field.At(seconds)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Value) != 32 || len(got.Rate) != 32 {
			t.Fatal("missing Cauchy state")
		}
		var qError, vError float64
		for i := range got.Value {
			qError = math.Hypot(qError, cmplx.Abs(got.Value[i]-want.Value[i]))
			vError = math.Hypot(vError, cmplx.Abs(got.Rate[i]-want.Rate[i]))
		}
		q, v := waveMetalNorm(initial.Value), waveMetalNorm(initial.Rate)
		if math.IsNaN(qError) || math.IsInf(qError, 0) || math.IsNaN(vError) || math.IsInf(vError, 0) || qError > 3e-5*(q+math.Abs(seconds)*v) || vError > 3e-5*(4*q+v) {
			t.Fatal("generated Cauchy comparison exceeded fixed bounds")
		}
	})
}

// TestWaveMetalTimeCoefficientBoundary fixes both sides of the generated
// fixture's 2^-126 edge, including zero and the smallest float64 value.
func TestWaveMetalTimeCoefficientBoundary(t *testing.T) {
	initial, frequency := waveState(), waveFrequencies()
	field := openWave(t, frequency, initial)
	plan := openWaveMetal(t, field)
	for _, seconds := range []float64{0, 0x1p-126, -0x1p-126} {
		got, err := plan.At(seconds)
		if err != nil {
			t.Fatalf("representable boundary %g refused: %v", seconds, err)
		}
		want, err := field.At(seconds)
		if err != nil {
			t.Fatal(err)
		}
		requireWaveMetal(t, got, want, initial, frequency, seconds)
	}
	want, err := field.At(0.25)
	if err != nil {
		t.Fatal(err)
	}
	for _, seconds := range []float64{math.Nextafter(0x1p-126, 0),
		math.Nextafter(-0x1p-126, 0), math.SmallestNonzeroFloat64} {
		got, err := plan.At(seconds)
		if err == nil || got.Value != nil || got.Rate != nil {
			t.Fatalf("unrepresentable boundary %g was admitted", seconds)
		}
		got, err = plan.At(0.25)
		if err != nil {
			t.Fatalf("refusal poisoned the plan: %v", err)
		}
		requireWaveMetal(t, got, want, initial, frequency, 0.25)
	}
}

// Closed, nil or zero Metal plans cannot dispatch or return initial copies.
func TestWaveMetalMissingResourcesRefuse(t *testing.T) {
	initial, frequency := waveState(), waveFrequencies()
	closed := openWaveMetal(t, openWave(t, frequency, initial))
	closed.Close()
	closed.Close()
	for _, plan := range []*MetalField{nil, {}, closed} {
		plan.Close()
		for _, seconds := range []float64{0, 1} {
			state, err := plan.At(seconds)
			if err == nil || state.Value != nil || state.Rate != nil {
				t.Fatal("non-live plan accepted a query")
			}
		}
	}
	for _, field := range []*Field{nil, {}} {
		plan, err := field.OpenMetal()
		if plan != nil {
			plan.Close()
		}
		if err == nil || plan != nil {
			t.Fatal("uninitialized wave field opened Metal")
		}
	}
}

// anti-test of TestWaveMetalMissingResourcesRefuse
// Finite CPU data beyond the GPU sum range must be refused before packing.
func TestWaveMetalRefusesUnrepresentableSpectra(t *testing.T) {
	initial, frequency := waveState(), waveFrequencies()
	for i := range initial.Value {
		initial.Value[i] = 1e100
	}
	field := openWave(t, frequency, initial)
	plan, err := field.OpenMetal()
	if plan != nil {
		plan.Close()
	}
	if err == nil || plan != nil {
		t.Fatal("Metal accepted a spectrum outside float32 range")
	}
}

// Full calls include coefficient construction, upload, two inverses and
// readback; construction and one warm-up call are outside the measured loop.
func BenchmarkWaveMetalComparison(b *testing.B) {
	shape := spectral.Shape{Width: 256, Height: 256, MaxCells: 256 * 256}
	initial, frequency := waveMetalFixture(shape)
	field, err := New(shape, frequency, initial)
	if err != nil {
		b.Fatal(err)
	}
	for _, useGPU := range []bool{false, true} {
		name := "cpu"
		if useGPU {
			name = "metal"
		}
		b.Run(name, func(b *testing.B) {
			at := field.At
			if useGPU {
				at = openWaveMetal(b, field).At
			}
			if _, err := at(0.125); err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			for range b.N {
				got, err := at(0.125)
				if err != nil || len(got.Value) != len(initial.Value) || len(got.Rate) != len(initial.Rate) {
					b.Fatalf("propagate: %v", err)
				}
			}
		})
	}
}
