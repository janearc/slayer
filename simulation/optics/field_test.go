package optics

import (
	"math"
	"math/cmplx"
	"testing"

	"github.com/janearc/slayer/simulation/spectral"
	"pgregory.net/rapid"
)

func config(width, height int, widthMetres, heightMetres, wavelength float64) Config {
	return Config{
		Shape:       spectral.Shape{Width: width, Height: height, MaxCells: 256},
		WidthMetres: widthMetres, HeightMetres: heightMetres,
		WavelengthMetres: wavelength,
	}
}

func checkComplexes(test *testing.T, got, want []complex128, tolerance float64) {
	test.Helper()
	if len(got) != len(want) {
		test.Fatalf("got %d samples, want %d", len(got), len(want))
	}
	for index := range want {
		if !finiteComplex(got[index]) {
			test.Errorf("sample %d is non-finite: %v", index, got[index])
			continue
		}
		limit := tolerance * math.Max(1, cmplx.Abs(want[index]))
		if difference := cmplx.Abs(got[index] - want[index]); difference > limit {
			test.Errorf("sample %d is %v, want %v (difference %g)",
				index, got[index], want[index], difference)
		}
	}
}

func finiteComplex(value complex128) bool {
	return finite(real(value)) && finite(imag(value))
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func mode(grid spectral.Shape, xMode, yMode int, sign float64) []complex128 {
	values := make([]complex128, grid.Width*grid.Height)
	for y := range grid.Height {
		for x := range grid.Width {
			turns := float64(xMode*x)/float64(grid.Width) +
				float64(yMode*y)/float64(grid.Height)
			values[y*grid.Width+x] = cmplx.Exp(complex(0, sign*2*math.Pi*turns))
		}
	}
	return values
}

func TestAtZeroReturnsExactIndependentCopy(test *testing.T) {
	cfg := config(4, 2, 1, 0.5, 0.1)
	input := []complex128{1 + 2i, 3 - 4i, -2 + 5i, 7, 8i, -9, 2 - 3i, 4 + 6i}
	want := append([]complex128(nil), input...)
	field, err := New(cfg, input)
	if err != nil {
		test.Fatal(err)
	}
	input[0] = 99
	got, err := field.At(0)
	if err != nil {
		test.Fatal(err)
	}
	if len(got) != len(want) {
		test.Fatalf("got %d samples, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			test.Errorf("sample %d is %v, want exact copy %v", index, got[index], want[index])
		}
	}
	got[0] = -100
	again, err := field.At(0)
	if err != nil {
		test.Fatal(err)
	}
	checkComplexes(test, again, want, 0)
}

func TestPlaneWaveRemainsConstantAfterCarrierIsFactored(test *testing.T) {
	cfg := config(4, 2, 1, 0.5, 0.1)
	input := make([]complex128, cfg.Shape.Width*cfg.Shape.Height)
	for index := range input {
		input[index] = complex(1.25, -0.75)
	}
	field, err := New(cfg, input)
	if err != nil {
		test.Fatal(err)
	}
	got, err := field.At(123.5)
	if err != nil {
		test.Fatal(err)
	}
	checkComplexes(test, got, input, 1e-12)
}

func TestPropagatingFourierModeGetsItsRelativePhase(test *testing.T) {
	cfg := config(4, 2, 1, 0.5, 0.2)
	input := mode(cfg.Shape, 1, 0, 1)
	field, err := New(cfg, input)
	if err != nil {
		test.Fatal(err)
	}
	const distance = 0.4
	wavelengthNumber := 2 * math.Pi / cfg.WavelengthMetres
	kx := 2 * math.Pi / cfg.WidthMetres
	kz := math.Sqrt(wavelengthNumber*wavelengthNumber - kx*kx)
	phase := (kz - wavelengthNumber) * distance
	wantFactor := cmplx.Exp(complex(0, phase))
	want := make([]complex128, len(input))
	for index := range input {
		want[index] = input[index] * wantFactor
	}
	got, err := field.At(distance)
	if err != nil {
		test.Fatal(err)
	}
	checkComplexes(test, got, want, 1e-10)
}

// anti-test of TestPropagatingFourierModeGetsItsRelativePhase
// An evanescent mode decays and keeps its relative carrier phase.
func TestEvanescentModeGetsDecayAndRelativePhase(test *testing.T) {
	cfg := config(4, 2, 0.5, 0.5, 0.5)
	input := mode(cfg.Shape, 2, 0, 1)
	field, err := New(cfg, input)
	if err != nil {
		test.Fatal(err)
	}
	const distance = 0.01
	wavelengthNumber := 2 * math.Pi / cfg.WavelengthMetres
	kx := 2 * math.Pi * 2 / cfg.WidthMetres
	decay := math.Sqrt(kx*kx - wavelengthNumber*wavelengthNumber)
	wantFactor := complex(math.Exp(-decay*distance), 0) *
		cmplx.Exp(complex(0, -wavelengthNumber*distance))
	want := make([]complex128, len(input))
	for index := range input {
		want[index] = input[index] * wantFactor
	}
	got, err := field.At(distance)
	if err != nil {
		test.Fatal(err)
	}
	checkComplexes(test, got, want, 1e-10)
}

func TestSmallAnglePhaseAvoidsSubtractiveCancellation(test *testing.T) {
	cfg := config(4, 2, 1e9, 1e9, 0.5)
	input := mode(cfg.Shape, 1, 0, 1)
	field, err := New(cfg, input)
	if err != nil {
		test.Fatal(err)
	}
	const distance = 1e15
	k := 2 * math.Pi / cfg.WavelengthMetres
	kx := 2 * math.Pi / cfg.WidthMetres
	// The paraxial term differs from the exact phase by less than 1e-20 rad
	// here; its independent expression stays resolved when sqrt(k*k-kx*kx)
	// rounds to k in float64.
	phase := -(kx * kx) * distance / (2 * k)
	factor := cmplx.Exp(complex(0, phase))
	want := make([]complex128, len(input))
	for index := range input {
		want[index] = input[index] * factor
	}
	got, err := field.At(distance)
	if err != nil {
		test.Fatal(err)
	}
	checkComplexes(test, got, want, 1e-10)
}

func TestPropagationHasTheSemigroupProperty(test *testing.T) {
	cfg := config(4, 2, 1, 0.75, 0.2)
	input := mode(cfg.Shape, 1, 1, 1)
	field, err := New(cfg, input)
	if err != nil {
		test.Fatal(err)
	}
	first, err := field.At(0.1)
	if err != nil {
		test.Fatal(err)
	}
	stepped, err := New(cfg, first)
	if err != nil {
		test.Fatal(err)
	}
	stepThen, err := stepped.At(0.2)
	if err != nil {
		test.Fatal(err)
	}
	direct, err := field.At(0.3)
	if err != nil {
		test.Fatal(err)
	}
	if len(stepThen) != len(input) || len(direct) != len(input) {
		test.Fatalf("propagation returned %d and %d samples, want %d", len(stepThen), len(direct), len(input))
	}
	checkComplexes(test, stepThen, direct, 1e-10)
}

func TestIntensityIsPerSampleSquaredMagnitude(test *testing.T) {
	got, err := Intensity([]complex128{3 + 4i, -2i, 1 - 2i})
	if err != nil {
		test.Fatal(err)
	}
	want := []float64{25, 4, 5}
	if len(got) != len(want) {
		test.Fatalf("got %d values, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			test.Errorf("intensity %d is %g, want %g", index, got[index], want[index])
		}
	}
}

func TestCoherentCancellationPrecedesIntensity(test *testing.T) {
	first := []complex128{1, 1i, complex(2, -1)}
	second := []complex128{-1, -1i, complex(-2, 1)}
	coherent := make([]complex128, len(first))
	for index := range first {
		coherent[index] = first[index] + second[index]
	}
	coherentPower, err := Intensity(coherent)
	if err != nil {
		test.Fatal(err)
	}
	if len(coherentPower) != len(coherent) {
		test.Fatalf("got %d coherent intensities, want %d", len(coherentPower), len(coherent))
	}
	for index, power := range coherentPower {
		if !finite(power) || power != 0 {
			test.Errorf("coherent intensity %d is %g, want cancellation to zero", index, power)
		}
	}
}

// anti-test of TestCoherentCancellationPrecedesIntensity
// Intensities summed separately stay positive for these opposite fields.
func TestIncoherentIntensitiesDoNotCancel(test *testing.T) {
	first := []complex128{1, 1i, complex(2, -1)}
	second := []complex128{-1, -1i, complex(-2, 1)}
	firstPower, err := Intensity(first)
	if err != nil {
		test.Fatal(err)
	}
	secondPower, err := Intensity(second)
	if err != nil {
		test.Fatal(err)
	}
	if len(firstPower) != len(first) || len(secondPower) != len(second) {
		test.Fatalf("got incoherent intensity lengths %d and %d, want %d",
			len(firstPower), len(secondPower), len(first))
	}
	want := []float64{2, 2, 10}
	for index, wantPower := range want {
		if !finite(firstPower[index]) || !finite(secondPower[index]) {
			test.Errorf("incoherent intensity %d is non-finite", index)
			continue
		}
		if incoherent := firstPower[index] + secondPower[index]; incoherent != wantPower {
			test.Errorf("incoherent intensity %d is %g, want %g", index, incoherent, wantPower)
		}
	}
}

func TestIntensityRefusesNonFiniteAndOverflowingSamples(test *testing.T) {
	invalid := map[string][]complex128{
		"nan real":                   {complex(math.NaN(), 0)},
		"nan imaginary":              {complex(0, math.NaN())},
		"positive infinity":          {complex(math.Inf(1), 0)},
		"squared magnitude overflow": {complex(math.MaxFloat64/2, 0)},
	}
	for name, values := range invalid {
		test.Run(name, func(test *testing.T) {
			got, err := Intensity(values)
			if err == nil || got != nil {
				test.Fatalf("got (%v, %v), want error and nil result", got, err)
			}
		})
	}
}

func TestNewRefusesInvalidConfigurationAndSamples(test *testing.T) {
	base := config(2, 2, 1, 1, 0.5)
	valid := []complex128{1, 2, 3, 4}
	copied := append([]complex128(nil), valid...)
	longer := append(copied, 5)
	invalid := []struct {
		name    string
		config  Config
		samples []complex128
	}{
		{"zero width metres", Config{Shape: base.Shape, HeightMetres: 1, WavelengthMetres: 0.5}, valid},
		{"negative width metres", Config{Shape: base.Shape, WidthMetres: -1, HeightMetres: 1, WavelengthMetres: 0.5}, valid},
		{"nan width metres", Config{Shape: base.Shape, WidthMetres: math.NaN(), HeightMetres: 1, WavelengthMetres: 0.5}, valid},
		{"zero height metres", Config{Shape: base.Shape, WidthMetres: 1, WavelengthMetres: 0.5}, valid},
		{"negative height metres", Config{Shape: base.Shape, WidthMetres: 1, HeightMetres: -1, WavelengthMetres: 0.5}, valid},
		{"nan height metres", Config{Shape: base.Shape, WidthMetres: 1, HeightMetres: math.NaN(), WavelengthMetres: 0.5}, valid},
		{"infinite height metres", Config{Shape: base.Shape, WidthMetres: 1, HeightMetres: math.Inf(1), WavelengthMetres: 0.5}, valid},
		{"zero wavelength", Config{Shape: base.Shape, WidthMetres: 1, HeightMetres: 1, WavelengthMetres: 0}, valid},
		{"negative wavelength", Config{Shape: base.Shape, WidthMetres: 1, HeightMetres: 1, WavelengthMetres: -0.5}, valid},
		{"nan wavelength", Config{Shape: base.Shape, WidthMetres: 1, HeightMetres: 1, WavelengthMetres: math.NaN()}, valid},
		{"infinite wavelength", Config{Shape: base.Shape, WidthMetres: 1, HeightMetres: 1, WavelengthMetres: math.Inf(1)}, valid},
		{"bad spectral shape", Config{Shape: spectral.Shape{Width: 3, Height: 2, MaxCells: 8}, WidthMetres: 1, HeightMetres: 1, WavelengthMetres: 0.5}, valid},
		{"over-budget shape", Config{Shape: spectral.Shape{Width: 4, Height: 4, MaxCells: 15}, WidthMetres: 1, HeightMetres: 1, WavelengthMetres: 0.5}, make([]complex128, 16)},
		{"short samples", base, valid[:3]},
		{"long samples", base, longer},
		{"non-finite sample", base, []complex128{1, 2, 3, complex(0, math.Inf(-1))}},
	}
	for _, each := range invalid {
		test.Run(each.name, func(test *testing.T) {
			field, err := New(each.config, each.samples)
			if err == nil || field != nil {
				test.Fatalf("got (%v, %v), want error and nil field", field, err)
			}
		})
	}
}

func TestAtRefusesInvalidDistancesWithoutChangingField(test *testing.T) {
	cfg := config(2, 2, 1, 1, 0.5)
	input := []complex128{1, 2i, -3, 4 + 5i}
	field, err := New(cfg, input)
	if err != nil {
		test.Fatal(err)
	}
	for name, distance := range map[string]float64{
		"negative":          math.Nextafter(0, -1),
		"nan":               math.NaN(),
		"positive infinity": math.Inf(1),
	} {
		test.Run(name, func(test *testing.T) {
			got, err := field.At(distance)
			if err == nil || got != nil {
				test.Fatalf("got (%v, %v), want error and nil result", got, err)
			}
		})
	}
	got, err := field.At(0)
	if err != nil {
		test.Fatal(err)
	}
	checkComplexes(test, got, input, 0)
	var nilField *Field
	if got, err := nilField.At(0); err == nil || got != nil {
		test.Fatalf("nil field returned (%v, %v), want error and nil result", got, err)
	}
}

func TestParsevalHoldsForPropagatingModes(test *testing.T) {
	cfg := config(4, 4, 1, 1, 0.2)
	input := make([]complex128, cfg.Shape.Width*cfg.Shape.Height)
	for index := range input {
		sine := math.Sin(float64(index) * 0.4)
		cosine := math.Cos(float64(index) * 0.23)
		input[index] = complex(sine, cosine)
	}
	field, err := New(cfg, input)
	if err != nil {
		test.Fatal(err)
	}
	got, err := field.At(0.75)
	if err != nil {
		test.Fatal(err)
	}
	if len(got) != len(input) {
		test.Fatalf("At returned %d samples, want %d", len(got), len(input))
	}
	var inputPower, propagatedPower float64
	for _, sample := range input {
		inputPower += cmplx.Abs(sample) * cmplx.Abs(sample)
	}
	for _, sample := range got {
		if !finiteComplex(sample) {
			test.Fatal("propagated field contains a non-finite sample")
		}
		propagatedPower += cmplx.Abs(sample) * cmplx.Abs(sample)
	}
	if difference := math.Abs(inputPower - propagatedPower); difference > 1e-10*inputPower {
		test.Errorf("propagating-mode powers are %g and %g", inputPower, propagatedPower)
	}
}

func TestSmallFieldsPreserveTheirInitialSamplesProperty(test *testing.T) {
	rapid.Check(test, func(draw *rapid.T) {
		width := 1 << rapid.IntRange(1, 2).Draw(draw, "width power")
		height := 1 << rapid.IntRange(1, 2).Draw(draw, "height power")
		cfg := config(width, height, 1.25, 0.75, 0.3)
		input := make([]complex128, width*height)
		for index := range input {
			input[index] = complex(
				rapid.Float64Range(-3, 3).Draw(draw, "real"),
				rapid.Float64Range(-3, 3).Draw(draw, "imaginary"))
		}
		before := append([]complex128(nil), input...)
		field, err := New(cfg, input)
		if err != nil {
			draw.Fatalf("New: %v", err)
		}
		if field == nil {
			draw.Fatal("New returned a nil field without an error")
		}
		got, err := field.At(0)
		if err != nil {
			draw.Fatalf("At(0): %v", err)
		}
		if len(got) != len(input) {
			draw.Fatalf("At(0) returned %d samples, want %d", len(got), len(input))
		}
		for index := range input {
			if got[index] != before[index] {
				draw.Fatalf("sample %d is %v, want exact %v", index, got[index], before[index])
			}
		}
		distance := rapid.Float64Range(0, 0.1).Draw(draw, "distance")
		propagated, err := field.At(distance)
		if err != nil {
			draw.Fatalf("At(%g): %v", distance, err)
		}
		if len(propagated) != len(input) {
			draw.Fatalf("At(%g) returned %d samples, want %d",
				distance, len(propagated), len(input))
		}
		var inputPower, propagatedPower float64
		for index, sample := range input {
			inputPower += cmplx.Abs(sample) * cmplx.Abs(sample)
			if !finiteComplex(propagated[index]) {
				draw.Fatalf("propagated sample %d is non-finite", index)
			}
			propagatedPower += cmplx.Abs(propagated[index]) *
				cmplx.Abs(propagated[index])
		}
		if difference := math.Abs(inputPower - propagatedPower); difference > 1e-9*inputPower {
			draw.Fatalf("power changed from %g to %g at z=%g",
				inputPower, propagatedPower, distance)
		}
	})
}

// Large relative optical phase must use accurate range reduction.
func TestLargeRelativeOpticalPhaseMatchesAnalyticCarrier(t *testing.T) {
	conf := config(8, 4, 2*math.Pi, 2*math.Pi, 2*math.Pi)
	initial := mode(conf.Shape, 1, 0, 1)
	field, err := New(conf, initial)
	if err != nil {
		t.Fatal(err)
	}
	got, err := field.At(1e16)
	if err != nil {
		t.Fatal(err)
	}
	sine, cosine := math.Sincos(-1e16)
	want := make([]complex128, len(initial))
	for i := range want {
		want[i] = initial[i] * complex(cosine, sine)
	}
	checkComplexes(t, got, want, 2e-11)
}
