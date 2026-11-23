//go:build draw

package trace

import (
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// The comparison's numbers: a picture 161 by 91, whose middle pixel
// looks straight along the eye's way; four metres of medium between the
// eye and a glowing wall; a Henyey-Greenstein asymmetry of 0.35; and a
// wall that glows 0.65.
const (
	studyWidth, studyHeight = 161, 91
	studyDistance           = 4.0
	studyPhase              = 0.35
	studyGlow               = 0.65
)

// What the comparison's medium absorbs and scatters a metre, and the
// sun that lights it from the side, square across the eye's way.
var (
	studyAbsorption = maths.RGB(0.06, 0.10, 0.14)
	studyScattering = maths.RGB(0.05, 0.08, 0.10)
	studyIncident   = maths.Grey(30)
)

// studyPicture is the comparison's scene drawn from inside a medium
// that scatters so much: an eye looking up at a glowing wall four metres
// away, under a sun from one side.
func studyPicture(t *testing.T, name string,
	scatter maths.Light) *frame.Frame {
	t.Helper()
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Name: "synthetic emissive reference",
			Glow: maths.Grey(studyGlow)},
		Motion: resting(maths.V(0, studyDistance, 0))})
	world.Lamp(thing.Sun(maths.V(1, 0, 0), studyIncident))
	medium, err := thing.NewClearMediumWithScattering(name,
		studyAbsorption, scatter, studyPhase, 1, 8, studyDistance)
	if err != nil {
		t.Fatal(err)
	}
	made, err := RenderWithInitialMedia(world, mediumEye(maths.Vec{},
		maths.Up), film.Pinhole{Wide: 40 * maths.Degree}, 0, studyWidth,
		studyHeight, []InitialMedium{{Index: 1, ClearMedium: medium}})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// studyPhaseAcross is Henyey and Greenstein's lobe at a right angle, a
// cosine of nought: (1 - g^2) / (4 pi (1 + g^2)^1.5).
func studyPhaseAcross() float64 {
	g := studyPhase
	return (1 - g*g) / (4 * math.Pi * math.Pow(1+g*g, 1.5))
}

// clearInscatterWant is the middle pixel worked by hand for a medium
// that scatters so much. The wall's glow comes through d metres of
// extinction t, absorption and scattering together, as glow e^(-t d).
// Each metre of the way scatters toward the eye the sun's light times
// the phase across times the scattering s, and that comes through what
// is left of the way: the integral of s e^(-t x) over the way is
// s (1 - e^(-t d)) / t. The sun is not dimmed on its way in.
func clearInscatterWant(scattering maths.Light) maths.Light {
	channel := func(absorb, scatter, lamp float64) float64 {
		total := absorb + scatter
		kept := math.Exp(-total * studyDistance)
		return studyGlow*kept +
			lamp*studyPhaseAcross()*scatter*(1-kept)/total
	}
	return maths.RGB(
		channel(studyAbsorption.R, scattering.R, studyIncident.R),
		channel(studyAbsorption.G, scattering.G, studyIncident.G),
		channel(studyAbsorption.B, scattering.B, studyIncident.B))
}

// pixelNear is whether a pixel's light is within tol of a light in every
// channel. A NaN is near nothing.
func pixelNear(got frame.Pixel, want maths.Light, tol float64) bool {
	return within(float64(got.R), want.R, tol) &&
		within(float64(got.G), want.G, tol) &&
		within(float64(got.B), want.B, tol)
}

// studyClose is how near the middle pixel is held to its worked value: a
// pixel is single precision, and its light is under one.
const studyClose = 2e-5

