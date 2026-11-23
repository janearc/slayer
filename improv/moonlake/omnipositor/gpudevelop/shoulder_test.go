//go:build draw

package gpudevelop

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
)

// The kernel finish restates the shoulder that ninjafilm/colour defines.
// These tests develop the same lights through the kernel and through the
// tone on the CPU and require them to agree to within float32's
// precision, which makes the kernel a mirror of the one definition.
//
// The bound. The kernel works in float32, whose unit roundoff u is two to
// the minus 24. Its shoulder is a short chain of roundings of numbers no
// larger than the light: the luminance, compensated to about u, the
// subtraction of the knee, the division by the scale, one plus that,
// the logarithm, the product and the sum, each u of what it rounds.
//
// Metal's fast logarithm adds at most two to the minus 21 near one, 8u,
// and three of its own units elsewhere. The knee and the scale are each
// rounded to float32 once. So the squeezed luminance is within about 20u
// of the tone's, its slope at most one on either side of the knee.
//
// Scaling it back to a channel multiplies that by the channel's share of
// the luminance, at most one over blue's weight, about 14, and the
// bleach toward white does not grow it. A channel's light is within
// 280u, and lightError takes 512u.
//
// The sRGB code is steepest on its straight toe, 255 times 12.92 codes
// to one of light, so the kernel's code before rounding is within about
// 0.10 of the tone's, the encode's own float32 roundings far less. Its
// byte is within codeBound, a half and that, of the tone's code before
// rounding: two bytes may part by one, and only near a half-step.

// How the kernel is held to the tone.
const (
	// unitRoundoff is float32's unit roundoff.
	unitRoundoff = 0x1p-24 // defined: float32's 24-bit significand
	// lightError is how far a channel's light out of the kernel may be
	// from the tone's: see the bound above.
	lightError = 512 * unitRoundoff // picked: past the 280u derived
	// steepestCode is the sRGB code's steepest slope in codes to one of
	// light: the toe's 12.92, times the 255 codes of a byte.
	steepestCode = 255 * 12.92 // defined: IEC 61966-2-1's toe slope
	// halfByte is how far a rounded byte is from the code it rounds.
	halfByte = 0.5 // defined: rounding to the nearest
	// codeBound is how far the kernel's byte may be from the tone's code
	// before rounding.
	codeBound = halfByte + steepestCode*lightError // defined: see above
	// rampPixels is how many greys the ramp runs through.
	rampPixels = 2048 // picked: a fine ramp, one row of a small stage
	// rampPastTop is how far past the top the ramp runs, as a multiple.
	rampPastTop = 2 // picked: past white, where every light is white
	// colourSteps is how many luminances each saturated colour is taken
	// at, from the knee to past the top.
	colourSteps = 64 // picked: enough to cross the bleach
	// scaleField is the place of the shoulder's scale among Params'
	// fields: after w, h, reach, exposure and knee.
	scaleField = 5 // defined: the kernel's struct Params
	// fieldBytes is how many bytes a field of Params takes.
	fieldBytes = 4 // defined: a Metal uint or float
	// channels is how many parts of light a pixel of the stage holds.
	channels = 3 // defined: red, green and blue
	// pixelBytes is how many bytes a pixel of the picture takes.
	pixelBytes = 4 // defined: red, green, blue and alpha
	// codeTop is the largest code of a byte.
	codeTop = 255 // defined: a byte's eight bits
)

// heldTones are the tones the kernel is held to the tone at: the session's,
// a hundred times white, and a low knee with a near top. Each says its
// top, so the ramp knows how far to run; a top of one or less read as a
// hundred is ninjafilm/colour's to test.
var heldTones = []colour.Tone{
	{Exposure: 1, Knee: 0.7, Top: 8},   // defined: hillside/run.go's tone
	{Exposure: 1, Knee: 0.5, Top: 100}, // picked: the default top's size
	{Exposure: 1, Knee: 0.3, Top: 2.5}, // picked: a low knee, a near top
}

