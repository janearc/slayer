package flow

import (
	"math"
	"math/cmplx"
	"strconv"
	"testing"

	"github.com/janearc/slayer/simulation/spectral"
	"pgregory.net/rapid"
)

func testConfig() Config {
	return Config{
		Shape:          spectral.Shape{Width: 8, Height: 8, MaxCells: 64},
		WidthMetres:    8,
		HeightMetres:   8,
		Viscosity:      0.1,
		MaxStepSeconds: 0.01,
	}
}

func field(test *testing.T, got Velocity) Velocity {
	test.Helper()
	if len(got.X) != 64 || len(got.Y) != 64 {
		test.Fatalf("velocity has component lengths %d and %d, want 64 each",
			len(got.X), len(got.Y))
	}
	return got
}

func makeFlow(test *testing.T, config Config, initial Velocity) *Flow {
	test.Helper()
	current, err := New(config, initial)
	if err != nil {
		test.Fatalf("New: %v", err)
	}
	if current == nil {
		test.Fatal("New returned a nil flow without an error")
	}
	return current
}

func mean(values []float64) float64 {
	var sum float64
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func kineticEnergy(velocity Velocity) float64 {
	var energy float64
	for index := range velocity.X {
		energy += 0.5 * (velocity.X[index]*velocity.X[index] +
			velocity.Y[index]*velocity.Y[index])
	}
	return energy / float64(len(velocity.X))
}

func expectedGuardNumber(config Config, velocity Velocity, seconds float64) float64 {
	maxX, maxY := 0.0, 0.0
	for index := range velocity.X {
		maxX = math.Max(maxX, math.Abs(velocity.X[index]))
		maxY = math.Max(maxY, math.Abs(velocity.Y[index]))
	}
	modeX := float64(config.Shape.Width / 3)
	modeY := float64(config.Shape.Height / 3)
	kx := 2 * math.Pi * modeX / config.WidthMetres
	ky := 2 * math.Pi * modeY / config.HeightMetres
	return seconds * (maxX*kx + maxY*ky +
		config.Viscosity*(kx*kx+ky*ky))
}

func checkClose(test *testing.T, got, want, tolerance float64) {
	test.Helper()
	if math.IsNaN(got) || math.IsInf(got, 0) ||
		math.Abs(got-want) > tolerance*math.Max(1, math.Abs(want)) {
		test.Errorf("got %.16g, want %.16g within %.3g", got, want, tolerance)
	}
}

func spectralDivergenceL2(test *testing.T, config Config, velocity Velocity) float64 {
	test.Helper()
	count := config.Shape.Width * config.Shape.Height
	u := make([]complex128, count)
	v := make([]complex128, count)
	for index := 0; index < count; index++ {
		u[index] = complex(velocity.X[index], 0)
		v[index] = complex(velocity.Y[index], 0)
	}
	ux, err := spectral.Forward(config.Shape, u)
	if err != nil {
		test.Fatalf("Forward(u): %v", err)
	}
	vy, err := spectral.Forward(config.Shape, v)
	if err != nil {
		test.Fatalf("Forward(v): %v", err)
	}
	if len(ux) != count || len(vy) != count {
		test.Fatalf("spectral transforms returned lengths %d and %d, want %d",
			len(ux), len(vy), count)
	}
	for y := 0; y < config.Shape.Height; y++ {
		ky := y
		if ky > config.Shape.Height/2 {
			ky -= config.Shape.Height
		} else if ky == config.Shape.Height/2 {
			ky = 0 // the real-valued Nyquist derivative is zero
		}
		for x := 0; x < config.Shape.Width; x++ {
			kx := x
			if kx > config.Shape.Width/2 {
				kx -= config.Shape.Width
			} else if kx == config.Shape.Width/2 {
				kx = 0 // the real-valued Nyquist derivative is zero
			}
			index := y*config.Shape.Width + x
			waveX := 2 * math.Pi * float64(kx) / config.WidthMetres
			waveY := 2 * math.Pi * float64(ky) / config.HeightMetres
			ux[index] *= complex(0, waveX)
			vy[index] *= complex(0, waveY)
		}
	}
	for index := range ux {
		ux[index] += vy[index]
	}
	divergence, err := spectral.Inverse(config.Shape, ux)
	if err != nil {
		test.Fatalf("Inverse(divergence): %v", err)
	}
	if len(divergence) != count {
		test.Fatalf("inverse transform returned %d values, want %d",
			len(divergence), count)
	}
	var square float64
	for _, value := range divergence {
		square += cmplx.Abs(value) * cmplx.Abs(value)
	}
	return math.Sqrt(square / float64(count))
}

// anti-test of TestNewRefusesInvalidConfigurationAndVelocity
func TestNewProjectsToSolenoidalFieldAndPreservesMeans(test *testing.T) {
	config := testConfig()
	initial := Velocity{X: make([]float64, 64), Y: make([]float64, 64)}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			index := y*8 + x
			initial.X[index] = 0.7 + math.Sin(2*math.Pi*float64(x)/8)
			initial.Y[index] = -0.2 + math.Cos(2*math.Pi*float64(y)/8)
		}
	}
	before := Velocity{
		X: append([]float64(nil), initial.X...),
		Y: append([]float64(nil), initial.Y...),
	}
	current := makeFlow(test, config, initial)
	projected := field(test, current.Velocity())
	checkClose(test, mean(projected.X), mean(before.X), 1e-12)
	checkClose(test, mean(projected.Y), mean(before.Y), 1e-12)
	if divergence := spectralDivergenceL2(test, config, projected); divergence > 1e-10 {
		test.Errorf("projected divergence RMS is %g, want at most 1e-10", divergence)
	}
	for index := range before.X {
		if initial.X[index] != before.X[index] || initial.Y[index] != before.Y[index] {
			test.Fatalf("New changed input velocity at cell %d", index)
		}
	}
}

