package emptyspace

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// Run describes existing sample positions on one ray in field-local units.
// Changing the sample ordinal must not change First, Stride or its jitter.
type Run struct {
	Ray           maths.Ray
	First, Stride float64
	Count         int
}

// SafePrefix verifies a suggested skip against every packet member's samples.
// Different rays may have different entry distances and fixed strides.
// The result skips existing ordinals; it never resamples a nonempty segment.
func (mask *Mask) SafePrefix(runs []Run, suggested int) (int, error) {
	count, err := mask.prefixCount(runs, suggested)
	if err != nil || count == 0 {
		return 0, err
	}
	empty, err := mask.prefixEmpty(runs, count)
	if err != nil {
		return 0, err
	}
	if empty {
		return count, nil
	}
	low, high := 0, count
	for high-low > 1 {
		middle := low + (high-low)/2
		empty, err := mask.prefixEmpty(runs, middle)
		if err != nil {
			return 0, err
		}
		if empty {
			low = middle
		} else {
			high = middle
		}
	}
	return low, nil
}

// prefixCount validates the batch before using its shortest remaining run.
func (mask *Mask) prefixCount(runs []Run, suggested int) (int, error) {
	if mask == nil || !mask.ready || len(runs) < 1 || len(runs) > mask.options.Rays || suggested < 0 {
		return 0, fmt.Errorf("emptyspace: invalid sample packet or suggested prefix")
	}
	count := suggested
	for _, run := range runs {
		if !finiteVector(run.Ray.At) || !finiteVector(run.Ray.Dir) || run.Ray.Dir == (maths.Vec{}) ||
			!finite(run.First) || run.First < 0 || !finite(run.Stride) || run.Stride <= 0 || run.Count < 0 {
			return 0, fmt.Errorf("emptyspace: invalid sample run")
		}
		count = min(count, run.Count)
	}
	return count, nil
}

// prefixEmpty encloses every sample in each member's proposed ordinal prefix.
func (mask *Mask) prefixEmpty(runs []Run, count int) (bool, error) {
	bounds := maths.Box{Min: maths.V(math.Inf(1), math.Inf(1), math.Inf(1)),
		Max: maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))}
	for _, run := range runs {
		last := run.First + float64(count-1)*run.Stride
		if !finite(last) {
			return false, fmt.Errorf("emptyspace: sample interval overflow")
		}
		for _, point := range [2]maths.Vec{run.Ray.Point(run.First), run.Ray.Point(last)} {
			if !finiteVector(point) {
				return false, fmt.Errorf("emptyspace: sample endpoint overflow")
			}
			bounds.Min = maths.V(min(bounds.Min.X, point.X), min(bounds.Min.Y, point.Y), min(bounds.Min.Z, point.Z))
			bounds.Max = maths.V(max(bounds.Max.X, point.X), max(bounds.Max.Y, point.Y), max(bounds.Max.Z, point.Z))
		}
	}
	return mask.EmptyBox(bounds)
}
