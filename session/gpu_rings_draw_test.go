//go:build draw && darwin

package session

import (
	"errors"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// the picture the ringed planet is drawn in: 160 by 90, one ray a pixel, and
// the lens as wide as the set's own rig's frame is at its focal length, 70.41
// millimetres across at 330, in degrees.
const (
	ringsWide, ringsHigh, ringsSide = 160, 90, 1
	ringsView                       = 12.19 * maths.Degree
)

// ringsApart is how far the second frame is from the first as the gpu tests
// measure it: the sum over pixels of the difference in each channel as a
// share of the first frame's light, the largest difference in any one pixel
// as a share of the brightest, and how many pixels' difference is not a
// number.
type ringsApart struct {
	mean, largest float64
	notFinite     int
}

// ringsMeasured is how far drawn is from truth.
func ringsMeasured(truth, drawn *frame.Frame) ringsApart {
	var found ringsApart
	var total, brightest float64
	for row := range truth.H {
		for column := range truth.W {
			one, other := truth.At(column, row), drawn.At(column, row)
			light := float64(one.R + one.G + one.B)
			difference := math.Abs(float64(one.R-other.R)) +
				math.Abs(float64(one.G-other.G)) +
				math.Abs(float64(one.B-other.B))
			total += light
			brightest = math.Max(brightest, light)
			if math.IsNaN(difference) || math.IsInf(difference, 0) {
				found.notFinite++
			}
			found.mean += difference
			found.largest = math.Max(found.largest, difference)
		}
	}
	if total > 0 {
		found.mean /= total
		found.largest /= brightest
	}
	return found
}

// ringsDrawn is a world drawn both ways at 160 by 90 through a lens, and how
// far apart the two are, and how long each took: the tracer, the gpu, and the
// gpu again with its kernel already built. When the gpu refuses the world, it
// is the refusal that comes back, and no frame.
type ringsDrawn struct {
	apart              ringsApart
	refused            error
	cpu, gpu, gpuAgain time.Duration
}

// ringsBoth draws a world on the tracer and on the gpu.
func ringsBoth(t *testing.T, world *thing.World, eye maths.Subspace,
	lens film.Sensor) ringsDrawn {
	t.Helper()
	var found ringsDrawn
	start := time.Now()
	truth, err := trace.Sampled(world, eye, lens, 0, ringsWide, ringsHigh,
		ringsSide)
	found.cpu = time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	start = time.Now()
	drawn, err := gpu.Sampled(world, eye, lens, 0, ringsWide, ringsHigh,
		ringsSide)
	found.gpu = time.Since(start)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		found.refused = err
		return found
	}
	start = time.Now()
	if _, err := gpu.Sampled(world, eye, lens, 0, ringsWide, ringsHigh,
		ringsSide); err != nil {
		t.Fatalf("the gpu refuses it the second time: %v", err)
	}
	found.gpuAgain = time.Since(start)
	found.apart = ringsMeasured(truth, drawn)
	return found
}

// the most the gpu may be from the tracer on the ringed giant, on average
// and at the largest pixel. The gpu's own tests hold a ring to 1e-5 and 3e-3,
// at a slab 1000 km thick; this set's is 200 km, and its fourteen rims lie
// 850,000 km from the eye, where a float32 sample is 64 m coarse against a
// step 2.7 km long, so a sample beside a rim falls on the other side of it
// from the tracer's in one of every few dozen. Measured on 2026-11-10: with
// the slab 200, 1000 and 4000 km thick the mean is 1.2e-4, 2.9e-5 and 6.2e-6.
// The bounds are two and a half times the measure at 200 km.
const (
	ringsMean    = 3e-4
	ringsLargest = 3.3e-2
)

// the ringed giant of sets/planetary-turbidity.yaml, with no ground and no
// terrain shelf, drawn on the gpu and by the tracer at 160 by 90. its seven
// rings are plain ones, the kernel asks them itself.
//
// the gpu draws it as the tracer does, to the bounds above.
func TestTheGPUDrawsARingedPlanet(t *testing.T) {
	given := mustParse(t)
	given.set = filepath.Join("..", "sets", "planetary-turbidity.yaml")
	world, _, given, err := staged(given)
	if err != nil {
		t.Skipf("the set cannot be staged here: %v", err)
	}
	if given.eye == nil {
		t.Fatal("the set has no camera")
	}
	eye, lens := *given.eye, film.Sensor(film.Pinhole{Wide: ringsView})
	found := ringsBoth(t, world, eye, lens)
	if found.refused != nil {
		t.Fatalf("the gpu refuses it: %v", found.refused)
	}
	t.Logf("things %d, volumes %d, lamps %d", len(world.Things),
		len(world.Volumes), len(world.Lamps))
	t.Logf("mean %g, largest %g, not finite %d", found.apart.mean,
		found.apart.largest, found.apart.notFinite)
	t.Logf("cpu %v, gpu %v, gpu again %v", found.cpu, found.gpu,
		found.gpuAgain)
	if found.apart.notFinite > 0 {
		t.Errorf("%d pixels of the gpu's picture are not finite",
			found.apart.notFinite)
	}
	if !(found.apart.mean <= ringsMean) ||
		!(found.apart.largest <= ringsLargest) {
		t.Errorf("the gpu is %g from the tracer on average and %g at most, "+
			"want within %g and %g", found.apart.mean, found.apart.largest,
			ringsMean, ringsLargest)
	}
	ringsBare(t, world, eye, lens)
}

// ringsBare draws the planet with its rings taken away, both ways, and logs how
// far apart they are: what the two disagree about before any ring is drawn.
func ringsBare(t *testing.T, world *thing.World, eye maths.Subspace,
	lens film.Sensor) {
	rings := world.Volumes
	world.Volumes = nil
	defer func() { world.Volumes = rings }()
	found := ringsBoth(t, world, eye, lens)
	if found.refused != nil {
		t.Fatalf("the gpu refuses the planet alone: %v", found.refused)
	}
	t.Logf("the planet alone: mean %g, largest %g, not finite %d",
		found.apart.mean, found.apart.largest, found.apart.notFinite)
}
