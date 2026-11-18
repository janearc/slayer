package waves

import (
	"math"
	"math/cmplx"
	"slices"
	"testing"
)

// A finite displacement can scale an underflowed acceleration coefficient
// back into a representable rate.
func TestWaveRetainsTinyAccelerationContribution(t *testing.T) {
	initial := State{Value: make([]complex128, 32), Rate: make([]complex128, 32)}
	frequency := make([]float64, 32)
	const displacement = 1e306
	for i := range initial.Value {
		initial.Value[i] = complex(displacement, 0)
		frequency[i] = 1e-200
	}
	field := openWave(t, frequency, initial)
	const seconds = 1.0
	got, err := field.At(seconds)
	if err != nil {
		t.Fatal(err)
	}
	phase := frequency[0] * seconds
	wantRate := -(displacement * frequency[0]) * math.Sin(phase)
	if wantRate == 0 || math.IsNaN(wantRate) || math.IsInf(wantRate, 0) {
		t.Fatalf("expected rate is not finite and nonzero: %g", wantRate)
	}
	for i, rate := range got.Rate {
		if math.IsNaN(real(rate)) || math.IsInf(real(rate), 0) ||
			math.IsNaN(imag(rate)) || math.IsInf(imag(rate), 0) {
			t.Fatalf("rate %d is non-finite: %v", i, rate)
		}
		relativeError := cmplx.Abs(rate-complex(wantRate, 0)) / math.Abs(wantRate)
		if relativeError > 2e-11 {
			t.Fatalf("rate %d is %v, want %g (relative error %g)", i, rate, wantRate, relativeError)
		}
	}
	zero, err := field.At(0)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(zero.Value, initial.Value) || !slices.Equal(zero.Rate, initial.Rate) {
		t.Fatal("query changed the stored initial state")
	}
}

// A phase that rounds to zero still has a representable rate contribution
// after scaling the small-angle limit by a large finite displacement.
func TestWaveRetainsRateWhenPhaseUnderflows(t *testing.T) {
	initial := State{Value: make([]complex128, 32), Rate: make([]complex128, 32)}
	frequency := make([]float64, 32)
	const (
		displacement = 1e306
		seconds      = 1e-200
	)
	for i := range initial.Value {
		initial.Value[i] = complex(displacement, 0)
		frequency[i] = 1e-200
	}
	field := openWave(t, frequency, initial)
	got, err := field.At(seconds)
	if err != nil {
		t.Fatal(err)
	}
	wantRate := -((displacement * frequency[0]) * frequency[0]) * seconds
	if wantRate == 0 || math.IsNaN(wantRate) || math.IsInf(wantRate, 0) {
		t.Fatalf("expected rate is not finite and nonzero: %g", wantRate)
	}
	for i, rate := range got.Rate {
		if math.IsNaN(real(rate)) || math.IsInf(real(rate), 0) ||
			math.IsNaN(imag(rate)) || math.IsInf(imag(rate), 0) {
			t.Fatalf("rate %d is non-finite: %v", i, rate)
		}
		relativeError := cmplx.Abs(rate-complex(wantRate, 0)) / math.Abs(wantRate)
		if relativeError > 2e-11 {
			t.Fatalf("rate %d is %v, want %g (relative error %g)", i, rate, wantRate, relativeError)
		}
	}
	zero, err := field.At(0)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(zero.Value, initial.Value) || !slices.Equal(zero.Rate, initial.Rate) {
		t.Fatal("query changed the stored initial state")
	}
}

// A nonzero subnormal phase still needs accurate scaling of the acceleration
// term by a large finite displacement.
func TestWaveRetainsRateAtSubnormalPhase(t *testing.T) {
	initial := State{Value: make([]complex128, 32), Rate: make([]complex128, 32)}
	frequency := make([]float64, 32)
	const displacement = 1e290
	for i := range initial.Value {
		initial.Value[i] = complex(displacement, 0)
		frequency[i] = 100000000.5
	}
	field := openWave(t, frequency, initial)
	seconds := math.SmallestNonzeroFloat64
	phase := frequency[0] * seconds
	if phase == 0 || math.Abs(phase) >= 0x1p-1022 {
		t.Fatalf("test phase is %g, want nonzero subnormal", phase)
	}
	got, err := field.At(seconds)
	if err != nil {
		t.Fatal(err)
	}
	wantRate := -((displacement * seconds) * frequency[0]) * frequency[0]
	if wantRate == 0 || math.IsNaN(wantRate) || math.IsInf(wantRate, 0) {
		t.Fatalf("expected rate is not finite and nonzero: %g", wantRate)
	}
	for i, rate := range got.Rate {
		if math.IsNaN(real(rate)) || math.IsInf(real(rate), 0) ||
			math.IsNaN(imag(rate)) || math.IsInf(imag(rate), 0) {
			t.Fatalf("rate %d is non-finite: %v", i, rate)
		}
		relativeError := cmplx.Abs(rate-complex(wantRate, 0)) / math.Abs(wantRate)
		if relativeError > 2e-11 {
			t.Fatalf("rate %d is %v, want %g (relative error %g)", i, rate, wantRate, relativeError)
		}
	}
	zero, err := field.At(0)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(zero.Value, initial.Value) || !slices.Equal(zero.Rate, initial.Rate) {
		t.Fatal("query changed the stored initial state")
	}
}

// A finite rate can scale an underflowed drift coefficient back into a
// representable displacement.
func TestWaveRetainsTinyDriftContribution(t *testing.T) {
	initial := State{Value: make([]complex128, 32), Rate: make([]complex128, 32)}
	frequency := make([]float64, 32)
	const initialRate = 1e306
	for i := range initial.Rate {
		initial.Rate[i] = complex(initialRate, 0)
		frequency[i] = 1e308
	}
	field := openWave(t, frequency, initial)
	seconds := math.Pi / frequency[0]
	phase := frequency[0] * seconds
	got, err := field.At(seconds)
	if err != nil {
		t.Fatal(err)
	}
	wantValue := (initialRate / frequency[0]) * math.Sin(phase)
	if wantValue == 0 || math.IsNaN(wantValue) || math.IsInf(wantValue, 0) {
		t.Fatalf("expected displacement is not finite and nonzero: %g", wantValue)
	}
	for i, value := range got.Value {
		if math.IsNaN(real(value)) || math.IsInf(real(value), 0) ||
			math.IsNaN(imag(value)) || math.IsInf(imag(value), 0) {
			t.Fatalf("value %d is non-finite: %v", i, value)
		}
		relativeError := cmplx.Abs(value-complex(wantValue, 0)) / math.Abs(wantValue)
		if relativeError > 2e-11 {
			t.Fatalf("value %d is %v, want %g (relative error %g)", i, value, wantValue, relativeError)
		}
	}
	zero, err := field.At(0)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(zero.Value, initial.Value) || !slices.Equal(zero.Rate, initial.Rate) {
		t.Fatal("query changed the stored initial state")
	}
}
