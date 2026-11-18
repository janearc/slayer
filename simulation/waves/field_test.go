package waves

import (
	"math"
	"math/cmplx"
	"slices"
	"testing"

	"github.com/janearc/slayer/simulation/spectral"
	"pgregory.net/rapid"
)

func waveShape() spectral.Shape { return spectral.Shape{Width: 8, Height: 4, MaxCells: 32} }
func waveState() State {
	s := State{Value: make([]complex128, 32), Rate: make([]complex128, 32)}
	for i := range s.Value {
		s.Value[i] = complex(float64(i%7)-2, float64(i%5)/3)
		s.Rate[i] = complex(float64(i%3)/4-0.25, float64(i%11)/13)
	}
	return s
}
func waveFrequencies() []float64 {
	f := make([]float64, 32)
	for i := range f {
		f[i] = float64(i % 5)
	}
	return f
}
func openWave(t testing.TB, frequency []float64, initial State) *Field {
	t.Helper()
	field, err := New(waveShape(), frequency, initial)
	if err != nil || field == nil {
		t.Fatalf("New: %v, %v", field, err)
	}
	return field
}
func requireWave(t testing.TB, got, want State) {
	t.Helper()
	for _, pair := range [][2][]complex128{{got.Value, want.Value}, {got.Rate, want.Rate}} {
		if len(pair[0]) != len(pair[1]) || len(pair[0]) == 0 {
			t.Fatalf("state lengths %d/%d", len(pair[0]), len(pair[1]))
		}
		for i, want := range pair[1] {
			delta := cmplx.Abs(pair[0][i] - want)
			if math.IsNaN(delta) || math.IsInf(delta, 0) || delta > 2e-11*math.Max(1, cmplx.Abs(want)) {
				t.Fatalf("sample %d: got %v want %v", i, pair[0][i], want)
			}
		}
	}
}

// At zero preserves the exact initial values, with independent copies.
func TestWaveZeroTimeCopiesBothFields(t *testing.T) {
	initial := waveState()
	before := State{Value: slices.Clone(initial.Value), Rate: slices.Clone(initial.Rate)}
	frequencies := waveFrequencies()
	field := openWave(t, frequencies, initial)
	initial.Value[0] = 99
	initial.Rate[0] = 123
	frequencies[1] = 99
	got, err := field.At(0)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Value, before.Value) || !slices.Equal(got.Rate, before.Rate) {
		t.Fatal("zero time lost exact initial data")
	}
	got.Value[0] = 456
	got.Rate[0] = 789
	again, err := field.At(0)
	if err != nil {
		t.Fatal(err)
	}
	requireWave(t, again, before)
}

// anti-test of TestWaveZeroTimeCopiesBothFields
// A traveling complex Fourier mode needs both its initial value and rate.
func TestWaveTravelingModeHasCorrectPhaseAndDerivative(t *testing.T) {
	shape := waveShape()
	initial := waveState()
	frequency := make([]float64, 32)
	for y := range shape.Height {
		for x := range shape.Width {
			i := y*shape.Width + x
			phase := 2 * math.Pi * (float64(x)/8 + float64(y)/4)
			initial.Value[i] = complex(0.75, -0.25) * cmplx.Exp(complex(0, phase))
			initial.Rate[i] = complex(0, -3) * initial.Value[i]
		}
	}
	for i := range frequency {
		frequency[i] = 3
	}
	field := openWave(t, frequency, initial)
	for _, time := range []float64{0.25, -0.5, 2} {
		got, err := field.At(time)
		if err != nil {
			t.Fatal(err)
		}
		want := State{Value: make([]complex128, 32), Rate: make([]complex128, 32)}
		for i := range want.Value {
			want.Value[i] = initial.Value[i] * cmplx.Exp(complex(0, -3*time))
			want.Rate[i] = complex(0, -3) * want.Value[i]
		}
		requireWave(t, got, want)
	}
}

// Zero-frequency modes drift without dividing by zero.
func TestWaveZeroFrequencyDriftsAtConstantRate(t *testing.T) {
	initial := waveState()
	field := openWave(t, make([]float64, 32), initial)
	got, err := field.At(0.75)
	if err != nil {
		t.Fatal(err)
	}
	want := State{Value: make([]complex128, 32), Rate: slices.Clone(initial.Rate)}
	for i := range want.Value {
		want.Value[i] = initial.Value[i] + 0.75*initial.Rate[i]
	}
	requireWave(t, got, want)
}

