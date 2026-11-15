package colour

import (
	"math"
	"math/rand"
	"testing"

	"github.com/janearc/slayer/maths"
)

// oldBloomed is the bloom as it was before infinite light was handled,
// written out so that the finite path can be held to it bit for bit.
func oldBloomed(lights []maths.Light, w, h int,
	exposure, radius float64) []maths.Light {
	over := make([]maths.Light, len(lights))
	any := false
	for i, l := range lights {
		l = l.Scale(exposure)
		if y := l.Luminance(); y > 1 {
			over[i] = l.Scale((y - 1) / y / 3)
			any = true
		}
	}
	if !any {
		return lights
	}
	reach := int(math.Ceil(3 * radius))
	weights := make([]float64, 2*reach+1)
	sum := 0.0
	for i := range weights {
		d := float64(i - reach)
		weights[i] = math.Exp(-d * d / (2 * radius * radius))
		if d == 0 {
			weights[i] = 1
		}
		sum += weights[i]
	}
	for i := range weights {
		weights[i] /= sum
	}
	across := make([]maths.Light, len(lights))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var acc maths.Light
			for k, wt := range weights {
				sx := x + k - reach
				if sx >= 0 && sx < w {
					acc = acc.Add(over[y*w+sx].Scale(wt))
				}
			}
			across[y*w+x] = acc
		}
	}
	out := make([]maths.Light, len(lights))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var acc maths.Light
			for k, wt := range weights {
				sy := y + k - reach
				if sy >= 0 && sy < h {
					acc = acc.Add(across[sy*w+x].Scale(wt))
				}
			}
			out[y*w+x] = lights[y*w+x].Add(acc.Scale(1 / exposure))
		}
	}
	return out
}

// On finite light the bloom is bit for bit the one written out above,
// over seeded random frames at several exposures and radii.
func TestTheBloomOfFiniteLightIsUnchangedBitForBit(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for n := 0; n < 300; n++ {
		w, h := 3+rng.Intn(10), 3+rng.Intn(10)
		lights := make([]maths.Light, w*h)
		for i := range lights {
			lights[i] = maths.RGB(rng.Float64()*4, rng.Float64()*4,
				rng.Float64()*4)
		}
		exposure := []float64{0.3, 1, 1.7, 3}[n%4]
		radius := []float64{0.5, 1, 2}[n%3]
		got := bloomed(lights, w, h, exposure, radius)
		want := oldBloomed(lights, w, h, exposure, radius)
		for i := range got {
			for c, pair := range [][2]float64{{got[i].R, want[i].R},
				{got[i].G, want[i].G}, {got[i].B, want[i].B}} {
				if math.Float64bits(pair[0]) != math.Float64bits(pair[1]) {
					t.Fatalf("frame %d (%dx%d, exposure %v, radius %v) "+
						"pixel %d part %d: got %v, want %v", n, w, h,
						exposure, radius, i, c, pair[0], pair[1])
				}
			}
		}
	}
}
