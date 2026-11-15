// Package colour develops linear light into display colours: exposure, optional
// bloom, highlight compression, and sRGB encoding. Think of it as the darkroom
// between the rendered frame and the screen.
package colour

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
)

// Develop is the frame as a picture, after multiplying its light by
// exposure, with a hard clamp: anything brighter than the screen's
// white is white. It is Tone with no knee.
func Develop(f *frame.Frame, exposure float64) *image.RGBA {
	return Tone{Exposure: exposure}.Develop(f)
}

// Tone is how light becomes a picture: Exposure multiplies it, zero read as
// one, and Knee, when there is one, is where a film's shoulder starts.
//
// Below the knee light is as it is; above it, it rolls off by the logarithm, so
// that Top, the brightest light the picture holds, lands on white and
// everything between keeps its detail. a Top of one or less, zero among them,
// is read as a hundred times white.
//
// The roll-off is by luminance, so a colour keeps its hue as it brightens. No
// knee is the clamp.
type Tone struct {
	Exposure float64
	Knee     float64
	Top      float64
	// Bloom is a glow like a lens's or a film's halation: the light a
	// pixel holds beyond white spills into its neighbours by a gaussian
	// of this many pixels' sigma, reaching three times as far, so a star
	// brighter than the picture holds is a glow. Nought is none.
	Bloom float64
}

// Develop is the frame as a picture through the tone.
func (t Tone) Develop(f *frame.Frame) *image.RGBA {
	lights := make([]maths.Light, f.W*f.H)
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			lights[y*f.W+x] = f.At(x, y).Light()
		}
	}
	return t.picture(lights, f.W, f.H)
}

// picture is lights, w by h, as the screen's bytes through the tone,
// bloomed first if the tone blooms.
func (t Tone) picture(lights []maths.Light, w, h int) *image.RGBA {
	exposure := t.Exposure
	if exposure == 0 {
		exposure = 1
	}
	if t.Bloom > 0 {
		lights = bloomed(lights, w, h, exposure, t.Bloom)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, t.pixel(lights[y*w+x]))
		}
	}
	return img
}

// bloomed is the lights with what each holds beyond white, at the
// exposure, spread to its neighbours by a gaussian of sigma radius and
// added back, so a point too bright to hold glows instead of clipping.
// The spill is a third of the light beyond white; the rest clips.
//
// An infinite pixel spills an infinite third of itself, the limit of
// that third as the light grows, so the pixels within its reach come out
// infinite in the parts it holds light in, and the shoulder bleaches
// them to white as it does any infinite light. A pixel that is no number
// spills nothing.
//
// No NaN comes out of lights whose parts are finite or +Inf, at a
// finite exposure. Two cases are not covered: a +Inf pixel
// within reach of a pixel with a -Inf part, which sums to NaN, and an
// infinite exposure with an infinite pixel.
//
// A radius past widest, or endless, is refused by name.
func bloomed(lights []maths.Light, w, h int,
	exposure, radius float64) []maths.Light {
	over := make([]maths.Light, len(lights))
	any := false
	for i, l := range lights {
		l = l.Scale(exposure)
		if y := l.Luminance(); y > 1 {
			spill := 1.0 / 3
			if !math.IsInf(y, 1) {
				spill = (y - 1) / y / 3
			}
			over[i] = l.Scale(spill)
			any = true
		}
	}
	if !any {
		return lights
	}
	if !(radius <= widest) {
		panic(fmt.Sprintf("colour: a bloom of %v pixels is past %v, "+
			"wider than any picture", radius, widest))
	}
	reach := int(math.Ceil(3 * radius))
	weights := make([]float64, 2*reach+1)
	sum := 0.0
	for i := range weights {
		d := float64(i - reach)
		weights[i] = math.Exp(-d * d / (2 * radius * radius))
		if d == 0 {
			// the middle is one, also for a radius whose square is
			// past what a float holds, where it would be 0/0
			weights[i] = 1
		}
		sum += weights[i]
	}
	for i := range weights {
		weights[i] /= sum
	}
	// separable: across, then down
	across := make([]maths.Light, len(lights))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var acc maths.Light
			for k, wt := range weights {
				sx := x + k - reach
				// a zero weight carries nothing, and infinity
				// times it would be no number
				if wt != 0 && sx >= 0 && sx < w {
					acc = acc.Add(over[y*w+sx].Scale(wt))
				}
			}
			across[y*w+x] = acc
		}
	}
	// the glow goes back to the scale of the light by the reciprocal of
	// the exposure, unless that is infinite, for an exposure too small to
	// invert
	inverse := 1 / exposure
	flat := math.IsInf(inverse, 0)
	out := make([]maths.Light, len(lights))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var acc maths.Light
			for k, wt := range weights {
				sy := y + k - reach
				if wt != 0 && sy >= 0 && sy < h {
					acc = acc.Add(across[sy*w+x].Scale(wt))
				}
			}
			if flat {
				// the reciprocal is infinite, and times a part of zero
				// would be no number, so the glow is divided
				acc = maths.Light{R: acc.R / exposure, G: acc.G / exposure,
					B: acc.B / exposure}
				out[y*w+x] = lights[y*w+x].Add(acc)
				continue
			}
			out[y*w+x] = lights[y*w+x].Add(acc.Scale(inverse))
		}
	}
	return out
}

