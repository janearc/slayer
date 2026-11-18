package census

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/render/frame"
)

// Difference describes raw linear radiance and first-hit differences. A nil
// ratio means the reference contains no light, so division is not defined.
// The peak is the largest absolute reference channel, not display white.
type Difference struct {
	Pixels         int      `json:"pixels"`
	Absolute       float64  `json:"absolute"`
	Reference      float64  `json:"reference_energy"`
	Relative       *float64 `json:"relative,omitempty"`
	RMSE           float64  `json:"rmse"`
	ReferencePeak  float64  `json:"reference_peak"`
	MaxAbs         float64  `json:"max_absolute"`
	MaxRelative    *float64 `json:"max_relative,omitempty"`
	MaxX           int      `json:"max_x"`
	MaxY           int      `json:"max_y"`
	MaxChannel     string   `json:"max_channel"`
	CoverDifferent int      `json:"cover_different"`
	DepthDifferent int      `json:"depth_different"`
}

// Compare compares the two frames without clamping or developing HDR light.
// A reference with no energy is reported without a relative error; it is not
// proof of agreement. Nonfinite radiance or coverage and invalid depth are
// errors. Positive infinite depth is the renderer's normal miss sentinel.
func Compare(reference, candidate *frame.Frame) (Difference, error) {
	var d Difference
	if err := validFrame(reference); err != nil {
		return d, fmt.Errorf("reference: %w", err)
	}
	if err := validFrame(candidate); err != nil {
		return d, fmt.Errorf("candidate: %w", err)
	}
	if reference.W != candidate.W || reference.H != candidate.H {
		return d, fmt.Errorf("frame dimensions differ: %dx%d and %dx%d", reference.W, reference.H, candidate.W, candidate.H)
	}
	d.Pixels = len(reference.Pix)
	var squares float64
	names := [3]string{"red", "green", "blue"}
	for i, one := range reference.Pix {
		other := candidate.Pix[i]
		if err := validPixel(one); err != nil {
			return Difference{}, fmt.Errorf("reference pixel %d,%d: %w", i%reference.W, i/reference.W, err)
		}
		if err := validPixel(other); err != nil {
			return Difference{}, fmt.Errorf("candidate pixel %d,%d: %w", i%reference.W, i/reference.W, err)
		}
		a := [3]float32{one.R, one.G, one.B}
		b := [3]float32{other.R, other.G, other.B}
		for channel := range a {
			value := math.Abs(float64(a[channel]))
			gap := math.Abs(float64(a[channel]) - float64(b[channel]))
			d.Reference += value
			d.ReferencePeak = max(d.ReferencePeak, value)
			d.Absolute += gap
			squares += gap * gap
			if gap > d.MaxAbs {
				d.MaxAbs, d.MaxX, d.MaxY, d.MaxChannel = gap, i%reference.W, i/reference.W, names[channel]
			}
		}
		if math.Abs(float64(one.Cover)-float64(other.Cover)) > 1e-3 {
			d.CoverDifferent++
		}
		if !depthEqual(one.Depth, other.Depth) {
			d.DepthDifferent++
		}
	}
	d.RMSE = math.Sqrt(squares / (float64(d.Pixels) * 3))
	if d.Reference > 0 {
		ratio := d.Absolute / d.Reference
		d.Relative = &ratio
	}
	if d.ReferencePeak > 0 {
		ratio := d.MaxAbs / d.ReferencePeak
		d.MaxRelative = &ratio
	}
	return d, nil
}

// validFrame reports whether a frame's size matches its pixel count.
func validFrame(f *frame.Frame) error {
	if f == nil || f.W <= 0 || f.H <= 0 || f.W > int(^uint(0)>>1)/f.H || len(f.Pix) != f.W*f.H {
		return fmt.Errorf("missing or invalid frame")
	}
	return nil
}

// validPixel reports whether a pixel has finite light, coverage in [0,1]
// and a depth that is not negative.
func validPixel(p frame.Pixel) error {
	for _, v := range [...]float32{p.R, p.G, p.B, p.Cover} {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return fmt.Errorf("nonfinite light or coverage")
		}
	}
	if p.Cover < 0 || p.Cover > 1 {
		return fmt.Errorf("coverage outside [0,1]")
	}
	if math.IsNaN(float64(p.Depth)) || p.Depth < 0 {
		return fmt.Errorf("invalid depth")
	}
	return nil
}

// depthEqual reports whether two depths match. Infinite depths match only
// when they are equal.
func depthEqual(a, b float32) bool {
	if math.IsInf(float64(a), 1) || math.IsInf(float64(b), 1) {
		return a == b
	}
	return math.Abs(float64(a)-float64(b)) <= 1e-3*math.Abs(float64(a))
}
