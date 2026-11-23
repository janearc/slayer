package over

import (
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
)

// gridMap is a map of heights made from a function of the column and row.
type gridMap struct {
	across, down int
	step         float64
	height       func(column, row int) float64
}

// Size is the map's samples across and down and how far apart they are.
func (made gridMap) Size() (int, int, float64) {
	return made.across, made.down, made.step
}

// At is the height of a sample.
func (made gridMap) At(column, row int) float64 {
	return made.height(column, row)
}

// wallRise is how many metres the wall rises from one sample to the next
// eastward: ten over a step of four is a slope of two and a half, past the
// slope the rules' flutes and beds are whole at (1.7 and a quarter more,
// 2.125) with the wall's break of 0.3 more or less on it. The wall once
// rose six, a slope of 1.5, which is under where either rule begins, so
// that nothing was ever cut or stood out of it.
const wallRise = 10

// wall is a map that rises steeply eastward, a little broken, with a
// shallow rise toward the south so that water has somewhere to go.
func wall() gridMap {
	return gridMap{across: 14, down: 14, step: 4,
		height: func(column, row int) float64 {
			return wallRise*float64(column) + 0.3*float64(row) +
				0.6*math.Sin(float64(column*7+row*3))
		}}
}

// levelSide is how many samples a side the flat and the gentle map are:
// twelve, eleven cells, so that within's stretch, cells 2 to 9, is cells of
// them with one to spare.
const levelSide = 12

// flat is a map every sample of which is the same height.
func flat() gridMap {
	return gridMap{across: levelSide, down: levelSide, step: 4,
		height: func(int, int) float64 { return 10 }}
}

// gentle is a map that rises one metre in every five, well under the slope
// any rule begins at.
func gentle() gridMap {
	return gridMap{across: levelSide, down: levelSide, step: 4,
		height: func(column, row int) float64 {
			return 0.8 * float64(column+row)
		}}
}

// everything is a set of rules with every rule on.
func everything() Rules {
	return Rules{Seed: 7, Spacing: 0.5, Faces: 400_000, Departure: 0.25,
		Margin: 8,
		Flutes: Flutes{Slope: 1.7, Depth: 1, Drainage: 100, Width: 1.5,
			Length: 8},
		Beds:      Beds{Slope: 1.7, Spacing: 3, Standout: 0.4},
		Roughness: Roughness{Gain: 1, Stand: 0.6}}
}

// broken is a map that is a wall with rock in it: heights that scatter by
// a metre or two from sample to sample over a smooth steep rise.
func broken() gridMap {
	return gridMap{across: 20, down: 20, step: 4,
		height: func(column, row int) float64 {
			return 6*float64(column) + 1.5*math.Sin(float64(column*column+
				row*7)) + 1.5*math.Cos(float64(row*row*3+column))
		}}
}

// within is the stretch of the test maps: all but a cell's width at the
// edge.
func within() Stretch {
	return Stretch{Column: 2, Row: 2, Across: 8, Down: 8}
}

// TestDrainageWorked is flow accumulation on a grid that can be worked by
// hand. Rows fall to the south, so each sample passes what it has to the one
// below: the top row has itself, the middle two, the bottom three.
func TestDrainageWorked(t *testing.T) {
	slope := gridMap{across: 3, down: 3, step: 2,
		height: func(_, row int) float64 { return float64(3 - row) }}
	want := []float64{4, 4, 4, 8, 8, 8, 12, 12, 12}
	if got := Drainage(slope); !slices.Equal(got, want) {
		t.Fatalf("drainage %v, worked by hand %v", got, want)
	}
}

// TestDrainageDiagonal is worked by hand too: a plateau with a shallow dip
// in one corner and a deep pit in the other. The middle sample falls to the
// pit across a diagonal, as steeply as it can. The dip gathers two samples
// and itself, the pit three and itself, and the two others keep their own.
func TestDrainageDiagonal(t *testing.T) {
	heights := []float64{5, 9, 9, 9, 9, 9, 9, 9, 0}
	corner := gridMap{across: 3, down: 3, step: 1,
		height: func(column, row int) float64 { return heights[row*3+column] }}
	want := []float64{3, 1, 1, 1, 1, 1, 1, 1, 4}
	if got := Drainage(corner); !slices.Equal(got, want) {
		t.Fatalf("drainage %v, worked by hand %v", got, want)
	}
}

