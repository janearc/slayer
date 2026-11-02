package divergence_test

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/burning-trousers/divergence"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
)

// A light is no way apart from itself.
func TestALightIsNotApartFromItself(t *testing.T) {
	for _, light := range []maths.Light{maths.Grey(0), maths.Grey(0.18),
		maths.RGB(0.9, 0.2, 0.05), maths.Grey(40)} {
		if apart := divergence.Apart(light, light); apart != 0 {
			t.Errorf("%v is %g JND from itself", light, apart)
		}
	}
}

// OKLab's lightness of a grey is the cube root of its light, and a grey
// has no colour, so two greys are apart by the cube roots' difference
// over 0.02.
func TestGreysAreApartByTheCubeRoots(t *testing.T) {
	for _, pair := range [][2]float64{{0.18, 0.2}, {0.01, 0.012},
		{1, 4}} {
		want := math.Abs(math.Cbrt(pair[0])-math.Cbrt(pair[1])) / 0.02
		got := float64(divergence.Apart(maths.Grey(pair[0]),
			maths.Grey(pair[1])))
		if math.Abs(got/want-1) > 1e-4 {
			t.Errorf("greys %g and %g are %.5f JND; cube roots say %.5f",
				pair[0], pair[1], got, want)
		}
	}
}

// A light that is not a number is infinitely far from any other.
func TestNaNIsInfinitelyFar(t *testing.T) {
	nan := maths.RGB(math.NaN(), 0, 0)
	if apart := divergence.Apart(nan, maths.Grey(1)); !math.IsInf(
		float64(apart), 1) {
		t.Errorf("a NaN is %g JND from white", apart)
	}
}

// greyFrame is a frame of one grey, with some of its pixels another.
func greyFrame(wide, high int, grey float32, others int) *frame.Frame {
	picture := frame.New(wide, high)
	for i := range picture.Pix {
		picture.Pix[i].R, picture.Pix[i].G, picture.Pix[i].B = grey, grey,
			grey
	}
	for i := range others {
		picture.Pix[i].R = 1
	}
	return picture
}

// One pixel in a hundred moved is a hundredth of the picture, within
// what is allowed; three in a hundred is not.
func TestTheShareMovedIsCounted(t *testing.T) {
	still := greyFrame(10, 10, 0.2, 0)
	for _, moved := range []struct {
		pixels int
		share  divergence.Share
		within bool
	}{{0, 0, true}, {1, 0.01, true}, {2, 0.02, true}, {3, 0.03, false}} {
		seen, err := divergence.Compare(still,
			greyFrame(10, 10, 0.2, moved.pixels))
		if err != nil {
			t.Fatal(err)
		}
		if seen.Past(1) != moved.share || seen.Within() != moved.within {
			t.Errorf("%d moved: %g past one JND, within %v", moved.pixels,
				seen.Past(1), seen.Within())
		}
	}
}

// The worst pixel is the one farthest apart.
func TestTheWorstIsTheFarthest(t *testing.T) {
	seen, err := divergence.Compare(greyFrame(4, 4, 0.2, 0),
		greyFrame(4, 4, 0.2, 1))
	if err != nil {
		t.Fatal(err)
	}
	// the frame holds its light as float32, so the grey is that one
	grey := float64(float32(0.2))
	want := divergence.Apart(maths.Grey(grey), maths.RGB(1, grey, grey))
	if seen.Worst() != want {
		t.Errorf("the worst is %g JND; the moved pixel is %g", seen.Worst(),
			want)
	}
}

// Pictures of different sizes are refused.
func TestPicturesOfTwoSizesAreRefused(t *testing.T) {
	_, err := divergence.Compare(frame.New(4, 4), frame.New(4, 5))
	if !errors.Is(err, divergence.ErrSizes) {
		t.Errorf("4x4 against 4x5 gave %v", err)
	}
}
