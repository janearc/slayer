//go:build darwin

package metal

import (
	"math"
	"testing"
)

// These are integer representation boundaries, not large allocations or GPU jobs.
func TestAccelerationStagingChecksBothNativeAndHostCounts(test *testing.T) {
	nodes := min(math.MaxInt32, math.MaxInt/3)
	instances := min(math.MaxInt32, math.MaxInt/12)
	if !accelerationCounts(nodes, instances) {
		test.Fatal("the exact representable capacities were refused")
	}
	for _, counts := range [][2]int{
		{nodes + 1, 0}, {0, instances + 1}, {-1, 0}, {0, -1},
	} {
		if accelerationCounts(counts[0], counts[1]) {
			test.Fatal("unrepresentable staging count accepted", counts)
		}
	}
}

func TestAccelerationNodeWordsAreCheckedBeforeNarrowing(test *testing.T) {
	if !accelerationRange(math.MaxInt32, math.MaxInt32) {
		test.Fatal("exact signed native words refused")
	}
	for _, words := range [][2]int{
		{math.MaxInt32 + 1, 0}, {0, math.MaxInt32 + 1},
		{-1, 0}, {0, -1},
	} {
		if accelerationRange(words[0], words[1]) {
			test.Fatal("node word would truncate at the C boundary", words)
		}
	}
}
