// Package regionjudge compares the linear light of a region and a reference.
package regionjudge

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
)

// Channels is one value for each linear RGB channel.
type Channels struct {
	R, G, B float64
}

// Report holds per-channel mean absolute error, candidate-minus-reference
// mean difference, and the count of isolated positive luminance residuals.
type Report struct {
	Absolute     Channels
	Signed       Channels
	MeanAbsolute float64
	Isolated     int
}

// Compare compares candidate with reference using threshold for the isolated
// positive residual count. See the package README for the count and refusals.
func Compare(candidate, reference *frame.Frame, threshold float64) (Report, error) {
	if !finite(threshold) || threshold <= 0 {
		return Report{}, fmt.Errorf("regionjudge: threshold must be finite and positive")
	}
	if !validFrame(candidate) || !validFrame(reference) {
		return Report{}, fmt.Errorf("regionjudge: frames must be nonempty and whole")
	}
	if candidate.W != reference.W || candidate.H != reference.H {
		return Report{}, fmt.Errorf("regionjudge: frame dimensions differ")
	}
	var result Report
	for i, pixel := range candidate.Pix {
		other := reference.Pix[i]
		if !finiteLight(pixel) || !finiteLight(other) {
			return Report{}, fmt.Errorf("regionjudge: nonfinite light at pixel %d", i)
		}
		dr := float64(pixel.R) - float64(other.R)
		dg := float64(pixel.G) - float64(other.G)
		db := float64(pixel.B) - float64(other.B)
		result.Absolute.R += math.Abs(dr)
		result.Absolute.G += math.Abs(dg)
		result.Absolute.B += math.Abs(db)
		result.Signed.R += dr
		result.Signed.G += dg
		result.Signed.B += db
	}
	n := float64(len(candidate.Pix))
	result.Absolute.R /= n
	result.Absolute.G /= n
	result.Absolute.B /= n
	result.MeanAbsolute = (result.Absolute.R + result.Absolute.G + result.Absolute.B) / 3
	result.Signed.R /= n
	result.Signed.G /= n
	result.Signed.B /= n
	for y := 1; y < candidate.H-1; y++ {
		for x := 1; x < candidate.W-1; x++ {
			if isolated(candidate, reference, x, y, threshold) {
				result.Isolated++
			}
		}
	}
	return result, nil
}

// validFrame checks the dimensions and pixel storage of a frame.
func validFrame(picture *frame.Frame) bool {
	return picture != nil && picture.W > 0 && picture.H > 0 &&
		picture.W <= math.MaxInt/picture.H && len(picture.Pix) == picture.W*picture.H
}

// finite reports whether value is neither infinite nor NaN.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// finiteLight checks all channels of one pixel.
func finiteLight(pixel frame.Pixel) bool {
	return finite(float64(pixel.R)) && finite(float64(pixel.G)) && finite(float64(pixel.B))
}

// residual is the candidate-minus-reference luminance at a pixel.
func residual(candidate, reference *frame.Frame, x, y int) float64 {
	a, b := candidate.At(x, y), reference.At(x, y)
	return maths.RGB(float64(a.R)-float64(b.R), float64(a.G)-float64(b.G),
		float64(a.B)-float64(b.B)).Luminance()
}

// isolated detects a bright residual with quiet immediate neighbours.
func isolated(candidate, reference *frame.Frame, x, y int, threshold float64) bool {
	if residual(candidate, reference, x, y) <= threshold {
		return false
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			if residual(candidate, reference, x+dx, y+dy) > threshold {
				return false
			}
		}
	}
	return true
}
