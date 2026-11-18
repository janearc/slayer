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

// TestClearInscatterComparisonDraw renders the same bounded scene with a
// synthetic clear medium's scattering coefficient set to zero and nonzero.
// It is opt-in so ordinary draw-tag runs do not write into a user's gallery.
func TestClearInscatterComparisonDraw(t *testing.T) {
	directory := os.Getenv("SLAYER_CLEAR_INSCATTER_COMPARISON_OUT")
	if directory == "" {
		t.Skip("set SLAYER_CLEAR_INSCATTER_COMPARISON_OUT to write the comparison")
	}
	const (
		width, height = 161, 91
		distance      = 4.0
		phase         = 0.35
		glow          = 0.65
	)
	absorption := maths.RGB(0.06, 0.10, 0.14)
	scattering := maths.RGB(0.05, 0.08, 0.10)
	incident := maths.Grey(30)
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Name: "synthetic emissive reference", Glow: maths.Grey(glow)},
		Motion:   resting(maths.V(0, distance, 0))})
	world.Lamp(thing.Sun(maths.V(1, 0, 0), incident))
	eye := mediumEye(maths.Vec{}, maths.Up)

	makeMedium := func(name string, scatter maths.Light) *thing.ClearMedium {
		t.Helper()
		made, err := thing.NewClearMediumWithScattering(name,
			absorption, scatter, phase, 1, 8, distance)
		if err != nil {
			t.Fatal(err)
		}
		return made
	}
	render := func(medium *thing.ClearMedium) *frame.Frame {
		t.Helper()
		made, err := RenderWithInitialMedia(world, eye, film.Pinhole{Wide: 40 * maths.Degree},
			0, width, height,
			[]InitialMedium{{Index: 1, ClearMedium: medium}})
		if err != nil {
			t.Fatal(err)
		}
		return made
	}
	off := render(makeMedium("synthetic-scattering-off", maths.Light{}))
	on := render(makeMedium("synthetic-scattering-on", scattering))
	assertClearInscatterAnalytic(t, "scattering-off", off.At(width/2, height/2),
		absorption, maths.Light{}, incident, glow, phase, distance)
	assertClearInscatterAnalytic(t, "scattering-on", on.At(width/2, height/2),
		absorption, scattering, incident, glow, phase, distance)
	if off.At(width/2, height/2) == on.At(width/2, height/2) {
		t.Fatal("center pixels are identical; comparison does not reveal the scattering change")
	}

	writeClearInscatterImage(t, directory, "scattering-off.png", off)
	writeClearInscatterImage(t, directory, "scattering-on.png", on)
	if err := os.WriteFile(filepath.Join(directory, "analytic.txt"), []byte(
		clearInscatterAnalyticReport(off.At(width/2, height/2),
			on.At(width/2, height/2), absorption, scattering, incident,
			glow, phase, distance)), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote CPU comparison at %dx%d; synthetic absorption=%v/m scattering-off=0/m scattering-on=%v/m path=%.1fm phase=%.2f",
		width, height, absorption, scattering, distance, phase)
}

func assertClearInscatterAnalytic(test *testing.T, label string,
	got frame.Pixel, absorption, scattering, incident maths.Light,
	glow, g, distance float64,
) {
	test.Helper()
	phase := (1 - g*g) / (4 * math.Pi * math.Pow(1+g*g, 1.5)) // HG at cos=0
	for _, channel := range []struct {
		name            string
		got             float64
		absorb, scatter float64
		lamp            float64
	}{{"red", float64(got.R), absorption.R, scattering.R, incident.R},
		{"green", float64(got.G), absorption.G, scattering.G, incident.G},
		{"blue", float64(got.B), absorption.B, scattering.B, incident.B}} {
		total := channel.absorb + channel.scatter
		want := glow * math.Exp(-total*distance)
		if channel.scatter > 0 {
			want += channel.lamp * phase * channel.scatter *
				(1 - math.Exp(-total*distance)) / total
		}
		if math.Abs(channel.got-want) > 2e-5 {
			test.Errorf("%s %s center=%.8g want analytic %.8g", label,
				channel.name, channel.got, want)
		}
	}
}

func clearInscatterAnalyticReport(off, on frame.Pixel, absorption,
	scattering, incident maths.Light, glow, g, distance float64,
) string {
	phase := (1 - g*g) / (4 * math.Pi * math.Pow(1+g*g, 1.5))
	return fmt.Sprintf(`Synthetic homogeneous medium comparison (all RGB coefficients are inverse metres)
Camera path: %.3f m; phase: HG g=%.3f, evaluated at cos(theta)=0; phase density=%.9g sr^-1
Absorption RGB: %.6g %.6g %.6g
Scattering off RGB: 0 0 0
Scattering on RGB: %.6g %.6g %.6g
Directional lamp RGB: %.6g %.6g %.6g; endpoint glow RGB: %.6g %.6g %.6g
Analytic per channel: glow*exp(-(sigma_a+sigma_s)*d) + lamp*phase*sigma_s*(1-exp(-(sigma_a+sigma_s)*d))/(sigma_a+sigma_s)
When sigma_s=0 this reduces to glow*exp(-sigma_a*d). Incoming lamp illumination is constant, unattenuated and unrefracted at each sample; the only modeled attenuation is outgoing camera-path extinction.
Center pixel off raw RGB: %.9g %.9g %.9g
Center pixel on raw RGB: %.9g %.9g %.9g
`, distance, g, phase, absorption.R, absorption.G, absorption.B,
		scattering.R, scattering.G, scattering.B, incident.R, incident.G,
		incident.B, glow, glow, glow, off.R, off.G, off.B, on.R, on.G, on.B)
}

func writeClearInscatterImage(test *testing.T, directory, name string,
	made *frame.Frame,
) {
	test.Helper()
	if err := os.MkdirAll(directory, 0o755); err != nil {
		test.Fatal(err)
	}
	file, err := os.OpenFile(filepath.Join(directory, name),
		os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
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