// anti-test of TestNewProjectsToSolenoidalFieldAndPreservesMeans
// This oracle input must retain nonzero divergence before projection.
func TestProjectionOracleDetectsUnprojectedInput(test *testing.T) {
	config := testConfig()
	velocity := Velocity{X: make([]float64, 64), Y: make([]float64, 64)}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			velocity.X[y*8+x] = math.Sin(2 * math.Pi * float64(x) / 8)
		}
	}
	if divergence := spectralDivergenceL2(test, config, velocity); divergence < 0.1 {
		test.Fatalf("raw field divergence RMS is %g, want a detectable divergence",
			divergence)
	}
}

// anti-test of TestStepRefusesInvalidIntervalsWithoutChangingState
func TestUniformFlowRemainsUniform(test *testing.T) {
	config := testConfig()
	initial := Velocity{X: make([]float64, 64), Y: make([]float64, 64)}
	for index := range initial.X {
		initial.X[index], initial.Y[index] = 1.25, -0.75
	}
	current := makeFlow(test, config, initial)
	if _, err := current.Step(0.01); err != nil {
		test.Fatal(err)
	}
	got := field(test, current.Velocity())
	for index := range got.X {
		checkClose(test, got.X[index], 1.25, 1e-12)
		checkClose(test, got.Y[index], -0.75, 1e-12)
	}
	checkClose(test, current.Diagnostics().TimeSeconds, 0.01, 1e-12)
}

func TestViscousFourierModeDecaysAnalytically(test *testing.T) {
	config := testConfig()
	config.WidthMetres = 4
	config.HeightMetres = 4
	config.Viscosity = 0.2
	config.MaxStepSeconds = 0.01
	initial := Velocity{X: make([]float64, 64), Y: make([]float64, 64)}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			initial.X[y*8+x] = math.Sin(2 * math.Pi * float64(y) / 8)
		}
	}
	current := makeFlow(test, config, initial)
	for range 10 {
		if _, err := current.Step(0.01); err != nil {
			test.Fatal(err)
		}
	}
	got := field(test, current.Velocity())
	decay := math.Exp(-config.Viscosity * math.Pow(2*math.Pi/config.HeightMetres, 2) * 0.1)
	for y := 0; y < 8; y++ {
		want := decay * math.Sin(2*math.Pi*float64(y)/8)
		for x := 0; x < 8; x++ {
			checkClose(test, got.X[y*8+x], want, 2e-4)
			checkClose(test, got.Y[y*8+x], 0, 2e-4)
		}
	}
}

