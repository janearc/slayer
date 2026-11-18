//go:build draw && darwin

package flow

import (
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/simulation/spectral"
)

// metalFlowConfig fixes physical units and a modest accepted interval.
func metalFlowConfig(size int) Config {
	return Config{Shape: spectral.Shape{Width: size, Height: size, MaxCells: size * size},
		WidthMetres: 2 * math.Pi, HeightMetres: 2 * math.Pi, Viscosity: 0.1, MaxStepSeconds: 0.001}
}

// metalFlowInitial combines two solenoidal Fourier patterns and uniform
// translation. Different wave magnitudes generate a nontrivial interaction.
func metalFlowInitial(conf Config) Velocity {
	n := conf.Shape.Width
	values := Velocity{X: make([]float64, n*n), Y: make([]float64, n*n)}
	for y := range n {
		for x := range n {
			px, py := 2*math.Pi*float64(x)/float64(n), 2*math.Pi*float64(y)/float64(n)
			values.X[y*n+x] = 0.4 + math.Sin(px)*math.Cos(py) + 0.75*math.Cos(2*px+3*py)
			values.Y[y*n+x] = -0.25 - math.Cos(px)*math.Sin(py) - 0.5*math.Cos(2*px+3*py)
		}
	}
	return values
}

func openMetalFlow(t testing.TB, conf Config, initial Velocity) *MetalFlow {
	t.Helper()
	flow, err := OpenMetal(conf, initial)
	if err != nil || flow == nil {
		t.Fatalf("OpenMetal: %v, %v", flow, err)
	}
	t.Cleanup(flow.Close)
	return flow
}

func requireMetalVelocity(t testing.TB, got, want Velocity) {
	t.Helper()
	if len(got.X) != len(want.X) || len(got.Y) != len(want.Y) || len(got.X) == 0 {
		t.Fatal("invalid velocity lengths")
	}
	var difference, norm float64
	for i := range got.X {
		for _, pair := range [][2]float64{{got.X[i], want.X[i]}, {got.Y[i], want.Y[i]}} {
			if !flowFinite(pair[0]) {
				t.Fatalf("non-finite velocity at %d", i)
			}
			difference = math.Hypot(difference, pair[0]-pair[1])
			norm = math.Hypot(norm, pair[1])
		}
	}
	if difference > 1e-4*math.Max(norm, 1e-8) {
		t.Fatalf("velocity relative L2 error %g", difference/norm)
	}
}

// Full nonlinear RK evolution agrees with the independent float64 path.
func TestMetalFlowMatchesCPUAcrossRepeatedSteps(t *testing.T) {
	for _, size := range []int{8, 32, 64} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			conf := metalFlowConfig(size)
			initial := metalFlowInitial(conf)
			cpu, err := New(conf, initial)
			if err != nil {
				t.Fatal(err)
			}
			gpu := openMetalFlow(t, conf, initial)
			for range 8 {
				want, err := cpu.Step(0.001)
				if err != nil {
					t.Fatal(err)
				}
				got, err := gpu.Step(0.001)
				if err != nil {
					t.Fatal(err)
				}
				requireMetalVelocity(t, gpu.Velocity(), cpu.Velocity())
				checkClose(t, got.TimeSeconds, want.TimeSeconds, 1e-12)
				checkClose(t, got.KineticEnergy, want.KineticEnergy, 1e-4)
				if !flowFinite(got.DivergenceL2) || got.DivergenceL2 > 3e-5 {
					t.Fatalf("divergence %g", got.DivergenceL2)
				}
			}
			actual := spectralDivergenceL2(t, conf, gpu.Velocity())
			if actual > 3e-5 {
				t.Fatalf("independent divergence %g", actual)
			}
		})
	}
}

// anti-test of TestMetalFlowMatchesCPUAcrossRepeatedSteps
// Uniform velocity must not acquire vorticity, decay or spatial variation.
func TestMetalUniformFlowPreservesMeans(t *testing.T) {
	conf := metalFlowConfig(8)
	values := metalFlowInitial(conf)
	for i := range values.X {
		values.X[i], values.Y[i] = 1.25, -0.75
	}
	gpu := openMetalFlow(t, conf, values)
	if _, err := gpu.Step(0.001); err != nil {
		t.Fatal(err)
	}
	got := gpu.Velocity()
	requireMetalVelocity(t, got, values)
	checkClose(t, mean(got.X), 1.25, 1e-7)
	checkClose(t, mean(got.Y), -0.75, 1e-7)
}

