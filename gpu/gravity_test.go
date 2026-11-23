//go:build draw && darwin

package gpu

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/hole"
	"github.com/janearc/slayer/metal"
)

const gravityProbe = `
kernel void probe(device const float *points [[buffer(0)]],
 device float *out [[buffer(1)]], constant float *p [[buffer(2)]],
 uint id [[thread_position_in_grid]]) {
 float4 x = float4(p[4],p[5],p[6],p[7]);
 float4 q = float4(p[8],p[9],p[10],p[11]);
 for (int i=0;i<int(p[3]);i++) kerr_step(x,q,p[0],p[1],p[2]);
 int part = int(points[id*3]);
 out[id] = part<4 ? x[part] : q[part-4];
}`

func gravityBytes(values ...float64) []byte {
	var bytes []byte
	for _, value := range values {
		bytes = binary.LittleEndian.AppendUint32(bytes, math.Float32bits(float32(value)))
	}
	return bytes
}

// The GPU must integrate the owning library's equations, including spin and
// non-equatorial motion, rather than drawing a screen-space distortion.
func TestGravityMetalStepsMatchCelestial(t *testing.T) {
	kernel, err := metal.Open(metal.GeodesicLibrary()+gravityProbe, "probe")
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	points := make([][3]float32, 8)
	for i := range points {
		points[i][0] = float32(i)
	}
	for _, spin := range []float64{0, .5, .9} {
		for _, steps := range []int{1, 32, 256} {
			black, err := hole.NewHole(1, spin)
			if err != nil {
				t.Fatal(err)
			}
			at := hole.NewVec3(7, 2, 3)
			momentum := black.Null(at, black.FrameOf(at, black.Static(at)), hole.NewVec3(-.8, .3, -.2)).Momentum()
			x := hole.NewVec4(0, 7, 2, 3)
			ray := hole.NewRay(x, momentum)
			parts := func(v hole.Vec4) []float64 { return []float64{v.Time(), v.AlongX(), v.AlongY(), v.AlongZ()} }
			data := []float64{1, spin, .01, float64(steps)}
			data = append(data, parts(x)...)
			data = append(data, parts(momentum)...)
			got, err := kernel.Points(points, gravityBytes(data...))
			if err != nil {
				t.Fatal(err)
			}
			for range steps {
				ray = black.Step(ray, .01)
			}
			want := append(parts(ray.Position()), parts(ray.Momentum())...)
			for i, value := range got {
				if math.IsNaN(float64(value)) || math.Abs(float64(value)-want[i]) > 3e-4*math.Max(1, math.Abs(want[i])) {
					t.Fatalf("gravity step mismatch spin=%g steps=%d part=%d GPU=%g CPU=%g", spin, steps, i, value, want[i])
				}
			}
		}
	}
}

// A photon close-up stresses trajectories that spend a long time near the
// critical orbit, as well as backward tracing and the flat ring-disc branch.
func TestGravityMetalCriticalAndBackwardSteps(t *testing.T) {
	kernel, err := metal.Open(metal.GeodesicLibrary()+gravityProbe, "probe")
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	points := make([][3]float32, 8)
	for i := range points {
		points[i][0] = float32(i)
	}
	for _, backward := range []bool{false, true} {
		black, _ := hole.NewHole(1, 0)
		at := hole.NewVec3(3, 0, 0)
		ray := black.Null(at, black.FrameOf(at, black.Static(at)), hole.NewVec3(0, 1, 0))
		p := ray.Momentum()
		if backward {
			p = hole.NewVec4(-p.Time(), -p.AlongX(), -p.AlongY(), -p.AlongZ())
			ray = hole.NewRay(ray.Position(), p)
		}
		got, err := kernel.Points(points, gravityBytes(1, 0, .01, 1024, 0, 3, 0, 0, p.Time(), p.AlongX(), p.AlongY(), p.AlongZ()))
		if err != nil {
			t.Fatal(err)
		}
		for range 1024 {
			ray = black.Step(ray, .01)
		}
		x, q := ray.Position(), ray.Momentum()
		want := []float64{x.Time(), x.AlongX(), x.AlongY(), x.AlongZ(), q.Time(), q.AlongX(), q.AlongY(), q.AlongZ()}
		for i, value := range got {
			if math.IsNaN(float64(value)) || math.Abs(float64(value)-want[i]) > .003*math.Max(1, math.Abs(want[i])) {
				t.Fatalf("critical gravity drift backward=%v part=%d GPU=%g CPU=%g", backward, i, value, want[i])
			}
		}
	}
	// On the oblate coordinate disc the owning CPU step defines flat motion.
	got, err := kernel.Points(points, gravityBytes(1, .9, .01, 1, 0, 0, 0, 0, -1, 1, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	black, _ := hole.NewHole(1, .9)
	ray := black.Step(hole.NewRay(hole.NewVec4(0, 0, 0, 0), hole.NewVec4(-1, 1, 0, 0)), .01)
	if math.IsNaN(float64(got[0])) || math.Abs(float64(got[0])-ray.Position().Time()) > 1e-6 {
		t.Fatalf("ring-disc branch differs GPU=%g CPU=%g", got[0], ray.Position().Time())
	}
}

// This measures the integration kernel alone, not scene FPS. Every GPU
// invocation advances one ray and reports one component, so the CPU benchmark
// deliberately performs the same number of integrations.
func BenchmarkGravitySteps(b *testing.B) {
	black, _ := hole.NewHole(1, .9)
	at := hole.NewVec3(7, 2, 3)
	ray := black.Null(at, black.FrameOf(at, black.Static(at)), hole.NewVec3(-.8, .3, -.2))
	const count = 4096
	const steps = 256
	b.Run("CPU", func(b *testing.B) {
		for b.Loop() {
			for range count {
				stepped := ray
				for range steps {
					stepped = black.Step(stepped, .01)
				}
				if math.IsNaN(stepped.Position().Time()) {
					b.Fatal("invalid CPU ray")
				}
			}
		}
	})
	b.Run("Metal", func(b *testing.B) {
		kernel, err := metal.Open(metal.GeodesicLibrary()+gravityProbe, "probe")
		if errors.Is(err, metal.ErrNoDevice) {
			b.Skip(err)
		}
		if err != nil {
			b.Fatal(err)
		}
		defer kernel.Close()
		p := ray.Momentum()
		data := gravityBytes(1, .9, .01, steps, 0, 7, 2, 3, p.Time(), p.AlongX(), p.AlongY(), p.AlongZ())
		points := make([][3]float32, count)
		for i := range points {
			points[i][0] = float32(i % 8)
		}
		for b.Loop() {
			if _, err := kernel.Points(points, data); err != nil {
				b.Fatal(err)
			}
		}
	})
}