func TestTaylorGreenVortexDecaysAnalytically(test *testing.T) {
	config := testConfig()
	initial := Velocity{X: make([]float64, 64), Y: make([]float64, 64)}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			phaseX, phaseY := 2*math.Pi*float64(x)/8, 2*math.Pi*float64(y)/8
			initial.X[y*8+x] = math.Sin(phaseX) * math.Cos(phaseY)
			initial.Y[y*8+x] = -math.Cos(phaseX) * math.Sin(phaseY)
		}
	}
	current := makeFlow(test, config, initial)
	for range 10 {
		if _, err := current.Step(0.01); err != nil {
			test.Fatal(err)
		}
	}
	got := field(test, current.Velocity())
	decay := math.Exp(-2 * config.Viscosity * math.Pow(2*math.Pi/8, 2) * 0.1)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			phaseX, phaseY := 2*math.Pi*float64(x)/8, 2*math.Pi*float64(y)/8
			checkClose(test, got.X[y*8+x], decay*math.Sin(phaseX)*math.Cos(phaseY), 3e-4)
			checkClose(test, got.Y[y*8+x], -decay*math.Cos(phaseX)*math.Sin(phaseY), 3e-4)
		}
	}
}

func checkGalileanTaylorGreen(test *testing.T, meanX, meanY float64) {
	test.Helper()
	config := testConfig()
	initial := Velocity{X: make([]float64, 64), Y: make([]float64, 64)}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			phaseX, phaseY := 2*math.Pi*float64(x)/8, 2*math.Pi*float64(y)/8
			initial.X[y*8+x] = meanX + math.Sin(phaseX)*math.Cos(phaseY)
			initial.Y[y*8+x] = meanY - math.Cos(phaseX)*math.Sin(phaseY)
		}
	}
	current := makeFlow(test, config, initial)
	for range 10 {
		if _, err := current.Step(0.01); err != nil {
			test.Fatal(err)
		}
	}
	got := field(test, current.Velocity())
	time := current.Diagnostics().TimeSeconds
	decay := math.Exp(-2 * config.Viscosity * math.Pow(2*math.Pi/8, 2) * time)
	shiftX := 2 * math.Pi * meanX * time / config.WidthMetres
	shiftY := 2 * math.Pi * meanY * time / config.HeightMetres
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			phaseX := 2*math.Pi*float64(x)/8 - shiftX
			phaseY := 2*math.Pi*float64(y)/8 - shiftY
			checkClose(test, got.X[y*8+x], meanX+
				decay*math.Sin(phaseX)*math.Cos(phaseY), 5e-4)
			checkClose(test, got.Y[y*8+x], meanY-
				decay*math.Cos(phaseX)*math.Sin(phaseY), 5e-4)
		}
	}
}

func TestGalileanTranslatedTaylorGreenVortex(test *testing.T) {
	checkGalileanTaylorGreen(test, 0.6, -0.35)
}

// anti-test of TestGalileanTranslatedTaylorGreenVortex
func TestGalileanTranslationWithOppositeMeanVelocity(test *testing.T) {
	checkGalileanTaylorGreen(test, -0.6, 0.35)
}

func TestPositiveViscosityDoesNotIncreaseKineticEnergy(test *testing.T) {
	config := testConfig()
	initial := Velocity{X: make([]float64, 64), Y: make([]float64, 64)}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			index := y*8 + x
			initial.X[index] = math.Sin(2*math.Pi*float64(x)/8) +
				0.25*math.Cos(2*math.Pi*float64(y)/8)
			initial.Y[index] = math.Cos(2*math.Pi*float64(y)/8) -
				0.25*math.Sin(2*math.Pi*float64(x)/8)
		}
	}
	current := makeFlow(test, config, initial)
	velocity := field(test, current.Velocity())
	before := kineticEnergy(velocity)
	for range 10 {
		if _, err := current.Step(0.01); err != nil {
			test.Fatal(err)
		}
	}
	after := current.Diagnostics().KineticEnergy
	if after > before+1e-10 {
		test.Errorf("kinetic energy rose from %g to %g with positive viscosity",
			before, after)
	}
}

