package main

import (
	"fmt"
	"math"
	"sort"

	"github.com/janearc/slayer/improv/cascades/flatland"
)

// relativeFloor keeps relative error finite where the reference is nearly
// zero. Picked: far below anything visible in the picture.
const relativeFloor = 1e-6

// percentile is the quantile of error reported besides the mean.
const percentile = 0.99

var bandNames = [flatland.Bands]string{"red", "green", "blue"}

// printPlan prints how many cascades there are and, for each, its probes,
// spacing, directions, shell, and rays.
func printPlan(plan flatland.Plan) {
	fmt.Printf("cascades used: %d\n", len(plan))
	for _, level := range plan {
		fmt.Printf("  cascade %d: probes %dx%d (spacing %d cells), "+
			"directions %d, shell [%.2f, %s], rays %d\n",
			level.Index, level.ProbesPerSide, level.ProbesPerSide, level.Spacing,
			level.Directions, level.Start, endText(level.End), level.Rays())
	}
}

// endText is the end of a cascade's shell with two decimals, or "edge of
// grid" for an end at infinity.
func endText(end float64) string {
	if math.IsInf(end, 1) {
		return "edge of grid"
	}
	return fmt.Sprintf("%.2f", end)
}

// printStats prints the rays traced, march steps, and wall time of the
// method called name.
func printStats(name string, stats flatland.Stats) {
	fmt.Printf("%s: rays traced %d, march steps %d, wall time %.2fs\n",
		name, stats.Rays, stats.Steps, stats.Elapsed.Seconds())
}

// printErrors compares a field with the reference over cells outside the
// opaque circles.
//
// Two measures per band. Relative error divides by the reference value
// (floored), which is harsh in deep shadow where the reference is almost
// zero. Error against the band mean divides by the reference's mean over
// the same cells instead, so a shadow cannot inflate it.
func printErrors(name string, field, reference flatland.Field, opaque []bool) {
	for band := 0; band < flatland.Bands; band++ {
		var relative, scaled []float64
		sum, counted := 0.0, 0
		for cell, inside := range opaque {
			if !inside {
				sum += reference.Values[cell*flatland.Bands+band]
				counted++
			}
		}
		average := sum / float64(counted)
		for cell, inside := range opaque {
			if inside {
				continue
			}
			want := reference.Values[cell*flatland.Bands+band]
			gap := math.Abs(field.Values[cell*flatland.Bands+band] - want)
			relative = append(relative, gap/math.Max(want, relativeFloor))
			scaled = append(scaled, gap/average)
		}
		printSpread(name, bandNames[band]+" relative", relative)
		printSpread(name, bandNames[band]+" against band mean", scaled)
	}
}

// printSpread sorts errors, in place, and prints their mean, 99th
// percentile, and largest under name and label. errors must not be empty.
func printSpread(name, label string, errors []float64) {
	sort.Float64s(errors)
	fmt.Printf("%s error, %s: mean %.4f, 99th percentile %.4f, largest %.4f\n",
		name, label, mean(errors),
		errors[int(math.Ceil(percentile*float64(len(errors))))-1],
		errors[len(errors)-1])
}

// mean is the average of values, and NaN if there are none.
func mean(values []float64) float64 {
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}
