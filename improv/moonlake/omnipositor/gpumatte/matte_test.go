//go:build draw

package gpumatte

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/matte"
	"github.com/janearc/slayer/metal"
)

func TestApplyMatchesCPUWithSignedCancellationInMoonReduction(t *testing.T) {
	raw := Image{Width: 4, Height: 1, Scale: -1, Pixels: []float32{
		0.13, 0.27, 0.41, 0.55, 0.69, 0.83, 0.21, 0.34, 0.47, 0.58, 0.72, 0.86,
	}}
	sea := Image{Width: 4, Height: 1, Scale: 1, Pixels: []float32{
		0.91, 0.79, 0.67, 0.53, 0.41, 0.29, 0.17, 0.05, 0.23, 0.35, 0.49, 0.63,
	}}
	moon := Image{Width: 257, Height: 1, Scale: -1, Pixels: make([]float32, 257*3)}
	huge := float32(math.MaxFloat32)
	moon.Pixels[0] = huge
	moon.Pixels[1] = -huge
	moon.Pixels[2] = 2
	moon.Pixels[3] = math.Nextafter32(1, 2)
	moon.Pixels[6] = 1
	moon.Pixels[9] = math.Nextafter32(1, 0)
	moon.Pixels[12] = math.Nextafter32(1, float32(math.Inf(1)))
	cfg := Config{Top: 8, Bottom: -8, Shore: 0, MoonRadius: 0.35, MoonRef: &moon}
	got, err := Apply(raw, sea, cfg)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatal(err)
	}
	want, err := matte.Apply(raw, sea, cfg)
	if err != nil {
		t.Fatal(err)
	}
	assertImagesWithinULP(t, got, want, 1)
	assertImageFinite(t, got)
	assertImageFinite(t, want)
	assertImageHasNonZeroPixel(t, got)
}

func TestApplyMatchesCPUWithExtremeFiniteNearThresholdRGB(t *testing.T) {
	raw := Image{Width: 4, Height: 1, Scale: -1, Pixels: []float32{
		0.18, 0.28, 0.38, 0.48, 0.58, 0.68, 0.78, 0.88, 0.98, 0.12, 0.22, 0.32,
	}}
	sea := Image{Width: 4, Height: 1, Scale: 1, Pixels: []float32{
		0.81, 0.71, 0.61, 0.51, 0.41, 0.31, 0.21, 0.11, 0.01, 0.91, 0.89, 0.87,
	}}
	half := float32(math.MaxFloat32 / 2)
	moon := Image{Width: 4, Height: 1, Scale: -1, Pixels: []float32{
		half, half, 0,
		half, 0, 0,
		math.Nextafter32(half, 0), 0, 0,
		math.Nextafter32(half, float32(math.Inf(1))), 0, 0,
	}}
	cfg := Config{Top: 6, Bottom: -6, Shore: 0, MoonRadius: 0.5, MoonRef: &moon}
	got, err := Apply(raw, sea, cfg)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		if strings.Contains(err.Error(), "moon red reduction produced nonfinite value") {
			if want, wantErr := matte.Apply(raw, sea, cfg); wantErr != nil {
				t.Fatal(wantErr)
			} else if len(want.Pixels) == 0 {
				t.Fatal("cpu output unexpectedly empty")
			}
			return
		}
		t.Fatal(err)
	}
	want, err := matte.Apply(raw, sea, cfg)
	if err != nil {
		t.Fatal(err)
	}
	assertImagesWithinULP(t, got, want, 1)
	assertImageFinite(t, got)
	assertImageFinite(t, want)
}

