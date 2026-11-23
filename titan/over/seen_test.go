package over

import (
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
)

// wanting is an eye that wants the same spacing everywhere, metres.
type wanting float64

// Wanted is the spacing wanted at any point.
func (spacing wanting) Wanted(maths.Vec) float64 { return float64(spacing) }

// seeing is the rules the eye tests lay by: every rule on, a centimetre
// at finest, and faces enough for the small maps.
func seeing() Rules {
	rules := everything()
	rules.Spacing = 0.01
	rules.Faces = 4_000_000
	rules.Roughness.Continued = true
	rules.Roughness.Stand = mostSeenStand
	return rules
}

// TestSeenUnseenIsTheMap is an eye that sees nothing laying the map's own
// triangles and nothing made.
func TestSeenUnseenIsTheMap(t *testing.T) {
	unbounded := wanting(math.Inf(1))
	built, receipt, err := Seen(wall(), 0, unbounded, seeing())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Cells != 0 || receipt.Made != 0 || receipt.Faces != 0 {
		t.Fatalf("an unseen map was laid finer: %+v", receipt)
	}
	if faces := len(built.Faces()); faces != 2*13*13 {
		t.Fatalf("%d faces, the map has %d", faces, 2*13*13)
	}
}

// TestSeenCutAsTheEyeWants is every cell cut so that no edge of it is
// longer than the spacing the eye wants, in halves each way.
func TestSeenCutAsTheEyeWants(t *testing.T) {
	read := survey(wall(), 0)
	for row := range read.down - 1 {
		for column := range read.across - 1 {
			level := read.level(column, row, wanting(0.5), 0.01)
			longest := read.longestEdge(column, row)
			if longest/float64(int(1)<<level) > 0.5*(1+1e-9) {
				t.Fatalf("cell %d, %d: %g m in %d parts", column, row,
					longest, 1<<level)
			}
			if level > 0 && longest/float64(int(1)<<(level-1)) <= 0.5 {
				t.Fatalf("cell %d, %d is cut finer than wanted", column, row)
			}
		}
	}
}

// Anti: an eye that wants no finer than the survey's own spacing leaves a
// cell whole.
func TestSeenCoarseEyeLeavesCellsWhole(t *testing.T) {
	read := survey(flat(), 0)
	if level := read.level(3, 3, wanting(4), 0.01); level != 0 {
		t.Fatalf("a flat cell 4 m across was cut %d times for a 4 m eye",
			level)
	}
}

// seenGround is the made detail of broken ground for an eye, every rule
// on and no margin, so detail is made right up to the cells tested.
func seenGround(eye Fineness) (*shape, ground) {
	read := survey(broken(), 0)
	rules := seeing()
	rules.Margin = 0
	whole := Stretch{Across: read.across - 1, Down: read.down - 1}
	return read, newGround(read, whole, rules, rules.Spacing).seenBy(eye, 1)
}

// laidCell is one cell of a map laid at a level, as Seen lays it, before
// it meets its neighbours.
func laidCell(read *shape, detail ground, column, row, level int) laid {
	parts := 1 << level
	made := grid{stretch: Stretch{Column: column, Row: row, Across: 1,
		Down: 1}, eastward: newAxis([]int{parts}),
		southward: newAxis([]int{parts})}
	made.cut()
	return made.layWith(read, detail)
}

// offChord is the most any place of a cell's edge stands from the straight
// line between the places either side of it a ratio of places apart.
func offChord(placed laid, edge side, parts, ratio int) float64 {
	var most float64
	for k := range parts + 1 {
		low := k / ratio * ratio
		if k == low {
			continue
		}
		one := placed.corners[edge.first+low*edge.next].point
		other := placed.corners[edge.first+(low+ratio)*edge.next].point
		share := float64(k-low) / float64(ratio)
		point := placed.corners[edge.first+k*edge.next].point
		most = max(most, point.Sub(between(one, other, share)).Len())
	}
	return most
}

// TestSeenMeetsCoarserNeighbours is a finely cut cell's edge, made rough
// by its own detail, laid by meet onto the edge of its coarser neighbour:
// off it before, on it after, and at every place the two share, the same
// point from both cells, so the two meet without a gap.
func TestSeenMeetsCoarserNeighbours(t *testing.T) {
	read, detail := seenGround(wanting(0.05))
	columns := read.across - 1
	fine, coarse := 4, 1
	levels := make([]int, columns*(read.down-1))
	levels[6*columns+4], levels[6*columns+5] = fine, coarse
	placed := laidCell(read, detail, 4, 6, fine)
	east, ratio := sides(1 << fine)[3], 1<<(fine-coarse)
	if offChord(placed, east, 1<<fine, ratio) == 0 {
		t.Fatal("the fine edge lay on the coarse one before meeting it; " +
			"the test asks nothing")
	}
	placed.meet(read, levels, fine)
	if gap := offChord(placed, east, 1<<fine, ratio); gap > 1e-12 {
		t.Fatalf("after meeting, a place stands %g m off the coarse edge", gap)
	}
	neighbour := laidCell(read, detail, 5, 6, coarse)
	west := sides(1 << coarse)[2]
	for k := range (1 << coarse) + 1 {
		mine := placed.corners[east.first+k*ratio*east.next].point
		theirs := neighbour.corners[west.first+k*west.next].point
		if mine != theirs {
			t.Fatalf("shared place %d: %v from the fine cell, %v from the "+
				"coarse", k, mine, theirs)
		}
	}
}

