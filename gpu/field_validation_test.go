//go:build draw && darwin

package gpu

import (
	"errors"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"testing"
	"time"
)

// Invalid host data must be refused before a Metal command can be submitted.
func TestGravityCompleteRejectsInvalidInputs(t *testing.T) {
	renderer, err := NewRenderer(32)
	if err != nil {
		t.Fatal(err)
	}
	if renderer.LastTiming() != (FrameTiming{}) {
		t.Fatal("timing exists before a frame")
	}
	if _, _, err := fieldSampled(thing.NewWorld(), maths.Subspace{}, nil, 0, 0, 0, 1); err == nil {
		t.Fatal("missing field accepted")
	}
	cases := map[string]*thing.Field{
		"nil":             nil,
		"mass":            {Mass: 0, Reach: 40},
		"spin":            {Mass: 1, Spin: math.NaN(), Reach: 40},
		"reach":           {Mass: 1, Reach: math.Inf(1)},
		"negative scale":  {Mass: 1, Reach: 40, Motion: thing.Still(maths.Subspace{Scale: -1})},
		"infinite scale":  {Mass: 1, Reach: 40, Motion: thing.Still(maths.Subspace{Scale: math.Inf(1)})},
		"infinite origin": {Mass: 1, Reach: 40, Motion: thing.Still(maths.Subspace{Origin: maths.V(math.Inf(1), 0, 0)})},
		"float spin":      {Mass: 1, Spin: 1 - 1e-10, Reach: 40},
		"float reach":     {Mass: 1, Reach: 1e100},
	}
	for name, field := range cases {
		t.Run(name, func(t *testing.T) {
			world := thing.NewWorld()
			world.Fields = append(world.Fields, field)
			if _, _, err := fieldSampled(world, maths.Subspace{}, nil, 0, 0, 0, 1); err == nil {
				t.Fatal("invalid field reached GPU setup")
			}
		})
	}
}

func TestGravityCompleteBeamValidation(t *testing.T) {
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1}, Beam: &thing.Beam{Speed: .4, Round: true}})
	encoded := newScene()
	encoded.gravity = true
	if err := encoded.volumesOf(world, 0); err != nil {
		t.Fatal(err)
	}
	world.Volumes[0].Beam.Round = false
	if err := encoded.volumesOf(world, 0); err == nil {
		t.Fatal("unsupported axial field beam accepted")
	}
}

type failedFieldKernel struct {
	err    error
	closed bool
	rows   int
}

type countedFieldLens struct{ calls int }

func (lens *countedFieldLens) Ray(x, y, w, h int, band maths.Band) maths.Ray {
	lens.calls++
	return (film.Pinhole{}).Ray(x, y, w, h, band)
}

func TestGravityCompleteDoesNotGenerateTheLensTwice(t *testing.T) {
	world := thing.NewWorld()
	world.Fields = append(world.Fields, &thing.Field{Mass: 1, Reach: 40})
	lens := &countedFieldLens{}
	failure := errors.New("stop before dispatch")
	ops := fieldDefaults()
	ops.open = func(string, string) (fieldKernel, error) { return &failedFieldKernel{err: failure}, nil }
	_, _, err := fieldSampledUsing(world, maths.Subspace{Origin: maths.V(0, 0, 60)}, lens, 0, 2, 2, 1, ops)
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if lens.calls != 4 {
		t.Fatal("field generated lens rays more than once", lens.calls)
	}
}

func (k *failedFieldKernel) Grid(width, height, rows, per int, inputs [][]byte) ([]float32, error) {
	k.rows = rows
	return nil, k.err
}
func (k *failedFieldKernel) GPUTime() time.Duration { return 0 }
func (k *failedFieldKernel) Close()                 { k.closed = true }

func TestGravityCompletePropagatesStageFailures(t *testing.T) {
	failure := errors.New("field stage failed")
	for _, stage := range []string{"encode", "bake", "compile", "dispatch"} {
		t.Run(stage, func(t *testing.T) {
			world := thing.NewWorld()
			world.Fields = append(world.Fields, &thing.Field{Mass: 1, Reach: 40, EyeTime: 1.2, Eye: maths.V(.1, 0, 0)})
			ops := fieldDefaults()
			ops.rows = 7
			device := &failedFieldKernel{err: failure}
			switch stage {
			case "encode":
				ops.encode = func(*thing.World, []grown, maths.Subspace, film.Sensor, time.Duration, int, int, ...int) (*scene, error) {
					return nil, failure
				}
			case "bake":
				ops.bake = func(*scene) (time.Duration, error) { return 0, failure }
			case "compile":
				ops.open = func(string, string) (fieldKernel, error) { return nil, failure }
			case "dispatch":
				ops.open = func(string, string) (fieldKernel, error) { return device, nil }
			}
			_, _, err := fieldSampledUsing(world, maths.Subspace{Origin: maths.V(0, 0, 10)}, nil, 0, 2, 2, 1, ops)
			if !errors.Is(err, failure) {
				t.Fatal("field stage error lost", err)
			}
			if stage == "dispatch" && !device.closed {
				t.Fatal("failed dispatch leaked its kernel")
			}
			if stage == "dispatch" && device.rows != 7 {
				t.Fatal("field dispatch band ignored", device.rows)
			}
		})
	}
}