// anti-test of TestNewRefusesFiniteVelocityWhoseEnergyOverflows
func TestConstructionAndVelocityAccessUseIndependentCopies(test *testing.T) {
	config := testConfig()
	initial := Velocity{X: make([]float64, 64), Y: make([]float64, 64)}
	for index := range initial.X {
		initial.X[index], initial.Y[index] = 2, -3
	}
	current := makeFlow(test, config, initial)
	initial.X[0], initial.Y[0] = 100, 100
	first := field(test, current.Velocity())
	checkClose(test, first.X[0], 2, 1e-12)
	checkClose(test, first.Y[0], -3, 1e-12)
	first.X[0], first.Y[0] = -90, -90
	second := field(test, current.Velocity())
	checkClose(test, second.X[0], 2, 1e-12)
	checkClose(test, second.Y[0], -3, 1e-12)
	firstDiagnostics := current.Diagnostics()
	firstDiagnostics.TimeSeconds = 99
	checkClose(test, current.Diagnostics().TimeSeconds, 0, 1e-12)
}

func TestNewRefusesInvalidConfigurationAndVelocity(test *testing.T) {
	base := testConfig()
	valid := Velocity{X: make([]float64, 64), Y: make([]float64, 64)}
	invalid := []struct {
		name     string
		config   Config
		velocity Velocity
	}{
		{"zero width", func() Config { c := base; c.WidthMetres = 0; return c }(), valid},
		{"negative height", func() Config { c := base; c.HeightMetres = -1; return c }(), valid},
		{"NaN width", func() Config { c := base; c.WidthMetres = math.NaN(); return c }(), valid},
		{"infinite height", func() Config { c := base; c.HeightMetres = math.Inf(1); return c }(), valid},
		{"overflowing wavenumber", func() Config { c := base; c.WidthMetres = math.SmallestNonzeroFloat64; return c }(), valid},
		{"negative viscosity", func() Config { c := base; c.Viscosity = -0.1; return c }(), valid},
		{"NaN viscosity", func() Config { c := base; c.Viscosity = math.NaN(); return c }(), valid},
		{"zero step limit", func() Config { c := base; c.MaxStepSeconds = 0; return c }(), valid},
		{"infinite step limit", func() Config { c := base; c.MaxStepSeconds = math.Inf(1); return c }(), valid},
		{"short x", base, Velocity{X: make([]float64, 63), Y: make([]float64, 64)}},
		{"nil components", base, Velocity{}},
		{"long y", base, Velocity{X: make([]float64, 64), Y: make([]float64, 65)}},
		{"NaN x", base, Velocity{X: append([]float64{math.NaN()}, make([]float64, 63)...), Y: make([]float64, 64)}},
		{"infinite y", base, Velocity{X: make([]float64, 64), Y: append([]float64{math.Inf(-1)}, make([]float64, 63)...)}},
	}
	for _, each := range invalid {
		test.Run(each.name, func(test *testing.T) {
			got, err := New(each.config, each.velocity)
			if err == nil || got != nil {
				test.Fatalf("New returned (%v, %v), want nil flow and error", got, err)
			}
		})
	}
}

func TestNewRefusesFiniteVelocityWhoseEnergyOverflows(test *testing.T) {
	config := testConfig()
	initial := Velocity{X: make([]float64, 64), Y: make([]float64, 64)}
	for index := range initial.X {
		initial.X[index] = 1e200
	}
	got, err := New(config, initial)
	if err == nil || got != nil {
		test.Fatalf("New returned (%v, %v), want nil flow and overflow error",
			got, err)
	}
}

func TestNewRefusesInvalidSpectralShapes(test *testing.T) {
	base := testConfig()
	initial := Velocity{X: make([]float64, 64), Y: make([]float64, 64)}
	for _, shape := range []spectral.Shape{
		{Width: 0, Height: 8, MaxCells: 64},
		{Width: 3, Height: 8, MaxCells: 64},
		{Width: 8, Height: 8, MaxCells: 63},
	} {
		config := base
		config.Shape = shape
		got, err := New(config, initial)
		if err == nil || got != nil {
			test.Errorf("New(%+v) returned (%v, %v), want nil flow and error",
				shape, got, err)
		}
	}
}

