package scrooge

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// Frugal checks a node against scrooge's contract at a footprint, as each
// kind's tests ask: its children lie within its box; none it gives is
// finer than the footprint asks unless it gives nothing further itself;
// and the children keep the parent's light, so a node does not brighten
// or dim as it opens.
//
// It checks one layer; a test walks deeper by asking
// it again of each child. Tolerance is the share by which summed light
// may differ, for rounding.
func Frugal(node Node, at Footprint, tolerance float64) error {
	made := node.Aggregate()
	var sum maths.Light
	given := 0
	for child := range node.Detail(at) {
		given++
		part := child.Aggregate()
		if !within(part.Box, made.Box) {
			return fmt.Errorf("scrooge: child %d's box is outside its "+
				"parent's", given)
		}
		width := at.WidthAt(at.Nearest(part.Box))
		if part.Size() < width {
			for range child.Detail(at) {
				return fmt.Errorf("scrooge: child %d is %g m across, "+
					"finer than the footprint's %g m, and gives a layer "+
					"under it", given, part.Size(), width)
			}
		}
		sum = sum.Add(part.Emission)
	}
	if given == 0 {
		return nil
	}
	if !near(sum, made.Emission, tolerance) {
		return fmt.Errorf("scrooge: the children's light %v is not the "+
			"parent's %v", sum, made.Emission)
	}
	return nil
}

// within is whether a box lies inside another.
func within(inner, outer maths.Box) bool {
	return inner.Min.X >= outer.Min.X && inner.Min.Y >= outer.Min.Y &&
		inner.Min.Z >= outer.Min.Z && inner.Max.X <= outer.Max.X &&
		inner.Max.Y <= outer.Max.Y && inner.Max.Z <= outer.Max.Z
}

// near is whether two lights are the same within a share of the larger.
func near(one, other maths.Light, tolerance float64) bool {
	close := func(a, b float64) bool {
		return math.Abs(a-b) <= tolerance*math.Max(math.Abs(a), math.Abs(b))
	}
	return close(one.R, other.R) && close(one.G, other.G) &&
		close(one.B, other.B)
}
