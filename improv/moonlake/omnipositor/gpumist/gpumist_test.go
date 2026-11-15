//go:build draw

package gpumist

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/improv/moonlake/mist"
	"github.com/janearc/slayer/metal"
)

type frozenHashCase struct {
	sample   Sample
	raw      uint64
	normBits uint32
}

func openDevice(t *testing.T) *Device {
	t.Helper()
	d, err := New()
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(d.Close)
	return d
}

func TestHashMatchesFrozenManualValuesAndRepeats(t *testing.T) {
	d := openDevice(t)
	const minInt64 = int64(-1 << 63)
	const maxInt64 = int64(1<<63 - 1)
	cases := []frozenHashCase{
		{sample: Sample{X: 0, Y: 0, Seed: 0}, raw: 0x0000000000000000, normBits: 0x00000000},
		{sample: Sample{X: 1, Y: -1, Seed: 0x0123456789abcdef}, raw: 0x5aada6f7e7e63719, normBits: 0x3eb55b4e},
		{sample: Sample{X: -123456789, Y: 987654321, Seed: 0x8000000000000000}, raw: 0xc855a2654b0912e2, normBits: 0x3f4855a2},
		{sample: Sample{X: maxInt64, Y: minInt64, Seed: ^uint64(0)}, raw: 0x2d95b482b8ddc189, normBits: 0x3e3656d2},
	}
	batches := [][]Sample{
		{cases[0].sample, cases[1].sample, cases[2].sample, cases[3].sample},
		{cases[0].sample, cases[1].sample},
		{cases[0].sample, cases[1].sample, cases[2].sample, cases[3].sample},
	}
	for round, batch := range batches {
		got, err := d.Hash(batch)
		if err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		if len(got) != len(batch) {
			t.Fatalf("round %d: result length %d, want %d", round, len(got), len(batch))
		}
		for i, sample := range batch {
			want := cases[i]
			if sample != want.sample {
				t.Fatalf("round %d sample %d changed unexpectedly: %#v", round, i, sample)
			}
			if got[i].Raw != want.raw {
				t.Fatalf("round %d sample %d raw %016x, want %016x", round, i, got[i].Raw, want.raw)
			}
			if bits := math.Float32bits(got[i].Normalized); bits != want.normBits {
				t.Fatalf("round %d sample %d normalized bits %08x, want %08x", round, i, bits, want.normBits)
			}
		}
		if round == 0 {
			t.Log("frozen expectations cover only a few manually checked values and do not prove broader distribution behavior")
		}
	}
}