// TestClearInscatterComparisonDraw draws the same bounded scene from
// inside a clear medium that scatters nothing and one that scatters,
// and holds each middle pixel to its worked value (clearInscatterWant).
// The pictures are written only where
// SLAYER_CLEAR_INSCATTER_COMPARISON_OUT names a folder, so that a draw
// run does not write into a gallery; the pixels are held in every run.
func TestClearInscatterComparisonDraw(t *testing.T) {
	off := studyPicture(t, "synthetic-scattering-off", maths.Light{})
	on := studyPicture(t, "synthetic-scattering-on", studyScattering)
	offMiddle := off.At(studyWidth/2, studyHeight/2)
	onMiddle := on.At(studyWidth/2, studyHeight/2)
	if want := clearInscatterWant(maths.Light{}); !pixelNear(offMiddle,
		want, studyClose) {
		t.Errorf("scattering off: the middle is %+v, want %v", offMiddle,
			want)
	}
	if want := clearInscatterWant(studyScattering); !pixelNear(onMiddle,
		want, studyClose) {
		t.Errorf("scattering on: the middle is %+v, want %v", onMiddle,
			want)
	}
	directory := os.Getenv("SLAYER_CLEAR_INSCATTER_COMPARISON_OUT")
	writeClearInscatterImage(t, directory, "scattering-off.png", off)
	writeClearInscatterImage(t, directory, "scattering-on.png", on)
	if directory == "" {
		return
	}
	if err := os.WriteFile(filepath.Join(directory, "analytic.txt"), []byte(
		clearInscatterAnalyticReport(offMiddle, onMiddle)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Anti: the worked value tells a medium that scatters from one that
// does not. The picture drawn with no scattering is not near the value
// worked for scattering, in any channel: with scattering the glow is
// dimmed more, to 0.42 in red, and the sun adds 0.29, so 0.70 against
// the 0.51 of no scattering.
func TestClearInscatterWorkedValueTellsScatteringFromNone(t *testing.T) {
	off := studyPicture(t, "synthetic-scattering-off", maths.Light{}).At(
		studyWidth/2, studyHeight/2)
	want := clearInscatterWant(studyScattering)
	if within(float64(off.R), want.R, studyClose) ||
		within(float64(off.G), want.G, studyClose) ||
		within(float64(off.B), want.B, studyClose) {
		t.Errorf("a medium that scatters nothing drew %+v, which passes "+
			"for the scattering one's %v", off, want)
	}
}

// clearInscatterAnalyticReport is the comparison's numbers as text, for
// the folder its pictures are written to.
func clearInscatterAnalyticReport(off, on frame.Pixel) string {
	return fmt.Sprintf(`Synthetic homogeneous medium comparison (all RGB coefficients are inverse metres)
Camera path: %.3f m; phase: HG g=%.3f, evaluated at cos(theta)=0; phase density=%.9g sr^-1
Absorption RGB: %.6g %.6g %.6g
Scattering off RGB: 0 0 0
Scattering on RGB: %.6g %.6g %.6g
Directional lamp RGB: %.6g %.6g %.6g; endpoint glow: %.6g
Analytic per channel: glow*exp(-(sigma_a+sigma_s)*d) + lamp*phase*sigma_s*(1-exp(-(sigma_a+sigma_s)*d))/(sigma_a+sigma_s)
When sigma_s=0 this reduces to glow*exp(-sigma_a*d). Incoming lamp illumination is constant, unattenuated and unrefracted at each sample; the only modeled attenuation is outgoing camera-path extinction.
Center pixel off raw RGB: %.9g %.9g %.9g
Center pixel on raw RGB: %.9g %.9g %.9g
`, studyDistance, studyPhase, studyPhaseAcross(), studyAbsorption.R,
		studyAbsorption.G, studyAbsorption.B, studyScattering.R,
		studyScattering.G, studyScattering.B, studyIncident.R,
		studyIncident.G, studyIncident.B, studyGlow, off.R, off.G, off.B,
		on.R, on.G, on.B)
}

// writeClearInscatterImage writes a picture, toned, into a folder under
// a name, over one already there; with no folder named it writes
// nothing.
func writeClearInscatterImage(test *testing.T, directory, name string,
	made *frame.Frame,
) {
	test.Helper()
	if directory == "" {
		return
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		test.Fatal(err)
	}
	file, err := os.OpenFile(filepath.Join(directory, name),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		test.Fatal(err)
	}
	if err := png.Encode(file,
		colour.Tone{Exposure: 1, Knee: 0.8, Top: 2.2}.Develop(made)); err != nil {
		_ = file.Close()
		test.Fatal(err)
	}
	if err := file.Close(); err != nil {
		test.Fatal(err)
	}
	test.Logf("wrote %s", filepath.Join(directory, name))
}