func TestApplyMatchesCPUWithDifferentSizedMoonReference(t *testing.T) {
	raw := Image{Width: 3, Height: 2, Scale: -1, Pixels: []float32{
		0.1, 0.2, 0.3, 0.9, 0.8, 0.7, 0.4, 0.5, 0.6,
		0.7, 0.6, 0.5, 0.3, 0.2, 0.1, 0.8, 0.7, 0.6,
	}}
	sea := Image{Width: 3, Height: 2, Scale: 1, Pixels: []float32{
		0.8, 0.6, 0.4, 0.2, 0.4, 0.6, 0.1, 0.3, 0.5,
		0.5, 0.7, 0.9, 0.3, 0.5, 0.7, 0.9, 0.2, 0.4,
	}}
	moon := Image{Width: 1, Height: 2, Scale: -1, Pixels: []float32{
		3.0, 1.0, 0.5,
		0.2, 0.9, 2.1,
	}}
	rawBefore := append([]float32(nil), raw.Pixels...)
	seaBefore := append([]float32(nil), sea.Pixels...)
	moonBefore := append([]float32(nil), moon.Pixels...)
	cfg := Config{Top: 6, Bottom: -2, Shore: 1, MoonRadius: 0.35, MoonRef: &moon}
	got, err := Apply(raw, sea, cfg)
	if err != nil {
		t.Fatal(err)
	}
	want, err := matte.Apply(raw, sea, cfg)
	if err != nil {
		t.Fatal(err)
	}
	assertImagesClose(t, got, want)
	assertFloat32sEqual(t, raw.Pixels, rawBefore, "raw input mutated")
	assertFloat32sEqual(t, sea.Pixels, seaBefore, "sea input mutated")
	assertFloat32sEqual(t, moon.Pixels, moonBefore, "moon reference input mutated")
	if sameFloat32s(got.Pixels, raw.Pixels) {
		t.Fatal("gpu matte returned the raw input unchanged")
	}
}

