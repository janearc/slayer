package finish

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/frame"
)

// Cut is what fitting a plate to a film's shape cost: how much it was
// scaled, and how much of the plate, in the plate's own pixels, was cut
// from each of its sides. Nothing cut is a plate of the film's shape.
type Cut struct {
	scale         float64
	across, down  float64
	width, height int
	filmW, filmH  int
}

// Scale is how many pixels of film a pixel of plate became.
func (made Cut) Scale() float64 { return made.scale }

// Across is how many pixels of plate were cut from the left, and as
// many from the right.
func (made Cut) Across() float64 { return made.across }

// Down is how many pixels of plate were cut from the top, and as many
// from the bottom.
func (made Cut) Down() float64 { return made.down }

// Cuts is whether any of the plate was cut.
func (made Cut) Cuts() bool { return made.across > none || made.down > none }

// String says what was cut, in words for a log: the sizes, the scale and
// the sides.
func (made Cut) String() string {
	if !made.Cuts() {
		return fmt.Sprintf("%dx%d plate scaled %.4gx to %dx%d, nothing "+
			"cut", made.width, made.height, made.scale, made.filmW,
			made.filmH)
	}
	return fmt.Sprintf("%dx%d plate scaled %.4gx to cover %dx%d; cut "+
		"%.1f pixels from the left and right and %.1f from the top and "+
		"bottom", made.width, made.height, made.scale, made.filmW,
		made.filmH, made.across, made.down)
}

// Fit is how a job's plates are made the film's size: nothing, when the
// job names no size or the plates are already it, or a scale to cover
// the size and a cut of the middle, when it names one the plates are
// not. A job that names half a size is refused.
type Fit struct {
	cut           Cut
	width, height int
}

// NewFit is the fit of plates to the size the job's film says.
func NewFit(made job.Job, plates Plates) (Fit, error) {
	wide, saidWide := made.Film().Width()
	high, saidHigh := made.Film().Height()
	across, down := plates.Size()
	if saidWide != saidHigh {
		return Fit{}, errors.New("finish: the film says a width or a " +
			"height but not both; say both, or neither to keep the " +
			"frames' size")
	}
	if !saidWide {
		wide, high = across, down
	}
	return Fit{cut: covering(across, down, wide, high), width: wide,
		height: high}, nil
}

// Size is the width and height of a frame after the fit.
func (made Fit) Size() (width, height int) { return made.width, made.height }

// Cut is what the fit cuts of every plate. The same is cut of each.
func (made Fit) Cut() Cut { return made.cut }

// covering is the cut that scales a plate to cover a film's size, one
// scale on both sides so nothing is stretched, and keeps its middle.
func covering(width, height, filmW, filmH int) Cut {
	scale := math.Max(float64(filmW)/float64(width),
		float64(filmH)/float64(height))
	keptW, keptH := float64(filmW)/scale, float64(filmH)/scale
	return Cut{scale: scale, across: (float64(width) - keptW) / pair,
		down: (float64(height) - keptH) / pair, width: width,
		height: height, filmW: filmW, filmH: filmH}
}

// Apply is the plate at the film's size. A plate that is the film's size
// is returned as it is; others are resampled in linear light, by the
// mean of the plate under each film pixel when the plate is the larger,
// and by bilinear weights when it is the smaller. A plate that is not
// the size the fit was made for is refused.
func (made Fit) Apply(plate *frame.Frame) (*frame.Frame, error) {
	if plate.W != made.cut.width || plate.H != made.cut.height {
		return nil, fmt.Errorf("finish: a %dx%d frame came to a fit made "+
			"for %dx%d; fit the frames the plates were opened with",
			plate.W, plate.H, made.cut.width, made.cut.height)
	}
	if made.cut.width == made.width && made.cut.height == made.height {
		return plate, nil
	}
	fitted := frame.New(made.width, made.height)
	footprint := whole / made.cut.scale
	for y := range made.height {
		for x := range made.width {
			left := made.cut.across + float64(x)*footprint
			top := made.cut.down + float64(y)*footprint
			fitted.Pix[y*made.width+x] = sampled(plate, left, top,
				footprint)
		}
	}
	return fitted, nil
}

// sampled is the light of the plate under a square of a size at a
// corner, in plate pixels: the mean over it when it takes in a pixel or
// more, the bilinear light at its middle when it is smaller.
func sampled(plate *frame.Frame, left, top, size float64) frame.Pixel {
	if size < whole {
		return bilinear(plate, left+size*half, top+size*half)
	}
	return area(plate, left, top, size)
}

// area is the mean light of the plate under a square, each pixel
// weighted by how much of it the square covers.
func area(plate *frame.Frame, left, top, size float64) frame.Pixel {
	var sum [3]float64
	var weight float64
	for y := int(math.Floor(top)); float64(y) < top+size; y++ {
		for x := int(math.Floor(left)); float64(x) < left+size; x++ {
			share := overlap(float64(x), left, size) *
				overlap(float64(y), top, size)
			pixel := plate.Pix[clamped(y, plate.H)*plate.W+clamped(x,
				plate.W)]
			sum[0] += share * float64(pixel.R)
			sum[1] += share * float64(pixel.G)
			sum[2] += share * float64(pixel.B)
			weight += share
		}
	}
	return lit(sum[0]/weight, sum[1]/weight, sum[2]/weight)
}

// overlap is how much of the pixel starting at cell, one wide, a span of
// a size from start covers.
func overlap(cell, start, size float64) float64 {
	return math.Max(none, math.Min(cell+whole, start+size)-
		math.Max(cell, start))
}

// bilinear is the light of the plate at a point, weighing the four
// pixel centres round it by nearness.
func bilinear(plate *frame.Frame, at, down float64) frame.Pixel {
	x, y := at-half, down-half
	x0, y0 := math.Floor(x), math.Floor(y)
	shareX, shareY := x-x0, y-y0
	var sum [3]float64
	for _, corner := range [][3]float64{{0, 0, (1 - shareX) * (1 - shareY)},
		{1, 0, shareX * (1 - shareY)}, {0, 1, (1 - shareX) * shareY},
		{1, 1, shareX * shareY}} {
		pixel := plate.Pix[clamped(int(y0+corner[1]),
			plate.H)*plate.W+clamped(int(x0+corner[0]), plate.W)]
		sum[0] += corner[2] * float64(pixel.R)
		sum[1] += corner[2] * float64(pixel.G)
		sum[2] += corner[2] * float64(pixel.B)
	}
	return lit(sum[0], sum[1], sum[2])
}

// clamped is a place held inside a line of a length, so the edge's
// pixel stands in for what lies past it.
func clamped(place, length int) int {
	return max(none, min(length-one, place))
}

// lit is a pixel of the light, covering nothing, met at no distance.
func lit(red, green, blue float64) frame.Pixel {
	return frame.Pixel{R: float32(red), G: float32(green),
		B: float32(blue), Depth: float32(math.Inf(one))}
}
