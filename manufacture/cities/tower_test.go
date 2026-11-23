package cities

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// yes and no are pins to hand to a tower.
var (
	yes, no = true, false
)

// towerOf is a tower of a seed at an irregularity, with pins, or the
// test's end.
func towerOf(t *testing.T, seed Seed, irregularity float64,
	pins Pins) *Tower {
	t.Helper()
	tower, err := NewTower(seed, Irregularity{Value: irregularity}, pins)
	if err != nil {
		t.Fatalf("seed %d at %v: %v", seed, irregularity, err)
	}
	return tower
}

// drawnTower is a tower drawn at a level and a time, or the test's end.
func drawnTower(t *testing.T, tower *Tower, level Detail,
	seconds float64) []Rendered {
	t.Helper()
	design, err := tower.Design(level, seconds)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Rendered(level)
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

// named are the drawn parts with a mesh of a name.
func named(drawn []Rendered, name string) []Rendered {
	var found []Rendered
	for _, one := range drawn {
		if one.Mesh.Name == name {
			found = append(found, one)
		}
	}
	return found
}

// towerNumber is a fixed number of the tower data.
func towerNumber(t *testing.T, name string) float64 {
	t.Helper()
	return drawerOf(t, towerCulture, 1).fixed(name)
}

// edgeLengths are the lengths of the edges of the one part of a plan.
func edgeLengths(t *testing.T, shape footprintShape) []float64 {
	t.Helper()
	parts, err := shape.parts()
	if err != nil || len(parts) != 1 {
		t.Fatalf("%d parts, %v", len(parts), err)
	}
	var lengths []float64
	for index, corner := range parts[0] {
		next := parts[0][(index+1)%len(parts[0])]
		lengths = append(lengths, math.Hypot(next.X-corner.X,
			next.Z-corner.Z))
	}
	return lengths
}

// plantStoreys is how many of a shaft's storeys are plant floors.
func plantStoreys(t *testing.T, storeys int) int {
	t.Helper()
	every := tower1Every(t)
	count := 0
	for storey := range storeys {
		if (storey+1)%every == 0 && storey+1 < storeys {
			count++
		}
	}
	return count
}

// tower1Every is the mechanical interval of the tower of seed one.
func tower1Every(t *testing.T) int {
	t.Helper()
	return drawerOf(t, towerCulture, 1).count("mechanical-every", place{})
}

func TestAPlainBlockHasOneWindowPerModuleAcrossEachFaceAndOnePerStoreyUp(
	t *testing.T) {
	tower := towerOf(t, 1, 0, Pins{})
	shaft := tower.Shafts[0]
	if len(shaft.Segments) != 1 || tower.Plan != planRectangle {
		t.Fatalf("a plain block is %d segments of %s", len(shaft.Segments),
			tower.Plan)
	}
	drawn := drawnTower(t, tower, "quick", 0)
	module, margin := towerNumber(t, "window-module"),
		towerNumber(t, "edge-margin")
	want := 0
	for _, length := range edgeLengths(t, shaft.Segments[0].Bottom) {
		want += int(math.Floor((length - 2*margin) / module))
	}
	faces := 4
	if got := len(named(drawn, "mullion")) - faces; got != want {
		t.Errorf("%d windows across a storey, want %d by the module", got,
			want)
	}
	plant := plantStoreys(t, shaft.Storeys)
	if got := len(named(drawn, "spandrel")); got !=
		faces*(shaft.Storeys-plant) {
		t.Errorf("%d spandrels, want %d faces of %d storeys", got, faces,
			shaft.Storeys-plant)
	}
}

// leanSlack is how far a pane on a leaning face may reach past its storey
// in height, in metres: the thickness of the pane tilted with the face.
const leanSlack = 0.05

func TestMechanicalFloorsAreLouvresAndAreNeverLit(t *testing.T) {
	sawPlant, sawLit := false, false
	for seed := range Seed(6) {
		drawn := drawnTower(t, towerOf(t, seed, 0.3, Pins{}), "quick", 0)
		bands := named(drawn, "plant")
		sawPlant = sawPlant || len(bands) > 0
		for _, pane := range named(drawn, "pane") {
			sawLit = sawLit || pane.Material.Glow != maths.Light{}
			low, high := reach(pane)
			for _, band := range bands {
				bandLow, bandHigh := reach(band)
				if low.Y < bandHigh.Y-leanSlack && high.Y > bandLow.Y+
					leanSlack {
					t.Fatalf("seed %d: a lit pane at %v to %v is on a plant "+
						"floor at %v to %v", seed, low.Y, high.Y,
						bandLow.Y, bandHigh.Y)
				}
			}
		}
		if len(named(drawn, "louvre")) == 0 {
			t.Errorf("seed %d: no louvres", seed)
		}
	}
	if !sawPlant || !sawLit {
		t.Errorf("plant floors seen %v, lit panes seen %v", sawPlant,
			sawLit)
	}
}

// inside is whether a corner is inside a convex polygon, or on its edge.
func inside(corner outlineCorner, polygon []outlineCorner) bool {
	sign := math.Copysign(1, signedArea(polygon))
	for index, from := range polygon {
		to := polygon[(index+1)%len(polygon)]
		cross := (to.X-from.X)*(corner.Z-from.Z) -
			(to.Z-from.Z)*(corner.X-from.X)
		if sign*cross < -1e-6 {
			return false
		}
	}
	return true
}

func TestASetbackOutlineIsInsideTheOneBelowIt(t *testing.T) {
	steps := 0
	for seed := range Seed(12) {
		tower := towerOf(t, seed, 1, Pins{Plan: planRectangle, Setbacks: &yes,
			Twist: &no})
		shaft := tower.Shafts[0]
		for index := 1; index < len(shaft.Segments); index++ {
			_, below, err := shaft.layer(index - 1)
			if err != nil {
				t.Fatal(err)
			}
			above, _, err := shaft.layer(index)
			if err != nil {
				t.Fatal(err)
			}
			steps++
			for _, corner := range above[0] {
				if !inside(corner, below[0]) {
					t.Fatalf("seed %d: segment %d pokes out of the one "+
						"below at %v", seed, index, corner)
				}
			}
		}
	}
	if steps == 0 {
		t.Error("no tower had a setback to check")
	}
}

func TestWithoutSetbacksOrTaperEachSegmentStandsOnTheLastOnesTop(t *testing.T) {
	tower := towerOf(t, 3, 1, Pins{Plan: planRectangle, Setbacks: &no,
		Taper: &no, Stack: &no, Nest: &no})
	shaft := tower.Shafts[0]
	for index := 1; index < len(shaft.Segments); index++ {
		if shaft.Segments[index].Bottom != shaft.Segments[index-1].Top ||
			shaft.Segments[index].Top != shaft.Segments[index].Bottom {
			t.Errorf("segment %d is not a straight continuation", index)
		}
	}
}

func TestTheShaftsOfATowerDoNotOverlap(t *testing.T) {
	for seed := range Seed(10) {
		tower := towerOf(t, seed, 1, Pins{Shafts: 3})
		var lows, highs []float64
		for _, shaft := range tower.Shafts {
			low, high := math.Inf(1), math.Inf(-1)
			for index := range shaft.Segments {
				bottom, _, err := shaft.layer(index)
				if err != nil {
					t.Fatal(err)
				}
				for _, part := range bottom {
					for _, corner := range part {
						low, high = math.Min(low, corner.X),
							math.Max(high, corner.X)
					}
				}
			}
			lows, highs = append(lows, low), append(highs, high)
		}
		for index := 1; index < len(lows); index++ {
			if highs[index-1] >= lows[index] {
				t.Errorf("seed %d: shaft %d reaches to %v and the next "+
					"starts at %v", seed, index-1, highs[index-1],
					lows[index])
			}
		}
	}
}

func TestAnOverlappingPairOfShaftsIsRefused(t *testing.T) {
	tower := towerOf(t, 1, 1, Pins{Shafts: 2})
	tower.Shafts[1].Centre = tower.Shafts[0].Centre
	if tower.check() == nil {
		t.Error("shafts on one centre were accepted")
	}
}

// xRange is the lowest and highest x of a drawn part.
func xRange(one Rendered) (float64, float64) {
	low, high := reach(one)
	return low.X, high.X
}

func TestASkybridgeTouchesBothShafts(t *testing.T) {
	for seed := range Seed(8) {
		tower := towerOf(t, seed, 1, Pins{Shafts: 2, Span: SpanSkybridge})
		drawn := drawnTower(t, tower, "hull", 0)
		bridges := named(drawn, "bridge")
		if len(bridges) != 1 {
			t.Fatalf("seed %d: %d bridges between two shafts", seed,
				len(bridges))
		}
		low, high := xRange(bridges[0])
		at := tower.baseHeight() + float64(tower.SpanStorey)*
			tower.storeyHeight()
		_, firstHigh, ok := tower.Shafts[0].extent(at, tower.baseHeight(),
			tower.storeyHeight())
		secondLow, _, other := tower.Shafts[1].extent(at,
			tower.baseHeight(), tower.storeyHeight())
		if !ok || !other || low > firstHigh || high < secondLow {
			t.Errorf("seed %d: a bridge from %v to %v, shafts at %v and %v",
				seed, low, high, firstHigh, secondLow)
		}
	}
}

func TestSpanPinsAreKeptAndAreNothingForOneShaft(t *testing.T) {
	deck := drawnTower(t, towerOf(t, 2, 1, Pins{Shafts: 3, Span: SpanDeck}),
		"hull", 0)
	if len(named(deck, "deck")) != 1 || len(named(deck, "bridge")) != 0 {
		t.Error("a pinned deck did not make one deck and no bridge")
	}
	none := drawnTower(t, towerOf(t, 2, 1, Pins{Shafts: 3,
		Span: SpanNone}), "hull", 0)
	if len(named(none, "deck")) != 0 || len(named(none, "bridge")) != 0 {
		t.Error("shafts pinned to no span were joined")
	}
	single := towerOf(t, 2, 1, Pins{Shafts: 1, Span: SpanDeck})
	if single.Span != SpanNone {
		t.Errorf("one shaft has the span %s", single.Span)
	}
}

func TestBeaconsStandOnlyOverTheCulturesHeight(t *testing.T) {
	tower := towerOf(t, 1, 0, Pins{})
	looks, err := tower.looks()
	if err != nil {
		t.Fatal(err)
	}
	least := towerNumber(t, "beacon-height")
	at := []maths.Vec{maths.V(0, 0, 0)}
	if got := tower.beacons(looks, least-1, at); len(got) != 0 {
		t.Errorf("%d beacons under the height", len(got))
	}
	if got := tower.beacons(looks, least, at); len(got) != 1 {
		t.Errorf("%d beacons at the height, want 1", len(got))
	}
}

func TestEveryTowerHasABeaconExactlyWhenItsTopIsOverTheHeight(t *testing.T) {
	least := towerNumber(t, "beacon-height")
	for seed := range Seed(30) {
		drawn := drawnTower(t, towerOf(t, seed, 0, Pins{}), "hull", 0)
		top := 0.0
		for _, one := range drawn {
			if _, high := reach(one); one.Mesh.Name != "beacon" {
				top = math.Max(top, high.Y)
			}
		}
		if got := len(named(drawn, "beacon")) > 0; got != (top >= least) {
			t.Errorf("seed %d: top %v, beacons %v, height %v", seed, top,
				got, least)
		}
	}
}

func TestACrownsLightTravelsAndIsTheSameAtTheSameTime(t *testing.T) {
	const panels, period, band = 24, 20.0, 3.0
	if crownLight(0, panels, 0, period, band) != 1 {
		t.Error("the band does not start on the first panel")
	}
	later := period / panels * 6
	if crownLight(6, panels, later, period, band) != 1 ||
		crownLight(0, panels, later, period, band) != 0 {
		t.Error("the band did not travel six panels in six steps")
	}
	for panel := range panels {
		if crownLight(panel, panels, 7.5, period, band) !=
			crownLight(panel, panels, 7.5, period, band) ||
			math.Abs(crownLight(panel, panels, 7.5, period, band)-
				crownLight(panel, panels, 7.5+period, period, band)) > 1e-9 {
			t.Errorf("panel %d is not the same at the same time", panel)
		}
		if got := crownLight(panel, panels, 3, period, band); got < 0 ||
			got > 1 {
			t.Errorf("panel %d is %v bright", panel, got)
		}
	}
	if crownLight(0, 0, 0, period, band) != 0 ||
		crownLight(0, panels, 0, 0, band) != 0 {
		t.Error("a crown of no panels or a band of no period glowed")
	}
}

// glowNames are the names of the crown materials of a drawing, sorted.
func glowNames(drawn []Rendered) string {
	var names []string
	for _, one := range named(drawn, "crown panel") {
		names = append(names, one.Material.Name)
	}
	return strings.Join(names, ",")
}

func TestALitCrownChangesWithTimeAndAnUnlitOneDoesNot(t *testing.T) {
	lit := towerOf(t, 4, 0, Pins{Spire: &no, LitCrown: &yes})
	first := glowNames(drawnTower(t, lit, "hull", 0))
	again := glowNames(drawnTower(t, lit, "hull", 0))
	later := glowNames(drawnTower(t, lit, "hull", 7))
	if first != again || first == later {
		t.Errorf("a lit crown was the same at times 0 and 7, or differed "+
			"at time 0:\n%s\n%s", first, later)
	}
	dark := towerOf(t, 4, 0, Pins{Spire: &no, LitCrown: &no})
	if glowNames(drawnTower(t, dark, "hull", 0)) !=
		glowNames(drawnTower(t, dark, "hull", 7)) {
		t.Error("an unlit crown changed with time")
	}
}

func TestAtNoIrregularityEveryTowerIsAPlainBlock(t *testing.T) {
	for seed := range Seed(40) {
		tower := towerOf(t, seed, 0, Pins{})
		shaft := tower.Shafts[0]
		if tower.Plan != planRectangle || len(tower.Shafts) != 1 ||
			len(shaft.Segments) != 1 || tower.Span != SpanNone ||
			shaft.Segments[0].Top != shaft.Segments[0].Bottom ||
			shaft.Segments[0].Turn != 0 || shaft.Spire || shaft.Deck ||
			shaft.Penthouse || !shaft.Crown {
			t.Errorf("seed %d is not a plain block: %+v", seed, tower.Plan)
		}
	}
}

func TestAtFullIrregularityTheGeneratorReachesEveryFamily(t *testing.T) {
	plans, shafts, tapered, twisted, spired, lit := map[Plan]bool{}, 0, 0,
		0, 0, 0
	spans := map[SpanKind]bool{}
	for seed := range Seed(120) {
		tower := towerOf(t, seed, 1, Pins{})
		plans[tower.Plan] = true
		spans[tower.Span] = true
		shafts = max(shafts, len(tower.Shafts))
		for _, shaft := range tower.Shafts {
			for _, segment := range shaft.Segments {
				tapered += btoi(segment.Top != segment.Bottom)
				twisted += btoi(segment.Turn != 0)
			}
			spired += btoi(shaft.Spire)
			lit += btoi(shaft.LitCrown)
		}
	}
	if len(plans) < 5 || shafts < 3 || tapered == 0 || twisted == 0 ||
		spired == 0 || lit == 0 || !spans[SpanSkybridge] || !spans[SpanDeck] {
		t.Errorf("plans %v, most shafts %d, tapers %d, twists %d, spires "+
			"%d, lit crowns %d, spans %v", plans, shafts, tapered, twisted,
			spired, lit, spans)
	}
}

// btoi is one for true and nought for false.
func btoi(is bool) int {
	if is {
		return 1
	}
	return 0
}

func TestTheSkylinesIrregularityIsMostlyLowAndSometimesHigh(t *testing.T) {
	low, high := 0, 0
	const towers = 300
	for seed := range Seed(towers) {
		tower, err := NewTower(seed, Irregularity{Drawn: true}, Pins{})
		if err != nil {
			t.Fatal(err)
		}
		low += btoi(tower.Irregularity < 0.25)
		high += btoi(tower.Irregularity > 0.6)
	}
	if low < towers/2 || high == 0 || high > towers/4 {
		t.Errorf("of %d towers %d were under 0.25 and %d over 0.6", towers,
			low, high)
	}
}

func TestPinsOverrideTheDraw(t *testing.T) {
	for seed := range Seed(6) {
		tower := towerOf(t, seed, 0, Pins{Plan: planStar, Shafts: 2,
			Taper: &yes})
		if tower.Plan != planStar || len(tower.Shafts) != 2 {
			t.Errorf("seed %d: plan %s, %d shafts", seed, tower.Plan,
				len(tower.Shafts))
		}
		segment := tower.Shafts[0].Segments[0]
		if !(segment.Top.Width < segment.Bottom.Width) {
			t.Errorf("seed %d: a pinned taper did not taper", seed)
		}
	}
}

func TestATowerThatCannotBeDrawnIsRefused(t *testing.T) {
	flawed := map[string]struct {
		irregularity Irregularity
		pins         Pins
	}{
		"irregularity over one":     {Irregularity{Value: 1.5}, Pins{}},
		"irregularity under nought": {Irregularity{Value: -0.1}, Pins{}},
		"unknown plan":              {Irregularity{}, Pins{Plan: "dome"}},
		"too many shafts":           {Irregularity{}, Pins{Shafts: 99}},
		"negative shafts":           {Irregularity{}, Pins{Shafts: -1}},
	}
	for name, flaw := range flawed {
		if _, err := NewTower(1, flaw.irregularity, flaw.pins); err == nil {
			t.Errorf("%s: a tower was drawn", name)
		}
	}
}

func TestTheSameSeedIsTheSameTowerAndAnotherSeedAnother(t *testing.T) {
	digestOf := func(seed Seed) string {
		tower := towerOf(t, seed, 0.6, Pins{})
		return digest(drawnTower(t, tower, "hull", 0))
	}
	first, again := digestOf(5), digestOf(5)
	if first != again {
		t.Error("one seed made two towers")
	}
	if first == digestOf(6) {
		t.Error("two seeds made the same tower")
	}
}

func TestTheHullIsOnlyTheMassingAndTheQuickLevelHasTheFaces(t *testing.T) {
	tower := towerOf(t, 2, 0.4, Pins{})
	hull := drawnTower(t, tower, "hull", 0)
	quick := drawnTower(t, tower, "quick", 0)
	if len(named(hull, "mullion")) != 0 || len(named(quick, "mullion")) == 0 ||
		len(quick) <= len(hull) {
		t.Errorf("hull has %d parts, quick %d", len(hull), len(quick))
	}
}

func TestATallerTowerNeedsAMoreSpaciousCore(t *testing.T) {
	previous := 0.0
	for _, storeys := range []int{10, 30, 60, 120} {
		side, err := coreSide(storeys)
		if err != nil || !(side > previous) {
			t.Errorf("%d storeys give a core of %v after %v, %v", storeys,
				side, previous, err)
		}
		previous = side
	}
}

func TestEveryPlanHasConvexPartsThatAScalingKeepsCornerForCorner(t *testing.T) {
	shape := footprintShape{Width: 40, Depth: 30, Round: 5, Bow: 4, Sides: 6, Steps: 3,
		Lobe: 6, Wings: [wingCount]float64{20, 18, 16}, WingWidth: 12}
	for _, plan := range PlanNames() {
		shape.Plan = plan
		parts, err := shape.parts()
		if err != nil {
			t.Fatalf("%s: %v", plan, err)
		}
		smaller, err := shape.scaled(0.5).parts()
		if err != nil || len(smaller) != len(parts) {
			t.Fatalf("%s: %d parts then %d, %v", plan, len(parts),
				len(smaller), err)
		}
		for index, part := range parts {
			if !convex(part) || signedArea(part) >= 0 ||
				len(smaller[index]) != len(part) {
				t.Errorf("%s part %d is not convex, wound as the plans are, "+
					"or its scaling has other corners", plan, index)
			}
		}
	}
	if _, err := (footprintShape{Plan: "dome", Width: 1, Depth: 1}).parts(); err == nil {
		t.Error("a plan that is not one was drawn")
	}
}

func TestAStarIsTwoSquaresAndAYIsThreeWings(t *testing.T) {
	star, err := (footprintShape{Plan: planStar, Width: 30}).parts()
	if err != nil || len(star) != 2 {
		t.Fatalf("a star is %d parts, %v", len(star), err)
	}
	points := map[string]bool{}
	for _, part := range star {
		for _, corner := range part {
			points[fmt.Sprintf("%.3f,%.3f", corner.X, corner.Z)] = true
		}
	}
	if len(points) != 8 {
		t.Errorf("a star has %d distinct corners, want eight points", len(points))
	}
	y, err := (footprintShape{Plan: planY, Width: 40, WingWidth: 12,
		Wings: [wingCount]float64{20, 20, 20}}).parts()
	if err != nil || len(y) != wingCount {
		t.Fatalf("a Y is %d parts, %v", len(y), err)
	}
}

func TestAnEdgeCoveredByAnotherPartIsLeftOnlyWhatShows(t *testing.T) {
	square := boxCorners(10, 10)
	other := moved(boxCorners(10, 20), outlineCorner{X: 5})
	free := spans(outlineCorner{X: -5, Z: 5}, outlineCorner{X: 5, Z: 5}, [][]outlineCorner{other})
	if len(free) != 1 || math.Abs(free[0][0]) > 1e-9 ||
		math.Abs(free[0][1]-0.5) > 1e-9 {
		t.Errorf("the free stretch of a half covered edge is %v", free)
	}
	if got := spans(outlineCorner{X: -5, Z: 5}, outlineCorner{X: 5, Z: 5}, nil); len(got) != 1 {
		t.Errorf("an uncovered edge is %v", got)
	}
	if !slices.Equal(square, boxCorners(10, 10)) {
		t.Error("the corners of a box changed")
	}
}
