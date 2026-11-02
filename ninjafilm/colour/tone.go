// Package colour is the film darkroom: it develops linear light into display
// colours through exposure, optional bloom, and highlight compression.
package colour

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/frame"
)

// what the darkroom counts in, each defined
const (
	// none is no amount. It is defined as zero.
	none = 0
	// whole is the whole of a share, and one. It is defined as one.
	whole = 1
	// pair is a count of two. It is defined as two.
	pair = 2
	// byteMost is the largest value a byte holds. It is defined by the
	// eight bits of a byte.
	byteMost = 255
	// bytes is how many values a byte holds. It is defined by the same.
	bytes = 256
	// half is half of a share. It is defined as one over two.
	half = 0.5
	// byteWidth is how many bytes a pixel takes. They are red and green
	// and blue and alpha. It is defined by those four.
	byteWidth = 4
	// first is the place of the first of a list. It is defined as zero.
	first = 0
)

var (
	// topDefault is how many times white the brightest light is read
	// to be when none is given.
	topDefault = data.Picked("tone-top-default")
	// reachSigmas is how many sigmas out a bloom reaches.
	reachSigmas = data.Picked("bloom-reach-sigmas")
	// spillDivisor is what the light beyond white is divided by for
	// the share that spills.
	spillDivisor = data.Picked("bloom-spill-divisor")
	// widest is the widest bloom, in pixels of sigma.
	widest = data.Picked("bloom-widest")
)

// Develop is the frame as a picture after its light is multiplied by the
// exposure. It has a hard clamp. Anything brighter than the white of the
// screen is white. It is Tone with no knee.
//
//ninja:vow practice slayer develops its frames here
func Develop(grid *frame.Frame, exposure float64) *image.RGBA {
	return NewTone(exposure, none, none, none).Develop(grid)
}

// Tone is how light becomes a picture. Its exposure multiplies the
// light. Nought is read as one. Its knee is where the shoulder of a film
// starts, when there is one.
//
// Below the knee light is as it is. Above it the light rolls off by the
// logarithm. Its top is the brightest light the picture holds. That
// lands on white. Everything between keeps its detail. Tops of one or
// less are read as a hundred times white. Nought is among them.
//
// The roll-off is by luminance. So a colour keeps its hue as it
// brightens. No knee is the clamp.
//
// Its bloom is a glow like the glow of a lens or the halation of a film.
// The light a pixel holds beyond white spills into its neighbours.
//
// The spill is by a gaussian of this many pixels of sigma. The gaussian
// reaches three times as far. So a star brighter than the picture holds
// is a glow. Nought is none.
//
//ninja:vow practice slayer's pictures hold their tone
type Tone struct {
	exposure, knee, top, bloom float64
}

// NewTone is a tone of an exposure with a knee, a top and a bloom.
//
//ninja:vow practice slayer's pictures make their tone here
func NewTone(exposure, knee, top, bloom float64) Tone {
	return Tone{exposure, knee, top, bloom}
}

// exposed is the exposure, with zero read as one.
func (tone Tone) exposed() float64 {
	if tone.exposure == none {
		return whole
	}
	return tone.exposure
}

// Develop is the frame as a picture through the tone.
//
//ninja:vow practice slayer's pictures develop their frames here
func (tone Tone) Develop(grid *frame.Frame) *image.RGBA {
	lights := make([]maths.Light, grid.W*grid.H)
	for y := range grid.H {
		for x := range grid.W {
			lights[y*grid.W+x] = grid.At(x, y).Light()
		}
	}
	return tone.picture(lights, grid.W, grid.H)
}

// picture is the lights as the bytes of the screen through the tone.
// They are across by down. They are bloomed first if the tone blooms.
func (tone Tone) picture(lights []maths.Light, across, down int) *image.RGBA {
	if tone.bloom > none {
		lights = Bloom(lights, across, down, tone.exposed(), tone.bloom)
	}
	img := image.NewRGBA(image.Rect(none, none, across, down))
	for y := range down {
		for x := range across {
			img.SetRGBA(x, y, tone.pixel(lights[y*across+x]))
		}
	}
	return img
}

// beyondWhite is the light each of the lights holds beyond white at the
// exposure. It says whether any holds some. Of that a third spills and
// the rest clips.
func beyondWhite(lights []maths.Light, exposure float64) (over []maths.Light,
	found bool) {
	over = make([]maths.Light, len(lights))
	for index, light := range lights {
		light = light.Scale(exposure)
		if luminance := light.Luminance(); luminance > whole {
			spill := (luminance - whole) / luminance / spillDivisor
			over[index] = light.Scale(spill)
			found = true
		}
	}
	return over, found
}

// kernel is the weights of a gaussian of sigma radius out to the reach
// where it is cut off. It gives that reach too. The weights add up to
// one.
func kernel(radius float64) (weights []float64, reach int) {
	reach = int(math.Ceil(reachSigmas * radius))
	weights = make([]float64, pair*reach+whole)
	var sum float64
	for index := range weights {
		offset := float64(index - reach)
		square := pair * radius * radius
		weights[index] = math.Exp(-offset * offset / square)
		if offset == none {
			// the middle is one. That holds for a radius whose
			// square is past what a float holds, where it would be
			// 0/0
			weights[index] = whole
		}
		sum += weights[index]
	}
	for index := range weights {
		weights[index] /= sum
	}
	return weights, reach
}

