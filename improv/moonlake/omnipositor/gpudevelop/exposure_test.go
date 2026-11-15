//go:build draw

package gpudevelop

import (
	"bytes"
	"math"
	"testing"

	"github.com/janearc/slayer/render/colour"
)

func TestExposureChangesReuseStage(t *testing.T) {
	input := []float32{0.25, 0.5, 0.75, 1, 0.125, 0.25}
	stage, err := New(2, 1, colour.Tone{Exposure: 1, Knee: 0.7, Top: 8})
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Close()
	for _, exposure := range []float64{1, 4, 0.5, 1} {
		got, err := stage.DevelopWithExposure(input, exposure)
		if err != nil {
			t.Fatal(err)
		}
		fresh, err := New(2, 1, colour.Tone{Exposure: exposure, Knee: 0.7, Top: 8})
		if err != nil {
			t.Fatal(err)
		}
		want, err := fresh.Develop(input)
		fresh.Close()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got.Pix, want.Pix) {
			t.Fatalf("exposure %v: reused %v fresh %v", exposure, got.Pix, want.Pix)
		}
	}
}

func TestExposureRefusesNonfiniteWithoutClosingStage(t *testing.T) {
	stage, err := New(1, 1, colour.Tone{Exposure: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Close()
	if _, err := stage.DevelopWithExposure([]float32{1, 1, 1}, math.Inf(1)); err == nil {
		t.Fatal("nonfinite exposure accepted")
	}
	if _, err := stage.Develop([]float32{1, 1, 1}); err != nil {
		t.Fatal(err)
	}
}
