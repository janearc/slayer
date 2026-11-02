package colour

import (
	"image/color"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
)

// the bloom's weights for a radius, as a gaussian worked by hand: for
// each step d out to three radii, exp(-d*d / 2*sigma*sigma), over the
// sum of all of them, both sides and the middle.
//
// sigma 1: 1, e^-0.5, e^-2, e^-4.5 = 1, 0.60653066, 0.13533528,
// 0.01110900, and a sum of 1 + 2*(0.75297494) = 2.50594988.
// sigma 0.5: 1, e^-2, e^-8 = 1, 0.13533528, 0.00033546, and a sum of
// 1 + 2*(0.13567074) = 1.27134149.
var bloomWeights = map[float64][]float64{
	1:   {0.399050279652, 0.242036229376, 0.0540055826224, 0.00443304817524},
	0.5: {0.786570725887, 0.106450771974, 0.000263865082737},
}

// weight is the weight d steps from the middle, zero past the reach.
func weight(sigma float64, d int) float64 {
	w := bloomWeights[sigma]
	if d < 0 {
		d = -d
	}
	if d >= len(w) {
		return 0
	}
	return w[d]
}

// near says two lights are the same to a part in a billion.
func near(a, b maths.Light) bool {
	return math.Abs(a.R-b.R) < 1e-9 && math.Abs(a.G-b.G) < 1e-9 &&
		math.Abs(a.B-b.B) < 1e-9
}

// a bloom is a third of the light beyond white, spread by the gaussian
// across and again down, and added back at the exposure's scale. in a
// row of one pixel's height the pass down finds only the middle, so
// what a pixel d away gets is the spill times weight(d) times weight(0).
//
// the stars sit at both ends, so that the first and last columns are
// both a source of light and a place it falls. each spill is worked in
// the light's exposed amount, y: (y - 1) / 3 of grey, then divided by
// the exposure to be back in the light the picture was given.
func TestABloomIsAThirdOfWhatIsBeyondWhiteSpreadByAGaussian(t *testing.T) {
	for _, c := range []struct {
		name     string
		star     maths.Light
		exposure float64
		spill    float64
	}{
		// y = 4: 3 past white, a third is 1
		{"four times white", maths.Grey(4), 1, 1},
		// y = 1.5: half past white, a third is 1/6
		{"half past white", maths.Grey(1.5), 1, 1.0 / 6},
		// y = 0.75 * 2 = 1.5: past white only once exposed; 1/6 of
		// exposed light, 1/12 of the light given
		{"past white only at twice the exposure", maths.Grey(0.75), 2, 1.0 / 12},
		// y = 6 * 0.5 = 3: 2 past, a third is 2/3 of exposed light,
		// 4/3 of the light given
		{"under white only at half the exposure", maths.Grey(6), 0.5, 4.0 / 3},
	} {
		for _, sigma := range []float64{1, 0.5} {
			const n = 9
			base := maths.Grey(0.05)
			lights := make([]maths.Light, n)
			for i := range lights {
				lights[i] = base
			}
			lights[0], lights[n-1] = c.star, c.star
			want := make([]maths.Light, n)
			for i := range want {
				spill := c.spill * (weight(sigma, i) + weight(sigma, n-1-i)) * weight(sigma, 0)
				want[i] = lights[i].Add(maths.Grey(spill))
			}
			for shape, wh := range map[string][2]int{"row": {n, 1}, "column": {1, n}} {
				got := bloomed(lights, wh[0], wh[1], c.exposure, sigma)
				for i := range got {
					if !near(got[i], want[i]) {
						t.Errorf("%s, sigma %v, %s, light %d: %v, want %v",
							c.name, sigma, shape, i, got[i], want[i])
					}
				}
			}
		}
	}
}

// a star away from the edges in a picture wider and taller than its
// reach spills by the weight across times the weight down, to every
// pixel in reach, and what falls off the edge is lost, not given back.
//
// every pixel starts different, so a light read from the wrong place
// shows.
func TestABloomSpillsAcrossAndDown(t *testing.T) {
	const w, h = 7, 9
	lights := make([]maths.Light, w*h)
	for i := range lights {
		lights[i] = maths.Grey(0.005 * float64(1+i))
	}
	// the star at column 1, row 4: its reach runs off the left edge
	lights[4*w+1] = maths.Grey(4)
	got := bloomed(lights, w, h, 1, 1)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			spill := weight(1, x-1) * weight(1, y-4)
			want := lights[y*w+x].Add(maths.Grey(spill))
			if !near(got[y*w+x], want) {
				t.Errorf("(%d, %d) is %v, want %v", x, y, got[y*w+x], want)
			}
		}
	}
}

// a bloom of zero, or less, is no bloom at all: a star far past white through a
// knee is as bright as the knee makes it, and no brighter for a spill onto
// itself.
//
// the star is 4 times white; through a knee of 0.5 and the default top of 100
// it is 0.5 + 0.5*ln(8)/ln(200) of the screen's white, and 5 times white would
// be more.
func TestABloomOfNoughtOrLessAddsNothing(t *testing.T) {
	f := frame.New(3, 1)
	for x, v := range []float64{0.1, 4, 0.1} {
		f.Set(x, 0, frame.Pixel{R: float32(v), G: float32(v), B: float32(v), Cover: 1})
	}
	rolled := 0.5 + 0.5*math.Log(8)/math.Log(200)
	for _, bloom := range []float64{0, -1} {
		img := Tone{Knee: 0.5, Bloom: bloom}.Develop(f)
		want := color.RGBA{Encode(rolled), Encode(rolled), Encode(rolled), 255}
		if got := img.RGBAAt(1, 0); got != want {
			t.Errorf("a star under a bloom of %v is %v, want %v", bloom, got, want)
		}
		dim := Encode(0.1)
		if got := img.RGBAAt(0, 0); got != (color.RGBA{dim, dim, dim, 255}) {
			t.Errorf("its neighbour under a bloom of %v is %v, want %v", bloom, got, dim)
		}
	}
}

// a bloom of half a pixel is a bloom: a star at four times white lights
// the pixel beside it by 1 * weight(1) * weight(0) of sigma 0.5, on top
// of the 0.1 it had.
func TestASmallBloomStillBlooms(t *testing.T) {
	f := frame.New(3, 1)
	for x, v := range []float64{0.1, 4, 0.1} {
		f.Set(x, 0, frame.Pixel{R: float32(v), G: float32(v), B: float32(v), Cover: 1})
	}
	lit := Encode(0.1 + 0.106450771974*0.786570725887)
	img := Tone{Bloom: 0.5}.Develop(f)
	if got := img.RGBAAt(0, 0); got != (color.RGBA{lit, lit, lit, 255}) {
		t.Errorf("the pixel beside a star under a bloom of 0.5 is %v, want %d", got, lit)
	}
	if lit == Encode(0.1) {
		t.Fatal("the test cannot tell a lit pixel from a dark one")
	}
}
