// Package colour develops linear light into display colours: exposure, optional
// bloom, highlight compression, and sRGB encoding. Think of it as the darkroom
// between the rendered frame and the screen.
//
// The darkroom's arithmetic is ninjafilm/colour's, the one definition of
// slayer's print. This package keeps its names for its many callers and
// hands every picture there, holding no copy of its own.
package colour

import (
	"image"

	"github.com/janearc/slayer/maths"
	ninjacolour "github.com/janearc/slayer/ninjafilm/colour"
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
// The roll-off leaves the knee at the slope it had below it, one, so a
// smooth gradient that crosses the knee stays smooth: a corner there drew
// a contour across it.
//
// The roll-off is by luminance, so a colour keeps its hue as it brightens. No
// knee is the clamp. The tone is printed by ninjafilm's; see Print.
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

// Print is the tone as ninjafilm's print, which develops every picture
// of this package: the same exposure, knee, top and bloom, with the
// shoulder's scale solved once. Its Scale is what a restatement of the
// shoulder, as on the GPU, reads.
func (t Tone) Print() ninjacolour.Tone {
	return ninjacolour.NewTone(t.Exposure, t.Knee, t.Top, t.Bloom)
}

// Develop is the frame as a picture through the tone.
func (t Tone) Develop(f *frame.Frame) *image.RGBA {
	lights := make([]maths.Light, f.W*f.H)
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			lights[y*f.W+x] = f.At(x, y).Light()
		}
	}
	return t.Print().Picture(lights, f.W, f.H)
}

// Encode is one part of linear light, clamped to 0 through 1, as the srgb byte
// a screen expects. Screens are not linear: half the light is not half the
// byte, it is 188. The curve is libtheme's, and the encoding ninjafilm's.
//
// a part that is no number is drawn as zero, black, by rule and the same on
// every machine; the language leaves its byte to the machine.
func Encode(v float64) uint8 { return ninjacolour.Encode(v) }

// Decode is an srgb byte as linear light: Encode, undone.
func Decode(b uint8) float64 { return ninjacolour.Decode(b) }