func TestApplyRejectsZeroMaskButSupportsTinyMasks(t *testing.T) {
	raw := Image{Width: 2, Height: 2, Scale: -1, Pixels: []float32{
		0.12, 0.22, 0.32, 0.42, 0.52, 0.62,
		0.72, 0.82, 0.92, 0.18, 0.28, 0.38,
	}}
	sea := Image{Width: 2, Height: 2, Scale: 1, Pixels: []float32{
		0.91, 0.81, 0.71, 0.61, 0.51, 0.41,
		0.31, 0.21, 0.11, 0.19, 0.29, 0.39,
	}}
	zeroMask := Image{Width: 2, Height: 2, Scale: 1, Pixels: make([]float32, 12)}
	tinyMask := Image{Width: 2, Height: 2, Scale: 1, Pixels: []float32{
		1e-6, 0, 0, 0, 2e-6, 0,
		0, 0, 3e-6, 0, 0, 4e-6,
	}}
	t.Run("zero mask", func(t *testing.T) {
		if _, err := Apply(raw, sea, Config{Top: 5, Bottom: -3, Shore: 1.25, MoonRadius: 0.28, WaterMask: &zeroMask}); err == nil || !strings.Contains(err.Error(), "water mask must contain positive light") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("tiny mask", func(t *testing.T) {
		cfg := Config{Top: 5, Bottom: -3, Shore: 1.25, MoonRadius: 0.28, WaterMask: &tinyMask}
		got, err := Apply(raw, sea, cfg)
		if err != nil {
			t.Fatal(err)
		}
		want, err := matte.Apply(raw, sea, cfg)
		if err != nil {
			t.Fatal(err)
		}
		assertImagesClose(t, got, want)
		assertImageFinite(t, got)
	})
}

func TestApplyRejectsPositiveSubnormalWaterMask(t *testing.T) {
	raw := Image{Width: 2, Height: 2, Scale: -1, Pixels: []float32{
		0.12, 0.22, 0.32, 0.42, 0.52, 0.62,
		0.72, 0.82, 0.92, 0.18, 0.28, 0.38,
	}}
	sea := Image{Width: 2, Height: 2, Scale: 1, Pixels: []float32{
		0.91, 0.81, 0.71, 0.61, 0.51, 0.41,
		0.31, 0.21, 0.11, 0.19, 0.29, 0.39,
	}}
	subnormal := math.Float32frombits(0x007fffff)
	mask := Image{Width: 2, Height: 2, Scale: 1, Pixels: []float32{
		subnormal, 0, 0, 0, 1, 0,
		0, 0, 0, 2, 0, 0,
	}}
	if _, err := Apply(raw, sea, Config{Top: 5, Bottom: -3, Shore: 1.25, MoonRadius: 0.28, WaterMask: &mask}); err == nil || !strings.Contains(err.Error(), "water mask contains positive subnormal values") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyNearNormalWaterMaskStaysWithinOneULP(t *testing.T) {
	raw := Image{Width: 2, Height: 2, Scale: -1, Pixels: []float32{
		0.12, 0.22, 0.32, 0.42, 0.52, 0.62,
		0.72, 0.82, 0.92, 0.18, 0.28, 0.38,
	}}
	sea := Image{Width: 2, Height: 2, Scale: 1, Pixels: []float32{
		0.91, 0.81, 0.71, 0.61, 0.51, 0.41,
		0.31, 0.21, 0.11, 0.19, 0.29, 0.39,
	}}
	mask := Image{Width: 2, Height: 2, Scale: 1, Pixels: []float32{
		1e-37, 0, 0, 0, 2e-37, 0,
		0, 0, 3e-37, 0, 0, 4e-37,
	}}
	cfg := Config{Top: 5, Bottom: -3, Shore: 1.25, MoonRadius: 0.28, WaterMask: &mask}
	got, err := Apply(raw, sea, cfg)
	if err != nil {
		t.Fatal(err)
	}
	want, err := matte.Apply(raw, sea, cfg)
	if err != nil {
		t.Fatal(err)
	}
	assertImagesClose(t, got, want)
	if diff := maxFloat32ULPDiff(got.Pixels, want.Pixels); diff > 1 {
		t.Fatalf("expected GPU output to stay within one ULP of CPU output, got %d", diff)
	}
}

func TestApplyMatchesCPUWithFrontParity(t *testing.T) {
	raw := Image{Width: 3, Height: 2, Scale: -1, Pixels: []float32{
		0.12, 0.22, 0.32, 0.42, 0.52, 0.62, 0.72, 0.82, 0.92,
		0.18, 0.28, 0.38, 0.48, 0.58, 0.68, 0.78, 0.88, 0.98,
	}}
	sea := Image{Width: 3, Height: 2, Scale: 1, Pixels: []float32{
		0.91, 0.81, 0.71, 0.61, 0.51, 0.41, 0.31, 0.21, 0.11,
		0.19, 0.29, 0.39, 0.49, 0.59, 0.69, 0.79, 0.89, 0.99,
	}}
	moon := Image{Width: 1, Height: 2, Scale: -1, Pixels: []float32{
		1.5, 2.0, 2.5,
		0.2, 0.4, 0.8,
	}}
	front := Image{Width: 3, Height: 2, Scale: 1, Pixels: []float32{
		0.01, 0.02, 0.03, 0.04, 0.05, 0.06, 0.07, 0.08, 0.09,
		0.10, 0.11, 0.12, 0.13, 0.14, 0.15, 0.16, 0.17, 0.18,
	}}
	rawBefore := append([]float32(nil), raw.Pixels...)
	seaBefore := append([]float32(nil), sea.Pixels...)
	moonBefore := append([]float32(nil), moon.Pixels...)
	frontBefore := append([]float32(nil), front.Pixels...)
	cfg := Config{Top: 5, Bottom: -3, Shore: 1.25, MoonRadius: 0.28, MoonRef: &moon, Front: &front}
	got, err := Apply(raw, sea, cfg)
	if err != nil {
		t.Fatal(err)
	}
	want, err := matte.Apply(raw, sea, cfg)
	if err != nil {
		t.Fatal(err)
	}
	assertImagesClose(t, got, want)
	assertFloat32sEqual(t, raw.Pixels, rawBefore, "raw input mutated")
	assertFloat32sEqual(t, sea.Pixels, seaBefore, "sea input mutated")
	assertFloat32sEqual(t, moon.Pixels, moonBefore, "moon reference input mutated")
	assertFloat32sEqual(t, front.Pixels, frontBefore, "front input mutated")
}

func TestApplyIncludesExactHalfBrightnessMoonPixel(t *testing.T) {
	raw := Image{Width: 1, Height: 1, Scale: -1, Pixels: []float32{0, 0, 0}}
	sea := Image{Width: 1, Height: 1, Scale: 1, Pixels: []float32{1, 2, 3}}
	moon := Image{Width: 3, Height: 1, Scale: -1, Pixels: []float32{
		4, 0, 0,
		1, 1, 0,
		0.5, 0.5, 0.5,
	}}
	cfg := Config{Top: 2, Bottom: -2, Shore: 0, MoonRadius: 1, MoonRef: &moon}
	manual := func(includeEqual bool) Image {
		most := 0.0
		for i := 0; i < len(moon.Pixels); i += 3 {
			brightness := float64(moon.Pixels[i]) + float64(moon.Pixels[i+1]) + float64(moon.Pixels[i+2])
			if brightness > most {
				most = brightness
			}
		}
		var sum [3]float64
		count := 0.0
		for i := 0; i < len(moon.Pixels); i += 3 {
			brightness := float64(moon.Pixels[i]) + float64(moon.Pixels[i+1]) + float64(moon.Pixels[i+2])
			if brightness < most/2 || (!includeEqual && brightness == most/2) {
				continue
			}
			sum[0] += float64(moon.Pixels[i])
			sum[1] += float64(moon.Pixels[i+1])
			sum[2] += float64(moon.Pixels[i+2])
			count++
		}
		if count == 0 {
			t.Fatal("manual expectation selected no moon pixels")
		}
		for i := range sum {
			sum[i] /= count
		}
		lum := (sum[0] + sum[1] + sum[2]) / 3
		disc := math.Pi * math.Pow(cfg.MoonRadius*math.Pi/180, 2)
		shoreRow := (cfg.Top - cfg.Shore) / (cfg.Top - cfg.Bottom) * float64(raw.Height)
		w := math.Max(0, math.Min(1, (0-shoreRow+4)/(2*4)))
		pixels := make([]float32, 3)
		for c := range pixels {
			pixels[c] = float32((1-w)*float64(raw.Pixels[c]) + w*float64(sea.Pixels[c])*lum*disc)
		}
		return Image{Width: 1, Height: 1, Scale: raw.Scale, Pixels: pixels}
	}
	expected := manual(true)
	antiExpected := manual(false)
	if diff := maxFloat32ULPDiff(expected.Pixels, antiExpected.Pixels); diff == 0 {
		t.Fatal("excluding the equality pixel did not change the manual expectation")
	}
	cpuWant, err := matte.Apply(raw, sea, cfg)
	if err != nil {
		t.Fatal(err)
	}
	assertImagesWithinULP(t, cpuWant, expected, 1)
	if diff := maxFloat32ULPDiff(cpuWant.Pixels, antiExpected.Pixels); diff == 0 {
		t.Fatal("CPU matte output unexpectedly matched the anti expectation")
	}
	got, err := Apply(raw, sea, cfg)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatal(err)
	}
	assertImagesWithinULP(t, got, expected, 1)
}

func TestApplyMatchesCPUWithMultiLevelMoonReduction(t *testing.T) {
	raw := Image{Width: 1, Height: 1, Scale: -1, Pixels: []float32{0.1, 0.2, 0.3}}
	sea := Image{Width: 1, Height: 1, Scale: 1, Pixels: []float32{0.4, 0.5, 0.6}}
	const moonPixels = 65537
	moon := Image{Width: moonPixels, Height: 1, Scale: -1, Pixels: make([]float32, moonPixels*3)}
	above := [3]float32{3.25, 1.25, 0.625}
	below := [3]float32{2.5, 1.5, 0.75}
	peak := [3]float32{6, 3, 1}
	aboveCount := 0
	belowCount := 0
	for i := 0; i < moonPixels-1; i++ {
		j := i * 3
		if i%2 == 0 {
			moon.Pixels[j] = above[0]
			moon.Pixels[j+1] = above[1]
			moon.Pixels[j+2] = above[2]
			aboveCount++
		} else {
			moon.Pixels[j] = below[0]
			moon.Pixels[j+1] = below[1]
			moon.Pixels[j+2] = below[2]
			belowCount++
		}
	}
	last := (moonPixels - 1) * 3
	moon.Pixels[last] = peak[0]
	moon.Pixels[last+1] = peak[1]
	moon.Pixels[last+2] = peak[2]
	cfg := Config{Top: 2, Bottom: -2, Shore: 0, MoonRadius: 20, MoonRef: &moon}
	got, err := Apply(raw, sea, cfg)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatal(err)
	}
	expected := func(moon Image) (Image, int) {
		most := float64(0)
		for i := 0; i < len(moon.Pixels); i += 3 {
			brightness := float64(moon.Pixels[i] + moon.Pixels[i+1] + moon.Pixels[i+2])
			if brightness > most {
				most = brightness
			}
		}
		var sum [3]float64
		count := 0
		for i := 0; i < len(moon.Pixels); i += 3 {
			brightness := float64(moon.Pixels[i] + moon.Pixels[i+1] + moon.Pixels[i+2])
			if brightness < most/2 {
				continue
			}
			sum[0] += float64(moon.Pixels[i])
			sum[1] += float64(moon.Pixels[i+1])
			sum[2] += float64(moon.Pixels[i+2])
			count++
		}
		if count == 0 {
			t.Fatal("manual expectation selected no moon pixels")
		}
		for c := range sum {
			sum[c] /= float64(count)
		}
		const feather = 4.0
		shoreRow := (cfg.Top - cfg.Shore) / (cfg.Top - cfg.Bottom) * float64(raw.Height)
		w := math.Max(0, math.Min(1, (0-shoreRow+feather)/(2*feather)))
		lum := (sum[0] + sum[1] + sum[2]) / 3
		disc := math.Pi * math.Pow(cfg.MoonRadius*math.Pi/180, 2)
		return Image{Width: 1, Height: 1, Scale: raw.Scale, Pixels: []float32{
			float32((1-w)*float64(raw.Pixels[0]) + w*float64(sea.Pixels[0])*lum*disc),
			float32((1-w)*float64(raw.Pixels[1]) + w*float64(sea.Pixels[1])*lum*disc),
			float32((1-w)*float64(raw.Pixels[2]) + w*float64(sea.Pixels[2])*lum*disc),
		}}, count
	}
	want, selected := expected(moon)
	if selected != aboveCount+1 {
		t.Fatalf("manual expectation selected %d moon pixels, want %d", selected, aboveCount+1)
	}
	cpuWant, err := matte.Apply(raw, sea, cfg)
	if err != nil {
		t.Fatal(err)
	}
	assertImagesWithinULP(t, cpuWant, want, 1)
	assertImagesWithinULP(t, got, want, 1)
	assertImageFinite(t, got)
	assertImageFinite(t, want)
	assertImageHasNonZeroPixel(t, got)

	perturbedMoon := moon
	perturbedMoon.Pixels = append([]float32(nil), moon.Pixels...)
	perturbedMoon.Pixels[last] = below[0]
	perturbedMoon.Pixels[last+1] = below[1]
	perturbedMoon.Pixels[last+2] = below[2]
	perturbedGot, err := Apply(raw, sea, Config{Top: cfg.Top, Bottom: cfg.Bottom, Shore: cfg.Shore, MoonRadius: cfg.MoonRadius, MoonRef: &perturbedMoon})
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatal(err)
	}
	perturbedWant, perturbedSelected := expected(perturbedMoon)
	if perturbedSelected != moonPixels {
		t.Fatalf("manual expectation selected %d moon pixels after perturbation, want %d", perturbedSelected, moonPixels)
	}
	perturbedCPU, err := matte.Apply(raw, sea, Config{Top: cfg.Top, Bottom: cfg.Bottom, Shore: cfg.Shore, MoonRadius: cfg.MoonRadius, MoonRef: &perturbedMoon})
	if err != nil {
		t.Fatal(err)
	}
	assertImagesWithinULP(t, perturbedCPU, perturbedWant, 1)
	assertImagesWithinULP(t, perturbedGot, perturbedWant, 1)
	assertImageFinite(t, perturbedGot)
	assertImageFinite(t, perturbedWant)
	if diff := maxFloat32ULPDiff(got.Pixels, perturbedGot.Pixels); diff < 1024 {
		t.Fatalf("expected perturbing the last pixel to change the output materially, got max ULP diff %d", diff)
	}
}

