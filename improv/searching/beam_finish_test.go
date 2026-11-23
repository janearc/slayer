package searching

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

// TestMergeBeamLightAddsRadianceOnce keeps geometry metadata out of the
// radiance sum and checks that no input or output shares pixel ownership.
func TestMergeBeamLightAddsRadianceOnce(t *testing.T) {
	base := &frame.Frame{W: 1, H: 1, Pix: []frame.Pixel{
		{R: 0.1, G: 0.2, B: 0.3, Cover: 0.25, Depth: 4}}}
	beam := &frame.Frame{W: 1, H: 1, Pix: []frame.Pixel{
		{R: 0.4, G: 0.5, B: 0.6, Cover: 1, Depth: 2}}}
	baseBefore, beamBefore := base.Pix[0], beam.Pix[0]
	got, err := mergeBeamLight(base, beam)
	if err != nil || got == nil {
		t.Fatalf("merge did not return a picture: %v", err)
	}
	pixel := got.Pix[0]
	for _, pair := range [][2]float32{{pixel.R, 0.5},
		{pixel.G, 0.7}, {pixel.B, 0.9}} {
		if math.Abs(float64(pair[0]-pair[1])) > 1e-6 {
			t.Fatalf("linear light %g, want %g", pair[0], pair[1])
		}
	}
	if got.W != 1 || got.H != 1 || pixel.Cover != 0.25 || pixel.Depth != 4 {
		t.Fatal("beam replaced base geometry", pixel)
	}
	got.Pix[0] = frame.Pixel{}
	if base.Pix[0] != baseBefore || beam.Pix[0] != beamBefore {
		t.Fatal("merge or output mutation changed an input")
	}
}

// TestMergeBeamLightRefusesMismatchedStorage prevents a stale beam frame
// from being silently paired with a different camera resolution.
func TestMergeBeamLightRefusesMismatchedStorage(t *testing.T) {
	valid := frame.New(2, 1)
	for _, bad := range []*frame.Frame{nil, frame.New(0, 0),
		frame.New(1, 2), {W: 2, H: 1},
		{W: -1, H: -1, Pix: []frame.Pixel{{}}},
		{W: int(^uint(0) >> 1), H: 2, Pix: []frame.Pixel{{}}}} {
		for _, pair := range [][2]*frame.Frame{{valid, bad}, {bad, valid}} {
			got, err := mergeBeamLight(pair[0], pair[1])
			if err == nil || got != nil {
				t.Fatalf("malformed pair returned %v, %v", got, err)
			}
		}
	}
}

// TestMergeBeamLightRefusesNonfiniteRadiance checks both a bad input
// and overflow of two finite channels, without publishing partial output.
func TestMergeBeamLightRefusesNonfiniteRadiance(t *testing.T) {
	for _, value := range []float32{float32(math.NaN()),
		float32(math.Inf(1)), float32(math.Inf(-1)), math.MaxFloat32} {
		base := frame.New(2, 1)
		beam := frame.New(2, 1)
		base.Pix[0].R, beam.Pix[0].R = 0.2, 0.3
		base.Pix[1].G, beam.Pix[1].G = value, value
		got, err := mergeBeamLight(base, beam)
		if err == nil || got != nil || base.Pix[0].R != 0.2 || beam.Pix[0].R != 0.3 {
			t.Fatal("nonfinite merge published or modified input", got, err)
		}
	}
}

// TestMergeBeamLightPreservesSignedColour allows linear spectral RGB
// coordinates outside the display gamut; display mapping comes later.
func TestMergeBeamLightPreservesSignedColour(t *testing.T) {
	base, beam := frame.New(1, 1), frame.New(1, 1)
	base.Pix[0].R, beam.Pix[0].R = -0.5, 0.25
	got, err := mergeBeamLight(base, beam)
	if err != nil || got == nil {
		t.Fatalf("signed colour refused: %v", err)
	}
	unbounded := float32(math.Inf(1))
	if got.Pix[0].R != -0.25 || !reflect.DeepEqual(base.Pix, []frame.Pixel{
		{R: -0.5, Depth: unbounded}}) {
		t.Fatal("linear signed colour was clipped or input changed")
	}
}
