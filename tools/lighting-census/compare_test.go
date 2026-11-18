package census

import (
	"math"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

func TestCompareTracksRawHDRAndSpatialWorst(t *testing.T) {
	ref, got := frame.New(2, 2), frame.New(2, 2)
	ref.Pix[0] = frame.Pixel{R: 10, G: 1, B: 1, Cover: 1, Depth: 8}
	got.Pix[0] = frame.Pixel{R: 9, G: 1, B: 1, Cover: 1, Depth: 8.004}
	ref.Pix[3] = frame.Pixel{R: 0.5, G: 0.25, B: 0.25, Cover: 1, Depth: 2}
	got.Pix[3] = frame.Pixel{R: 0.5, G: 0.25, B: 3.25, Cover: 0.5, Depth: 3}
	d, err := Compare(ref, got)
	if err != nil {
		t.Fatal(err)
	}
	if d.Absolute != 4 || d.Relative == nil || math.Abs(*d.Relative-4.0/13) > 1e-12 ||
		d.MaxAbs != 3 || d.MaxX != 1 || d.MaxY != 1 || d.MaxChannel != "blue" ||
		d.ReferencePeak != 10 || d.MaxRelative == nil || math.Abs(*d.MaxRelative-0.3) > 1e-12 ||
		d.CoverDifferent != 1 || d.DepthDifferent != 1 {
		t.Fatalf("wrong HDR/spatial comparison: %+v", d)
	}
}

func TestCompareRefusesCorruptInputs(t *testing.T) {
	good := frame.New(1, 1)
	for _, test := range []struct {
		name string
		bad  *frame.Frame
	}{
		{"nil", nil},
		{"empty", frame.New(0, 0)},
		{"bad storage", &frame.Frame{W: 1, H: 1}},
		{"nan light", &frame.Frame{W: 1, H: 1, Pix: []frame.Pixel{{R: float32(math.NaN())}}}},
		{"infinite coverage", &frame.Frame{W: 1, H: 1, Pix: []frame.Pixel{{Cover: float32(math.Inf(1))}}}},
		{"negative coverage", &frame.Frame{W: 1, H: 1, Pix: []frame.Pixel{{Cover: -.1}}}},
		{"excess coverage", &frame.Frame{W: 1, H: 1, Pix: []frame.Pixel{{Cover: 1.1}}}},
		{"nan depth", &frame.Frame{W: 1, H: 1, Pix: []frame.Pixel{{Depth: float32(math.NaN())}}}},
		{"negative depth", &frame.Frame{W: 1, H: 1, Pix: []frame.Pixel{{Depth: -1}}}},
		{"negative infinite depth", &frame.Frame{W: 1, H: 1, Pix: []frame.Pixel{{Depth: float32(math.Inf(-1))}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Compare(good, test.bad); err == nil {
				t.Fatal("accepted malformed candidate")
			}
			if _, err := Compare(test.bad, good); err == nil {
				t.Fatal("accepted malformed reference")
			}
		})
	}
}

func TestCompareDarkReferenceAndInfiniteDepthAreHonest(t *testing.T) {
	ref, got := frame.New(1, 1), frame.New(1, 1)
	got.Pix[0].R = 2
	d, err := Compare(ref, got)
	if err != nil || d.Relative != nil || d.MaxRelative != nil || d.Absolute != 2 || d.DepthDifferent != 0 {
		t.Fatalf("dark reference: %+v, %v", d, err)
	}
	got.Pix[0].Depth = 3
	d, err = Compare(ref, got)
	if err != nil || d.DepthDifferent != 1 {
		t.Fatalf("hit and miss must differ: %+v, %v", d, err)
	}
}