// TestSeenSurveyKept is every surveyed height where the survey put it,
// however finely the eye wants the ground.
func TestSeenSurveyKept(t *testing.T) {
	_, receipt, err := Seen(broken(), 0, wanting(0.2), seeing())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Moved > 1e-9 {
		t.Fatalf("a surveyed corner moved %g m", receipt.Moved)
	}
	if receipt.Largest > receipt.Bound {
		t.Fatalf("a made corner moved %g m, past the bound %g m",
			receipt.Largest, receipt.Bound)
	}
}

// TestSeenFacesFit is the made faces kept to the rules' count, the eye's
// spacing doubled until they fit.
func TestSeenFacesFit(t *testing.T) {
	rules := seeing()
	rules.Faces = 5_000
	_, receipt, err := Seen(wall(), 0, wanting(0.1), rules)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Faces > rules.Faces || receipt.Doubled == 0 {
		t.Fatalf("%d faces of %d allowed, doubled %d times",
			receipt.Faces, rules.Faces, receipt.Doubled)
	}
}

// gapBetween is the most two details move one place apart, over places
// spread through the inner cells of the broken map, off its samples.
func gapBetween(read *shape, one, other ground) float64 {
	var most float64
	for east := 8.3; east < 64; east += 1.37 {
		for south := 8.7; south < 64; south += 1.53 {
			at := read.locate(east, south)
			point := maths.V(east, read.surface(at), south)
			normal := read.smoothed(at)
			gap := one.displacement(at, point, normal) -
				other.displacement(at, point, normal)
			most = max(most, math.Abs(gap))
		}
	}
	return most
}

// TestTwoEyesOneGround is the same place the same ground from two eyes,
// but for the octaves of roughness too fine for the coarser eye: what
// those add is under the coarser eye's spacing, a metre here.
func TestTwoEyesOneGround(t *testing.T) {
	read, near := seenGround(wanting(0.05))
	_, far := seenGround(wanting(1))
	gap := gapBetween(read, near, far)
	if gap >= 1 {
		t.Fatalf("two eyes see ground %g m apart, past the coarser's metre",
			gap)
	}
	if gap == 0 {
		t.Fatal("the eyes drew the same octaves; the test asks nothing")
	}
}

// Anti: with no roughness, flutes and beds alone, two eyes see the same
// ground exactly: the eye decides nothing of them.
func TestTwoEyesOneGroundWithoutRoughness(t *testing.T) {
	read, near := seenGround(wanting(0.05))
	_, far := seenGround(wanting(1))
	near.rules.Roughness.Gain, far.rules.Roughness.Gain = 0, 0
	if gap := gapBetween(read, near, far); gap != 0 {
		t.Fatalf("flutes and beds differ by %g m between two eyes", gap)
	}
}

// TestLayWithNothingMadeSaysNoSpread is a stretch cut no finer than the
// survey, so no corner is made, laid and its receipt given without a fault.
func TestLayWithNothingMadeSaysNoSpread(t *testing.T) {
	rules := everything()
	rules.Spacing = 100
	_, receipt, err := Lay(wall(), 0, within(), rules)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Made != 0 || receipt.Largest != 0 || receipt.Median != 0 {
		t.Fatalf("nothing made, yet the receipt says %+v", receipt)
	}
}

// TestPlanCountsBeforeItCuts is the count of faces made from the cuts,
// before cutting, never under what the cut makes, and exact for cells
// cut the same each way; and plan keeping to the cap by that count.
func TestPlanCountsBeforeItCuts(t *testing.T) {
	read := survey(wall(), 0)
	for _, spacing := range []float64{0.3, 0.7, 2, 9} {
		east := read.cuts(within(), spacing, [2]int{1, 0})
		south := read.cuts(within(), spacing, [2]int{0, 1})
		made := grid{stretch: within(), eastward: newAxis(east),
			southward: newAxis(south)}
		made.cut()
		if counted := mostFaces(east, south); len(made.faces) > counted {
			t.Fatalf("at %g m, %d faces made, %d counted", spacing,
				len(made.faces), counted)
		}
	}
	if got := mostFaces([]int{3, 3}, []int{3}); got != 2*2*3*3 {
		t.Fatalf("two cells cut three each way count %d faces", got)
	}
	rules := everything()
	rules.Faces = 3 * 2 * within().Across * within().Down
	if made := plan(read, within(), rules); len(made.faces) > rules.Faces {
		t.Fatalf("plan made %d faces of %d", len(made.faces), rules.Faces)
	}
}

