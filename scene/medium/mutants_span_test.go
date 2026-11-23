package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The files the span tests read: rays cut by planes and cylinders, probed
// gas, and the shapes whose spans are probed.
const (
	mutantsSpanPlanesFile = "testdata/mutants-span-planes.csv"
	mutantsSpanProbedFile = "testdata/mutants-span-probed.csv"
	mutantsSpanShapesFile = "testdata/mutants-span-shapes.csv"
	mutantsSpanDiscFile   = "testdata/mutants-span-disc.csv"
)

// wantSpan checks a span against one worked by hand: whether it is met,
// and where, to a billionth.
func wantSpan(t *testing.T, name string, enter, leave float64, ok bool,
	wantEnter, wantLeave float64, wantOK bool) {
	t.Helper()
	if ok != wantOK {
		t.Errorf("%s: met is %v, want %v", name, ok, wantOK)
		return
	}
	tolerance := picked(t, "mutants-span-tolerance")
	if wantOK && !(closeTo(enter, wantEnter, tolerance) &&
		closeTo(leave, wantLeave, tolerance)) {
		t.Errorf("%s: %v to %v, want %v to %v", name, enter, leave, wantEnter,
			wantLeave)
	}
}

var (
	alongX  = maths.XAxis
	alongY  = maths.YAxis
	endless = math.Inf(unity)
)

// mutantsSpanDirections are the directions a row's ray runs in, by name.
var mutantsSpanDirections = map[string]maths.Vec{
	"x":    alongX,
	"y":    alongY,
	"down": alongY.Neg(),
}

// mutantsSpanOrigin is the place the rays that start from nothing start from.
var mutantsSpanOrigin = maths.V(nought, nought, nought)

// mutantsSpanRayFrom is the ray from a place in a direction.
func mutantsSpanRayFrom(origin, direction maths.Vec) maths.Ray {
	return maths.Ray{At: origin, Dir: direction}
}

// mutantsSpanRay is the ray a row names: its place, and its direction by the
// name of one of the directions.
func mutantsSpanRay(t *testing.T, row map[string]string) maths.Ray {
	t.Helper()
	direction, known := mutantsSpanDirections[row["direction"]]
	if !known {
		t.Fatalf("%s: no direction called %q", row["name"], row["direction"])
	}
	x, y, z := cell(t, row, "x"), cell(t, row, "y"), cell(t, row, "z")
	return mutantsSpanRayFrom(maths.V(x, y, z), direction)
}

// mutantsSpanRow is the row of a file whose name is the stem and the name.
func mutantsSpanRow(t *testing.T, file, name string) map[string]string {
	t.Helper()
	return readSheet(t, file).row(t, file, "mutants-span-"+name)
}

// mutantsSpanPlaneFunc is a slab or a cylinder: a size, a ray and a range.
type mutantsSpanPlaneFunc func(size float64, ray maths.Ray, near,
	far float64) (float64, float64, bool)

// mutantsSpanPlane cuts the ray of a row of the planes file by a slab or a
// cylinder, and holds the span to the row's, if one is met.
func mutantsSpanPlane(t *testing.T, plane mutantsSpanPlaneFunc, label,
	name string, wantOK bool) {
	t.Helper()
	row := mutantsSpanRow(t, mutantsSpanPlanesFile, name)
	ray := mutantsSpanRay(t, row)
	near, far := cell(t, row, "near"), cell(t, row, "far")
	enter, leave, ok := plane(cell(t, row, "size"), ray, near, far)
	wantEnter, wantLeave := cell(t, row, "enter"), cell(t, row, "leave")
	wantSpan(t, label, enter, leave, ok, wantEnter, wantLeave, wantOK)
}

