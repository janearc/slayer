//go:build draw && darwin

package gpu

import (
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

type rcFieldKernel struct {
	failedFieldKernel
	inputs [][]byte
}

func (kernel *rcFieldKernel) Grid(w, h, rows, per int, inputs [][]byte) ([]float32, error) {
	kernel.inputs = inputs
	return nil, kernel.err
}

// TestGravityRCSceneBuffers keeps the gravity parameters after all ordinary
// scene buffers, so a height-field buffer cannot be mistaken for a Kerr mass.
func TestGravityRCSceneBuffers(t *testing.T) {
	world := thing.NewWorld()
	world.Fields = append(world.Fields, &thing.Field{Mass: 2, Reach: 80, LetGo: 17})
	failure := errors.New("stop after checking the dispatch")
	kernel := &rcFieldKernel{failedFieldKernel: failedFieldKernel{err: failure}}
	ops := fieldDefaults()
	ops.bake = func(*scene) (time.Duration, error) { return 0, nil }
	ops.open = func(source, _ string) (fieldKernel, error) {
		for _, declaration := range []string{
			"Field *fields [[buffer(15)]]", "float *heights [[buffer(16)]]",
			"float4 *field [[buffer(17)]]", "float *out [[buffer(18)]]",
		} {
			if !strings.Contains(source, declaration) {
				t.Fatalf("gravity kernel lost scene binding %q", declaration)
			}
		}
		return kernel, nil
	}
	_, _, err := fieldSampledUsing(world, maths.Subspace{Origin: maths.V(0, 0, 100)},
		nil, 0, 2, 3, 1, ops)
	if !errors.Is(err, failure) || !kernel.closed {
		t.Fatalf("dispatch error or cleanup lost: %v, closed=%v", err, kernel.closed)
	}
	if len(kernel.inputs) != 18 {
		t.Fatalf("got %d inputs, want 17 scene buffers then gravity parameters", len(kernel.inputs))
	}
	mass := math.Float32frombits(binary.LittleEndian.Uint32(kernel.inputs[17]))
	if mass != 1 {
		t.Fatalf("normalized gravity mass is %g, want 1", mass)
	}
	outward := math.Float32frombits(binary.LittleEndian.Uint32(kernel.inputs[17][96:]))
	if outward != 17 {
		t.Fatalf("field LetGo is %g, want 17", outward)
	}
}

// anti-test of TestGravityRCSceneBuffers
// Modes without curved transport must refuse fields before opening a device.
func TestGravityRCUnsupportedModesRefuseFields(t *testing.T) {
	world := thing.NewWorld()
	world.Fields = append(world.Fields, &thing.Field{Mass: 1, Reach: 40})
	_, _, err := SampledNative(world, maths.Subspace{}, nil, 0, 2, 2, 1)
	if err == nil || !strings.Contains(err.Error(), "gravity") {
		t.Fatalf("native mode silently ignored gravity: %v", err)
	}
	_, _, err = prepared(world, nil, maths.Subspace{}, nil, 0, [2]int{2, 2},
		softwareTraversal, 0, nil, &timing{})
	if err == nil || !strings.Contains(err.Error(), "gravity") {
		t.Fatalf("retained mode silently ignored gravity: %v", err)
	}
}

// TestGravityRCRefusesUnportedOptics prevents a field render from silently
// dropping refraction or reading an unprepared sky-illumination grid.
func TestGravityRCRefusesUnportedOptics(t *testing.T) {
	for _, feature := range []string{"refraction", "cached volume-sky lighting", "gas volume"} {
		t.Run(feature, func(t *testing.T) {
			world := thing.NewWorld()
			world.Fields = append(world.Fields, &thing.Field{Mass: 1, Reach: 40})
			if feature == "refraction" {
				world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
					Material: &thing.Material{Through: 1, Index: 1.5}})
			} else if feature == "gas volume" {
				world.AddVolume(thing.Volume{Medium: gas.Of{Shape: medium.Puff{Radius: 1}, In: gas.Gas{}}})
			} else {
				lighting, err := thing.NewVolumeIllumination(2)
				if err != nil {
					t.Fatal(err)
				}
				grid, err := thing.NewIncidentGrid(maths.Box{
					Min: maths.V(-1, -1, -1), Max: maths.V(1, 1, 1)}, [3]int{2, 2, 2})
				if err != nil {
					t.Fatal(err)
				}
				world.VolumeIllumination, err = lighting.WithGrid(grid)
				if err != nil {
					t.Fatal(err)
				}
			}
			ops := fieldDefaults()
			ops.bake = func(*scene) (time.Duration, error) { return 0, nil }
			ops.open = func(string, string) (fieldKernel, error) {
				return nil, errors.New("device should not be opened")
			}
			_, _, err := fieldSampledUsing(world, maths.Subspace{Origin: maths.V(0, 0, 60)},
				nil, 0, 2, 2, 1, ops)
			if err == nil || !strings.Contains(err.Error(), feature) {
				t.Fatalf("%s was not refused before device setup: %v", feature, err)
			}
		})
	}
}

// TestGravityRCRejectsOverflowingFrameSizes refuses unsafe camera products
// before encoding a scene or allocating its observer-seed tape.
func TestGravityRCRejectsOverflowingFrameSizes(t *testing.T) {
	world := thing.NewWorld()
	world.Fields = append(world.Fields, &thing.Field{Mass: 1, Reach: 40})
	for _, size := range [][3]int{{-1, 1, 1}, {math.MaxInt, 1, 1},
		{2, 2, math.MaxInt}, {math.MaxInt32, math.MaxInt32, 1}} {
		called := false
		ops := fieldDefaults()
		ops.encode = func(*thing.World, []grown, maths.Subspace, film.Sensor,
			time.Duration, int, int, ...int) (*scene, error) {
			called = true
			return nil, errors.New("encoding should not start")
		}
		_, _, err := fieldSampledUsing(world, maths.Subspace{}, nil, 0,
			size[0], size[1], size[2], ops)
		if err == nil || called {
			t.Fatalf("unsafe size %v reached encoding: %v", size, err)
		}
	}
}