// TestDrainageFlat is flat ground: nothing is lower, so nothing is passed
// on and each sample drains only its own cell.
func TestDrainageFlat(t *testing.T) {
	for _, each := range Drainage(flat()) {
		if each != 16 {
			t.Fatalf("flat ground drained %g m2 through a sample, want 16",
				each)
		}
	}
}

// TestSpreadSmooth is a plane of any slope, and a smooth bowl, having no
// spread round them, however steep, and one sample standing out of a
// neighbourhood having the spread it is worked out to by hand: a 3 m spike
// in 49 samples, less what a fitted surface takes, is a little under 3 over
// seven.
func TestSpreadSmooth(t *testing.T) {
	steep := gridMap{across: 9, down: 9, step: 4,
		height: func(column, row int) float64 {
			return 20*float64(column) + 0.3*float64(column*row) +
				0.1*float64(row*row)
		}}
	if got := survey(steep, 0).spread(4, 4); got != 0 {
		t.Fatalf("a smooth steep surface has spread %g m", got)
	}
	spike := gridMap{across: 9, down: 9, step: 4,
		height: func(column, row int) float64 {
			if column == 4 && row == 4 {
				return 3
			}
			return 0
		}}
	got := survey(spike, 0).spread(4, 4)
	if got < 0.3 || got > 3.0/7 {
		t.Fatalf("a spike of 3 m in 49 samples has spread %g m", got)
	}
}

// TestSpreadReadsBrokenGround is ground the survey shows broken over several
// samples having more spread than a smooth wall of the same steepness.
func TestSpreadReadsBrokenGround(t *testing.T) {
	rough, smooth := survey(broken(), 0), survey(wall(), 0)
	if !(rough.spread(10, 10) > 1) || smooth.spread(7, 7) > 0.8 {
		t.Fatalf("spread broken %g m, smooth %g m", rough.spread(10, 10),
			smooth.spread(7, 7))
	}
}

// TestNoFacetAcrossACrease is every facet of the stretch lying in one of
// the map's triangles: the facets stand nowhere off the map's surface but by
// the arithmetic's rounding, on a wall whose triangles crease.
func TestNoFacetAcrossACrease(t *testing.T) {
	_, receipt, err := Lay(wall(), 0, within(), everything())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Sag > 1e-9 {
		t.Fatalf("facets sag %g m off the survey", receipt.Sag)
	}
}

// TestRoughnessBandLimited is every wave of the made roughness shorter than
// the survey's own wavelength, two samples: none is as long, so made
// roughness has nothing at or above it.
func TestRoughnessBandLimited(t *testing.T) {
	read := survey(wall(), 0)
	made := newGround(read, within(), everything(), 0.5)
	if len(made.octaves) == 0 {
		t.Fatal("no octaves")
	}
	for _, size := range made.octaves {
		for _, each := range size.waves {
			length := 2 * math.Pi / each.number.Len()
			if length >= 2*read.step {
				t.Fatalf("a wave %g m long, the survey's is %g m", length,
					2*read.step)
			}
		}
	}
}

// TestRoughnessKeptUnderTheGround is made roughness leaning no more off the
// ground than the ground's own slope: on ground of slope 0.2 with a gain
// asking for far more, no corner moves further than the slope times the
// octaves' wavelengths over two pi, with a little for the waves adding.
func TestRoughnessKeptUnderTheGround(t *testing.T) {
	rules := everything()
	rules.Flutes.Depth, rules.Beds.Standout = 0, 0
	rules.Roughness.Gain = 1e6
	from := gridMap{across: 20, down: 20, step: 4,
		height: func(column, row int) float64 {
			return 0.8*float64(column) + 0.05*math.Sin(float64(column*row))
		}}
	_, receipt, err := Lay(from, 0, Stretch{Column: 4, Row: 4, Across: 10,
		Down: 10}, rules)
	if err != nil {
		t.Fatal(err)
	}
	const slopeNear = 0.2
	const sizes = 4 + 2 + 1
	if limit := slopeNear * sizes / (2 * math.Pi) * 1.25 * 2; receipt.Largest >
		limit {
		t.Fatalf("largest %g m, over %g m", receipt.Largest, limit)
	}
}

// TestDistribution is the made corners' moves reported in order: the median
// no more than the 90th, no more than the 99th, no more than the most, and
// that no more than the bound.
func TestDistribution(t *testing.T) {
	_, receipt, err := Lay(broken(), 0, Stretch{Column: 3, Row: 3, Across: 12,
		Down: 12}, everything())
	if err != nil {
		t.Fatal(err)
	}
	if !(receipt.Median <= receipt.Ninetieth &&
		receipt.Ninetieth <= receipt.Ninetyninth &&
		receipt.Ninetyninth <= receipt.Largest &&
		receipt.Largest <= receipt.Bound) {
		t.Fatalf("not in order: %+v", receipt)
	}
}

