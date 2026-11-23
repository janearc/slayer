package cities

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// plainStoreys is how many storeys the plain towers of the skin tests
// have: enough to hold a plant floor, a tracery band and a few balconies.
const plainStoreys = 30

// coolestShare is the least a cool light's blue is of its red: near white,
// where a warm light's is well under half.
const coolestShare = 0.85

// plainPins pin a plain block of a skin: one straight segment, no taper,
// turn, setback or spire, so that only the skin differs.
func plainPins(skin Skin) Pins {
	return Pins{Skin: skin, Taper: &no, Twist: &no, Setbacks: &no,
		Spire: &no, Storeys: plainStoreys}
}

// skinParts are the parts each skin draws that no other skin does.
var skinParts = map[Skin][]string{
	skinMirror:     {"panel", "mullion cap", "spandrel"},
	skinConcrete:   {"slab edge", "board mark", "pier", "window"},
	skinGothic:     {"buttress", "lancet", "pinnacle", "tracery"},
	skinCantilever: {"floor plate", "cable"},
	skinVaporwave:  {"pastel band", "block bar", "glass block"},
	skinStainless:  {"fin", "stone band", "ribbon window"},
}

// partKey is where a part stands and how it is turned and sized, so that
// two parts with one key are one part drawn twice, overlapping.
func partKey(one Rendered) string {
	var sum maths.Vec
	for _, face := range one.Mesh.Triangles {
		for _, corner := range face {
			sum = sum.Add(one.Place.Out(corner))
		}
	}
	cornerCount := 3 * len(one.Mesh.Triangles)
	middle := sum.Scale(1 / float64(cornerCount))
	return fmt.Sprintf("%s %.4f %.4f", one.Mesh.Name,
		one.Place.Out(maths.Vec{}), middle)
}

// repeats are the keys of parts of the names given that stand twice.
func repeats(drawn []Rendered, names []string) []string {
	seen := map[string]bool{}
	var twice []string
	for _, one := range drawn {
		if !slices.Contains(names, one.Mesh.Name) {
			continue
		}
		key := partKey(one)
		if seen[key] {
			twice = append(twice, key)
		}
		seen[key] = true
	}
	return twice
}

// counts are how many parts there are of each name.
func counts(drawn []Rendered) map[string]int {
	found := map[string]int{}
	for _, one := range drawn {
		found[one.Mesh.Name]++
	}
	return found
}

func TestEverySkinDrawsOnAPlainTowerWithoutOverlap(t *testing.T) {
	for _, skin := range SkinNames()[1:] {
		wanted := skinParts[skin]
		if skin == skinGranite {
			continue
		}
		tower := towerOf(t, 3, 0, plainPins(skin))
		drawn := drawnTower(t, tower, "quick", 0)
		found := counts(drawn)
		for _, name := range wanted {
			if found[name] == 0 {
				t.Errorf("%s: no %q among %v", skin, name, found)
			}
		}
		if twice := repeats(drawn, wanted); len(twice) > 0 {
			t.Errorf("%s: %d parts stand twice: %v", skin, len(twice),
				twice[:1])
		}
	}
}

func TestGraniteIsPunchedOrRibbonAndEachDrawsWithoutOverlap(t *testing.T) {
	styles := map[string][]string{stylePunched: {"pier", "sill", "window"},
		styleRibbon: {"stone band", "ribbon window"}}
	for style, wanted := range styles {
		pins := plainPins(skinGranite)
		pins.Style = style
		drawn := drawnTower(t, towerOf(t, 3, 0, pins), "quick", 0)
		for _, name := range wanted {
			if counts(drawn)[name] == 0 {
				t.Errorf("%s granite: no %q", style, name)
			}
		}
		if twice := repeats(drawn, wanted); len(twice) > 0 {
			t.Errorf("%s granite: %d parts stand twice", style, len(twice))
		}
	}
	ribbon := towerOf(t, 3, 0, func() Pins {
		pins := plainPins(skinGranite)
		pins.Style = styleRibbon
		return pins
	}())
	other := counts(drawnTower(t, ribbon, "quick", 0))
	if other["pier"] != 0 {
		t.Errorf("ribbon granite has %d piers", other["pier"])
	}
}