// Refused intervals preserve every accepted snapshot, and a later valid
// interval still evolves the flow.
func TestMetalFlowRefusedStepsAreTransactional(t *testing.T) {
	conf := metalFlowConfig(8)
	initial := metalFlowInitial(conf)
	gpu := openMetalFlow(t, conf, initial)
	before, diag := gpu.Velocity(), gpu.Diagnostics()
	for _, dt := range []float64{0, -1, math.NaN(), math.Inf(1), 0.002, 1e-100, 1e-45} {
		got, err := gpu.Step(dt)
		if err == nil || got != diag {
			t.Fatalf("refused step %g changed diagnostics: %+v, %v", dt, got, err)
		}
		after := gpu.Velocity()
		if !slices.Equal(before.X, after.X) || !slices.Equal(before.Y, after.Y) || gpu.Diagnostics() != diag {
			t.Fatal("refused step changed state")
		}
	}
	if _, err := gpu.Step(0.001); err != nil {
		t.Fatal(err)
	}
	after := gpu.Velocity()
	if slices.Equal(before.X, after.X) && slices.Equal(before.Y, after.Y) {
		t.Fatal("valid step failed to evolve fixture")
	}
}

// anti-test of TestMetalFlowRefusedStepsAreTransactional
// The guard also applies when a large uniform translation is hidden in
// an otherwise small interval permitted by the caller maximum.
func TestMetalFlowGuardCannotBeBypassed(t *testing.T) {
	conf := metalFlowConfig(8)
	values := metalFlowInitial(conf)
	for i := range values.X {
		values.X[i] = 1000
		values.Y[i] = 0
	}
	gpu := openMetalFlow(t, conf, values)
	before := gpu.Diagnostics()
	if got, err := gpu.Step(0.001); err == nil || got != before {
		t.Fatal("Metal bypassed the timestep guard")
	}
	if got := gpu.Velocity(); !slices.Equal(got.X, values.X) {
		t.Fatal("guard refusal changed velocity")
	}
}

// Closing refuses further work but keeps snapshots for diagnosis.
func TestMetalFlowClosedNilAndZeroRefuseSteps(t *testing.T) {
	conf := metalFlowConfig(8)
	closed := openMetalFlow(t, conf, metalFlowInitial(conf))
	before, diag := closed.Velocity(), closed.Diagnostics()
	closed.Close()
	closed.Close()
	for _, current := range []*MetalFlow{nil, {}, closed} {
		current.Close()
		if _, err := current.Step(0.001); err == nil {
			t.Fatal("non-live Metal flow accepted a step")
		}
	}
	after := closed.Velocity()
	if !slices.Equal(before.X, after.X) || !slices.Equal(before.Y, after.Y) || closed.Diagnostics() != diag {
		t.Fatal("close lost accepted snapshots")
	}
	var absent *MetalFlow
	if absent.Diagnostics() != (Diagnostics{}) || len(absent.Velocity().X) != 0 {
		t.Fatal("nil snapshot is not zero")
	}
}

// anti-test of TestMetalFlowClosedNilAndZeroRefuseSteps
// Snapshot mutation cannot rewrite the accepted state or caller input.
func TestMetalFlowSnapshotsAreIndependent(t *testing.T) {
	conf := metalFlowConfig(8)
	initial := metalFlowInitial(conf)
	before := slices.Clone(initial.X)
	gpu := openMetalFlow(t, conf, initial)
	initial.X[0] = 99
	first := gpu.Velocity()
	saved := slices.Clone(first.X)
	first.X[0] = 123
	if !slices.Equal(saved, gpu.Velocity().X) || saved[0] == 99 {
		t.Fatal("velocity snapshot aliases accepted or caller memory")
	}
	initial.X = before
}

// Invalid domains, shapes and finite values outside GPU arithmetic are refused.
func TestMetalFlowRejectsInvalidOrUnrepresentableInputs(t *testing.T) {
	for _, which := range []string{"shape", "budget", "units", "viscosity", "length", "nan", "range", "coefficient", "subnormal coefficient"} {
		t.Run(which, func(t *testing.T) {
			conf := metalFlowConfig(8)
			values := metalFlowInitial(conf)
			switch which {
			case "shape":
				conf.Shape.Width = 3
			case "budget":
				conf.Shape.MaxCells = 1
			case "units":
				conf.WidthMetres = 0
			case "viscosity":
				conf.Viscosity = -1
			case "length":
				values.X = values.X[:3]
			case "nan":
				values.X[0] = math.NaN()
			case "range":
				for i := range values.X {
					values.X[i] = 1e40
				}
			case "coefficient":
				conf.WidthMetres = 1e-20
				conf.HeightMetres = 1e-20
			case "subnormal coefficient":
				conf.WidthMetres = 1e20
				conf.HeightMetres = 1e20
			}
			gpu, err := OpenMetal(conf, values)
			if gpu != nil {
				gpu.Close()
			}
			if err == nil || gpu != nil {
				t.Fatal("invalid Metal flow accepted")
			}
		})
	}
}

