package regionjudge

import (
	"math"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

func TestEqualLightHasNoDifference(t *testing.T) {
	picture := frame.New(2, 2)
	picture.Pix[0] = frame.Pixel{R: 4, G: 2, B: 1}
	got, err := Compare(picture, picture, 0.5)
	if err != nil || got != (Report{}) {
		t.Fatalf("same picture: %+v, %v", got, err)
	}
}

func TestRemovingLightIsMeasured(t *testing.T) {
	reference, candidate := frame.New(2, 1), frame.New(2, 1)
	reference.Pix[0] = frame.Pixel{R: 4, G: 2, B: 1}
	got, err := Compare(candidate, reference, 0.5)
	want := Report{Absolute: Channels{2, 1, 0.5}, Signed: Channels{-2, -1, -0.5}, MeanAbsolute: 3.5 / 3}
	if err != nil || got != want {
		t.Fatalf("lost light: %+v, %v; want %+v", got, err, want)
	}
}

func TestIsolatedResidualNeedsEightQuietNeighbours(t *testing.T) {
	reference, candidate := frame.New(5, 5), frame.New(5, 5)
	candidate.Set(2, 2, frame.Pixel{R: 2, G: 2, B: 2})
	got, err := Compare(candidate, reference, 1)
	if err != nil || got.Isolated != 1 {
		t.Fatalf("one isolated residual: %+v, %v", got, err)
	}
	// A neighbouring positive residual makes both pixels part of a cluster.
	candidate.Set(3, 2, frame.Pixel{R: 2, G: 2, B: 2})
	got, err = Compare(candidate, reference, 1)
	if err != nil || got.Isolated != 0 {
		t.Fatalf("cluster: %+v, %v", got, err)
	}
}

func TestResidualThresholdAndFrameEdge(t *testing.T) {
	reference, candidate := frame.New(3, 3), frame.New(3, 3)
	candidate.Set(1, 1, frame.Pixel{R: 1, G: 1, B: 1})
	got, err := Compare(candidate, reference, 1)
	if err != nil || got.Isolated != 0 {
		t.Fatalf("threshold is strict: %+v, %v", got, err)
	}
	candidate.Set(1, 1, frame.Pixel{})
	candidate.Set(0, 0, frame.Pixel{R: 3, G: 3, B: 3})
	got, err = Compare(candidate, reference, 1)
	if err != nil || got.Isolated != 0 {
		t.Fatalf("edge is excluded: %+v, %v", got, err)
	}
}

func TestAnEdgeNeighbourPreventsIsolation(t *testing.T) {
	reference, candidate := frame.New(3, 3), frame.New(3, 3)
	candidate.Set(1, 1, frame.Pixel{R: 2, G: 2, B: 2})
	candidate.Set(0, 0, frame.Pixel{R: 2, G: 2, B: 2})
	got, err := Compare(candidate, reference, 1)
	if err != nil || got.Isolated != 0 {
		t.Fatalf("edge neighbour belongs to the cluster: %+v, %v", got, err)
	}
}

func TestDarkeningAReferenceIsNotAnIsolatedPositive(t *testing.T) {
	reference, candidate := frame.New(3, 3), frame.New(3, 3)
	reference.Set(1, 1, frame.Pixel{R: 8, G: 4, B: 2})
	candidate.Set(1, 1, frame.Pixel{R: 1, G: 1, B: 1})
	got, err := Compare(candidate, reference, 0.5)
	if err != nil || got.Isolated != 0 || got.Absolute.R <= 0 ||
		got.Absolute.G <= 0 || got.Absolute.B <= 0 ||
		got.Signed.R >= 0 || got.Signed.G >= 0 || got.Signed.B >= 0 {
		t.Fatalf("clipped reference: %+v, %v", got, err)
	}
}

func TestRefusesInvalidInputs(t *testing.T) {
	good := frame.New(3, 3)
	nan := frame.New(3, 3)
	nan.Pix[0].G = float32(math.NaN())
	inf := frame.New(3, 3)
	inf.Pix[0].B = float32(math.Inf(1))
	cases := []struct {
		name                 string
		candidate, reference *frame.Frame
		threshold            float64
	}{
		{"nil candidate", nil, good, 1},
		{"nil reference", good, nil, 1},
		{"empty", frame.New(0, 0), frame.New(0, 0), 1},
		{"mismatched dimensions", good, frame.New(2, 3), 1},
		{"short pixels", &frame.Frame{W: 3, H: 3}, good, 1},
		{"extra pixels", &frame.Frame{W: 1, H: 1, Pix: make([]frame.Pixel, 2)}, good, 1},
		{"overflowing dimensions", &frame.Frame{W: math.MaxInt, H: 2}, good, 1},
		{"candidate NaN", nan, good, 1},
		{"reference infinity", good, inf, 1},
		{"zero threshold", good, good, 0},
		{"negative threshold", good, good, -1},
		{"NaN threshold", good, good, math.NaN()},
		{"infinite threshold", good, good, math.Inf(1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Compare(tc.candidate, tc.reference, tc.threshold)
			if err == nil || got != (Report{}) {
				t.Fatalf("invalid input gave %+v, %v", got, err)
			}
		})
	}
}