func TestStepRefusesInvalidIntervalsWithoutChangingState(test *testing.T) {
	config := testConfig()
	initial := Velocity{X: make([]float64, 64), Y: make([]float64, 64)}
	for index := range initial.X {
		initial.X[index], initial.Y[index] = 0.25, -0.5
	}
	for _, interval := range []float64{0, -0.01, 0.0101, math.NaN(), math.Inf(1)} {
		test.Run(fmtInterval(interval), func(test *testing.T) {
			current := makeFlow(test, config, initial)
			before := field(test, current.Velocity())
			beforeDiagnostics := current.Diagnostics()
			got, err := current.Step(interval)
			if err == nil {
				test.Fatalf("Step(%g) returned no error", interval)
			}
			if got != beforeDiagnostics {
				test.Errorf("rejected step returned diagnostics %+v, want %+v",
					got, beforeDiagnostics)
			}
			after := field(test, current.Velocity())
			for index := range before.X {
				if before.X[index] != after.X[index] || before.Y[index] != after.Y[index] {
					test.Errorf("rejected step changed velocity at cell %d", index)
					break
				}
			}
			if current.Diagnostics() != beforeDiagnostics {
				test.Errorf("rejected step changed diagnostics from %+v to %+v",
					beforeDiagnostics, current.Diagnostics())
			}
		})
	}
}

func TestStepRefusesUnderresolvedIntervalWithoutChangingState(test *testing.T) {
	config := testConfig()
	config.MaxStepSeconds = 0.01
	initial := Velocity{X: make([]float64, 64), Y: make([]float64, 64)}
	for index := range initial.X {
		initial.X[index] = 100
	}
	current := makeFlow(test, config, initial)
	before := field(test, current.Velocity())
	beforeDiagnostics := current.Diagnostics()
	const interval = 0.005 // below MaxStepSeconds, above the picked guard
	if interval > config.MaxStepSeconds {
		test.Fatal("test interval must be below the caller's maximum")
	}
	if measure := expectedGuardNumber(config, initial, interval); measure <= 0.5 {
		test.Fatalf("test interval has guard measure %g, want above 0.5", measure)
	}
	if _, err := current.Step(interval); err == nil {
		test.Fatal("Step accepted an under-resolved interval below its maximum")
	}
	after := field(test, current.Velocity())
	for index := range before.X {
		if before.X[index] != after.X[index] || before.Y[index] != after.Y[index] {
			test.Fatalf("rejected under-resolved step changed velocity at cell %d", index)
		}
	}
	if current.Diagnostics() != beforeDiagnostics {
		test.Fatalf("rejected under-resolved step changed diagnostics: %+v -> %+v",
			beforeDiagnostics, current.Diagnostics())
	}
}

// anti-test of TestStepRefusesUnderresolvedIntervalWithoutChangingState
func TestStepAcceptsIntervalWithinPickedGuard(test *testing.T) {
	config := testConfig()
	config.MaxStepSeconds = 0.01
	initial := Velocity{X: make([]float64, 64), Y: make([]float64, 64)}
	for index := range initial.X {
		initial.X[index] = 100
	}
	current := makeFlow(test, config, initial)
	const interval = 0.001
	if measure := expectedGuardNumber(config, initial, interval); measure > 0.5 {
		test.Fatalf("test interval has guard measure %g, want at most 0.5", measure)
	}
	if _, err := current.Step(interval); err != nil {
		test.Fatalf("Step rejected an interval within the picked guard: %v", err)
	}
	checkClose(test, current.Diagnostics().TimeSeconds, interval, 1e-12)
}

func TestNilAndZeroFlowRefuseStepAndReturnZeroSnapshots(test *testing.T) {
	var nilFlow *Flow
	var zero Flow
	for name, current := range map[string]*Flow{
		"nil":  nilFlow,
		"zero": &zero,
	} {
		test.Run(name, func(test *testing.T) {
			if _, err := current.Step(0.01); err == nil {
				test.Fatal("Step returned no error for an uninitialized flow")
			}
			velocity := current.Velocity()
			if velocity.X != nil || velocity.Y != nil {
				test.Errorf("Velocity returned nonempty components: %+v", velocity)
			}
			if got := current.Diagnostics(); got != (Diagnostics{}) {
				test.Errorf("Diagnostics returned %+v, want zero value", got)
			}
		})
	}
}