// anti-test of TestWaveZeroFrequencyDriftsAtConstantRate
// A tiny frequency cannot erase a resolved drift when omega*time underflows.
func TestWaveTinyFrequencyRetainsDrift(t *testing.T) {
	initial := State{Value: make([]complex128, 32), Rate: make([]complex128, 32)}
	frequency := make([]float64, 32)
	for i := range frequency {
		frequency[i] = 1e-300
		initial.Rate[i] = 1e100
	}
	field := openWave(t, frequency, initial)
	got, err := field.At(1e-100)
	if err != nil {
		t.Fatal(err)
	}
	want := State{Value: make([]complex128, 32), Rate: slices.Clone(initial.Rate)}
	for i := range want.Value {
		want.Value[i] = 1
	}
	requireWave(t, got, want)
}

// Absolute-time evolution composes and reverses for all retained modes.
func TestWaveCompositionAndReversibility(t *testing.T) {
	frequency, initial := waveFrequencies(), waveState()
	field := openWave(t, frequency, initial)
	first, err := field.At(0.25)
	if err != nil {
		t.Fatal(err)
	}
	continuation := openWave(t, frequency, first)
	got, err := continuation.At(0.5)
	if err != nil {
		t.Fatal(err)
	}
	want, err := field.At(0.75)
	if err != nil {
		t.Fatal(err)
	}
	requireWave(t, got, want)
	back, err := continuation.At(-0.25)
	if err != nil {
		t.Fatal(err)
	}
	requireWave(t, back, initial)
}

// anti-test of TestWaveCompositionAndReversibility
// Modal oscillator energy must not drift under reversible propagation.
func TestWaveModalEnergyIsConserved(t *testing.T) {
	frequency, initial := waveFrequencies(), waveState()
	field := openWave(t, frequency, initial)
	got, err := field.At(7.25)
	if err != nil {
		t.Fatal(err)
	}
	energy := func(state State) float64 {
		q, err := spectral.Forward(waveShape(), state.Value)
		if err != nil {
			t.Fatal(err)
		}
		v, err := spectral.Forward(waveShape(), state.Rate)
		if err != nil {
			t.Fatal(err)
		}
		var total float64
		for i := range q {
			total += cmplx.Abs(v[i])*cmplx.Abs(v[i]) + frequency[i]*frequency[i]*cmplx.Abs(q[i])*cmplx.Abs(q[i])
		}
		return total
	}
	before, after := energy(initial), energy(got)
	if math.Abs(after-before) > 2e-12*before {
		t.Fatalf("energy %g -> %g", before, after)
	}
}

// Construction rejects invalid grids, frequencies and initial fields.
func TestWaveNewRefusesMalformedInput(t *testing.T) {
	for _, which := range []string{"shape", "budget", "count", "negative", "nan", "inf", "short value", "short rate", "bad value", "bad rate"} {
		t.Run(which, func(t *testing.T) {
			shape, frequency, initial := waveShape(), waveFrequencies(), waveState()
			switch which {
			case "shape":
				shape.Width = 3
			case "budget":
				shape.MaxCells = 1
			case "count":
				frequency = frequency[:31]
			case "negative":
				frequency[0] = -1
			case "nan":
				frequency[0] = math.NaN()
			case "inf":
				frequency[0] = math.Inf(1)
			case "short value":
				initial.Value = initial.Value[:31]
			case "short rate":
				initial.Rate = initial.Rate[:31]
			case "bad value":
				initial.Value[1] = complex(math.NaN(), 0)
			case "bad rate":
				initial.Rate[1] = complex(0, math.Inf(1))
			}
			field, err := New(shape, frequency, initial)
			if err == nil || field != nil {
				t.Fatal("invalid wave input accepted")
			}
		})
	}
}

