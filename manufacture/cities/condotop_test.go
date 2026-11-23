package cities

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// reachOf is the lowest and highest corner of a placed solid in the frame
// it is placed in, taken through its place, whatever the turn of it. A
// solid is a box, or a prism over a circle.
func reachOf(t *testing.T, part placedElement) (low, high maths.Vec) {
	t.Helper()
	solid, isSolid := part.Element.(solid)
	if !isSolid {
		t.Fatalf("a placed part that is not a solid: %T", part.Element)
	}
	var near, far maths.Vec
	switch form := solid.Form.(type) {
	case Box:
		far = maths.V(form.Width, form.Height, form.Depth)
	case prism:
		radius := form.Bottom.(outlineCircle).Radius
		near, far = maths.V(-radius, 0, -radius),
			maths.V(radius, form.Height, radius)
	default:
		t.Fatalf("a form that is neither box nor prism: %T", solid.Form)
	}
	low = maths.V(math.Inf(1), math.Inf(1), math.Inf(1))
	high = maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))
	for _, x := range []float64{near.X, far.X} {
		for _, y := range []float64{near.Y, far.Y} {
			for _, z := range []float64{near.Z, far.Z} {
				at := part.Place.Out(maths.V(x, y, z))
				low = maths.V(math.Min(low.X, at.X), math.Min(low.Y, at.Y),
					math.Min(low.Z, at.Z))
				high = maths.V(math.Max(high.X, at.X), math.Max(high.Y,
					at.Y), math.Max(high.Z, at.Z))
			}
		}
	}
	return low, high
}

// insideArea is whether a reach lies inside the walls of a rectangle on the
// ground, and at or over the floor.
func insideArea(low, high maths.Vec, area roofArea) bool {
	return low.X >= area.x0-tolerance && high.X <= area.x1+tolerance &&
		low.Z >= area.z0-tolerance && high.Z <= area.z1+tolerance &&
		low.Y >= area.y-tolerance
}

func TestEverythingARoofCarriesStandsInsideTheRoof(t *testing.T) {
	area := roofArea{x0: 0.2, z0: 0.2, x1: 28, z1: 11, y: 14}
	carried := 0
	for seed := range Seed(seeds) {
		drawing := homeDrawer(t, seed)
		for bay := range 4 {
			for _, part := range drawing.crown(place{Bay: bay}, area) {
				low, high := reachOf(t, part)
				carried++
				if !insideArea(low, high, area) {
					t.Errorf("seed %d bay %d: a roof part at %v to %v is "+
						"off the roof", seed, bay, low, high)
				}
			}
		}
	}
	if carried == 0 {
		t.Error("no roof carries anything")
	}
}

func TestAnEntranceCanopyStandsOverItsStairCore(t *testing.T) {
	drawing := homeDrawer(t, 1)
	left, front := drawing.fixed("core-offset"), 12.0
	side := drawing.fixed("canopy-side")
	for _, part := range drawing.canopy(front, left) {
		low, high := reachOf(t, part)
		if low.X < left-side-tolerance || high.X > left+drawing.fixed(
			"core-width")+side+tolerance || low.Z < front-tolerance {
			t.Errorf("a canopy part at %v to %v is not over its core", low,
				high)
		}
	}
}

func TestShopsInARowAreInsideTheFrontAndNeverOverlap(t *testing.T) {
	drawing := homeDrawer(t, 1)
	from, length := 6.0, 28.0
	for bay := range 4 {
		var signs [][2]float64
		for _, part := range drawing.podium(place{Bay: bay}, 11.5, from,
			length) {
			low, high := reachOf(t, part)
			if low.X < from-tolerance || high.X > length+tolerance {
				t.Errorf("bay %d: a shop part at %v to %v is off the front",
					bay, low.X, high.X)
			}
			if solid := part.Element.(solid); solid.Name == "shop sign" {
				signs = append(signs, [2]float64{low.X, high.X})
			}
		}
		for index := 1; index < len(signs); index++ {
			if signs[index][0] < signs[index-1][1] {
				t.Errorf("bay %d: two shop signs overlap", bay)
			}
		}
	}
}

func TestEverythingInTheCourtStandsInsideTheCourt(t *testing.T) {
	for seed := range Seed(homeSeeds) {
		drawing := homeDrawer(t, seed)
		shared := court{width: 30, depth: 26}
		for _, part := range shared.furnish(drawing) {
			low, high := reachOf(t, part)
			if low.X < -shared.width/half-tolerance ||
				high.X > shared.width/half+tolerance ||
				low.Z < -shared.depth/half-tolerance ||
				high.Z > shared.depth/half+tolerance {
				t.Errorf("seed %d: a court part at %v to %v is off the "+
					"court", seed, low, high)
			}
		}
	}
}

func TestTheCourtHoldsAFewCarsAndSomePlanting(t *testing.T) {
	drawing := homeDrawer(t, 1)
	parts := (court{width: 30, depth: 26}).furnish(drawing)
	cars, trees := 0, 0
	for _, part := range parts {
		switch part.Element.(solid).Name {
		case "car body":
			cars++
		case "crown":
			trees++
		}
	}
	if cars < 3 || cars > 5 || trees == 0 {
		t.Errorf("%d cars and %d trees in the court", cars, trees)
	}
}
