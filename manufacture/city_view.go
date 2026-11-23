package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// cityDistantLit keeps the ordinary mass and occupied panes while omitting
// the unoccupied glazing. It is an approximate representation for a view;
// it changes reflection and shadow geometry and must be compared as an image.
const cityDistantLit = "distant-lit"

// cityDetailView opts a clustered city into a camera-relative trial. Eye and
// each planned building's bounds are in the same city space; it is not a view
// of a city bent or moved after manufacture. Wide is horizontal radians, Width
// is output pixels for a square-pixel pinhole camera, and SmallPixels is the
// caller's maximum projected extent for an approximate distant building.
//
// There is deliberately no default.
// If the camera, bounds or projection are uncertain, the authored level stays.
// This trial applies to quick night buildings only.
type cityDetailView struct {
	Eye         maths.Subspace
	Width       int
	Wide        float64
	SmallPixels float64
}

// detail selects before the building is made. Off-screen buildings are not
// culled, but their simplified glazing can change reflections and shadows.
func (view cityDetailView) detail(building CityBuilding, authored string, night bool) string {
	if authored != "quick" || !night {
		return authored
	}
	pixels, ok := view.extent(building.Bounds)
	if ok && pixels <= view.SmallPixels {
		return cityDistantLit
	}
	return authored
}

// cityViewFinite is whether a number is a real one: not a not-a-number and
// not infinite.
func cityViewFinite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// A partial or skewed orientation is not a camera frame. The zero frame is
// the documented identity in maths. Do not simplify from an invalid view.
func cityViewOrientation(turn maths.Orientation) bool {
	if turn == (maths.Orientation{}) {
		return true
	}
	const tolerance = 1e-8
	for _, axis := range []maths.Vec{turn.X, turn.Y, turn.Z} {
		if !cityViewFinite(axis.X) || !cityViewFinite(axis.Y) ||
			!cityViewFinite(axis.Z) || math.Abs(axis.Dot(axis)-1) > tolerance {
			return false
		}
	}
	return math.Abs(turn.X.Dot(turn.Y)) < tolerance &&
		math.Abs(turn.Y.Dot(turn.Z)) < tolerance &&
		math.Abs(turn.Z.Dot(turn.X)) < tolerance &&
		math.Abs(turn.X.Cross(turn.Y).Dot(turn.Z)-1) < tolerance
}

// extent encloses the perspective projections of the box's eight corners.
// Any box reaching behind the eye is uncertain and keeps the authored detail.
func (view cityDetailView) extent(bounds maths.Box) (float64, bool) {
	if view.Width <= 0 || !cityViewFinite(view.Wide) || view.Wide <= 0 || view.Wide >= math.Pi || !cityViewFinite(view.SmallPixels) || view.SmallPixels <= 0 || !cityViewFinite(view.Eye.Scale) || view.Eye.Scale < 0 || !cityViewOrientation(view.Eye.Orientation) {
		return 0, false
	}
	for _, n := range []float64{bounds.Min.X, bounds.Min.Y, bounds.Min.Z, bounds.Max.X, bounds.Max.Y, bounds.Max.Z} {
		if !cityViewFinite(n) {
			return 0, false
		}
	}
	if bounds.Min.X > bounds.Max.X || bounds.Min.Y > bounds.Max.Y || bounds.Min.Z > bounds.Max.Z {
		return 0, false
	}
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, x := range []float64{bounds.Min.X, bounds.Max.X} {
		for _, y := range []float64{bounds.Min.Y, bounds.Max.Y} {
			for _, z := range []float64{bounds.Min.Z, bounds.Max.Z} {
				point := view.Eye.In(maths.V(x, y, z))
				if !cityViewFinite(point.X) || !cityViewFinite(point.Y) || !cityViewFinite(point.Z) || point.Z >= 0 {
					return 0, false
				}
				px, py := point.X/-point.Z, point.Y/-point.Z
				minX, minY = math.Min(minX, px), math.Min(minY, py)
				maxX, maxY = math.Max(maxX, px), math.Max(maxY, py)
			}
		}
	}
	extent := math.Max(maxX-minX, maxY-minY) * float64(view.Width) / (2 * math.Tan(view.Wide/2))
	if !cityViewFinite(extent) {
		return 0, false
	}
	return extent, true
}
