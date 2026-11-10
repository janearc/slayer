//go:build 386 || arm || mips || mipsle

package medium

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// At a 32-bit index limit, a radius-one canopy can have one endpoint
// outside the host range while the other still fits. Checking both is
// necessary even though the endpoints round together near 64-bit limits.
func TestGroveRejectsACanopyWithOnlyItsLowerIndexOutsideTheHost(t *testing.T) {
	weather := GroveWeather{Extinction: .1, Depth: 1, Coverage: 1}
	tree := Canopy{At: maths.V(float64(math.MinInt)+2, 0, 0),
		Radius: 1, Height: 1}
	ground := func(float64, float64) float64 { return 0 }
	if _, err := NewGrove(weather, []Canopy{tree}, ground); err != nil {
		t.Fatalf("nearby in-range canopy: %v", err)
	}
	tree.At.X = float64(math.MinInt)
	// A bad conversion would start at some host integer. The upper bound
	// is MinInt+1, so even a missing refusal walks at most two X buckets.
	// This fixture cannot reproduce the positive-overflow runaway loop.
	if _, err := NewGrove(weather, []Canopy{tree}, ground); err == nil ||
		!strings.Contains(err.Error(), "bucket index range") {
		t.Fatalf("partially out-of-range canopy: %v", err)
	}
}