// anti-test of TestWaveNewRefusesMalformedInput
// Valid generated complex fields remain finite and preserve initial copies.
func TestWaveGeneratedStatesHaveReversibleEvolution(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		initial := waveState()
		frequency := waveFrequencies()
		scale := rapid.Float64Range(-2, 2).Draw(t, "scale")
		time := rapid.Float64Range(-2, 2).Draw(t, "time")
		for i := range initial.Value {
			initial.Value[i] *= complex(scale, 0)
			initial.Rate[i] *= complex(scale, 0)
		}
		field, err := New(waveShape(), frequency, initial)
		if err != nil || field == nil {
			t.Fatalf("New: %v", err)
		}
		state, err := field.At(time)
		if err != nil {
			t.Fatal(err)
		}
		next, err := New(waveShape(), frequency, state)
		if err != nil || next == nil {
			t.Fatalf("restart: %v", err)
		}
		back, err := next.At(-time)
		if err != nil {
			t.Fatal(err)
		}
		if len(back.Value) != 32 || len(back.Rate) != 32 {
			t.Fatal("missing state")
		}
		for i := range initial.Value {
			if cmplx.Abs(back.Value[i]-initial.Value[i]) > 1e-10 || cmplx.Abs(back.Rate[i]-initial.Rate[i]) > 1e-10 {
				t.Fatalf("failed reverse at %d", i)
			}
		}
	})
}

// Invalid and overflowing requests return no partial state or mutation.
func TestWaveAtRefusesNonFiniteAndOverflowingEvolution(t *testing.T) {
	field := openWave(t, waveFrequencies(), waveState())
	for _, time := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), math.MaxFloat64} {
		got, err := field.At(time)
		if err == nil || got.Value != nil || got.Rate != nil {
			t.Fatalf("accepted time %g", time)
		}
	}
	initial := waveState()
	for i := range initial.Rate {
		initial.Rate[i] = 2
	}
	drifting := openWave(t, make([]float64, 32), initial)
	got, err := drifting.At(math.MaxFloat64)
	if err == nil || got.Value != nil || got.Rate != nil {
		t.Fatal("overflowing drift accepted")
	}
	for _, empty := range []*Field{nil, {}} {
		got, err := empty.At(0)
		if err == nil || got.Value != nil || got.Rate != nil {
			t.Fatal("uninitialized field accepted")
		}
	}
}

// anti-test of TestWaveAtRefusesNonFiniteAndOverflowingEvolution
// Refusals leave the exact initial state and later finite evolution intact.
func TestWaveRefusalsDoNotChangeInitialState(t *testing.T) {
	initial := waveState()
	field := openWave(t, waveFrequencies(), initial)
	_, _ = field.At(math.Inf(1))
	_, _ = field.At(math.MaxFloat64)
	got, err := field.At(0)
	if err != nil {
		t.Fatal(err)
	}
	requireWave(t, got, initial)
	got, err = field.At(0.1)
	if err != nil || len(got.Value) != 32 || len(got.Rate) != 32 {
		t.Fatalf("finite propagation after refusal: %v", err)
	}
}

// Large finite phases need accurate argument reduction, rather than a
// remainder formed with a rounded two-pi constant.
func TestWaveLargePhaseMatchesAnalyticSineCosine(t *testing.T) {
	initial := State{Value: make([]complex128, 32), Rate: make([]complex128, 32)}
	frequency := make([]float64, 32)
	for i := range frequency {
		frequency[i] = 1
		initial.Value[i] = 1
	}
	field := openWave(t, frequency, initial)
	got, err := field.At(1e16)
	if err != nil {
		t.Fatal(err)
	}
	want := State{Value: make([]complex128, 32), Rate: make([]complex128, 32)}
	sine, cosine := math.Sincos(1e16)
	for i := range want.Value {
		want.Value[i] = complex(cosine, 0)
		want.Rate[i] = complex(-sine, 0)
	}
	requireWave(t, got, want)
}

// anti-test of TestWaveLargePhaseMatchesAnalyticSineCosine
// Frequency storage must not alias a caller-owned table after construction.
func TestWaveFrequencyTableIsCopied(t *testing.T) {
	frequency := waveFrequencies()
	field := openWave(t, frequency, waveState())
	want, err := field.At(0.25)
	if err != nil {
		t.Fatal(err)
	}
	for i := range frequency {
		frequency[i] = 99
	}
	got, err := field.At(0.25)
	if err != nil {
		t.Fatal(err)
	}
	requireWave(t, got, want)
}
