//go:build darwin

package metal

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/janearc/slayer/maths/noise"
)

// within is how far the GPU's noise may be from maths/noise's at any point. It
// works in 64-bit floats and the kernel in 32, and the gap grows with the
// coordinate, since each octave scales it by 2.03. measured on an M3 Max on
// 2026-10-02, the worst over the cases below was 3.1e-5; this allows three
// times that, so a real change in the kernel fails rather than hides.
const within = 1e-4

// points is count points thrown about a cube span from the middle
func points(count int, span float64, random *rand.Rand) [][3]float32 {
	thrown := make([][3]float32, count)
	for index := range thrown {
		thrown[index] = [3]float32{along(random, span),
			along(random, span), along(random, span)}
	}
	return thrown
}

// along is somewhere from minus span to span
func along(random *rand.Rand, span float64) float32 {
	return float32((random.Float64()*2 - 1) * span)
}

// reference is maths/noise's own noise at a point
func reference(seed uint64, point [3]float32, octaves int) float64 {
	return noise.Fractal(seed, float64(point[0]), float64(point[1]),
		float64(point[2]), octaves)
}

// a case is a seed and a number of octaves, asked over a cube of a span
type noiseCase struct {
	seed    uint64
	octaves int
	span    float64
}

var noiseCases = []noiseCase{{3, 1, 10}, {7, 4, 50}, {29, 6, 50},
	{0xdeadbeef, 6, 5}, {1 << 63, 3, 200}}

// worstOf is the furthest the gpu lands from maths/noise over one case
func worstOf(t *testing.T, gpu *Noise, asked noiseCase,
	random *rand.Rand) float64 {
	thrown := points(20000, asked.span, random)
	got, err := gpu.Fractal(asked.seed, asked.octaves, thrown)
	if err != nil {
		t.Fatal(err)
	}
	worst := 0.0
	for index, point := range thrown {
		want := reference(asked.seed, point, asked.octaves)
		apart := math.Abs(float64(got[index]) - want)
		worst = math.Max(worst, apart)
		if apart > within {
			t.Fatalf("%+v at %v: gpu %v, noise %v, %g apart", asked,
				point, got[index], want, apart)
		}
	}
	return worst
}

// the kernel is maths/noise's noise: at every point, for several seeds and
// octaves, it lands within the stated distance of noise.Fractal
func TestTheKernelIsMathsNoise(t *testing.T) {
	gpu, err := OpenNoise()
	if err != nil {
		t.Fatal(err)
	}
	defer gpu.Close()
	t.Logf("on %s", gpu.Device())
	random := rand.New(rand.NewPCG(1, 2))
	worst := 0.0
	for _, asked := range noiseCases {
		found := worstOf(t, gpu, asked, random)
		t.Logf("%+v: worst %.2g", asked, found)
		worst = math.Max(worst, found)
	}
	t.Logf("worst of all: %.2g, allowed %.0g", worst, within)
}

// how much faster, said as a number rather than claimed
func TestHowMuchFaster(t *testing.T) {
	if testing.Short() {
		t.Skip("a timing, not a check")
	}
	gpu, err := OpenNoise()
	if err != nil {
		t.Fatal(err)
	}
	defer gpu.Close()
	thrown := points(1<<20, 50, rand.New(rand.NewPCG(3, 4)))
	start := time.Now()
	if _, err := gpu.Fractal(29, 6, thrown); err != nil {
		t.Fatal(err)
	}
	onGPU := time.Since(start)
	start = time.Now()
	for _, point := range thrown {
		reference(29, point, 6)
	}
	onCPU := time.Since(start)
	t.Logf("a million points, six octaves: gpu %v, one cpu core %v, "+
		"%.0f times", onGPU, onCPU, float64(onCPU)/float64(onGPU))
}
