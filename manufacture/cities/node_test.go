package cities

import (
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	"github.com/janearc/slayer/scene/thing"
)

// countedNode is a node of a metre box that counts how often it is drawn
// and unfurled, and opens at every level.
func countedNode(drawn, unfurled *int) *node {
	part := put("box", Box{1, 1, 1}, &thing.Material{Name: "box"},
		maths.Vec{})
	return &node{Bounds: maths.Box{Max: maths.V(1, 1, 1)},
		Draw: func(Detail) ([]placedElement, error) {
			*drawn++
			return []placedElement{part}, nil
		},
		Opens: func(Detail, func(maths.Box) Detail) bool { return true },
		Unfurl: func(Detail) ([]placedElement, error) {
			*unfurled++
			return []placedElement{part, part}, nil
		}}
}

func TestANodeOutOfViewNeverUnfurls(t *testing.T) {
	drawn, unfurled := 0, 0
	pass := Pass{Leveller: Constant("demo"), Finest: "demo",
		Sees: func(maths.Box) bool { return false }}
	element := deferred(countedNode(&drawn,
		&unfurled))
	flat, err := pass.levelled([]placedElement{element}, standingAt(maths.Vec{}), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if unfurled != 0 || drawn != 1 || len(flat) != 1 {
		t.Errorf("unseen, a node unfurled %d times, drew %d and left %d",
			unfurled, drawn, len(flat))
	}
}

func TestANodeInViewUnfurlsOnceWhenItsLevelCallsForIt(t *testing.T) {
	drawn, unfurled := 0, 0
	pass := Pass{Leveller: Constant("demo"), Finest: "demo"}
	element := deferred(countedNode(&drawn,
		&unfurled))
	flat, err := pass.levelled([]placedElement{element}, standingAt(maths.Vec{}), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if unfurled != 1 || drawn != 0 || len(flat) != 2 {
		t.Errorf("seen, a node unfurled %d times, drew %d and left %d",
			unfurled, drawn, len(flat))
	}
}

func TestAPassWithNoLevellerIsRefused(t *testing.T) {
	if _, err := (Design{}).Levelled(Pass{}, maths.Subspace{}); err == nil {
		t.Error("a pass with no leveller drew a design")
	}
	if _, err := (deferredNode{}).Rendered("quick"); err == nil {
		t.Error("a deferred node drew itself as one element")
	}
}

// longWindowPitch is how far apart, in metres, the windows of the long
// face are: picked, ten, so that the face has windows from its left end to
// its right and some of them near an eye at one end and far from it.
const longWindowPitch = 10.0

// drawnLongFacade is the face four hundred metres long that the culture
// draws, its windows as it draws them, made at the demo level.
func drawnLongFacade(t *testing.T) facade {
	t.Helper()
	drawing := drawerOf(t, "portofino", 1)
	drawing.level = demoLevel
	facade := drawing.facade(place{}, massShape{width: 400, depth: 12,
		storeys: 1}, &thing.Material{Name: "wall"})
	if drawing.trouble != nil {
		t.Fatal(drawing.trouble)
	}
	return facade
}

// The windows of a long face are spread over all of it: from one end to
// the first, between windows and from the last to the other end there is
// the same gap (house.go spread). The culture draws a few windows for a
// bay and a face this long gets no more, so the gap is long; whether a
// long face should have more windows is the culture's to say.
func TestTheWindowsOfALongFaceReachWithinOneGapOfEachEnd(t *testing.T) {
	facade := drawnLongFacade(t)
	windows := facade.Storeys[0]
	if len(windows) < 2 {
		t.Fatalf("%d windows on a face to look at", len(windows))
	}
	first, last := windows[0], windows[len(windows)-1]
	gap := windows[1].Left - windows[1].ShutterWidth -
		(first.Left + first.Width + first.ShutterWidth)
	start := first.Left - first.ShutterWidth
	end := facade.Width - (last.Left + last.Width + last.ShutterWidth)
	if start > gap+closeEnough || end > gap+closeEnough {
		t.Errorf("windows start %v m from the left end and stop %v m from "+
			"the right, with %v m between them", start, end, gap)
	}
}

// longFacade is a face four hundred metres long of one storey of the
// culture's windows, made at the demo level. The culture draws a few
// windows for a bay, which a face this long spreads over its whole length
// with none near either end, so the storey is the first window it draws
// repeated every longWindowPitch along the face.
func longFacade(t *testing.T) facade {
	t.Helper()
	facade := drawnLongFacade(t)
	window := facade.Storeys[0][0]
	var row []facadeOpening
	end := facade.Width - window.Width
	for left := longWindowPitch; left < end; left += longWindowPitch {
		window.Left = left
		row = append(row, window)
	}
	facade.Storeys[0] = row
	return facade
}

// streetEye is an eye in the street before a long facade's left end, at
// a window's height, looking along it, with a film four thousand across.
func streetEye() strategies.Eye {
	from := maths.V(-5, 1.5, 4)
	return strategies.Eye{Place: maths.Subspace{Origin: from,
		Orientation: maths.LookAtUp(from, maths.V(400, 1.5, 0), maths.Up)},
		Wide: 1.5, Across: 4000, Down: 2000}
}

// levelledParts is a node's parts levelled by an eye and a ladder.
func levelledParts(t *testing.T, node *node, eye strategies.Eye,
	ladder string) []levelled {
	t.Helper()
	rungs, err := Ladder(ladder)
	if err != nil {
		t.Fatal(err)
	}
	leveller, err := NewLeveller(eye, rungs)
	if err != nil {
		t.Fatal(err)
	}
	pass := Pass{Leveller: leveller, Sees: eye.Sees,
		Finest: rungs[len(rungs)-1].Level}
	flat, err := pass.levelled([]placedElement{deferred(node)},
		standingAt(maths.Vec{}), pass.Finest)
	if err != nil {
		t.Fatal(err)
	}
	return flat
}

func TestANearWindowIsDemoAndAFarOneHullUnderOneLeveller(t *testing.T) {
	facade := longFacade(t)
	flat := levelledParts(t, facade.node(), streetEye(), "buildings")
	var near, far []Detail
	for _, part := range flat {
		if solid, ok := part.Element.(solid); ok && solid.Name == "wall" {
			continue
		}
		x := part.Place.Origin.X
		if x < 10 {
			near = append(near, part.Level)
		}
		if x > 300 {
			far = append(far, part.Level)
		}
	}
	if !slices.Contains(near, demoLevel) {
		t.Errorf("the near windows are %v", near)
	}
	if slices.ContainsFunc(far, func(level Detail) bool {
		return level != hullLevel
	}) {
		t.Errorf("the far windows are %v", far)
	}
}

// litAndDark are two windows of a face made at the demo level, one lit
// with a lamp behind its glass and one dark.
func litAndDark(t *testing.T) (facade, facadeOpening, facadeOpening) {
	t.Helper()
	facade := longFacade(t)
	facade.Finish.LampStrength = 1
	glass := &thing.Material{Name: "glass"}
	lit := facadeOpening{Left: 1, Sill: 1, Width: 1, Height: 1.5, Pane: glass,
		Lamp: &thing.Material{Name: "lamp", Glow: maths.Grey(1)},
		Heat: 2700}
	dark := lit
	dark.Left, dark.Lamp = 3, nil
	facade.Storeys = [][]facadeOpening{{lit, dark}}
	return facade, lit, dark
}

func TestALitWindowDrawnCoarseKeepsItsLightAndDarkGlassGoes(t *testing.T) {
	facade, lit, dark := litAndDark(t)
	parts := facade.panes([]facadeOpening{lit, dark})
	if len(parts) != 1 {
		t.Fatalf("a lit and a dark window at the hull are %d parts",
			len(parts))
	}
	pane, ok := parts[0].Element.(solid)
	if !ok || pane.Material != lit.Lamp || pane.Lamp == nil ||
		pane.Lamp.Heat != lit.Heat {
		t.Errorf("the lit window at the hull is %+v", parts[0].Element)
	}
	drawn, err := facade.whole(hullLevel)
	if err != nil {
		t.Fatal(err)
	}
	lamps := 0
	for _, part := range drawn {
		if solid, ok := part.Element.(solid); ok && solid.Lamp != nil {
			lamps++
		}
	}
	if lamps == 0 {
		t.Error("a face made at the demo level drawn whole at the hull " +
			"keeps no window's light")
	}
}

// glowingIn is how many drawn parts glow or are lamps.
func glowingIn(drawn []Rendered) int {
	count := 0
	for _, one := range drawn {
		if one.Lamp != nil || one.Material.Glow != (maths.Light{}) {
			count++
		}
	}
	return count
}

// towerGlow is how many parts of a tower glow, made for a level and
// drawn at the hull everywhere.
func towerGlow(t *testing.T, made Detail) int {
	t.Helper()
	tower := towerOf(t, 2, 0, Pins{Storeys: 20})
	design, err := tower.Design(made, 0)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Levelled(Pass{Leveller: Constant(hullLevel),
		Finest: made}, standingAt(maths.Vec{}))
	if err != nil {
		t.Fatal(err)
	}
	return glowingIn(drawn)
}

func TestATowerDrawnBareKeepsItsLitWindowsOnlyWhenMadeFiner(t *testing.T) {
	bare, kept := towerGlow(t, hullLevel), towerGlow(t, "quick")
	if !(kept > bare) {
		t.Errorf("a tower drawn bare glows in %d parts made for the "+
			"quick level, %d made for the hull", kept, bare)
	}
}

func TestAFaceMadeAtTheHullDrawsEveryPaneAsItDid(t *testing.T) {
	facade, lit, dark := litAndDark(t)
	facade.Level = hullLevel
	parts := facade.panes([]facadeOpening{lit, dark})
	if len(parts) != 2 {
		t.Fatalf("two windows on a face made at the hull are %d parts",
			len(parts))
	}
	for _, part := range parts {
		if solid, ok := part.Element.(solid); !ok ||
			solid.Material != lit.Pane || solid.Lamp != nil {
			t.Errorf("a pane at the hull is %+v", part.Element)
		}
	}
}

func TestThePartsOfOneTerraceDifferInLevel(t *testing.T) {
	design, err := terraceAtDetail("portofino", 1, nil, demoLevel)
	if err != nil {
		t.Fatal(err)
	}
	root := &node{Draw: func(Detail) ([]placedElement, error) {
		return design.Elements, nil
	}}
	levels := map[Detail]bool{}
	for _, part := range levelledParts(t, root, streetEye(), "buildings") {
		levels[part.Level] = true
	}
	if len(levels) < 2 {
		t.Errorf("a terrace seen from its end is all at %v", levels)
	}
}

func TestUnfurlingTwiceGivesTheSameParts(t *testing.T) {
	design, err := terraceAtDetail("portofino", 2, nil, demoLevel)
	if err != nil {
		t.Fatal(err)
	}
	eye := streetEye()
	rungs, err := Ladder("buildings")
	if err != nil {
		t.Fatal(err)
	}
	leveller, err := NewLeveller(eye, rungs)
	if err != nil {
		t.Fatal(err)
	}
	pass := Pass{Leveller: leveller, Sees: eye.Sees, Finest: demoLevel}
	first, err := design.Levelled(pass, standingAt(maths.Vec{}))
	if err != nil {
		t.Fatal(err)
	}
	again, err := design.Levelled(pass, standingAt(maths.Vec{}))
	if err != nil {
		t.Fatal(err)
	}
	if digestOf(t, first, nil) != digestOf(t, again, nil) {
		t.Error("a terrace levelled twice is drawn two ways")
	}
}

// trianglesOf is how many triangles drawn parts hold.
func trianglesOf(drawn []Rendered) int {
	count := 0
	for _, one := range drawn {
		count += len(one.Mesh.Triangles)
	}
	return count
}

// terracesSeen is the triangles of a number of terraces in a row along
// the street, each levelled by the street eye, standing so far off.
func terracesSeen(t *testing.T, count int, off float64) int {
	t.Helper()
	design, err := terraceAtDetail("portofino", 1, nil, demoLevel)
	if err != nil {
		t.Fatal(err)
	}
	eye := streetEye()
	rungs, err := Ladder("buildings")
	if err != nil {
		t.Fatal(err)
	}
	leveller, err := NewLeveller(eye, rungs)
	if err != nil {
		t.Fatal(err)
	}
	pass := Pass{Leveller: leveller, Sees: eye.Sees, Finest: demoLevel}
	total := 0
	for index := range count {
		offset := maths.V(off+float64(index)*100, 0, -off)
		place := standingAt(offset)
		drawn, err := design.Levelled(pass, place)
		if err != nil {
			t.Fatal(err)
		}
		total += trianglesOf(drawn)
	}
	return total
}

func TestTrianglesGrowWithClosenessAndNotWithCount(t *testing.T) {
	near := terracesSeen(t, 1, 0)
	far := terracesSeen(t, 10, 3000)
	if !(far < near) {
		t.Errorf("ten terraces far off are %d triangles, one near %d",
			far, near)
	}
}

// towerSeen is the triangles of a tower of sixty storeys drawn by a pass.
func towerSeen(t *testing.T, pass Pass) int {
	t.Helper()
	tower := towerOf(t, 1, 0, Pins{Storeys: 60})
	design, err := tower.Design(pass.Finest, 0)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Levelled(pass, standingAt(maths.Vec{}))
	if err != nil {
		t.Fatal(err)
	}
	return trianglesOf(drawn)
}

func TestATowerSeenFromItsFootIsDressedBelowAndBareAbove(t *testing.T) {
	from := maths.V(30, 2, 30)
	eye := strategies.Eye{Place: maths.Subspace{Origin: from,
		Orientation: maths.LookAtUp(from, maths.V(0, 60, 0), maths.Up)},
		Wide: 1.5, Across: 100, Down: 100}
	rungs, err := Ladder("buildings")
	if err != nil {
		t.Fatal(err)
	}
	leveller, err := NewLeveller(eye, rungs)
	if err != nil {
		t.Fatal(err)
	}
	seen := towerSeen(t, Pass{Leveller: leveller, Sees: eye.Sees,
		Finest: "quick"})
	bare := towerSeen(t, Pass{Leveller: Constant(hullLevel),
		Finest: "quick"})
	dressed := towerSeen(t, Pass{Leveller: Constant("quick"),
		Finest: "quick"})
	if !(bare < seen && seen < dressed) {
		t.Errorf("a tower from its foot is %d triangles, bare %d, "+
			"dressed %d", seen, bare, dressed)
	}
}

func TestAHandedRunOfNodesIsTheRunOfTheirParts(t *testing.T) {
	part := put("box", Box{1, 2, 3}, &thing.Material{Name: "box"},
		maths.V(0.25, 0, 0.5))
	plain := repeat{Count: 3, Spacing: 4, Axis: maths.XAxis, Handed: true,
		Unit: func(int) ([]placedElement, error) { return []placedElement{part}, nil }}
	nodes := plain
	nodes.Unit = func(int) ([]placedElement, error) {
		node := &node{Bounds: maths.Box{Max: maths.V(4, 2, 4)},
			Draw: func(Detail) ([]placedElement, error) {
				return []placedElement{part}, nil
			}}
		return []placedElement{deferred(node)}, nil
	}
	var digests []string
	for _, run := range []repeat{plain, nodes} {
		parts, err := run.parts()
		if err != nil {
			t.Fatal(err)
		}
		design := Design{Name: "run", Elements: parts}
		drawn, err := design.Rendered("quick")
		if err != nil {
			t.Fatal(err)
		}
		digests = append(digests, digestOf(t, drawn, nil))
	}
	if digests[0] != digests[1] {
		t.Error("a handed run of nodes is not the run of their parts")
	}
}