func TestApplyWithLargeScalarParameters(t *testing.T) {
	raw := Image{Width: 2, Height: 2, Scale: -1, Pixels: []float32{
		0.1, 0.2, 0.3, 0.4, 0.5, 0.6,
		0.7, 0.8, 0.9, 1.0, 1.1, 1.2,
	}}
	sea := Image{Width: 2, Height: 2, Scale: 1, Pixels: []float32{
		0.9, 0.8, 0.7, 0.6, 0.5, 0.4,
		0.3, 0.2, 0.1, 0.4, 0.5, 0.6,
	}}
	moon := Image{Width: 2, Height: 2, Scale: -1, Pixels: []float32{
		0.8, 0.6, 0.4, 0.3, 0.5, 0.7,
		0.2, 0.4, 0.6, 0.9, 0.8, 0.7,
	}}
	cfg := Config{Top: 1e20, Bottom: 0, Shore: 5e19, MoonRadius: 1e10, MoonRef: &moon}
	got, err := Apply(raw, sea, cfg)
	if err != nil {
		t.Fatal(err)
	}
	want, err := matte.Apply(raw, sea, cfg)
	if err != nil {
		t.Fatal(err)
	}
	assertImagesClose(t, got, want)
	assertImageFinite(t, got)
}

func TestApplyRejectsUnsafeGeometryAndMasks(t *testing.T) {
	good := Image{Width: 2, Height: 2, Scale: -1, Pixels: []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 1.0, 1.1, 1.2, 1.3, 1.4, 1.5}}
	badMask := Image{Width: 2, Height: 2, Scale: 1, Pixels: []float32{0.1, -0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.2}}
	moon := good
	for _, tc := range []struct {
		name string
		cfg  Config
	}{
		{"top below bottom", Config{Top: 0, Bottom: 1, Shore: 0.5, MoonRadius: 1}},
		{"shore outside range", Config{Top: 2, Bottom: 0, Shore: 2, MoonRadius: 1}},
		{"moon radius overflow", Config{Top: 2, Bottom: 0, Shore: 1, MoonRadius: math.MaxFloat64}},
		{"moon radius underflow", Config{Top: 2, Bottom: 0, Shore: 1, MoonRadius: 1e-30}},
		{"mask negative", Config{Top: 2, Bottom: 0, Shore: 1, MoonRadius: 1, WaterMask: &badMask}},
		{"front mismatch", Config{Top: 2, Bottom: 0, Shore: 1, MoonRadius: 1, Front: &Image{Width: 1, Height: 1, Pixels: make([]float32, 3)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Apply(good, good, tc.cfg); err == nil {
				t.Fatal("accepted unsafe geometry or mask")
			}
		})
	}
	if _, err := Apply(good, good, Config{Top: 2, Bottom: 0, Shore: 1, MoonRadius: 1, MoonRef: &moon}); err != nil {
		t.Fatal(err)
	}
}