func TestAMirrorPanelLeansOutOfTrueByAboutADegreeAndNoTwoAlike(t *testing.T) {
	tower := towerOf(t, 3, 0, plainPins(skinMirror))
	drawn := drawnTower(t, tower, "quick", 0)
	leans := map[string]bool{}
	for _, one := range named(drawn, "panel") {
		leaning := one.Place.Orientation.Out(maths.V(1, 0, 0))
		leans[fmt.Sprintf("%.5f",
			leaning)] = true
	}
	if len(leans) < 20 {
		t.Errorf("%d leans among the panels, wanted waves", len(leans))
	}
}

func TestALobedOutlineIsClosedAndInsideItsBoundingStar(t *testing.T) {
	shape := footprintShape{Plan: planLobed, Width: 40, Lobe: 6, Steps: 5}
	parts, err := shape.parts()
	if err != nil {
		t.Fatal(err)
	}
	const notches = 8
	if len(parts) != 2+notches {
		t.Fatalf("%d parts, wanted two squares and %d arcs", len(parts),
			notches)
	}
	star := shape.Width * math.Sqrt2 / 2
	for index, part := range parts {
		if len(part) < leastSides || !convex(part) || !(signedArea(part) < 0) {
			t.Errorf("part %d is not a closed convex outline wound as the "+
				"plans are: %v", index, part)
		}
		for _, corner := range part {
			if away := math.Hypot(corner.X, corner.Z); away > star+1e-9 {
				t.Errorf("part %d reaches %v, past the star's %v", index,
					away, star)
			}
		}
	}
}

func TestALobedTowerAndAnEllipticalOneAreClosedSolids(t *testing.T) {
	for _, plan := range []Plan{planLobed, planEllipse} {
		pins := plainPins(skinMirror)
		pins.Plan = plan
		drawn := drawnTower(t, towerOf(t, 3, 0, pins), "quick", 0)
		bodies := named(drawn, "body")
		if len(bodies) == 0 {
			t.Fatalf("%s: no body", plan)
		}
		for _, one := range bodies {
			if open := openEdges(one.Mesh); open > 0 {
				t.Errorf("%s: a body has %d edges not shared by two faces",
					plan, open)
			}
		}
	}
}

// openEdges is how many edges of the mesh are not shared by exactly two
// of its faces: nought for a closed solid.
func openEdges(mesh *Mesh) int {
	shared := map[[2]maths.Vec]int{}
	for _, face := range mesh.Triangles {
		for corner := range face {
			from, to := face[corner], face[(corner+1)%len(face)]
			if fmt.Sprint(from) > fmt.Sprint(to) {
				from, to = to, from
			}
			shared[[2]maths.Vec{from, to}]++
		}
	}
	var open int
	for _, count := range shared {
		if count != 2 {
			open++
		}
	}
	return open
}

func TestALobedPlanWithNoArcAndAnEllipseWithTooFewSidesAreRefused(
	t *testing.T) {
	for name, shape := range map[string]footprintShape{
		"no arc":    {Plan: planLobed, Width: 40, Steps: 5},
		"no pieces": {Plan: planLobed, Width: 40, Lobe: 6},
		"two sides": {Plan: planEllipse, Width: 40, Depth: 20, Sides: 2},
	} {
		if _, err := shape.parts(); err == nil {
			t.Errorf("%s: a plan was drawn", name)
		}
	}
}

func TestAnEllipseIsClosedAndWithinItsWidthAndDepth(t *testing.T) {
	shape := footprintShape{Plan: planEllipse, Width: 60, Depth: 30, Sides: 28}
	parts, err := shape.parts()
	if err != nil || len(parts) != 1 {
		t.Fatalf("%d parts, %v", len(parts), err)
	}
	if len(parts[0]) != 28 || !convex(parts[0]) ||
		!(signedArea(parts[0]) < 0) {
		t.Errorf("an ellipse is not 28 convex corners wound as plans are")
	}
	for _, corner := range parts[0] {
		if math.Abs(corner.X) > 30+1e-9 || math.Abs(corner.Z) > 15+1e-9 {
			t.Errorf("%v is outside the ellipse's box", corner)
		}
	}
}