// spread is the lights blurred along one line of the picture by the
// weights. It is along the rows, or down the columns if vertical.
// What would fall off an edge is lost.
func spread(lights []maths.Light, weights []float64, reach, across,
	down int, vertical bool) []maths.Light {
	limit, stride := across, whole
	if vertical {
		limit, stride = down, across
	}
	blurred := make([]maths.Light, len(lights))
	for index := range lights {
		at := index % across
		if vertical {
			at = index / across
		}
		var sum maths.Light
		for step, weight := range weights {
			along := at + step - reach
			if along >= none && along < limit {
				near := lights[index+(along-at)*stride]
				sum = sum.Add(near.Scale(weight))
			}
		}
		blurred[index] = sum
	}
	return blurred
}

// Bloom is the lights with what each holds beyond white at the exposure
// spread to its neighbours. The spread is by a gaussian of sigma radius.
// It is added back. So a point too bright to hold glows instead of
// clipping.
//
// The spill is a third of the light beyond white. The rest clips.
//
// Radii past the widest are refused by name. So are endless ones. The
// refusal is a panic. The lights are across by down.
//
//ninja:vow practice slayer's gpu pictures bloom their lights here
func Bloom(lights []maths.Light, across, down int, exposure,
	radius float64) []maths.Light {
	over, found := beyondWhite(lights, exposure)
	if !found {
		return lights
	}
	if !(radius <= widest) {
		panic(fmt.Sprintf("colour: a bloom of %v pixels is past %v, "+
			"wider than any picture", radius, widest))
	}
	weights, reach := kernel(radius)
	sideways := spread(over, weights, reach, across, down, false)
	glow := spread(sideways, weights, reach, across, down, true)
	out := make([]maths.Light, len(lights))
	for index, light := range lights {
		out[index] = light.Add(glow[index].Scale(whole / exposure))
	}
	return out
}

// pixel is one light as the bytes of the screen through the tone.
func (tone Tone) pixel(light maths.Light) color.RGBA {
	exposed := light.Scale(tone.exposed())
	squeezed := tone.Squeeze(exposed)
	return color.RGBA{Encode(squeezed.R), Encode(squeezed.G),
		Encode(squeezed.B), byteMost}
}

// Squeeze is the shoulder. Light above the knee is rolled off toward
// white by its luminance. Its hue is kept.
//
// Colours too saturated to fit at that luminance are drawn toward white
// along their own hue until they do. That is how a bright thing
// bleaches. They are not clipped a part at a time, which turns them.
//
//ninja:vow practice slayer's gpu pictures squeeze their light here
func (tone Tone) Squeeze(light maths.Light) maths.Light {
	if tone.knee <= none || tone.knee >= whole {
		return light
	}
	luminance := light.Luminance()
	if luminance <= tone.knee {
		return light
	}
	if math.IsInf(luminance, whole) {
		// the end of the shoulder: as light grows without bound it
		// bleaches to white, whatever its hue
		return maths.Grey(whole)
	}
	room := whole - tone.knee
	top := tone.top
	if top <= whole {
		top = topDefault
	}
	rolled := math.Log1p((luminance - tone.knee) / room)
	squeezed := tone.knee + room*rolled/math.Log1p((top-tone.knee)/room)
	squeezed = math.Min(whole, squeezed)
	light = light.Scale(squeezed / luminance)
	if brightest := max(light.R, light.G, light.B); brightest > whole {
		// the part above the grey of the same luminance, shrunk
		// until the brightest part is white
		grey := maths.Grey(squeezed)
		shrink := (whole - squeezed) / (brightest - squeezed)
		light = grey.Add(light.Add(grey.Scale(-whole)).Scale(shrink))
	}
	return light
}

// Encode is one part of linear light as the srgb byte a screen expects.
// It is clamped to zero through one. Screens are not linear. Half the
// light is not half the byte. It is 188. The curve is libtheme's.
//
// Parts that are no number are drawn as zero. That is black by rule
// and the same on every machine. The language leaves their byte to the
// machine.
//
//ninja:vow practice slayer encodes its colours here
func Encode(amount float64) uint8 {
	if math.IsNaN(amount) {
		return none
	}
	amount = math.Max(none, math.Min(whole, amount))
	curve := srgb.FromLinear(amount)
	return uint8(math.Round(curve * byteMost))
}

// Decode is an srgb byte as linear light. It is Encode undone.
//
//ninja:vow practice slayer decodes the bytes it reads here
func Decode(code uint8) float64 { return decoded[code] }

// decoded is the linear light of each byte.
var decoded = decodings()

// decodings is the linear light of each of the bytes.
func decodings() (table [bytes]float64) {
	for code := range table {
		table[code] = srgb.ToLinear(float64(code) / byteMost)
	}
	return table
}
