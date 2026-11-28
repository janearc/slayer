//go:build draw && darwin

package gpu

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

// stormMean and stormLargest are how far the gpu may be from the tracer
// looking at a storm, 2026-11-26. Measured: 0.026 and 0.027 mean, and 0.33
// and 0.17 largest. The mean is as the clouds' own below; the largest is
// a pixel or two where a storm's arm edge falls either side of a cell
// between float32 and float64, so its bound is wider than theirs.
const stormMean, stormLargest = 0.035, 0.36

// TestNativeCloudStormsMatchTheTracer draws a layer of clouds looking
// straight at a northern storm and at a southern one, on the gpu and on
// slayer's tracer, and holds the two to each other: the weather's whorls,
// their eyes and arms, are in the kernel as they are in the medium.
func TestNativeCloudStormsMatchTheTracer(t *testing.T) {
	clouds := medium.Clouds{Radius: 1, Low: .1, High: .4, Cover: .5,
		Thick: 5, Seed: 5}
	storms := clouds.Weather().Storms
	for _, index := range []int{0, len(storms) - 1} {
		from := storms[index].Centre.Scale(4)
		eye := maths.Subspace{Origin: from,
			Orientation: maths.LookAtUp(from, maths.V(0, 0, 0), maths.Up)}
		world := thing.NewWorld()
		world.AddVolume(thing.Volume{Medium: clouds,
			Material: &thing.Material{Glow: maths.Grey(1)}})
		lens := film.Pinhole{Wide: 40 * maths.Degree}
		renderer, err := NewRenderer(32)
		if err != nil {
			t.Fatal(err)
		}
		drawn, err := renderer.Sampled(world, eye, lens, 0, 65, 49, 1)
		if err != nil {
			t.Fatal(err)
		}
		truth, err := slayer.Sampled(world, eye, lens, 0, 65, 49, 1)
		if err != nil {
			t.Fatal(err)
		}
		found := compare(truth, drawn, 65, 49)
		t.Logf("storm %d against the tracer: %+v", index, found)
		if found.mean > stormMean || found.largest > stormLargest {
			t.Errorf("storm %d drew apart from the tracer: %+v", index, found)
		}
	}
}

// TestNativeBodyShellsMatchTheTracer draws a body's air and its clouds,
// glowing so that only their density shows, on the gpu and on slayer's
// tracer, and holds the two pictures to each other. A shell tens of
// kilometres thick in a box thousands across drew from a grid as rings
// (air) and cubes (clouds) on like-earth (2026-11-26); the kernel now
// asks their density as the medium does, along the shell's own legs.
//
// The bounds are the differences measured on 2026-11-26, a little over:
// air 0.035 mean and 0.095 largest, clouds 0.018 and 0.17. The grid this
// replaced drew the same air 0.059 and 0.19 apart. What is left is, by
// my reading and not yet shown, the tracer's own march, which crosses
// air and clouds over their whole outer sphere and not their two legs:
// the difference fell by four as the shells were drawn thicker. Thin
// shells, as a planet's are, are drawn apart further for that reason.
//
// The clouds' bounds were loosened to 0.035 and 0.22 when they gained
// finer, billowing noise (2026-11-26): with that noise off, by its pick,
// the clouds are 0.021 and 0.13 apart as before, so the kernel and the
// medium agree and the rest is the two marches sampling finer noise.
func TestNativeBodyShellsMatchTheTracer(t *testing.T) {
	for _, shell := range []struct {
		name          string
		med           medium.Medium
		mean, largest float64
	}{
		{"air", medium.Air{Radius: 1, Height: .15, Thick: 2}, .04, .11},
		{"clouds", medium.Clouds{Radius: 1, Low: .1, High: .4,
			Cover: .5, Thick: 5, Seed: 5}, .035, .22},
	} {
		t.Run(shell.name, func(t *testing.T) {
			world := thing.NewWorld()
			world.AddVolume(thing.Volume{Medium: shell.med,
				Material: &thing.Material{Glow: maths.Grey(1)}})
			eye := maths.Subspace{Origin: maths.V(0, 0, 4)}
			lens := film.Pinhole{Wide: 40 * maths.Degree}
			renderer, err := NewRenderer(32)
			if err != nil {
				t.Fatal(err)
			}
			drawn, err := renderer.Sampled(world, eye, lens, 0, 65, 49, 1)
			if err != nil {
				t.Fatal(err)
			}
			truth, err := slayer.Sampled(world, eye, lens, 0, 65, 49, 1)
			if err != nil {
				t.Fatal(err)
			}
			found := compare(truth, drawn, 65, 49)
			t.Logf("%s against the tracer: %+v", shell.name, found)
			if found.mean > shell.mean || found.largest > shell.largest {
				t.Fatalf("%s drew apart from the tracer: %+v", shell.name,
					found)
			}
		})
	}
}