func TestTheCyberLayerAddsPartsAndNeverRemovesTheSkinsOwn(t *testing.T) {
	for _, skin := range SkinNames() {
		plainPin, cyberPin := plainPins(skin), plainPins(skin)
		plainPin.Cyber, cyberPin.Cyber = &no, &yes
		plainTower := towerOf(t, 3, 0, plainPin)
		plain := counts(drawnTower(t, plainTower, "quick", 0))
		cyberTower := towerOf(t, 3, 0, cyberPin)
		cyber := counts(drawnTower(t, cyberTower, "quick", 0))
		for name, count := range plain {
			if cyber[name] < count {
				t.Errorf("%s: %d %q with the cyber layer, %d without", skin,
					cyber[name], name, count)
			}
		}
		for _, added := range []string{"corner band", "searchlight lens"} {
			if cyber[added] == 0 || plain[added] != 0 {
				t.Errorf("%s: the cyber layer's %q: %d with, %d without",
					skin, added, cyber[added], plain[added])
			}
		}
	}
}

func TestTheCyberLayerHasGreeblesAndSignsAndItsSearchlightsAreLamps(t *testing.T) {
	pins := plainPins(skinConcrete)
	pins.Cyber = &yes
	var found map[string]int
	var lamps int
	for seed := Seed(1); seed < 30 && found["sign board"] == 0; seed++ {
		drawn := drawnTower(t, towerOf(t, seed, 0, pins), "quick", 0)
		found = counts(drawn)
		for _, one := range drawn {
			if one.Lamp != nil && one.Mesh.Name == "searchlight lens" {
				lamps++
			}
		}
	}
	if found["sign board"] == 0 || found["sign cell"] == 0 {
		t.Fatalf("no signs in thirty seeds: %v", found)
	}
	if found["pipe"]+found["vent"]+found["utility box"]+found["gantry"] == 0 {
		t.Errorf("no greebles: %v", found)
	}
	if lamps == 0 || lamps != found["searchlight lens"] {
		t.Errorf("%d of %d searchlight lenses are lamps", lamps,
			found["searchlight lens"])
	}
}

// signPattern is where the lit cells of a tower's signs and screens are.
func signPattern(drawn []Rendered) string {
	var cells []string
	for _, one := range drawn {
		if strings.HasSuffix(one.Mesh.Name, " cell") {
			cells = append(cells, partKey(one))
		}
	}
	return strings.Join(cells, "|")
}

func TestASignsPatternChangesWithTheTimeAndHoldsForTheSameTime(t *testing.T) {
	pins := plainPins(skinConcrete)
	pins.Cyber = &yes
	var tower *Tower
	for seed := Seed(1); seed < 30 && tower == nil; seed++ {
		candidate := towerOf(t, seed, 0, pins)
		if counts(drawnTower(t, candidate, "quick", 0))["sign cell"] > 0 {
			tower = candidate
		}
	}
	if tower == nil {
		t.Fatal("no tower with a sign in thirty seeds")
	}
	period := towerNumber(t, "cyber-sign-period")
	first := signPattern(drawnTower(t, tower, "quick", 0))
	again := signPattern(drawnTower(t, tower, "quick", period/2))
	later := signPattern(drawnTower(t, tower, "quick", 3*period))
	if first != again {
		t.Errorf("the pattern moved within one step of the time")
	}
	if first == later {
		t.Errorf("the pattern is the same three steps on")
	}
}

// fingerprint is what a drawn tower is, part by part, as one string.
func fingerprint(drawn []Rendered) string {
	keys := make([]string, len(drawn))
	for index, one := range drawn {
		keys[index] = partKey(one) + " " + one.Material.Name
	}
	return strings.Join(keys, "|")
}

func TestTheSameSeedMakesTheSameTowerAndOthersMakeOthers(t *testing.T) {
	seen := map[Skin]bool{}
	prints := map[string]bool{}
	const seeds = 8
	for seed := Seed(1); seed <= seeds; seed++ {
		one, other := towerOf(t, seed, 1, Pins{Room: 130}),
			towerOf(t, seed, 1, Pins{Room: 130})
		if one.Skin != other.Skin || one.Cyber != other.Cyber ||
			one.Planted != other.Planted {
			t.Fatalf("seed %d dressed two ways", seed)
		}
		first := fingerprint(drawnTower(t, one, "quick", 0))
		if first != fingerprint(drawnTower(t, other, "quick", 0)) {
			t.Errorf("seed %d drew two towers", seed)
		}
		seen[one.Skin], prints[first] = true, true
	}
	if len(seen) < 3 || len(prints) < seeds {
		t.Errorf("%d skins and %d towers in %d seeds", len(seen),
			len(prints), seeds)
	}
}

