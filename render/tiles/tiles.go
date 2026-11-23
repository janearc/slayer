// Package tiles partitions a full image into bounded rectangles while keeping
// their coordinates in the full image.
package tiles

import (
	"errors"
	"fmt"
	"image"
	"math"
)

// Each visits non-overlapping rectangles covering the image in deterministic
// order. side is the number of samples along each pixel axis. The initial
// grid has side columns and side rows, limited by the image dimensions.
// Oversized rectangles are subdivided further.
//
// The byte estimate is fixedBytes plus pixels times side squared times
// bytesPerSample. The caller supplies those costs. It must account separately
// for shared scene data, resident caches, temporary allocations and concurrent
// tiles. This estimate does not enforce a process memory limit.
func Each(width, height, side int, bytesPerSample, fixedBytes, budget uint64,
	visit func(image.Rectangle, uint64) error,
) error {
	if width <= 0 || height <= 0 || side <= 0 || bytesPerSample == 0 || visit == nil {
		return errors.New("tiles: positive dimensions, sampling and a visitor are required")
	}
	if fixedBytes >= budget {
		return errors.New("tiles: fixed bytes leave no space in the budget")
	}
	samples := uint64(side)
	if samples > math.MaxUint64/samples || samples*samples > math.MaxUint64/bytesPerSample {
		return errors.New("tiles: per-pixel byte estimate overflows")
	}
	perPixel := samples * samples * bytesPerSample
	maxPixels := (budget - fixedBytes) / perPixel
	if maxPixels == 0 {
		return fmt.Errorf("tiles: budget cannot hold one pixel")
	}
	columns, rows := min(side, width), min(side, height)
	for y := 0; y < rows; y++ {
		y0 := boundary(height, rows, y)
		y1 := boundary(height, rows, y+1)
		for x := 0; x < columns; x++ {
			x0 := boundary(width, columns, x)
			x1 := boundary(width, columns, x+1)
			if err := subdivide(image.Rect(x0, y0, x1, y1), maxPixels, perPixel, fixedBytes, visit); err != nil {
				return err
			}
		}
	}
	return nil
}

// subdivide visits rect whole when its pixels fit the budget, with the
// bytes it will take, and otherwise halves it across its longer side and
// visits each half the same way, the left or upper half first.
func subdivide(rect image.Rectangle, maxPixels, perPixel, fixedBytes uint64,
	visit func(image.Rectangle, uint64) error,
) error {
	width, height := uint64(rect.Dx()), uint64(rect.Dy())
	if width <= maxPixels/height {
		return visit(rect, fixedBytes+width*height*perPixel)
	}
	if width >= height && width > 1 {
		middle := rect.Min.X + rect.Dx()/2
		if err := subdivide(image.Rect(rect.Min.X, rect.Min.Y, middle, rect.Max.Y), maxPixels, perPixel, fixedBytes, visit); err != nil {
			return err
		}
		return subdivide(image.Rect(middle, rect.Min.Y, rect.Max.X, rect.Max.Y), maxPixels, perPixel, fixedBytes, visit)
	}
	middle := rect.Min.Y + rect.Dy()/2
	if err := subdivide(image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, middle), maxPixels, perPixel, fixedBytes, visit); err != nil {
		return err
	}
	return subdivide(image.Rect(rect.Min.X, middle, rect.Max.X, rect.Max.Y), maxPixels, perPixel, fixedBytes, visit)
}

// boundary places a grid boundary without multiplying two image dimensions.
func boundary(size, count, index int) int {
	return (size/count)*index + min(index, size%count)
}

// Refine partitions rect using the caller's estimate for each rectangle.
// The estimate can account for spatially varying work or resident data.
// An estimate error, including an unknown cost, stops iteration. If a
// single pixel exceeds budget, Refine returns an error.
func Refine(rect image.Rectangle, budget uint64,
	estimate func(image.Rectangle) (uint64, error),
	visit func(image.Rectangle, uint64) error,
) error {
	if rect.Empty() || rect.Dx() <= 0 || rect.Dy() <= 0 || budget == 0 || estimate == nil || visit == nil {
		return errors.New("tiles: a nonempty rectangle, budget, estimator and visitor are required")
	}
	bytes, err := estimate(rect)
	if err != nil {
		return err
	}
	if bytes <= budget {
		return visit(rect, bytes)
	}
	if rect.Dx() == 1 && rect.Dy() == 1 {
		return errors.New("tiles: one pixel exceeds the estimated budget")
	}
	if rect.Dx() >= rect.Dy() && rect.Dx() > 1 {
		middle := rect.Min.X + rect.Dx()/2
		if err := Refine(image.Rect(rect.Min.X, rect.Min.Y, middle, rect.Max.Y), budget, estimate, visit); err != nil {
			return err
		}
		return Refine(image.Rect(middle, rect.Min.Y, rect.Max.X, rect.Max.Y), budget, estimate, visit)
	}
	middle := rect.Min.Y + rect.Dy()/2
	if err := Refine(image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, middle), budget, estimate, visit); err != nil {
		return err
	}
	return Refine(image.Rect(rect.Min.X, middle, rect.Max.X, rect.Max.Y), budget, estimate, visit)
}