func TestApplyRejectsReductionOverflow(t *testing.T) {
	raw := Image{Width: 2, Height: 2, Scale: -1, Pixels: []float32{
		0.1, 0.2, 0.3, 0.4, 0.5, 0.6,
		0.7, 0.8, 0.9, 1.0, 1.1, 1.2,
	}}
	sea := Image{Width: 2, Height: 2, Scale: 1, Pixels: []float32{
		0.9, 0.8, 0.7, 0.6, 0.5, 0.4,
		0.3, 0.2, 0.1, 0.4, 0.5, 0.6,
	}}
	moon := Image{Width: 2, Height: 2, Scale: -1, Pixels: []float32{
		1.5e38, 1.5e38, 1.5e38, 1.5e38, 1.5e38, 1.5e38,
		1.5e38, 1.5e38, 1.5e38, 1.5e38, 1.5e38, 1.5e38,
	}}
	_, err := Apply(raw, sea, Config{Top: 4, Bottom: 0, Shore: 2, MoonRadius: 1, MoonRef: &moon})
	if err == nil {
		t.Fatal("accepted a moon reference whose reduction overflowed")
	}
	if !strings.Contains(err.Error(), "reduction produced nonfinite value") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertImagesWithinULP(t *testing.T, got, want Image, maxDiff uint32) {
	t.Helper()
	if got.Width != want.Width || got.Height != want.Height || got.Scale != want.Scale {
		t.Fatalf("metadata mismatch: got %dx%d scale %v want %dx%d scale %v", got.Width, got.Height, got.Scale, want.Width, want.Height, want.Scale)
	}
	if diff := maxFloat32ULPDiff(got.Pixels, want.Pixels); diff > maxDiff {
		t.Fatalf("pixel difference exceeded %d ULPs: got %d", maxDiff, diff)
	}
}

func assertImagesClose(t *testing.T, got, want Image) {
	t.Helper()
	if got.Width != want.Width || got.Height != want.Height || got.Scale != want.Scale {
		t.Fatalf("metadata mismatch: got %dx%d scale %v want %dx%d scale %v", got.Width, got.Height, got.Scale, want.Width, want.Height, want.Scale)
	}
	if len(got.Pixels) != len(want.Pixels) {
		t.Fatalf("pixel length mismatch: got %d want %d", len(got.Pixels), len(want.Pixels))
	}
	for i := range got.Pixels {
		gv := float64(got.Pixels[i])
		wv := float64(want.Pixels[i])
		if math.IsNaN(gv) || math.IsInf(gv, 0) {
			t.Fatalf("pixel %d: gpu produced nonfinite value %v", i, got.Pixels[i])
		}
		if math.IsNaN(wv) || math.IsInf(wv, 0) {
			t.Fatalf("pixel %d: cpu produced nonfinite value %v", i, want.Pixels[i])
		}
		delta := math.Abs(gv - wv)
		scale := math.Max(1, math.Max(math.Abs(gv), math.Abs(wv)))
		tol := 1e-4 + 1e-5*scale
		if delta > tol {
			t.Fatalf("pixel %d: gpu %v cpu %v delta %v tol %v", i, got.Pixels[i], want.Pixels[i], delta, tol)
		}
	}
}

func assertFloat32sExact(t *testing.T, got, want []float32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("pixel length mismatch: got %d want %d", len(got), len(want))
	}
	for i := range got {
		if math.Float32bits(got[i]) != math.Float32bits(want[i]) {
			t.Fatalf("pixel %d: got %v (%08x) want %v (%08x)", i, got[i], math.Float32bits(got[i]), want[i], math.Float32bits(want[i]))
		}
	}
}

func assertFloat32sEqual(t *testing.T, got, want []float32, message string) {
	t.Helper()
	if !sameFloat32s(got, want) {
		t.Fatalf("%s", message)
	}
}

func assertImageFinite(t *testing.T, image Image) {
	t.Helper()
	for i, value := range image.Pixels {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			t.Fatalf("pixel %d is nonfinite: %v", i, value)
		}
	}
}

func assertImageHasNonZeroPixel(t *testing.T, image Image) {
	t.Helper()
	for _, value := range image.Pixels {
		if value != 0 {
			return
		}
	}
	t.Fatal("image is entirely zero")
}

func sameFloat32s(got, want []float32) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func maxFloat32ULPDiff(got, want []float32) uint32 {
	if len(got) != len(want) {
		return ^uint32(0)
	}
	var max uint32
	for i := range got {
		ga := orderedFloat32Bits(got[i])
		wa := orderedFloat32Bits(want[i])
		diff := ga - wa
		if wa > ga {
			diff = wa - ga
		}
		if diff > max {
			max = diff
		}
	}
	return max
}

func orderedFloat32Bits(value float32) uint32 {
	bits := math.Float32bits(value)
	if bits&0x80000000 != 0 {
		return ^bits
	}
	return bits | 0x80000000
}