// litMaterials are the materials of the lit panes of a tower, by name.
func litMaterials(drawn []Rendered) map[string]maths.Light {
	found := map[string]maths.Light{}
	for _, one := range named(drawn, "pane") {
		found[one.Material.Name] = one.Material.Glow
	}
	return found
}

func TestNightLightingDiffersByUseOfficesCoolAndEvenHomesWarmAndCurtained(
	t *testing.T) {
	office, home := plainPins(skinCurtain), plainPins(skinCurtain)
	office.Use, home.Use = useOffice, useHome
	offices := drawnTower(t, towerOf(t, 3, 0, office), "quick", 0)
	homes := drawnTower(t, towerOf(t, 3, 0, home), "quick", 0)
	cool, warm := litMaterials(offices), litMaterials(homes)
	if len(cool) != 1 {
		t.Errorf("offices are lit %d ways: %v", len(cool), cool)
	}
	for name, glow := range cool {
		if !(glow.B/glow.R > coolestShare) {
			t.Errorf("an office light, %s, is not cool: %v", name, glow)
		}
	}
	if len(warm) < 2 {
		t.Errorf("homes are lit %d ways, wanted uneven: %v", len(warm), warm)
	}
	for name, glow := range warm {
		if !(glow.R > glow.B) {
			t.Errorf("a home light, %s, is not warm: %v", name, glow)
		}
	}
	if len(named(offices, "curtain")) != 0 || len(named(homes, "curtain")) == 0 {
		officeCurtains := len(named(offices, "curtain"))
		homeCurtains := len(named(homes, "curtain"))
		t.Errorf("curtains: %d in offices, %d in homes",
			officeCurtains, homeCurtains)
	}
	coolKeys := slices.Sorted(maps.Keys(cool))
	warmKeys := slices.Sorted(maps.Keys(warm))
	if slices.Equal(coolKeys,
		warmKeys) {
		t.Errorf("offices and homes are lit alike")
	}
}

func TestAnOfficeFloorIsLitAcrossOrNotAtAll(t *testing.T) {
	pins := plainPins(skinCurtain)
	pins.Use = useOffice
	tower := towerOf(t, 3, 0, pins)
	storey := tower.storeyHeight()
	floors := map[int]int{}
	for _, one := range named(drawnTower(t, tower, "quick", 0), "pane") {
		floors[int(math.Floor(one.Place.Origin.Y/storey))]++
	}
	if len(floors) < 3 || len(floors) > plainStoreys-1 {
		t.Errorf("%d floors lit of %d", len(floors), plainStoreys)
	}
	for floor, panes := range floors {
		for _, other := range floors {
			if panes != other {
				t.Fatalf("floor %d has %d lit windows and another %d: an "+
					"office floor is lit across", floor, panes, other)
			}
		}
	}
}

func TestAPlantFloorIsNeverLitInAnyUse(t *testing.T) {
	for _, use := range UseNames() {
		pins := plainPins(skinCurtain)
		pins.Use = use
		tower := towerOf(t, 3, 0, pins)
		storey := tower.storeyHeight()
		drawn := drawnTower(t, tower, "quick", 0)
		slabs := named(drawn, "plant")
		if len(slabs) == 0 {
			t.Fatalf("%s: no plant floor in %d storeys", use, plainStoreys)
		}
		for _, pane := range named(drawn, "pane") {
			for _, slab := range slabs {
				low := slab.Place.Origin.Y
				if y := pane.Place.Origin.Y; y > low-storey/2 &&
					y < low+storey/2 {
					t.Fatalf("%s: a pane at %v is on the plant floor at %v",
						use, y, low)
				}
			}
		}
	}
}

func TestATowerWhoseFloorsTurnAsItRisesTurnsMoreEachBlock(t *testing.T) {
	pins := Pins{Spiral: &yes, Storeys: 60, Width: 24, Skin: skinMirror}
	tower := towerOf(t, 3, 0, pins)
	segments := tower.Shafts[0].Segments
	if len(segments) < 10 {
		t.Fatalf("%d segments in 60 storeys", len(segments))
	}
	var storeys int
	for index, segment := range segments {
		storeys += segment.Storeys
		if !(segment.Turn > 0) || segment.Bottom.Width != 24 {
			t.Errorf("segment %d turns %v at width %v", index, segment.Turn,
				segment.Bottom.Width)
		}
	}
	if storeys != 60 {
		t.Errorf("%d storeys in the blocks, wanted 60", storeys)
	}
}

