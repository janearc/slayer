package gpu

import (
	"github.com/janearc/slayer/maths"
	"math"
	"math/bits"
	"strings"
	"testing"
)

func TestInstanceBudgetCountsSharedLocalMembersAndRefusesBeforePacking(t *testing.T) {
	limit := math.MaxInt32 / instanceStruct.size()
	for _, counts := range [][]int{{0}, {1}, {limit}, {1000, 2000, 3000}} {
		if err := instanceBudget(math.MaxInt32, counts...); err != nil {
			t.Fatalf("valid counts %v: %v", counts, err)
		}
	}
	for _, counts := range [][]int{{limit + 1}, {limit, 1}, {1000, limit - 999}} {
		if err := instanceBudget(math.MaxInt32, counts...); err == nil || !strings.Contains(err.Error(), "instance buffer") {
			t.Fatalf("bad counts %v: %v", counts, err)
		}
	}
	encoded := newScene()
	if err := encoded.instancesWithin(nodeStruct.size() - 1); err == nil || !strings.Contains(err.Error(), "hierarchy") {
		t.Fatalf("empty root budget: %v", err)
	}
	bounds := box{low: maths.V(-1, -1, -1), high: maths.V(1, 1, 1)}
	encoded.standings = []standing{{bounds: bounds}, {bounds: bounds}}
	encoded.groups = []meshGroup{{members: []standing{{bounds: bounds}, {bounds: bounds}, {bounds: bounds}}}}
	if err := encoded.instancesWithin(4 * instanceStruct.size()); err == nil || !strings.Contains(err.Error(), "instance buffer") {
		t.Fatalf("local member budget: %v", err)
	}
	if len(encoded.instances.bytes) != 0 || len(encoded.tops.bytes) != 0 {
		t.Fatal("refusal partially packed the scene")
	}
	if err := encoded.instancesWithin(5 * instanceStruct.size()); err != nil {
		t.Fatal(err)
	}
	if encoded.counts.instances != 5 || len(encoded.instances.bytes) != 5*instanceStruct.size() {
		t.Fatal("budget and actual layout disagree")
	}
}
func TestMedianTreesFitTheKernelStackAtTheirPackedIndexLimits(t *testing.T) {
	// All three walks have separate stacks. A median split increases
	// pending siblings by at most one per level. This bound needs no
	// allocation of a scene near the two-gigabyte buffer limit.
	for _, stride := range []int{triangleStruct.size(), instanceStruct.size()} {
		count := math.MaxInt32 / stride
		maximumPending := bits.Len(uint(count-1)) + 1
		if maximumPending > treeStack {
			t.Fatalf("%d-byte entries need %d pending nodes", stride, maximumPending)
		}
	}
	for _, n := range []int{1, 2, 3, 5, 64, 1024} {
		standings := make([]standing, n)
		for i := range standings {
			standings[i].bounds = box{low: maths.V(float64(i), 0, 0), high: maths.V(float64(i)+.5, 1, 1)}
		}
		_, nodes := topped(standings)
		pending := []int{0}
		peak := 1
		visited := 0
		for len(pending) > 0 {
			index := pending[len(pending)-1]
			pending = pending[:len(pending)-1]
			visited++
			node := nodes[index]
			if node.count == 0 {
				pending = append(pending, node.first, node.first+1)
			}
			peak = max(peak, len(pending))
		}
		if visited != len(nodes) || peak > bits.Len(uint(n-1))+1 {
			t.Fatalf("%d members: %d nodes, peak %d", n, visited, peak)
		}
	}
}