// saturated are colours the shoulder bleaches: the primaries alone and
// a deep orange, each at unit luminance before it is scaled.
var saturated = []maths.Light{
	maths.RGB(1, 0, 0),   // picked: red alone
	maths.RGB(0, 1, 0),   // picked: green alone
	maths.RGB(0, 0, 1),   // picked: blue alone, the steepest share
	maths.RGB(1, 0.3, 0), // picked: a deep orange
}

// heldLights is the stage's light for a tone, packed as the kernel takes
// it: a grey ramp from black to past the top, then each saturated colour
// at luminances from a step past the knee to past the top.
//
// Not at the knee itself: the print is continuous there, so a float32
// rounding of the luminance that puts the kernel on either side of it
// changes nothing, and the steps start a step past it.
func heldLights(tone colour.Tone) []float32 {
	var rgb []float32
	for step := range rampPixels {
		grey := float32(rampPastTop * tone.Top * float64(step) / rampPixels)
		rgb = append(rgb, grey, grey, grey)
	}
	for _, hue := range saturated {
		unit := hue.Scale(1 / hue.Luminance()) // defined: unit luminance
		for step := range colourSteps {
			past := float64(step + 1) // defined: a step past the knee
			y := tone.Knee + (rampPastTop*tone.Top-tone.Knee)*
				past/colourSteps
			l := unit.Scale(y)
			rgb = append(rgb, float32(l.R), float32(l.G), float32(l.B))
		}
	}
	return rgb
}

// heldWidth is how many pixels heldLights makes, in one row.
func heldWidth() int { return rampPixels + len(saturated)*colourSteps }

// codeOf is the sRGB code, before rounding, of a part of light as the
// tone prints it.
func codeOf(part float64) float64 {
	clamped := math.Max(0, math.Min(1, part)) // defined: black to white
	return srgb.FromLinear(clamped) * codeTop
}

// shoulderExcess develops the lights through a stage and through the
// tone's print, and gives the worst distance of a kernel byte from the
// tone's code before rounding, and the byte where it was.
func shoulderExcess(t *testing.T, stage *Developer, tone colour.Tone,
	rgb []float32) (worst float64, at int) {
	t.Helper()
	got, err := stage.Develop(rgb)
	if err != nil {
		t.Fatal(err)
	}
	printing := tone.Print()
	for pixel := range len(rgb) / channels {
		first := channels * pixel
		light := maths.RGB(float64(rgb[first]), float64(rgb[first+1]),
			float64(rgb[first+2])) // defined: green and blue follow red
		squeezed := printing.Squeeze(light)
		for part, value := range []float64{squeezed.R, squeezed.G,
			squeezed.B} {
			index := pixelBytes*pixel + part
			distance := math.Abs(float64(got.Pix[index]) - codeOf(value))
			if distance > worst {
				worst, at = distance, index
			}
		}
	}
	return worst, at
}

// the kernel's shoulder is the tone's: across a grey ramp from black to
// past the top and four saturated colours, every byte the kernel draws
// is within codeBound of the tone's code before rounding.
func TestTheKernelsShoulderIsTheTones(t *testing.T) {
	for _, tone := range heldTones {
		stage, err := New(heldWidth(), 1, tone)
		if err != nil {
			t.Fatal(err)
		}
		worst, at := shoulderExcess(t, stage, tone, heldLights(tone))
		stage.Close()
		if worst > codeBound {
			t.Errorf("knee %v, top %v: byte %d is %.4f from the tone's "+
				"code, past %.4f", tone.Knee, tone.Top, at, worst, codeBound)
		}
		t.Logf("knee %v, top %v: worst %.4f of %.4f", tone.Knee, tone.Top,
			worst, codeBound)
	}
}