func TestGothicFiguresStandOnlyOnTheLedgesOfSetbacks(t *testing.T) {
	plainTower := towerOf(t, 3, 0, plainPins(skinGothic))
	plain := counts(drawnTower(t, plainTower,
		"quick", 0))
	stepped := plainPins(skinGothic)
	stepped.Setbacks, stepped.Segments = &yes, 3
	steppedTower := towerOf(t, 3, 0, stepped)
	with := counts(drawnTower(t, steppedTower, "quick", 0))
	if plain["figure"] != 0 || plain["pinnacle"] == 0 {
		t.Errorf("a plain gothic tower has %d figures and %d pinnacles",
			plain["figure"], plain["pinnacle"])
	}
	if with["figure"] == 0 || with["plinth"] != with["figure"] {
		t.Errorf("a stepped gothic tower has %d figures on %d plinths",
			with["figure"], with["plinth"])
	}
}

func TestAPlantedTowerHoldsTreesOfTheCataloguesKindsOnItsBalconies(t *testing.T) {
	pins := plainPins(skinConcrete)
	pins.Planted = &yes
	tower := towerOf(t, 3, 0, pins)
	drawn := drawnTower(t, tower, "quick", 0)
	trees := tower.Trees()
	if len(trees) < 8 {
		t.Fatalf("%d trees in %d storeys", len(trees), plainStoreys)
	}
	kinds := namesOf(tower.drawing.options("planted-tree"))
	for _, tree := range trees {
		if !slices.Contains(kinds, tree.Kind) || !(tree.At.Y > 0) ||
			!(tree.Height > 0) {
			t.Errorf("tree %v is not of a kind in %v, above ground", tree,
				kinds)
		}
	}
	found := counts(drawn)
	for _, name := range []string{"balcony", "planter wall", "hedge",
		"climbing plant", "hanging plant"} {
		if found[name] == 0 {
			t.Errorf("no %q on a planted tower: %v", name, found)
		}
	}
	bare := towerOf(t, 3, 0, plainPins(skinConcrete))
	drawnTower(t, bare, "quick", 0)
	if len(bare.Trees()) != 0 {
		trees := len(bare.Trees())
		t.Errorf("a tower that is not planted holds %d trees",
			trees)
	}
}

func TestTheLipstickStudyIsAnEllipseInThreeRedGraniteTiersOfRibbons(
	t *testing.T) {
	tower := towerOf(t, 1, 0, Pins{Plan: planEllipse, Segments: 3,
		Setbacks: &yes, Taper: &no, Twist: &no, Skin: skinGranite,
		Style: styleRibbon, Colour: "red", Storeys: 34, Spire: &no})
	if len(tower.Shafts[0].Segments) != 3 || tower.Plan != planEllipse {
		t.Fatalf("%d segments of %s", len(tower.Shafts[0].Segments),
			tower.Plan)
	}
	last, first := tower.Shafts[0].Segments[2], tower.Shafts[0].Segments[0]
	if !(last.Bottom.radius() < first.Top.radius()) {
		t.Errorf("the tiers do not telescope")
	}
	drawn := drawnTower(t, tower, "quick", 0)
	body := named(drawn, "body")
	if len(body) != 3 || !strings.Contains(body[0].Material.Name, "red") {
		t.Errorf("%d bodies of %v", len(body), body[0].Material.Name)
	}
	if counts(drawn)["ribbon window"] == 0 {
		t.Errorf("no ribbon windows")
	}
}

func TestATowerRefusesASkinAUseAStyleOrAColourThereIsNot(t *testing.T) {
	cases := map[string]Pins{
		"no skin": {Skin: "chrome"}, "no use": {Use: "factory"},
		"no style":  {Style: "arched"},
		"no colour": {Skin: skinGranite, Colour: "chartreuse"},
		"no mirror": {Skin: skinMirror, Colour: "red"},
	}
	for name, pins := range cases {
		if _, err := NewTower(1, Irregularity{}, pins); err == nil {
			t.Errorf("%s: a tower was drawn", name)
		}
	}
	if _, err := NewTower(1, Irregularity{}, Pins{Skin: skinStainless,
		Colour: "anything"}); err != nil {
		t.Errorf("a skin with one colour reads no colour pin: %v", err)
	}
}