// TestLocateKeepsSamples is the map's own surface read back at its samples
// and on its triangles: at a sample the height is the sample's, and halfway
// along an edge it is the mean of the two ends.
func TestLocateKeepsSamples(t *testing.T) {
	read := survey(wall(), 0)
	for _, column := range []int{0, 3, 13} {
		for _, row := range []int{0, 5, 13} {
			at := read.locate(float64(column)*read.step, float64(row)*read.step)
			if got := read.surface(at); got != read.at(column, row) {
				t.Fatalf("sample (%d, %d): surface %g, survey %g", column,
					row, got, read.at(column, row))
			}
		}
	}
	middle := read.locate(5*read.step+read.step/2, 4*read.step)
	want := (read.at(5, 4) + read.at(6, 4)) / 2
	if got := read.surface(middle); math.Abs(got-want) > 1e-12 {
		t.Fatalf("middle of an edge: %g, want %g", got, want)
	}
}

// TestSurveyedHeightsKept is the survey kept: every sample of the map stands
// in the mesh exactly where the map has it, measured, and the receipt counts
// every one of them.
func TestSurveyedHeightsKept(t *testing.T) {
	from := wall()
	built, receipt, err := Lay(from, 0, within(), everything())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Moved != 0 {
		t.Fatalf("surveyed corners moved %g m", receipt.Moved)
	}
	points := built.Points()
	for row := range from.down {
		for column := range from.across {
			want := maths.V(float64(column)*from.step,
				from.At(column, row), float64(row)*from.step)
			if points[row*from.across+column] != want {
				t.Fatalf("sample (%d, %d) moved", column, row)
			}
		}
	}
	if receipt.Surveyed != from.across*from.down || receipt.Made == 0 {
		t.Fatalf("counts: %+v", receipt)
	}
}

// TestDepartureBound is no corner moving further than the share of the
// map's spacing the rules allow, with rules asking for far more than that.
func TestDepartureBound(t *testing.T) {
	rules := everything()
	rules.Flutes.Depth, rules.Beds.Standout = 50, 50
	_, receipt, err := Lay(wall(), 0, within(), rules)
	if err != nil {
		t.Fatal(err)
	}
	bound := rules.Departure * 4
	if receipt.Bound != bound || receipt.Largest > bound {
		t.Fatalf("largest departure %g m, bound %g m", receipt.Largest, bound)
	}
	if receipt.Largest == 0 {
		t.Fatal("rules asking for 50 m moved nothing")
	}
}

// TestFlatGetsNothing is a flat window getting no flutes, no beds and no
// roughness: nothing moves, and every made corner is on the plane.
func TestFlatGetsNothing(t *testing.T) {
	built, receipt, err := Lay(flat(), 0, within(), everything())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Largest != 0 {
		t.Fatalf("flat ground moved %g m", receipt.Largest)
	}
	for _, point := range built.Points() {
		if point.Y != 10 {
			t.Fatalf("a corner of flat ground is at %g m, not 10", point.Y)
		}
	}
}

// TestGentleGetsNoLedges is ground below the slope the rules begin at getting
// no flutes and no beds, though a made roughness may still be asked of it:
// ledges do not stand out of nothing.
func TestGentleGetsNoLedges(t *testing.T) {
	rules := everything()
	rules.Roughness.Gain = 0
	_, receipt, err := Lay(gentle(), 0, within(), rules)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Largest != 0 {
		t.Fatalf("ground at a slope of 0.2 moved %g m", receipt.Largest)
	}
}

// TestFlutesCutAndBedsStand is the sign of each rule: flutes only ever cut
// in, beds only ever stand out, at every corner of a steep wall.
func TestFlutesCutAndBedsStand(t *testing.T) {
	read := survey(wall(), 0)
	flutes, beds := everything(), everything()
	flutes.Beds.Standout, flutes.Roughness.Gain = 0, 0
	beds.Flutes.Depth, beds.Roughness.Gain = 0, 0
	for _, kind := range []struct {
		rules Rules
		sign  float64
	}{{flutes, -1}, {beds, 1}} {
		grid := plan(read, within(), kind.rules)
		right := 0
		for _, each := range grid.lay(read, kind.rules).corners {
			if each.moved*kind.sign < 0 {
				t.Fatalf("a corner moved %g m, the wrong way for %+v",
					each.moved, kind.rules)
			}
			if each.moved*kind.sign > 0 {
				right++
			}
		}
		if right == 0 {
			t.Errorf("no corner moved the right way for %+v: the rule did "+
				"nothing", kind.rules)
		}
	}
}

