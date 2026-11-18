//go:build darwin

package gpu

import (
	"cmp"
	"fmt"
	"math"
	"os"
	"slices"
	"strings"

	"github.com/janearc/slayer/maths"
)

// Two diagnostics for finding a sample that holds the device, both off
// unless asked for by the environment, and neither in a picture to keep.
//
// SLAYER_WORK_COUNTS, set to anything, builds the trace counting what
// each sample does, in place of its light, and says on standard error,
// for every band, how those counts are spread and its heaviest samples.
//
// SLAYER_ONLY, as "x0-x1,y0-y1", draws only the samples in that window,
// half open, at their own places in the whole picture; the rest are
// nothing. Bisecting the window finds the sample that holds a band.

// workCounting is whether the trace counts each sample's work.
var workCounting = os.Getenv("SLAYER_WORK_COUNTS") != ""

// onlyWindow is the window of SLAYER_ONLY, or empty to draw everything.
var onlyWindow = os.Getenv("SLAYER_ONLY")

// workSlots is how many counts a sample keeps, in the trace's own order.
const workSlots = 12

// workNames names each slot of the counts, as trace.metal fills them.
// Each is a whole float of its own, so none carries into another. A
// refusal is a native shadow's candidate the software check refused,
// counted by where the device met it along the shadow; "refused floats"
// is the farthest a sample's refused candidates lay from the shadow's
// start, in floats of the size of the start's distance from the origin.
var workNames = [workSlots]string{"march steps", "lamp evaluations",
	"refused floats", "refused at the end", "refused at the start",
	"density reads", "wake path visits", "refused between",
	"native shadows", "shadow refinements", "refusals",
	"software nodes"}

// diagnosticDefines is the kernel text that turns on what was asked for.
func diagnosticDefines() string {
	text := ""
	if workCounting {
		text += fmt.Sprintf("#define SLAYER_WORK_COUNTS 1\n"+
			"constant constexpr int work_slots = %d;\n", workSlots)
	}
	var x0, x1, y0, y1 int
	if _, err := fmt.Sscanf(onlyWindow, "%d-%d,%d-%d", &x0, &x1, &y0,
		&y1); err == nil {
		text += fmt.Sprintf("#define SLAYER_ONLY_FROM_X %du\n"+
			"#define SLAYER_ONLY_TO_X %du\n#define SLAYER_ONLY_FROM_Y %du\n"+
			"#define SLAYER_ONLY_TO_Y %du\n", x0, x1, y0, y1)
	}
	return text
}

// sceneReceipt says, when counting, how far from the world's origin the
// eye is and so how coarse a float is there: the scale a precision
// disagreement between the device and the software check would have.
func sceneReceipt(eye maths.Subspace) {
	if !workCounting {
		return
	}
	far := eye.Origin.Len()
	fmt.Fprintf(os.Stderr, "work scene: the eye is %.6g from the origin; "+
		"a float there is %.3g apart\n", far,
		math.Nextafter32(float32(far), float32(math.Inf(1)))-float32(far))
}

// heaviestShown is how many of a band's heaviest samples are said.
const heaviestShown = 3

// workReceipt says, when counting, how a band's counts are spread: for
// each slot its mean, its 50th, 99th and 99.9th centiles and its most;
// then the band's heaviest samples by march steps and lamp evaluations.
func workReceipt(first, down, width, each int, samples []float32) {
	if !workCounting {
		return
	}
	var lines []string
	count := width * down
	column := make([]float64, count)
	for slot := range workSlots {
		sum := 0.0
		for index := range count {
			column[index] = float64(samples[index*each+slot])
			sum += column[index]
		}
		sorted := slices.Clone(column)
		slices.Sort(sorted)
		lines = append(lines, fmt.Sprintf("  %-17s mean %.1f p50 %.0f "+
			"p99 %.0f p99.9 %.0f most %.0f", workNames[slot],
			sum/float64(count), centile(sorted, .5), centile(sorted, .99),
			centile(sorted, .999), sorted[count-1]))
	}
	for _, slot := range weighedBy {
		lines = append(lines, heaviest(first, width, each, count, slot,
			samples)...)
	}
	fmt.Fprintf(os.Stderr, "work rows %d to %d:\n%s\n", first,
		first+down-1, strings.Join(lines, "\n"))
}

// centile is the value a share of the way up sorted values.
func centile(sorted []float64, share float64) float64 {
	return sorted[min(len(sorted)-1, int(share*float64(len(sorted))))]
}

// weighedBy is the slots a band's heaviest samples are found by: density
// reads, wake path visits and software nodes, the counts that multiply.
var weighedBy = []int{5, 6, 11}

// heaviest is a line for each of the band's heaviest samples by one
// slot, at its place in the picture and with all its counts.
func heaviest(first, width, each, count, slot int,
	samples []float32) []string {
	order := make([]int, count)
	for index := range order {
		order[index] = index
	}
	weight := func(index int) float32 { return samples[index*each+slot] }
	slices.SortFunc(order, func(a, b int) int {
		return cmp.Compare(weight(b), weight(a))
	})
	var lines []string
	for _, index := range order[:min(heaviestShown, count)] {
		counts := samples[index*each : index*each+workSlots]
		lines = append(lines, fmt.Sprintf("  heaviest by %s at %d,%d: %v",
			workNames[slot], index%width, first+index/width, counts))
	}
	return lines
}
