package skeleton

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/maths"
)

// webRays are five rays of a fin standing up from a line, each a little
// shorter than the last, carried by no motion.
func webRays(count int) []WebRay {
	rays := make([]WebRay, count)
	for i := range rays {
		base := maths.V(-float64(i)*.01, 0, 0)
		reach := maths.V(-.02, .05-float64(i)*.005, 0)
		tip := base.Add(reach)
		rays[i] = NewWebRay(base, tip, Frame{})
	}
	return rays
}

// TestWebIsClosed holds a web to being one closed solid: every edge run
// once each way, and no triangle with a corner twice.
func TestWebIsClosed(t *testing.T) {
	sheet := NewSheet()
	if err := sheet.Web(webRays(5), 4, .002, maths.V(0, 0, 1)); err != nil {
		t.Fatal(err)
	}
	edges := map[[2]int]int{}
	for _, face := range sheet.Faces() {
		if face[0] == face[1] || face[1] == face[2] || face[0] == face[2] {
			t.Fatalf("a triangle names a corner twice: %v", face)
		}
		for k := range 3 {
			edges[[2]int{face[k], face[(k+1)%3]}]++
		}
	}
	for edge, count := range edges {
		back := edges[[2]int{edge[1], edge[0]}]
		if count != 1 || back != 1 {
			t.Fatalf("edge %v is run %d times and back %d times", edge,
				count, back)
		}
	}
	if len(sheet.Points()) != len(sheet.Rest()) {
		pointCount := len(sheet.Points())
		restCount := len(sheet.Rest())
		t.Fatalf("%d points and %d rest points", pointCount, restCount)
	}
}

// TestWebMovesWithItsRays holds a web carried by a shift to lie shifted,
// its rest points where they were.
func TestWebMovesWithItsRays(t *testing.T) {
	shift := maths.V(1, 2, 3)
	moved := NewFrame(maths.Identity, shift)
	rays := webRays(3)
	for i := range rays {
		rays[i].carry = moved
	}
	sheet := NewSheet()
	if err := sheet.Web(rays, 2, .002, maths.V(0, 0, 1)); err != nil {
		t.Fatal(err)
	}
	for i, p := range sheet.Points() {
		if !p.Near(sheet.Rest()[i].Add(shift), 1e-12) {
			t.Fatalf("point %d at %v, rest %v", i, p, sheet.Rest()[i])
		}
	}
}

// TestWebRefusesTooFewRays is the anti-test: two rays make no web, nor
// does a web of no steps.
func TestWebRefusesTooFewRays(t *testing.T) {
	sheet := NewSheet()
	if err := sheet.Web(webRays(2), 4, .002, maths.V(0, 0, 1)); !errors.Is(err,
		errWebRays) {
		t.Fatalf("two rays made a web: %v", err)
	}
	if err := sheet.Web(webRays(4), 0, .002, maths.V(0, 0, 1)); !errors.Is(err,
		errWebRays) {
		t.Fatalf("no steps made a web: %v", err)
	}
	if len(sheet.Faces()) != 0 {
		faceCount := len(sheet.Faces())
		t.Fatalf("a refused web left %d faces", faceCount)
	}
}