// slab is where a ray is between two planes. a ray along them is in it
// or not by its height, the planes themselves in; a ray across them
// crosses at the two heights, whichever way it goes; and a span of no
// length is no span.
func TestSlabIsBetweenTwoPlanes(t *testing.T) {
	mutantsSpanPlane(t, slab, "along, inside", "slab-along-inside", true)
	mutantsSpanPlane(t, slab, "along, outside", "slab-along-outside", false)
	mutantsSpanPlane(t, slab, "along, on the plane",
		"slab-along-on-the-plane", true)
	mutantsSpanPlane(t, slab, "along, on the lower plane",
		"slab-along-on-the-lower-plane", true)
	mutantsSpanPlane(t, slab, "along, no length", "slab-along-no-length", false)

	// up from y = -5, the planes at y = -2 and 2 are met at 3 and 7
	mutantsSpanPlane(t, slab, "up", "slab-up", true)
	// down from y = 5, at 3 and 7 the same
	mutantsSpanPlane(t, slab, "down", "slab-down", true)
	// up from the middle, the planes at y = -1 and 1 are met at -1 and 1:
	// held to 1 and 5 it leaves where it enters, at 1
	mutantsSpanPlane(t, slab, "touching", "slab-touching", false)
}

// cylinder is where a ray is inside a cylinder round y. a ray along y is
// in it or not by how far it is from the axis, the wall itself in; a ray
// across it crosses the chord; a ray that only touches has no length.
//
// a ray a hair off the axis is on the quadratic, not taken for one along
// it: at 1e-6 across to 1 along, it leaves a cylinder 1e-4 in radius
// where it has gone 1e-4 over 1e-6 = 100 along.
func TestCylinderIsInsideARadius(t *testing.T) {
	mutantsSpanPlane(t, cylinder, "along, inside", "cylinder-along-inside",
		true)
	mutantsSpanPlane(t, cylinder, "along, outside", "cylinder-along-outside",
		false)
	// 2.1 squared less 2 squared is 0.41, outside by less than one
	mutantsSpanPlane(t, cylinder, "along, just outside",
		"cylinder-along-just-outside", false)
	mutantsSpanPlane(t, cylinder, "along, on the wall",
		"cylinder-along-on-the-wall", true)
	mutantsSpanPlane(t, cylinder, "along, a unit", "cylinder-along-a-unit",
		true)
	mutantsSpanPlane(t, cylinder, "along, no length",
		"cylinder-along-no-length", false)

	// from x = -10 along x, a cylinder of radius 6 about y is met at
	// 4 and 16 by a ray at z = 0, and only touched at 10 by one at z = 6
	mutantsSpanPlane(t, cylinder, "across", "cylinder-across", true)
	mutantsSpanPlane(t, cylinder, "touching", "cylinder-touching", false)

	// the sideways part squared is 1e-12 to the last bit, which is the
	// bound under which a ray is taken to run along the axis
	side := picked(t, "mutants-span-hair-side")
	direction := maths.V(side, unity, nought)
	ray := mutantsSpanRayFrom(mutantsSpanOrigin, direction)
	radius := picked(t, "mutants-span-hair-radius")
	enter, leave, ok := cylinder(radius, ray, nought,
		picked(t, "mutants-span-hair-far"))
	wantSpan(t, "a hair off the axis", enter, leave, ok, nought,
		picked(t, "mutants-span-hair-leave"), true)
}

// gasFrom is a medium that is something past a place along x and nothing
// before it: the simplest thing to probe a ray into.
type gasFrom struct{ x float64 }

// Bounds is no box, for the probe asks for none.
func (gas gasFrom) Bounds() maths.Box { return maths.Box{} }

// Density is one past the place and nothing before it.
func (gas gasFrom) Density(point maths.Vec) float64 {
	if point.X > gas.x {
		return unity
	}
	return nought
}

// mutantsSpanProbed probes the gas of a row of the probed file along x, holds
// the span to the row's, and gives where the span enters and where the gas is.
func mutantsSpanProbed(t *testing.T, label, name string) (enter, gas float64) {
	t.Helper()
	row := mutantsSpanRow(t, mutantsSpanProbedFile, name)
	gas = cell(t, row, "gas")
	ray := mutantsSpanRayFrom(mutantsSpanOrigin, alongX)
	near, far := cell(t, row, "near"), cell(t, row, "far")
	enter, leave, ok := probed(gasFrom{gas}, ray, near, far,
		cell(t, row, "finest"))
	wantEnter, wantLeave := cell(t, row, "enter"), cell(t, row, "leave")
	wantSpan(t, label, enter, leave, ok, wantEnter, wantLeave, true)
	return enter, gas
}

