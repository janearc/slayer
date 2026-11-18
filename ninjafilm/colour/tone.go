// Package colour is the film darkroom: it develops linear light into display
// colours through exposure, optional bloom, and highlight compression.
//
// It is the single definition of slayer's own print: exposure, bloom, the
// shoulder and its solved scale, the byte encoding, false colour, the
// gauge's mask and its grain. render/colour keeps its names for its
// callers and hands each picture here, holding no copy of its own.
//
// It is an ordinary package and not an internal one only because
// render/colour and improv/ must import it, and Go lets an internal
// package be imported from under ninjafilm alone. It is not a public
// surface. The sRGB curve under Encode and Decode is libtheme's.
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
// The roll-off is knee + a ln(1 + (y - knee)/a), a the scale solved once
// when the tone is made. It leaves the knee at the slope it had below
// it, one, so a smooth gradient that crosses the knee stays smooth: a
// corner there drew a contour across it.
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
	// scale is the shoulder's, solved by NewTone; see Scale
	scale float64
}

// NewTone is a tone of an exposure with a knee, a top and a bloom. The
// scale of its shoulder is solved here, once a tone and not once a pixel.
//
//ninja:vow practice slayer's pictures make their tone here
func NewTone(exposure, knee, top, bloom float64) Tone {
	made := Tone{exposure: exposure, knee: knee, top: top, bloom: bloom}
	if made.shouldered() {
		made.scale = scaleOf(knee, made.topped())
	}
	return made
}

// shouldered is whether the tone has a shoulder: a knee between nought
// and one. A knee of one has no room to roll in.
func (tone Tone) shouldered() bool {
	return tone.knee > none && tone.knee < whole
}

// topped is the tone's top, with a top of one or less read as the
// default.
func (tone Tone) topped() float64 {
	if tone.top <= whole {
		return topDefault
	}
	return tone.top
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
	return tone.Picture(lights, grid.W, grid.H)
}

// Picture is the lights as the bytes of the screen through the tone.
// They are across by down, a row at a time. They are bloomed first if
// the tone blooms. render/colour develops its frames through it.
//
//ninja:vow practice slayer's pictures develop their lights here
func (tone Tone) Picture(lights []maths.Light, across, down int) *image.RGBA {
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
			spill := whole / spillDivisor
			if !math.IsInf(luminance, whole) {
				spill = (luminance - whole) / luminance / spillDivisor
			}
			over[index] = light.Scale(spill)
			found = true
		}
	}
	return over, found
}

// BloomWeights is the weights of a gaussian of sigma radius out to the
// reach where it is cut off. It gives that reach too. The weights add up
// to one. The radius is above nought; Bloom refuses one past the widest
// before it asks.
//
// The Metal developer of improv/moonlake/omnipositor/gpudevelop blooms by
// these weights, as float32s.
//
//ninja:vow practice slayer's gpu pictures weigh their bloom here
func BloomWeights(radius float64) (weights []float64, reach int) {
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
			// a weight of zero carries nothing, and infinity times it
			// is no number
			if weight != none && along >= none && along < limit {
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
// An infinite light spills an infinite third of itself, which is what
// that third comes to as the light grows. So the pixels within its reach
// come out infinite in the parts it holds light in. The shoulder bleaches
// them to white as it does any infinite light. A light that is no number
// spills nothing.
//
// No NaN comes out of lights whose parts are finite or positive
// infinity, at a finite exposure. Two cases are not covered. One
// is an infinite light within reach of a light with a part of negative
// infinity, which sums to NaN. The other is an infinite exposure with an
// infinite light.
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
	weights, reach := BloomWeights(radius)
	sideways := spread(over, weights, reach, across, down, false)
	glow := spread(sideways, weights, reach, across, down, true)
	// the glow goes back to the scale of the light by the reciprocal of
	// the exposure, unless that is infinite, for an exposure too small to
	// invert
	inverse := whole / exposure
	flat := math.IsInf(inverse, whole) || math.IsInf(inverse, -whole)
	out := make([]maths.Light, len(lights))
	for index, light := range lights {
		if flat {
			// times a part of zero the reciprocal would be no number,
			// so the glow is divided
			back := maths.Light{R: glow[index].R / exposure,
				G: glow[index].G / exposure, B: glow[index].B / exposure}
			out[index] = light.Add(back)
			continue
		}
		out[index] = light.Add(glow[index].Scale(inverse))
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
// white by its luminance, as knee + a ln(1 + (y - knee)/a), a the
// tone's Scale. Its hue is kept.
//
// Colours too saturated to fit at that luminance are drawn toward white
// along their own hue until they do. That is how a bright thing
// bleaches. They are not clipped a part at a time, which turns them.
//
//ninja:vow practice slayer's gpu pictures squeeze their light here
func (tone Tone) Squeeze(light maths.Light) maths.Light {
	if !tone.shouldered() {
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
	rolled := math.Log1p((luminance - tone.knee) / tone.scale)
	squeezed := math.Min(whole, tone.knee+tone.scale*rolled)
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
// The byte is the count of steps at or below the amount, the steps being
// where libtheme's curve steps. A table says how many are at or below
// the start of the amount's slot, and the few inside it are counted from
// there. The amount times a power of two is exact, so the slot is too.
//
//ninja:vow practice slayer encodes its colours here
func Encode(amount float64) uint8 {
	if math.IsNaN(amount) {
		return none
	}
	amount = math.Max(none, math.Min(whole, amount))
	count := int(slotted[int(amount*slots)])
	for count < len(steps) && steps[count] <= amount {
		count++
	}
	return uint8(count)
}

// slots is how many slots from nought to one Encode sorts an amount into.
const slots = 4096 // defined: a power of two, so an amount's slot is exact

// slotted is, for each slot's start, how many steps are at or below it.
var slotted = func() (table [slots + whole]uint8) {
	count := none
	for slot := range table {
		start := float64(slot) / slots
		for count < len(steps) && steps[count] <= start {
			count++
		}
		table[slot] = uint8(count)
	}
	return table
}()

// byCurve is the byte of an amount already clamped to nought through
// one, by libtheme's curve itself.
func byCurve(amount float64) uint8 {
	return uint8(math.Round(srgb.FromLinear(amount) * byteMost))
}

// steps are the smallest amounts that encode to each byte from one
// through the most, found by bisection on the curve over the floats
// themselves. An amount is at or above steps[i] exactly when byCurve of
// it is at least i + 1.
var steps = func() (table [byteMost]float64) {
	for index := range table {
		want := uint8(index + whole)
		low, high := uint64(none), math.Float64bits(whole)
		for high-low > whole {
			middle := low + (high-low)/pair
			if byCurve(math.Float64frombits(middle)) >= want {
				high = middle
			} else {
				low = middle
			}
		}
		table[index] = math.Float64frombits(high)
	}
	return table
}()

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
