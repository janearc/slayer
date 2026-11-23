package manufacture

import (
	"math"
	"testing"
)

// edgeOf is the largest x of any surface in an assembly's parts.
func edgeOf(assembly *instrumentAssembly) float64 {
	edge := math.Inf(-1)
	for _, piece := range assembly.object.parts {
		for _, triangle := range piece.skin() {
			for _, corner := range triangle {
				edge = math.Max(edge, corner.X)
			}
		}
	}
	return edge
}

// boxFaces is defined by a box: six faces, and two triangles to a face.
const boxFaces = 12

// sidesOf is how far across the run, either way, each box of a matter
// reaches in an assembly's parts: the largest distance from the run's
// middle of any corner of it.
func sidesOf(assembly *instrumentAssembly, matter slot) []float64 {
	var sides []float64
	for _, piece := range assembly.object.parts {
		skin := piece.skin()
		if piece.of != matter || len(skin) != boxFaces {
			continue
		}
		side := 0.0
		for _, triangle := range skin {
			for _, corner := range triangle {
				side = math.Max(side, math.Abs(corner.Z))
			}
		}
		sides = append(sides, side)
	}
	return sides
}

// fittingsAt counts the fittings of an array whose outer face is a
// distance across the run from its middle.
func fittingsAt(array solarArray, side float64) int {
	count := 0
	for _, reach := range sidesOf(laid(array), spaceGold) {
		if math.Abs(reach-side) <= solarTolerance {
			count++
		}
	}
	return count
}

// TestAHingeFittingStandsInsideThePanelsSide checks the fittings of a
// small array's two hinge lines. Its panel is .6 m across the run, so the
// panels' side is .3 m from the middle. A barrel is three quarters of the
// .02 m thickness in radius, .015 m, and a fitting three radii across,
// .045 m: the four fittings each have their outer face at .3 m, and none
// reaches the .3225 m a fitting centred on the barrel's end would. The
// fitting at the yoke's root is on the middle and reaches half of three
// of the yoke's .02 m radii, .03 m.
func TestAHingeFittingStandsInsideThePanelsSide(t *testing.T) {
	const side, past, root, hinges = .3, .3225, .03, 4
	array := smallSolarArray()
	if got := fittingsAt(array, side); got != hinges {
		t.Errorf("%d fittings have a face at the panels' side, want %d",
			got, hinges)
	}
	if got := fittingsAt(array, past); got != 0 {
		t.Errorf("%d fittings stand past the panels' side", got)
	}
	if got := fittingsAt(array, root); got != 1 {
		t.Errorf("%d fittings on the middle, want the yoke's root", got)
	}
}

// Anti-test: the fittings follow the panels and not a fixed place. A panel
// a metre across the run has its four at .5 m and none at .3 m.
func TestHingeFittingsFollowTheWidthOfThePanels(t *testing.T) {
	const side, narrow, hinges = .5, .3, 4
	array := smallSolarArray()
	array.panels.depth = 2 * side
	if got := fittingsAt(array, side); got != hinges {
		t.Errorf("%d fittings at the side of a wider panel, want %d", got,
			hinges)
	}
	if got := fittingsAt(array, narrow); got != 0 {
		t.Errorf("%d fittings stayed where a narrower panel's are", got)
	}
}

// TestAPulledInArrayEndsAtItsSpan checks the far edge of a small array.
// Its panels reach .1 m and run 2 by 1 m with a .05 m gap, so the end of the
// substrate is at .1 + 2.05 = 2.15 m. The margins of the cells are .04 m
// along the run and .035 m across it, the rail is .6 of the less, .021 m,
// and the frame stands proud by a tenth of that, .0021 m. The array as
// it is has its frame at 2.1521 m, and pulled in it ends at 2.15 m.
func TestAPulledInArrayEndsAtItsSpan(t *testing.T) {
	const span, proud = 2.15, .0021
	array := smallSolarArray()
	if got := edgeOf(laid(array)); math.Abs(got-(span+proud)) > solarTolerance {
		t.Errorf("the array ends at %v, want %v", got, span+proud)
	}
	pulled := array.pulledIn()
	if got := edgeOf(laid(pulled)); math.Abs(got-span) >
		solarTolerance {
		t.Errorf("the array pulled in ends at %v, want %v", got, span)
	}
}