func TestHashRefusesOversizedBatchAndStaysUsable(t *testing.T) {
	d := openDevice(t)
	base := []Sample{{X: 0, Y: 0, Seed: 0}}
	got, err := d.Hash(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Raw != 0 || math.Float32bits(got[0].Normalized) != 0 {
		t.Fatalf("baseline hash = %#v, want zero result", got)
	}
	tooMany := make([]Sample, maxSamples+1)
	if _, err := d.Hash(tooMany); err == nil || !strings.Contains(err.Error(), "too many samples") {
		t.Fatalf("oversized batch error = %v, want too many samples", err)
	}
	again, err := d.Hash(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 1 || again[0] != got[0] {
		t.Fatalf("hash after refusal = %#v, want %#v", again, got)
	}
}

func TestHashAndCloseAreSynchronized(t *testing.T) {
	d := openDevice(t)
	samples := make([]Sample, maxSamples)
	for i := range samples {
		samples[i] = Sample{X: int64(i), Y: -int64(i), Seed: uint64(i) * 0x9e3779b97f4a7c15}
	}
	start := make(chan struct{})
	hashErr := make(chan error, 1)
	closeDone := make(chan struct{})
	go func() {
		<-start
		_, err := d.Hash(samples)
		hashErr <- err
	}()
	go func() {
		<-start
		d.Close()
		close(closeDone)
	}()
	close(start)
	select {
	case err := <-hashErr:
		if err != nil && !strings.Contains(err.Error(), "device is closed") {
			t.Fatalf("concurrent hash returned %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("concurrent hash timed out")
	}
	select {
	case <-closeDone:
	case <-time.After(10 * time.Second):
		t.Fatal("concurrent close timed out")
	}
	if _, err := d.Hash(samples[:1]); err == nil || !strings.Contains(err.Error(), "device is closed") {
		t.Fatalf("hash after close = %v, want device closed", err)
	}
}

func TestApplyMatchesCPUOnTinySyntheticScene(t *testing.T) {
	openDevice(t)
	image := syntheticMistImage()
	moonRef := syntheticMoonReference()
	water := syntheticWaterMask()
	trees := syntheticTreeMask()
	cases := []struct {
		name   string
		config Config
	}{
		{
			name: "plain sky",
			config: Config{
				Top: 6, Bottom: -6, Shore: 0, Strength: 0.43, Seconds: 0.75, Seed: 23,
				Colour: [3]float64{0.78, 0.88, 1.08}, Layers: 1.5,
				MoonRef: &moonRef,
			},
		},
		{
			name: "water and trees",
			config: Config{
				Top: 6, Bottom: -6, Shore: -0.5, Strength: 0.52, Seconds: 3.25, Seed: 71,
				Colour: [3]float64{0.72, 0.9, 1.05}, Layers: 2.5,
				MoonRef: &moonRef, WaterMask: &water, TreeMask: &trees,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Apply(image, tc.config)
			if err != nil {
				t.Fatal(err)
			}
			want, err := mist.Apply(image, tc.config)
			if err != nil {
				t.Fatal(err)
			}
			assertImagesClose(t, got, want)
			assertImageHasNonZeroPixel(t, got)
			assertImageChanged(t, image.Pixels, got.Pixels)
		})
	}
}

func TestApplyRespondsToSeedAndTime(t *testing.T) {
	openDevice(t)
	image := syntheticMistImage()
	moonRef := syntheticMoonReference()
	base := Config{
		Top: 6, Bottom: -6, Shore: 0, Strength: 0.43, Seconds: 0.75, Seed: 23,
		Colour: [3]float64{0.78, 0.88, 1.08}, Layers: 1.5, MoonRef: &moonRef,
	}
	changed := base
	changed.Seconds = 5.75
	changed.Seed++
	gotBase, err := Apply(image, base)
	if err != nil {
		t.Fatal(err)
	}
	wantBase, err := mist.Apply(image, base)
	if err != nil {
		t.Fatal(err)
	}
	assertImagesClose(t, gotBase, wantBase)
	gotChanged, err := Apply(image, changed)
	if err != nil {
		t.Fatal(err)
	}
	wantChanged, err := mist.Apply(image, changed)
	if err != nil {
		t.Fatal(err)
	}
	assertImagesClose(t, gotChanged, wantChanged)
	if samePixels(gotBase.Pixels, gotChanged.Pixels) {
		t.Fatal("seed and time changes were ignored")
	}
}

func TestApplyRejectsUnsupportedInputs(t *testing.T) {
	openDevice(t)
	image := syntheticMistImage()
	moonRef := syntheticMoonReference()
	bad := Config{Top: 6, Bottom: -6, Shore: 0, Strength: math.NaN(), Seconds: 0, Seed: 1, Colour: [3]float64{0.78, 0.88, 1.08}, Layers: 1, MoonRef: &moonRef}
	if _, err := Apply(image, bad); err == nil {
		t.Fatal("accepted nonfinite strength")
	}
	bad = Config{Top: 6, Bottom: 6, Shore: 0, Strength: 0.1, Seconds: 0, Seed: 1, Colour: [3]float64{0.78, 0.88, 1.08}, Layers: 1, MoonRef: &moonRef}
	if _, err := Apply(image, bad); err == nil {
		t.Fatal("accepted reversed shoreline geometry")
	}
	badMask := syntheticTreeMask()
	badMask.Pixels[0] = -1
	bad = Config{Top: 6, Bottom: -6, Shore: 0, Strength: 0.1, Seconds: 0, Seed: 1, Colour: [3]float64{0.78, 0.88, 1.08}, Layers: 1, MoonRef: &moonRef, TreeMask: &badMask}
	if _, err := Apply(image, bad); err == nil {
		t.Fatal("accepted negative tree mask value")
	}
}

func syntheticMistImage() Image {
	return syntheticColourImage(6, 5, 1.25, func(x, y int) [3]float32 {
		base := 0.08 + 0.02*float32(x) + 0.04*float32(y)
		return [3]float32{base, base * 1.1, base * 1.25}
	})
}

func syntheticMoonReference() Image {
	return syntheticColourImage(6, 5, 1.25, func(x, y int) [3]float32 {
		if y == 0 && x >= 2 && x <= 3 {
			return [3]float32{0.8, 0.78, 0.72}
		}
		if y == 1 && x == 2 {
			return [3]float32{0.45, 0.42, 0.38}
		}
		return [3]float32{0.03 * float32(x+1), 0.025 * float32(y+1), 0.02 * float32(x+y+1)}
	})
}

func syntheticWaterMask() Image {
	return syntheticColourImage(6, 5, 1, func(x, y int) [3]float32 {
		if y >= 3 {
			v := float32(0.12 + 0.08*float32(x%3) + 0.05*float32(y-2))
			return [3]float32{v, v * 1.2, v * 1.4}
		}
		return [3]float32{}
	})
}

func syntheticTreeMask() Image {
	return syntheticColourImage(6, 5, 1, func(x, y int) [3]float32 {
		if y == 2 && (x == 1 || x == 4) {
			return [3]float32{0.9, 0.7, 0.5}
		}
		if y == 1 && x == 3 {
			return [3]float32{0.5, 0.6, 0.7}
		}
		return [3]float32{}
	})
}

func syntheticColourImage(width, height int, scale float64, fn func(x, y int) [3]float32) Image {
	pixels := make([]float32, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			v := fn(x, y)
			i := (y*width + x) * 3
			pixels[i] = v[0]
			pixels[i+1] = v[1]
			pixels[i+2] = v[2]
		}
	}
	return Image{Width: width, Height: height, Scale: scale, Pixels: pixels}
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

func assertImageHasNonZeroPixel(t *testing.T, image Image) {
	t.Helper()
	for _, value := range image.Pixels {
		if value != 0 {
			return
		}
	}
	t.Fatal("image is entirely zero")
}

func assertImageChanged(t *testing.T, before, after []float32) {
	t.Helper()
	if samePixels(before, after) {
		t.Fatal("mist output matched its input exactly")
	}
}

func samePixels(got, want []float32) bool {
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