// anti-test of TestMetalFlowRejectsInvalidOrUnrepresentableInputs
// Positive viscosity damps a resolved shear mode by its analytic rate.
func TestMetalFlowViscousModeDecaysAnalytically(t *testing.T) {
	conf := metalFlowConfig(32)
	values := metalFlowInitial(conf)
	for y := range 32 {
		for x := range 32 {
			values.X[y*32+x] = math.Sin(2 * math.Pi * float64(y) / 32)
			values.Y[y*32+x] = 0
		}
	}
	gpu := openMetalFlow(t, conf, values)
	for range 10 {
		if _, err := gpu.Step(0.001); err != nil {
			t.Fatal(err)
		}
	}
	want := Velocity{X: slices.Clone(values.X), Y: slices.Clone(values.Y)}
	for i := range want.X {
		want.X[i] *= math.Exp(-0.1 * 0.01)
	}
	requireMetalVelocity(t, gpu.Velocity(), want)
}

// The comparison times full steps with final readback and diagnostics.
func BenchmarkMetalFlowComparison(b *testing.B) {
	conf := metalFlowConfig(128)
	initial := metalFlowInitial(conf)
	for _, useGPU := range []bool{false, true} {
		name := "cpu"
		if useGPU {
			name = "metal"
		}
		b.Run(name, func(b *testing.B) {
			var step func(float64) (Diagnostics, error)
			if useGPU {
				current := openMetalFlow(b, conf, initial)
				step = current.Step
			} else {
				current, err := New(conf, initial)
				if err != nil {
					b.Fatal(err)
				}
				step = current.Step
			}
			if _, err := step(0.0001); err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			for range b.N {
				if _, err := step(0.0001); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Both FFT stage parities and both rectangular orientations must retain
// independent components when shared scratch is reused.
func TestMetalFlowRectanglesPreserveIndependentTransforms(t *testing.T) {
	for _, dimensions := range [][2]int{{16, 8}, {8, 16}, {32, 8}, {8, 32}} {
		t.Run(fmt.Sprint(dimensions), func(t *testing.T) {
			conf := metalFlowConfig(8)
			width, height := dimensions[0], dimensions[1]
			conf.Shape = spectral.Shape{Width: width, Height: height, MaxCells: width * height}
			initial := Velocity{X: make([]float64, width*height), Y: make([]float64, width*height)}
			for y := range height {
				for x := range width {
					px, py := 2*math.Pi*float64(x)/float64(width), 2*math.Pi*float64(y)/float64(height)
					initial.X[y*width+x] = 0.4 + math.Sin(px)*math.Cos(py)
					initial.Y[y*width+x] = -0.25 - math.Cos(px)*math.Sin(py)
				}
			}
			cpu, err := New(conf, initial)
			if err != nil {
				t.Fatal(err)
			}
			gpu := openMetalFlow(t, conf, initial)
			for range 4 {
				if _, err := cpu.Step(0.001); err != nil {
					t.Fatal(err)
				}
				if _, err := gpu.Step(0.001); err != nil {
					t.Fatal(err)
				}
				requireMetalVelocity(t, gpu.Velocity(), cpu.Velocity())
			}
		})
	}
}

// anti-test of TestMetalFlowRectanglesPreserveIndependentTransforms
// The nonlinear reference fixture must change enough to expose a frozen solver.
func TestMetalFlowComparisonFixtureHasNonzeroComponents(t *testing.T) {
	conf := metalFlowConfig(32)
	initial := metalFlowInitial(conf)
	if len(initial.X) != 1024 || len(initial.Y) != 1024 || kineticEnergy(initial) < 0.1 {
		t.Fatal("comparison fixture has no resolved flow")
	}
	before, err := New(conf, initial)
	if err != nil {
		t.Fatal(err)
	}
	original := before.Velocity()
	for range 8 {
		if _, err := before.Step(0.001); err != nil {
			t.Fatal(err)
		}
	}
	evolved := before.Velocity()
	var change float64
	for i := range evolved.X {
		change = math.Hypot(change, evolved.X[i]-original.X[i])
		change = math.Hypot(change, evolved.Y[i]-original.Y[i])
	}
	if change < 0.01 {
		t.Fatalf("fixture cannot expose a frozen solver: change %g", change)
	}
}