// pixel is one light as the screen's bytes, through the tone.
func (t Tone) pixel(l maths.Light) color.RGBA {
	exposure := t.Exposure
	if exposure == 0 {
		exposure = 1
	}
	l = t.squeeze(l.Scale(exposure))
	return color.RGBA{Encode(l.R), Encode(l.G), Encode(l.B), 255}
}

// squeeze is the shoulder: light above the knee rolled off toward white
// by its luminance, its hue kept.
//
// A colour too saturated to fit at that luminance is drawn toward white
// along its own hue until it does, as a bright thing bleaches, rather
// than clipped a part at a time, which turns it.
func (t Tone) squeeze(l maths.Light) maths.Light {
	if t.Knee <= 0 || t.Knee >= 1 {
		return l
	}
	y := l.Luminance()
	if y <= t.Knee {
		return l
	}
	if math.IsInf(y, 1) {
		// the shoulder's own end: as light grows without bound it
		// bleaches to white, whatever its hue
		return maths.Grey(1)
	}
	room := 1 - t.Knee
	top := t.Top
	if top <= 1 {
		top = 100
	}
	squeezed := t.Knee + room*math.Log1p((y-t.Knee)/room)/
		math.Log1p((top-t.Knee)/room)
	squeezed = math.Min(1, squeezed)
	l = l.Scale(squeezed / y)
	if top := max(l.R, l.G, l.B); top > 1 {
		// the part above the grey of the same luminance, shrunk
		// until the brightest part is white
		grey := maths.Grey(squeezed)
		k := (1 - squeezed) / (top - squeezed)
		l = grey.Add(l.Add(grey.Scale(-1)).Scale(k))
	}
	return l
}

// Encode is one part of linear light, clamped to 0 through 1, as the srgb byte
// a screen expects. Screens are not linear: half the light is not half the
// byte, it is 188. The curve is libtheme's.
//
// a part that is no number is drawn as zero, black, by rule and the same on
// every machine; the language leaves its byte to the machine.
func Encode(v float64) uint8 {
	if math.IsNaN(v) {
		return 0
	}
	v = math.Max(0, math.Min(1, v))
	// the byte is the number of steps at or below v. the table says how
	// many are at or below the start of v's slot, and the few steps
	// inside the slot are counted from there; v times a power of two is
	// exact, so the slot is too.
	n := int(slotted[int(v*slots)])
	for n < len(steps) && steps[n] <= v {
		n++
	}
	return uint8(n)
}

// slots is how many slots from 0 to 1 Encode sorts a part into, a power of
// two so that a part's slot is found exactly.
const slots = 4096

// slotted is, for each slot's start, how many steps are at or below it.
var slotted = func() (t [slots + 1]uint8) {
	n := 0
	for j := range t {
		for n < len(steps) && steps[n] <= float64(j)/slots {
			n++
		}
		t[j] = uint8(n)
	}
	return t
}()

// byCurve is the byte for a part already clamped to 0 through 1, by the
// curve itself.
func byCurve(v float64) uint8 {
	return uint8(math.Round(srgb.FromLinear(v) * 255))
}

// steps are the smallest parts that encode to each byte from 1 through
// 255, found by bisection on the curve over the floats themselves, so a
// part is at or above steps[i] exactly when byCurve of it is at least
// i+1.
var steps = func() (t [255]float64) {
	for i := range t {
		want := uint8(i + 1)
		lo, hi := uint64(0), math.Float64bits(1)
		for hi-lo > 1 {
			mid := lo + (hi-lo)/2
			if byCurve(math.Float64frombits(mid)) >= want {
				hi = mid
			} else {
				lo = mid
			}
		}
		t[i] = math.Float64frombits(hi)
	}
	return t
}()

// Decode is an srgb byte as linear light: Encode, undone.
func Decode(b uint8) float64 { return decoded[b] }

var decoded = func() (t [256]float64) {
	for i := range t {
		t[i] = srgb.ToLinear(float64(i) / 255)
	}
	return t
}()

// widest is the widest bloom, in pixels' sigma: far past any picture,
// and near enough that its table of weights is a few megabytes.
const widest = 1e5
