// Package divergence is how far apart two pictures of one scene are,
// as an eye would see them: each pixel's distance in just-noticeable
// differences, and the share of the pixels past a limit. It is the
// yardstick for anything that draws a scene another way.
//
// A shortcut, a cache or a kernel, is near enough while no more than
// Allowed of the pixels move past one JND. Each light is linear srgb at
// its own level, with no exposure and no grade.
package divergence

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
)

// JNDs is a distance between colours in just-noticeable differences:
// their distance in OKLab over libtheme's ok.Eye.
type JNDs float64

// Share is a part of a picture's pixels, from zero to one.
type Share float64

// Allowed is the most of a picture another way of drawing it may move
// past one JND.
const Allowed Share = 0.02

// ErrSizes is two pictures of different sizes, which have no pixels to
// compare.
var ErrSizes = errors.New("divergence: the pictures are not one size")

// Apart is how far two lights are, in JNDs. A part of either that is
// not a number puts them infinitely far apart.
func Apart(one, other maths.Light) JNDs {
	distance := ok.Distance(oklab(one), oklab(other)) / ok.Eye
	if math.IsNaN(distance) {
		return JNDs(math.Inf(1))
	}
	return JNDs(distance)
}

// oklab is a linear srgb light in OKLab.
func oklab(light maths.Light) ok.OKLab {
	return ok.FromSwatch(srgb.FromLight(light.R, light.G, light.B))
}

// Divergence is two pictures of one scene compared, pixel by pixel.
type Divergence struct{ apart []JNDs }

// Compare is two pictures of one scene, pixel by pixel, or ErrSizes.
func Compare(one, other *frame.Frame) (Divergence, error) {
	if one.W != other.W || one.H != other.H {
		return Divergence{}, fmt.Errorf("%w: %dx%d and %dx%d", ErrSizes,
			one.W, one.H, other.W, other.H)
	}
	apart := make([]JNDs, len(one.Pix))
	for i := range one.Pix {
		apart[i] = Apart(one.Pix[i].Light(), other.Pix[i].Light())
	}
	return Divergence{apart: apart}, nil
}

// Past is the share of the pixels more than limit apart; zero for pictures with
// no pixels.
func (seen Divergence) Past(limit JNDs) Share {
	if len(seen.apart) == 0 {
		return 0
	}
	past := 0
	for _, apart := range seen.apart {
		if apart > limit {
			past++
		}
	}
	return Share(float64(past) / float64(len(seen.apart)))
}

// Worst is how far apart the farthest pixel is.
func (seen Divergence) Worst() JNDs {
	var worst JNDs
	for _, apart := range seen.apart {
		worst = max(worst, apart)
	}
	return worst
}

// Within is whether the pictures are near enough: no more than Allowed
// of the pixels past one JND.
func (seen Divergence) Within() bool { return seen.Past(1) <= Allowed }