// probed steps its range in cells of no more than finest, and no fewer
// than forty-eight, and as many as 2048, and keeps from half a cell before
// the first cell whose middle is in gas to half a cell after the last,
// within the range.
//
// 4096 long in steps of 1 is 2048 cells of 2, the cap; gas past 1000.9
// first has a middle in the cell from 1000 (middle 1001), so it enters
// half a cell, 1, before it, at 999, and leaves at the end, 4096.
func TestProbedStopsAtTwoThousandAndFortyEight(t *testing.T) {
	mutantsSpanProbed(t, "capped", "probed-capped")
}

// with no finest to go by, forty-eight cells: 48 long is cells of 1, and
// gas past 20.9 first has a middle in the cell from 21 (middle 21.5), so
// the span enters half a cell before it, at 20.5.
func TestProbedWithNoFinestIsFortyEightCells(t *testing.T) {
	mutantsSpanProbed(t, "no finest", "probed-no-finest")
}

// gas that begins in the upper half of a cell is in the span. 4096 long in
// steps of 1 is cells of 2; gas past 1001.1 is in the upper half of the cell
// from 1000, whose middle, 1001, is clear, and first has a middle in the
// cell from 1002. The span enters half a cell, 1, before that cell, at 1001,
// before the gas.
func TestProbedKeepsGasThatBeginsInTheUpperHalfOfACell(t *testing.T) {
	enter, gas := mutantsSpanProbed(t, "upper half", "probed-upper-half")
	if enter > gas {
		t.Errorf("the span starts at %v, past the gas at %v", enter, gas)
	}
}

// the span is kept within the range it was given: gas through the whole of
// a range is the whole of it, not half a cell more at each end.
func TestProbedIsKeptWithinItsRange(t *testing.T) {
	mutantsSpanProbed(t, "whole", "probed-whole")
}

// mutantsSpanner is the span of a shape along a ray, in a range.
type mutantsSpanner func(ray maths.Ray, near, far float64) (float64,
	float64, bool)

// mutantsSpanHold cuts a row's ray by a shape from the near end to the
// endless one, and holds the span to the row's, if one is met.
func mutantsSpanHold(t *testing.T, spanner mutantsSpanner, label string,
	row map[string]string, wantOK bool) {
	t.Helper()
	enter, leave, ok := spanner(mutantsSpanRay(t, row), nought, endless)
	wantEnter, wantLeave := cell(t, row, "enter"), cell(t, row, "leave")
	wantSpan(t, label, enter, leave, ok, wantEnter, wantLeave, wantOK)
}

// mutantsSpanJet is the jet a row of the shapes file describes.
func mutantsSpanJet(t *testing.T, row map[string]string) Jet {
	t.Helper()
	return Jet{
		Length: cell(t, row, "length"),
		Base:   cell(t, row, "base"),
		Spread: cell(t, row, "spread"),
		Thick:  cell(t, row, "thick"),
		From:   cell(t, row, "from"),
	}
}

// mutantsSpanTail is the tail a row of the shapes file describes.
func mutantsSpanTail(t *testing.T, row map[string]string) Tail {
	t.Helper()
	return Tail{
		Length: cell(t, row, "length"),
		Base:   cell(t, row, "base"),
		Spread: cell(t, row, "spread"),
		Curve:  cell(t, row, "curve"),
		Thick:  cell(t, row, "thick"),
	}
}

// a jet's span is probed in cells no coarser than half its base. a jet
// of base 1, 100 long, with a ray up its axis from the origin and gas
// past 30.9: 100 over 0.5 is 200 cells of 0.5.
//
// the first cell with a middle past 30.9 is the one from 31 (middle
// 31.25), so the span enters half a cell, 0.25, before it, at 30.75, and
// leaves at 100.
func TestAJetsSpanIsProbedByHalfItsBase(t *testing.T) {
	row := mutantsSpanRow(t, mutantsSpanShapesFile, "jet-half-base")
	jet := mutantsSpanJet(t, row)
	mutantsSpanHold(t, jet.Span, "base", row, true)
}

