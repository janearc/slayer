//go:build darwin

package weather

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/metal"
)

// stirred is weather on a small grid, stirred for a while on the cpu so
// it holds eddies, jets and torn bands, for the gpu to be held to.
func stirred(t *testing.T) *Weather {
	t.Helper()
	grid, err := NewGrid(64, 32, 70*math.Pi/180, 6e7)
	if err != nil {
		t.Fatal(err)
	}
	stirring, err := NewStirring(4e-17, 4e6, 3e-5)
	if err != nil {
		t.Fatal(err)
	}
	made, err := New(grid, 18000, 3e6, 2e6, stirring, 3)
	if err != nil {
		t.Fatal(err)
	}
	made.Run(2e5)
	return made
}

// spread is the root mean square of a field.
func spread(field []float64) float64 {
	sum := 0.0
	for _, value := range field {
		sum += value * value
	}
	return math.Sqrt(sum / float64(len(field)))
}

// a step on the gpu, in float32, is the cpu's step, in float64, to a
// part in ten thousand of the field's spread: vorticity and the bands'
// air both.
func TestTheGPUStepsAsTheCPUDoes(t *testing.T) {
	cpu := stirred(t)
	gpu := &Weather{}
	*gpu = *cpu
	gpu.vorticity = append([]float64(nil), cpu.vorticity...)
	gpu.origin = append([]float64(nil), cpu.origin...)
	gpu.age = append([]float64(nil), cpu.age...)
	on, err := gpu.onDevice()
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer on.session.Close()
	on.load(gpu)
	const step = 1200.0
	if err := on.stepOn(gpu, step); err != nil {
		t.Fatal(err)
	}
	on.store(gpu)
	cpu.step(step)
	for _, field := range []struct {
		name      string
		want, got []float64
	}{{"vorticity", cpu.vorticity, gpu.vorticity},
		{"bands", cpu.origin, gpu.origin}, {"age", cpu.age, gpu.age}} {
		worst, scale := 0.0, spread(field.want)
		for k := range field.want {
			worst = math.Max(worst, math.Abs(field.got[k]-field.want[k]))
		}
		if worst > 1e-4*scale {
			t.Errorf("%s: off by %g of a spread of %g", field.name, worst,
				scale)
		}
	}
}

// The public GPU runner must advance and return actual fields, not only
// dispatch kernels. A second request for its current clock takes no steps.
func TestPublicGPURunnerReturnsTheWeatherItAdvanced(t *testing.T) {
	cpu, gpu := small(t, 11), small(t, 11)
	cpu.Run(3600)
	steps, err := gpu.RunOnGPU(3600)
	if err != nil {
		t.Fatal(err)
	}
	if steps != 1 || gpu.Clock() != cpu.Clock() {
		t.Fatalf("%d steps to %g, cpu %g", steps, gpu.Clock(), cpu.Clock())
	}
	for i, pair := range [][2][]float64{{cpu.vorticity, gpu.vorticity}, {cpu.origin, gpu.origin}, {cpu.age, gpu.age}} {
		most := 0.0
		for k, want := range pair[0] {
			most = math.Max(most, math.Abs(pair[1][k]-want))
		}
		if most > 1e-5*math.Max(1, spread(pair[0])) {
			t.Fatalf("field%d error %g", i, most)
		}
	}
	if steps, err := gpu.RunOnGPU(gpu.Clock()); err != nil || steps != 0 {
		t.Fatalf("already reached: %d %v", steps, err)
	}
	grid, err := NewGrid(4096, 4, 1, 6e7)
	if err != nil {
		t.Fatal(err)
	}
	wide, err := New(grid, 18000, 3e6, 2e6, cpu.stirring, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wide.RunOnGPU(1); err == nil {
		t.Fatal("GPU accepted a row wider than its kernels hold")
	}
}
