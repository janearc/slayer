package cities

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// bandedSegment is the first segment of a straight sixty storey tower
// that has more bands than one, as its node, and the node of its bands.
func bandedSegment(t *testing.T) (segment, bands *node) {
	t.Helper()
	tower := towerOf(t, 1, 0, Pins{Storeys: 60})
	design, err := tower.Design(quickLevel, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range design.Elements {
		held, isNode := part.Element.(deferredNode)
		if !isNode || held.Node.Unfurl == nil {
			continue
		}
		opened, err := held.Node.Unfurl(quickLevel)
		if err != nil {
			t.Fatal(err)
		}
		last, isRun := opened[len(opened)-1].Element.(deferredNode)
		if isRun && last.Node.Unfurl != nil {
			return held.Node, last.Node
		}
	}
	t.Fatal("the tower has no segment of more bands than one")
	return nil, nil
}

// drawnAt is a node drawn whole at a level, or the test's end.
func drawnAt(t *testing.T, made *node, level Detail) []placedElement {
	t.Helper()
	parts, err := made.Draw(level)
	if err != nil {
		t.Fatal(err)
	}
	return parts
}

// apart is a run of bands opened all the way and each band drawn at a
// level, and how many bands there were.
func apart(t *testing.T, run *node, level Detail) (parts []placedElement,
	bands int) {
	t.Helper()
	if run.Unfurl == nil {
		return drawnAt(t, run, level), 1
	}
	halves, err := run.Unfurl(level)
	if err != nil {
		t.Fatal(err)
	}
	for _, half := range halves {
		drawn, count := apart(t, half.Element.(deferredNode).Node, level)
		parts, bands = append(parts, drawn...), bands+count
	}
	return parts, bands
}

// countedByName is how many solids of each name are among parts.
func countedByName(parts []placedElement) map[string]int {
	counts := map[string]int{}
	for _, part := range parts {
		if one, isSolid := part.Element.(solid); isSolid {
			counts[one.Name]++
		}
	}
	return counts
}

// TestARunOfBandsStandsItsMullionsOnce: all a segment's bands drawn as one
// run are the segment's storeys, so they stand the mullions the segment
// stands drawn whole, and no more.
func TestARunOfBandsStandsItsMullionsOnce(t *testing.T) {
	segment, bands := bandedSegment(t)
	whole := countedByName(drawnAt(t, segment, quickLevel))["mullion"]
	run := countedByName(drawnAt(t, bands, quickLevel))["mullion"]
	if whole == 0 || run != whole {
		t.Errorf("a run of all the bands stands %d mullions, the segment "+
			"whole %d", run, whole)
	}
}

// TestBandsDrawnApartStandAMullionEach is its anti-test: the same bands
// each drawn on its own stand a mullion apiece where the run stood one,
// so the count above is the run's doing and not the face's.
func TestBandsDrawnApartStandAMullionEach(t *testing.T) {
	_, bands := bandedSegment(t)
	run := countedByName(drawnAt(t, bands, quickLevel))["mullion"]
	each, count := apart(t, bands, quickLevel)
	if got := countedByName(each)["mullion"]; count < leastRun ||
		got != count*run {
		t.Errorf("%d bands apart stand %d mullions, the run %d", count, got,
			run)
	}
}

// TestARunOfBandsIsItsBandsButForItsMullions: on a straight tower a run
// drawn as one band has every other part its bands have drawn apart,
// dressed and bare, so joining a run changes nothing that is seen.
func TestARunOfBandsIsItsBandsButForItsMullions(t *testing.T) {
	_, bands := bandedSegment(t)
	for _, level := range []Detail{quickLevel, hullLevel} {
		run := countedByName(drawnAt(t, bands, level))
		each, _ := apart(t, bands, level)
		separate := countedByName(each)
		delete(run, "mullion")
		delete(separate, "mullion")
		if len(run) == 0 || len(run) != len(separate) {
			t.Fatalf("at %s the run has %d kinds of part, its bands %d",
				level, len(run), len(separate))
		}
		for name, count := range separate {
			if run[name] != count {
				t.Errorf("at %s the run has %d of %q, its bands %d", level,
					run[name], name, count)
			}
		}
	}
}

// levelBelow is a leveller's answer for a box: dressed where the box's
// middle is under a height, bare above it.
func levelBelow(height float64) func(maths.Box) Detail {
	return func(box maths.Box) Detail {
		if (box.Min.Y+box.Max.Y)/2 < height {
			return quickLevel
		}
		return hullLevel
	}
}

// TestARunOpensOnlyWhereABandIsAtAnotherLevel: a run whose bands are all
// at its level stays one band; a run with a bare band in it opens, into a
// lower and an upper run that meet and fill its height.
func TestARunOpensOnlyWhereABandIsAtAnotherLevel(t *testing.T) {
	_, bands := bandedSegment(t)
	foot, head := bands.Bounds.Min.Y, bands.Bounds.Max.Y
	if bands.Opens(quickLevel, levelBelow(head+1)) {
		t.Error("a run whose bands are all dressed opened")
	}
	if bands.Opens(hullLevel, levelBelow(foot-1)) {
		t.Error("a run whose bands are all bare opened")
	}
	if !bands.Opens(quickLevel, levelBelow((foot+head)/2)) {
		t.Fatal("a run dressed below and bare above did not open")
	}
	halves, err := bands.Unfurl(quickLevel)
	if err != nil {
		t.Fatal(err)
	}
	if len(halves) != 2 {
		t.Fatalf("a run opened into %d runs", len(halves))
	}
	lower := halves[0].Element.(deferredNode).Node.Bounds
	upper := halves[1].Element.(deferredNode).Node.Bounds
	if lower.Min.Y != foot || lower.Max.Y != upper.Min.Y ||
		upper.Max.Y != head {
		t.Errorf("the halves stand %v to %v and %v to %v in a run of %v "+
			"to %v", lower.Min.Y, lower.Max.Y, upper.Min.Y, upper.Max.Y,
			foot, head)
	}
}

// TestARunOfOneBandNeverOpens is its anti-test: a band has no halves, so
// it has no way to open, whatever the levels about it.
func TestARunOfOneBandNeverOpens(t *testing.T) {
	_, bands := bandedSegment(t)
	one := bands
	for one.Unfurl != nil {
		halves, err := one.Unfurl(quickLevel)
		if err != nil {
			t.Fatal(err)
		}
		one = halves[0].Element.(deferredNode).Node
	}
	if one.Opens != nil {
		t.Error("a run of one band has a way to open")
	}
}

// TestJoinedIsTheBandsAsOne: three bands of four, four and three storeys
// from the foot are one band of eleven from the foot, under the first's
// index. A band alone is itself.
func TestJoinedIsTheBandsAsOne(t *testing.T) {
	cut := []storeyBand{{index: 2, from: 8, storeys: 4},
		{index: 3, from: 12, storeys: 4}, {index: 4, from: 16, storeys: 3}}
	if got, want := joined(cut), (storeyBand{index: 2, from: 8,
		storeys: 11}); got != want {
		t.Errorf("three bands joined are %+v; want %+v", got, want)
	}
	if got := joined(cut[:1]); got != cut[0] {
		t.Errorf("a band alone joined is %+v; want %+v", got, cut[0])
	}
}