// a jet of no base is probed by an eighth of its widest. 64 long and
// spreading by an eighth is 8 wide at the end, an eighth of 8 is 1, so
// 64 cells of 1; the gas is past 29.25, the first cell with a middle
// past it is from 29 (middle 29.5), the span enters half a cell before it,
// at 28.5, and the last ends at 64.
func TestAJetOfNoBaseIsProbedByAnEighthOfItsSpread(t *testing.T) {
	row := mutantsSpanRow(t, mutantsSpanShapesFile, "jet-no-base")
	jet := mutantsSpanJet(t, row)
	mutantsSpanHold(t, jet.Span, "spread", row, true)
}

// the same of a tail: no base, 64 long, spread an eighth, cells of 1.
// it bends by Curve times y squared over the length, and the gas goes
// while the bend is within two and a half widths: 0.8 y^2 / 64 over y / 8
// is 0.1 y, so to y = 25.
//
// the last cell with a middle within it is the one from 24 (middle
// 24.5), and the span leaves half a cell after it, at 25.5.
func TestATailOfNoBaseIsProbedByAnEighthOfItsSpread(t *testing.T) {
	row := mutantsSpanRow(t, mutantsSpanShapesFile, "tail-no-base")
	tail := mutantsSpanTail(t, row)
	mutantsSpanHold(t, tail.Span, "spread", row, true)
}

// a ray along a tail's head, at y of zero, is in its slab: the head is as dense
// as anywhere, Thick, and the tail is 2 across there, with gas to two and a
// half half-widths either side, so a ray from x = -10 is met at 7.5 and 12.5.
func TestARayAlongATailsHeadIsInIt(t *testing.T) {
	row := mutantsSpanRow(t, mutantsSpanShapesFile, "tail-head")
	tail := mutantsSpanTail(t, row)
	mutantsSpanHold(t, tail.Span, "along the head", row, true)
}

// a ray straight up a jet is met where it crosses the slab, the jet
// being gas all the way: 2 long either side of the origin and 1 across,
// so from y = -5 it is met at 3 and 7.
func TestARayUpAJetIsInIt(t *testing.T) {
	row := mutantsSpanRow(t, mutantsSpanShapesFile, "jet-up")
	jet := mutantsSpanJet(t, row)
	mutantsSpanHold(t, jet.Span, "up the jet", row, true)
}

// mutantsSpanDisc is the disc whose span the disc file's rays are cut by.
func mutantsSpanDisc(t *testing.T) Disc {
	t.Helper()
	return Disc{
		Inner: picked(t, "mutants-span-disc-inner"),
		Outer: picked(t, "mutants-span-disc-outer"),
		Thick: picked(t, "mutants-span-disc-thick"),
	}
}

// mutantsSpanDiscHold cuts the ray of a row of the disc file by the disc.
func mutantsSpanDiscHold(t *testing.T, disc Disc, label, name string,
	wantOK bool) {
	t.Helper()
	row := mutantsSpanRow(t, mutantsSpanDiscFile, name)
	mutantsSpanHold(t, disc.Span, label, row, wantOK)
}

// a disc's span is its slab clipped to its outer cylinder. a disc 0.4
// thick has planes at y = -0.2 and 0.2.
func TestADiscsSpan(t *testing.T) {
	disc := mutantsSpanDisc(t)
	// straight up from y = -5, at 4.8 and 5.2, inside the cylinder or on
	// its wall; down from 5 the same; and outside it, a miss
	mutantsSpanDiscHold(t, disc, "up", "disc-up", true)
	mutantsSpanDiscHold(t, disc, "down", "disc-down", true)
	mutantsSpanDiscHold(t, disc, "up the wall", "disc-up-the-wall", true)
	mutantsSpanDiscHold(t, disc, "up, outside in z", "disc-up-outside-in-z",
		false)
	mutantsSpanDiscHold(t, disc, "up, outside in x", "disc-up-outside-in-x",
		false)

	// along x, above the disc, a miss; on its face, in: the chord of
	// the outer cylinder from x = -10 is 4 to 16; touching it, no length
	mutantsSpanDiscHold(t, disc, "along, above", "disc-along-above", false)
	mutantsSpanDiscHold(t, disc, "along, on the face",
		"disc-along-on-the-face", true)
	mutantsSpanDiscHold(t, disc, "along, touching", "disc-along-touching",
		false)
}