// the anti-test: the kernel fed the old shoulder's denominator in place
// of the scale draws another curve, and the check sees it.
func TestTheOldDenominatorIsSeenToDisagree(t *testing.T) {
	for _, tone := range heldTones {
		stage, err := New(heldWidth(), 1, tone)
		if err != nil {
			t.Fatal(err)
		}
		// the old shoulder's denominator, as New computed it before
		// 2026-11-15: ln(1 + (top - knee)/(1 - knee))
		old := math.Log1p((tone.Top - tone.Knee) / (1 - tone.Knee))
		binary.LittleEndian.PutUint32(stage.params[scaleField*fieldBytes:],
			math.Float32bits(float32(old)))
		worst, _ := shoulderExcess(t, stage, tone, heldLights(tone))
		stage.Close()
		if !(worst > codeBound) {
			t.Errorf("knee %v, top %v: the old denominator came within "+
				"%.4f, inside %.4f", tone.Knee, tone.Top, worst, codeBound)
		}
	}
}

// underKneeLights is the stage's light for a tone as the saturated colours
// take it under the knee: each at luminances from a step over black to a
// step under the knee, well away from the knee itself. Those whose red,
// green or blue is past white there are the ones the shoulder bleaches
// without rolling their luminance off.
func underKneeLights(tone colour.Tone) []float32 {
	var rgb []float32
	for _, hue := range saturated {
		unit := hue.Scale(1 / hue.Luminance()) // defined: unit luminance
		for step := range colourSteps {
			past := float64(step + 1) // defined: a step over black
			l := unit.Scale(tone.Knee * past / (colourSteps + 1))
			rgb = append(rgb, float32(l.R), float32(l.G), float32(l.B))
		}
	}
	return rgb
}

// pastWhite is whether any part of any light is past white.
func pastWhite(rgb []float32) bool {
	for _, part := range rgb {
		if part > 1 { // defined: white
			return true
		}
	}
	return false
}

// the kernel bleaches a saturated colour under the knee as the tone does:
// a part past white is drawn toward white along the colour's hue, and not
// clipped a part at a time. Every byte the kernel draws is within
// codeBound of the tone's code before rounding.
func TestTheKernelBleachesASaturatedColourUnderTheKneeAsTheToneDoes(
	t *testing.T) {
	for _, tone := range heldTones {
		rgb := underKneeLights(tone)
		if !pastWhite(rgb) {
			t.Fatalf("knee %v: no light under the knee has a part past "+
				"white", tone.Knee)
		}
		stage, err := New(len(rgb)/channels, 1, tone)
		if err != nil {
			t.Fatal(err)
		}
		worst, at := shoulderExcess(t, stage, tone, rgb)
		stage.Close()
		if worst > codeBound {
			t.Errorf("knee %v, top %v: byte %d is %.4f from the tone's "+
				"code, past %.4f", tone.Knee, tone.Top, at, worst, codeBound)
		}
	}
}

// the anti-test: a clip of the same lights, each part held at white, is
// further from the tone's bleach than codeBound, so the test above can
// tell a kernel that clips under the knee from one that bleaches.
func TestAClipUnderTheKneeIsToldFromTheTonesBleach(t *testing.T) {
	for _, tone := range heldTones {
		rgb := underKneeLights(tone)
		printing := tone.Print()
		worst := 0.0
		for first := 0; first < len(rgb); first += channels {
			light := maths.RGB(float64(rgb[first]), float64(rgb[first+1]),
				float64(rgb[first+2])) // defined: green and blue follow red
			bleached := printing.Squeeze(light)
			clipped := maths.RGB(min(light.R, 1), min(light.G, 1),
				min(light.B, 1)) // defined: white is one
			for _, pair := range [][2]float64{{bleached.R, clipped.R},
				{bleached.G, clipped.G}, {bleached.B, clipped.B}} {
				worst = max(worst, math.Abs(codeOf(pair[0])-codeOf(pair[1])))
			}
		}
		if !(worst > codeBound) {
			t.Errorf("knee %v: a clip came within %.4f of the bleach, "+
				"inside %.4f", tone.Knee, worst, codeBound)
		}
	}
}
