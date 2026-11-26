package regionjudge

import (
	"math"
	"testing"

	"github.com/janearc/slayer/render/frame"
	"pgregory.net/rapid"
)

// Swapping the pictures preserves absolute error and reverses signed error.
// The isolated positive count is directional and is deliberately not compared.
func TestComparisonIsSymmetricExceptForDirection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(1, 7).Draw(t, "width")
		height := rapid.IntRange(1, 7).Draw(t, "height")
		first, second := frame.New(width, height), frame.New(width, height)
		for i := range first.Pix {
			first.Pix[i] = randomLight(t)
			second.Pix[i] = randomLight(t)
		}
		forward, err := Compare(first, second, 1)
		if err != nil {
			t.Fatal(err)
		}
		backward, err := Compare(second, first, 1)
		if err != nil {
			t.Fatal(err)
		}
		if forward.Absolute != backward.Absolute || forward.MeanAbsolute != backward.MeanAbsolute ||
			forward.Signed.R != -backward.Signed.R ||
			forward.Signed.G != -backward.Signed.G ||
			forward.Signed.B != -backward.Signed.B {
			t.Fatalf("forward %+v, backward %+v", forward, backward)
		}
		for _, number := range []float64{forward.Absolute.R, forward.Absolute.G,
			forward.Absolute.B, forward.MeanAbsolute, forward.Signed.R, forward.Signed.G, forward.Signed.B} {
			if math.IsNaN(number) || math.IsInf(number, 0) {
				t.Fatalf("finite light gave nonfinite result %+v", forward)
			}
		}
	})
}

// randomLight spans signs, zeros, and very small to very large finite light.
func randomLight(t *rapid.T) frame.Pixel {
	part := func() float32 {
		mantissa := rapid.IntRange(-1024, 1024).Draw(t, "mantissa")
		exponent := rapid.IntRange(-120, 110).Draw(t, "exponent")
		return float32(math.Ldexp(float64(mantissa), exponent))
	}
	return frame.Pixel{R: part(), G: part(), B: part()}
}

func TestExtremeFiniteLightKeepsFiniteAverages(t *testing.T) {
	first, second := frame.New(3, 3), frame.New(3, 3)
	for i := range first.Pix {
		first.Pix[i] = frame.Pixel{R: math.MaxFloat32, G: -math.MaxFloat32, B: math.MaxFloat32}
		second.Pix[i] = frame.Pixel{R: -math.MaxFloat32, G: math.MaxFloat32, B: -math.MaxFloat32}
	}
	got, err := Compare(first, second, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := 2 * float64(math.MaxFloat32)
	if got.Absolute != (Channels{want, want, want}) ||
		got.Signed != (Channels{want, -want, want}) || got.Isolated != 0 {
		t.Fatalf("extreme finite values: %+v", got)
	}
}