// TestNoWaveAsLongAsTheSurveyAlongTheGround is every wave whose wavelength
// along the ground is the survey's or longer weighed at nothing, for
// ground level, upright either way and leaning; and a wave that runs
// straight along the normal infinitely long along the ground.
func TestNoWaveAsLongAsTheSurveyAlongTheGround(t *testing.T) {
	const survey = 8.0 // picked: two samples of 4 m
	normals := []maths.Vec{maths.V(0, 1, 0), maths.V(1, 0, 0),
		maths.V(0, 0, 1), maths.V(1, 1, 0).Unit(), maths.V(1, 2, 3).Unit()}
	for _, size := range octaves(7, 4, 0.1) {
		for _, normal := range normals {
			for _, each := range size.waves {
				long := alongGround(each.number, normal)
				kept := 1 - maths.Smooth(alongFrom*survey, survey, long)
				if long >= survey && kept != 0 {
					t.Fatalf("a wave %g m long along the ground kept %g",
						long, kept)
				}
			}
		}
	}
	along := alongGround(maths.V(0, 2, 0), maths.V(0, 1, 0))
	if !math.IsInf(along, 1) {
		t.Fatal("a wave along the normal has a length along the ground")
	}
}

// TestSeenSameSeedSameGround is the same eye, rules and seed giving the
// same mesh, and another seed another.
func TestSeenSameSeedSameGround(t *testing.T) {
	first, _, err := Seen(broken(), 0, wanting(0.5), seeing())
	if err != nil {
		t.Fatal(err)
	}
	second, _, _ := Seen(broken(), 0, wanting(0.5), seeing())
	if !slices.Equal(first.Points(), second.Points()) {
		t.Fatal("the same seed gave different ground")
	}
	other := seeing()
	other.Seed++
	third, _, _ := Seen(broken(), 0, wanting(0.5), other)
	if slices.Equal(first.Points(), third.Points()) {
		t.Fatal("another seed gave the same ground")
	}
}

// TestSeenRefusals is no eye, no spacing and no faces each refused.
func TestSeenRefusals(t *testing.T) {
	spaceless, faceless := seeing(), seeing()
	spaceless.Spacing, faceless.Faces = 0, 0
	for name, test := range map[string]struct {
		eye   Fineness
		rules Rules
	}{
		"no eye":     {nil, seeing()},
		"no spacing": {wanting(1), spaceless},
		"no faces":   {wanting(1), faceless},
	} {
		if _, _, err := Seen(wall(), 0, test.eye, test.rules); err == nil {
			t.Errorf("%s: laid", name)
		}
	}
}

// TestScalingKeptInBounds is the survey's fall of roughness with scale
// read from broken ground between the square root and one, and a smooth
// map, which has no roughness to read, taken as one.
func TestScalingKeptInBounds(t *testing.T) {
	big := gridMap{across: 40, down: 40, step: 4,
		height: broken().height}
	whole := Stretch{Across: 39, Down: 39}
	scaling := survey(big, 0).scaling(whole)
	if scaling < leastScaling || scaling > mostScaling {
		t.Fatalf("scaling %g is out of its bounds", scaling)
	}
	smooth := gridMap{across: 40, down: 40, step: 4,
		height: gentle().height}
	if got := survey(smooth, 0).scaling(whole); got != mostScaling {
		t.Fatalf("a smooth map read a scaling of %g", got)
	}
}

// Anti: relief not continued is as steep at a sixteenth of a metre as at
// four metres, for the same spread, which is the law Lay made before the
// survey's scaling was read; continued at the square root, the small wave
// is four times as steep, the sixteenth's root over the whole's.
func TestNotContinuedIsTheSameSlope(t *testing.T) {
	read := survey(broken(), 0)
	same := newGround(read, within(), everything(), 0.5)
	unbounded := math.Inf(1)
	big, small := same.steepness(1, 4, unbounded),
		same.steepness(1, 0.25, unbounded)
	if big != small {
		t.Fatalf("not continued, %g at 4 m and %g at a quarter", big, small)
	}
	falling := same
	falling.scaling = leastScaling
	ratio := falling.steepness(1, 0.25, unbounded) /
		falling.steepness(1, 4, unbounded)
	if math.Abs(ratio-4) > 1e-12 {
		t.Fatalf("at the square root the small wave is %g times as steep",
			ratio)
	}
}

// Seen refuses a stand or a departure past what keeps two eyes' ground
// within a pixel.
func TestSeenRefusesTooSteepOrTooFar(t *testing.T) {
	steep, far := seeing(), seeing()
	steep.Roughness.Stand = mostSeenStand * 1.2
	far.Departure = mostSeenDeparture * 1.5
	for name, rules := range map[string]Rules{"steep": steep, "far": far} {
		if _, _, err := Seen(wall(), 0, wanting(1), rules); err == nil {
			t.Errorf("%s: laid", name)
		}
	}
}