func fmtInterval(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}

func TestProjectionPreservesMeansAcrossGeneratedFields(test *testing.T) {
	rapid.Check(test, func(property *rapid.T) {
		config := testConfig()
		initial := Velocity{X: make([]float64, 64), Y: make([]float64, 64)}
		for index := range initial.X {
			initial.X[index] = rapid.Float64Range(-1, 1).Draw(property, "u")
			initial.Y[index] = rapid.Float64Range(-1, 1).Draw(property, "v")
		}
		wantMeanX, wantMeanY := mean(initial.X), mean(initial.Y)
		current, err := New(config, initial)
		if err != nil || current == nil {
			property.Fatalf("New returned (%v, %v)", current, err)
		}
		got := current.Velocity()
		if len(got.X) != 64 || len(got.Y) != 64 {
			property.Fatalf("velocity has component lengths %d and %d",
				len(got.X), len(got.Y))
		}
		for index := range got.X {
			for label, value := range map[string]float64{
				"u": got.X[index], "v": got.Y[index],
			} {
				if math.IsNaN(value) || math.IsInf(value, 0) {
					property.Fatalf("projected %s at cell %d is non-finite: %g",
						label, index, value)
				}
			}
		}
		diagnostics := current.Diagnostics()
		for label, value := range map[string]float64{
			"time":           diagnostics.TimeSeconds,
			"divergence":     diagnostics.DivergenceL2,
			"kinetic energy": diagnostics.KineticEnergy,
		} {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				property.Fatalf("%s diagnostic is non-finite: %g", label, value)
			}
		}
		if difference := math.Abs(mean(got.X) - wantMeanX); difference >
			1e-10*math.Max(1, math.Abs(wantMeanX)) {
			property.Fatalf("projected x mean differs by %g", difference)
		}
		if difference := math.Abs(mean(got.Y) - wantMeanY); difference >
			1e-10*math.Max(1, math.Abs(wantMeanY)) {
			property.Fatalf("projected y mean differs by %g", difference)
		}
		count := config.Shape.Width * config.Shape.Height
		u, v := make([]complex128, count), make([]complex128, count)
		for index := range got.X {
			u[index] = complex(got.X[index], 0)
			v[index] = complex(got.Y[index], 0)
		}
		ux, err := spectral.Forward(config.Shape, u)
		if err != nil || len(ux) != count {
			property.Fatalf("Forward(u) returned %d values and error %v", len(ux), err)
		}
		vy, err := spectral.Forward(config.Shape, v)
		if err != nil || len(vy) != count {
			property.Fatalf("Forward(v) returned %d values and error %v", len(vy), err)
		}
		for y := 0; y < config.Shape.Height; y++ {
			ky := y
			if ky > config.Shape.Height/2 {
				ky -= config.Shape.Height
			} else if ky == config.Shape.Height/2 {
				ky = 0
			}
			for x := 0; x < config.Shape.Width; x++ {
				kx := x
				if kx > config.Shape.Width/2 {
					kx -= config.Shape.Width
				} else if kx == config.Shape.Width/2 {
					kx = 0
				}
				index := y*config.Shape.Width + x
				ux[index] *= complex(0,
					2*math.Pi*float64(kx)/config.WidthMetres)
				vy[index] *= complex(0,
					2*math.Pi*float64(ky)/config.HeightMetres)
			}
		}
		for index := range ux {
			ux[index] += vy[index]
		}
		divergence, err := spectral.Inverse(config.Shape, ux)
		if err != nil || len(divergence) != count {
			property.Fatalf("Inverse(divergence) returned %d values and error %v",
				len(divergence), err)
		}
		var square float64
		for _, value := range divergence {
			square += cmplx.Abs(value) * cmplx.Abs(value)
		}
		rms := math.Sqrt(square / float64(count))
		if math.IsNaN(rms) || math.IsInf(rms, 0) || rms > 1e-9 {
			property.Fatalf("projected divergence RMS is %g", rms)
		}
	})
}