// TestDeeperWhereMoreDrains is the flutes' depth rising with the ground
// draining through: the same place cut with a small and then a large
// drainage asked for gets shallower, then deeper.
func TestDeeperWhereMoreDrains(t *testing.T) {
	rules := everything()
	rules.Beds.Standout, rules.Roughness.Gain = 0, 0
	rules.Departure = 1
	var largest []float64
	for _, drainage := range []float64{1e6, 1} {
		rules.Flutes.Drainage = drainage
		_, receipt, err := Lay(wall(), 0, within(), rules)
		if err != nil {
			t.Fatal(err)
		}
		largest = append(largest, receipt.Largest)
	}
	if !(largest[0] < largest[1]) {
		t.Fatalf("flutes cut %v m: more drainage wanted should cut less",
			largest)
	}
}

// TestSameSeedSameGround is the same rules and seed giving the same mesh,
// corner for corner, and another seed giving another.
func TestSameSeedSameGround(t *testing.T) {
	first, _, err := Lay(wall(), 0, within(), everything())
	if err != nil {
		t.Fatal(err)
	}
	second, _, _ := Lay(wall(), 0, within(), everything())
	if !slices.Equal(first.Points(), second.Points()) {
		t.Fatal("the same seed gave different ground")
	}
	other := everything()
	other.Seed++
	third, _, _ := Lay(wall(), 0, within(), other)
	if slices.Equal(first.Points(), third.Points()) {
		t.Fatal("another seed gave the same ground")
	}
}

// TestFacesFit is the stretch having no more faces than it may, and the
// spacing it is cut to made coarser to fit.
func TestFacesFit(t *testing.T) {
	rules := everything()
	rules.Faces = 20_000
	_, receipt, err := Lay(wall(), 0, within(), rules)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Faces > rules.Faces {
		t.Fatalf("%d faces, only %d allowed", receipt.Faces, rules.Faces)
	}
	read := survey(wall(), 0)
	if plan(read, within(), rules).spacing <= rules.Spacing {
		t.Fatal("the spacing was not made coarser to fit")
	}
}

// TestCellsCutFine is every cell of the stretch cut so that no edge of it,
// measured up the survey's rises, is longer than the spacing.
func TestCellsCutFine(t *testing.T) {
	read := survey(wall(), 0)
	rules := everything()
	grid := plan(read, within(), rules)
	counts := read.cuts(within(), grid.spacing, [2]int{1, 0})
	for cell, count := range counts {
		for line := range within().Down + 1 {
			rise := read.at(2+cell+1, 2+line) - read.at(2+cell, 2+line)
			if length := math.Hypot(read.step, rise); length/float64(count) >
				grid.spacing*(1+1e-9) {
				t.Fatalf("cell %d: %g m in %d parts", cell, length, count)
			}
		}
	}
}

// TestEdgeMeetsPlainGround is the stretch's outer edge being the survey's own
// triangles' edges: every corner on it is at no distance from the survey's
// surface, so the stretch and the plain ground meet with no gap.
func TestEdgeMeetsPlainGround(t *testing.T) {
	read := survey(wall(), 0)
	rules := everything()
	grid := plan(read, within(), rules)
	for place, each := range grid.lay(read, rules).corners {
		if grid.onEdge(place) && each.point != each.base {
			t.Fatalf("an edge corner moved %g m", each.moved)
		}
	}
}

// TestRefusals is a stretch off the map, too few faces and rules that
// cut nowhere each refused, not laid.
func TestRefusals(t *testing.T) {
	rules := everything()
	short := rules
	short.Faces = 10
	nowhere := rules
	nowhere.Flutes.Width = 0
	for name, test := range map[string]struct {
		stretch Stretch
		rules   Rules
	}{
		"off the map": {Stretch{Column: 10, Row: 2, Across: 8, Down: 3}, rules},
		"too few":     {within(), short},
		"nowhere":     {within(), nowhere},
	} {
		if _, _, err := Lay(wall(), 0, test.stretch, test.rules); err == nil {
			t.Errorf("%s: laid", name)
		}
	}
}
