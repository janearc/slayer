// Package bench times slayer's tracer through its own front doors, Render and
// Sampled, on scenes made of the harness's fakes.
//
// Each benchmark changes one thing at a time, the things in the scene, the
// lamps, the rays per pixel, and says per pixel how long it took, how much it
// allocated, and how often it asked a shape or a lamp.
package bench_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/janearc/slayer/burning-trousers/fake"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// scene is a square of ground with balls standing on it in a grid, and
// lamps above, under a grey sky.
func scene(balls, lamps int, counts *fake.Counts) *thing.World {
	grey := &thing.Material{Colour: maths.Grey(0.5)}
	world := &thing.World{Sky: fake.NewSky(maths.Grey(0.1))}
	world.Things = append(world.Things,
		&thing.Thing{Shape: fake.NewPlane(50, counts), Material: grey})
	side := 1
	for side*side < balls {
		side++
	}
	for i := range balls {
		x := float64(i%side) - float64(side-1)/2
		z := float64(i/side) - float64(side-1)/2
		ball := fake.NewSphere(maths.V(x*2, 0.6, z*2), 0.5, counts)
		world.Things = append(world.Things,
			&thing.Thing{Shape: ball, Material: grey})
	}
	for i := range lamps {
		at := maths.V(float64(i%4)*3-4.5, 8, float64(i/4)*3-4.5)
		world.Lamps = append(world.Lamps,
			fake.NewLamp(at, maths.Grey(60), counts))
	}
	return world
}

// eye looks at the middle of the ground from up and away.
func eye() maths.Subspace {
	from := maths.V(0, 6, 18)
	return maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 0, 0))}
}

// timed counts one frame's asks with counting on, then times frames with
// counting off, so the counting does not slow what is timed, and
// reports both per pixel. With BURNING_TROUSERS_COUNTS=off there is no
// counted frame at all, for a profile the counting would cloud.
func timed(b *testing.B, balls, lamps, pixels int,
	draw func(*thing.World) error) {
	counts := &fake.Counts{}
	if os.Getenv("BURNING_TROUSERS_COUNTS") != "off" {
		if err := draw(scene(balls, lamps, counts)); err != nil {
			b.Fatal(err)
		}
	}
	world := scene(balls, lamps, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := draw(world); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	frames := float64(b.N * pixels)
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/frames, "ns/pixel")
	b.ReportMetric(float64(counts.Hits())/float64(pixels), "hits/pixel")
	b.ReportMetric(float64(counts.Lights())/float64(pixels), "lights/pixel")
}

// render times Render on one scene at one size.
func render(b *testing.B, balls, lamps, wide, high int) {
	timed(b, balls, lamps, wide*high, func(world *thing.World) error {
		_, err := trace.Render(world, eye(), film.Pinhole{}, 0, wide, high)
		return err
	})
}

// More balls on the ground, one lamp: how the cost grows with things.
func BenchmarkThings(b *testing.B) {
	for _, balls := range []int{1, 16, 256, 1024} {
		b.Run(fmt.Sprintf("balls=%d", balls), func(b *testing.B) {
			render(b, balls, 1, 128, 128)
		})
	}
}

// More lamps over sixteen balls: how the cost grows with lights.
func BenchmarkLamps(b *testing.B) {
	for _, lamps := range []int{1, 8, 64} {
		b.Run(fmt.Sprintf("lamps=%d", lamps), func(b *testing.B) {
			render(b, 16, lamps, 128, 128)
		})
	}
}

// A finer grid of rays per pixel through Sampled, rays by rays, which is what
// its rays number means: how the cost grows with samples.
func BenchmarkGrid(b *testing.B) {
	for _, rays := range []int{1, 4, 16} {
		b.Run(fmt.Sprintf("grid=%dx%d", rays, rays), func(b *testing.B) {
			timed(b, 16, 1, 64*64, func(world *thing.World) error {
				_, err := trace.Sampled(world, eye(), film.Pinhole{}, 0,
					64, 64, rays)
				return err
			})
		})
	}
}
