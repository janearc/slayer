package session

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/sources"
)

// The canal of the frontage tests: its middle runs north on x nought, 14 m
// between walls half a metre thick, its water at 1.5 m under the ground.
const (
	canalHalfWidth  = 7.0
	canalWall       = 0.5
	canalWaterLevel = -1.5
	canalSetback    = 5.0
	canalCloseTo    = 1e-6
)

// canalStandsText is a scene on no terrain with a canal, its water, a row
// of terraces on its west bank and sampans moored on the water by its east
// wall. The depth of the canal, in metres, may be changed.
func canalStandsText(depth string) string {
	return `scene:
  name: canal stands
  light: sun
  air: earth
  groundless-air:
    coordinate-origin: earth-mean-radius
    reference-altitude: {value: 0, unit: m}
    boundary: earth-standard-land
  sun: {azimuth: 120, altitude: 35}
  film: {width: 1, height: 1, rays: 1}
  cameras:
    view:
      title: view
      orientation: {look_at: [0, 0, 0]}
      location: {at: [0, -150, 5]}
  world:
    unit: 1
    materials:
      water:
        clear: {through: 1, index: 1.333, gloss: 1, rank: 0}
        source: "synthetic validation"
      brick:
        colour: {srgb_linear: [0.3, 0.1, 0.05]}
        source: eye
      stone:
        colour: {srgb_linear: [0.3, 0.3, 0.3]}
        source: eye
      paving:
        colour: {srgb_linear: [0.1, 0.1, 0.1]}
        source: eye
      silt:
        colour: {srgb_linear: [0.02, 0.02, 0.01]}
        source: eye
    ocean:
      material: water
      middle: [0, 0, -1.5]
      extent: {value: 256, unit: m}
      samples: 3
      sea:
        wind: {value: 2, unit: m/s}
        fetch: {value: 100, unit: m}
        depth: {value: 3, unit: m}
        toward: {value: 0, unit: rad}
      seed: 1
      time: {value: 0, unit: s}
      source: "synthetic validation"
    channels:
      canal:
        line: [[0, -100], [0, 100]]
        width: {value: 14, unit: m}
        depth: {value: ` + depth + `, unit: m}
        top: {value: 0, unit: m}
        reach: {value: 20, unit: m}
        ground: paving
        bed: silt
        wall: {material: brick, thickness: {value: 0.5, unit: m}}
        coping:
          material: stone
          height: {value: 0.15, unit: m}
          overhang: {value: 0.1, unit: m}
        source: eye
  set:
    stands:
      houses:
        kind: terrace-groningen
        habit: frontage
        count: 4
        seed: 3
        complexity: hull
        frontage:
          of: canal
          side: left
          setback: {value: 5, unit: m}
          gap: {value: 8, unit: m}
      moored:
        kind: sampan-at-anchor
        habit: frontage
        count: 3
        seed: 5
        complexity: hull
        frontage:
          of: canal
          side: right
          setback: {value: -2, unit: m}
          gap: {value: 0, unit: m}
`
}

// canalStood is the places a canal's stands stood, by the stand's name,
// or why a stand was refused.
func canalStood(t *testing.T, text string) (map[string][]standing, error) {
	t.Helper()
	read := groundlessAirRead(t, text)
	stage := testStaging()
	stage.recipes = read
	stage.groundlessAir = true
	for _, each := range read.Stands() {
		if err := stage.groundlessStand(each); err != nil {
			return nil, err
		}
	}
	stood := map[string][]standing{}
	for _, each := range stage.stood {
		stood[each.of] = append(stood[each.of], each)
	}
	return stood, nil
}

// Terraces along a canal on a scene with no terrain stand on the flat
// ground at the scene's zero, on the west of the canal, back from the
// street the setback leaves, and boats float at the level of the water,
// inside the walls. A terrace of twelve bays is at least 60 m long, so
// the 200 m of the canal's line, short of half a gap at each end and a gap
// between, take two and no more.
func TestAFrontageStandsOnTheBankAndBoatsOnTheWater(t *testing.T) {
	stood, err := canalStood(t, canalStandsText("4.5"))
	if err != nil {
		t.Fatal(err)
	}
	if len(stood["houses"]) != 2 || len(stood["moored"]) != 3 {
		t.Fatalf("%d runs and %d boats", len(stood["houses"]),
			len(stood["moored"]))
	}
	front := canalHalfWidth + canalWall + canalSetback
	for _, house := range stood["houses"] {
		if house.height != 0 || house.east > -front {
			t.Errorf("a run stands %v up at %v e, behind the fronts at %v",
				house.height, house.east, -front)
		}
	}
	for _, boat := range stood["moored"] {
		if math.Abs(boat.height-canalWaterLevel) > canalCloseTo ||
			boat.east < canalHalfWidth-2 || boat.east > canalHalfWidth {
			t.Errorf("a boat floats at %v up, %v e", boat.height, boat.east)
		}
	}
}

// A frontage says what it laid, a line for each stand, with where each
// run of terrace stands along each reach.
func TestAFrontageSaysWhatItLaid(t *testing.T) {
	read := groundlessAirRead(t, canalStandsText("4.5"))
	stage := testStaging()
	stage.recipes = read
	stage.groundlessAir = true
	for _, each := range read.Stands() {
		if err := stage.groundlessStand(each); err != nil {
			t.Fatal(err)
		}
	}
	if len(stage.given.departures) != len(read.Stands()) {
		stands := len(read.Stands())
		t.Fatalf("%d lines for %d stands", len(stage.given.departures),
			stands)
	}
	for _, line := range stage.given.departures {
		if !strings.Contains(line, "a frontage laid") ||
			!strings.Contains(line, "reach 0 from") {
			t.Errorf("a line of what a frontage laid: %q", line)
		}
	}
}

// The same stand and seed make the same places each time.
func TestAFrontageIsTheSameEachTime(t *testing.T) {
	var runs [2]map[string][]standing
	for index := range runs {
		stood, err := canalStood(t, canalStandsText("4.5"))
		if err != nil {
			t.Fatal(err)
		}
		runs[index] = stood
	}
	if len(runs[0]) == 0 || len(runs[0]) != len(runs[1]) {
		t.Fatalf("%d stands, then %d", len(runs[0]), len(runs[1]))
	}
	for name, first := range runs[0] {
		if len(first) == 0 || len(first) != len(runs[1][name]) {
			t.Fatalf("%s laid %d, then %d", name, len(first),
				len(runs[1][name]))
		}
		for index, place := range first {
			if runs[1][name][index] != place {
				t.Fatalf("%s %d was %v, then %v", name, index, place,
					runs[1][name][index])
			}
		}
	}
}

// Anti-test: the water of a canal too shallow for a boat takes no boat
// stand of its kind, and the refusal says how deep the water is and how
// much the boat draws.
func TestABoatStandInWaterTooShallowIsRefused(t *testing.T) {
	_, err := canalStood(t, canalStandsText("1.6"))
	if err == nil || !strings.Contains(err.Error(), "0.10 m deep") {
		t.Fatalf("a boat in a hand of water: %v", err)
	}
}

// Anti-test: a frontage along a channel the world does not have is refused
// by the name it says.
func TestAFrontageAlongNoChannelIsRefused(t *testing.T) {
	text := strings.Replace(canalStandsText("4.5"), "of: canal",
		"of: moat", 1)
	_, err := canalStood(t, text)
	if err == nil || !strings.Contains(err.Error(), `"moat"`) {
		t.Fatalf("a frontage along a moat: %v", err)
	}
}

// Anti-test: a boat stand of another habit on a scene with no terrain is
// refused, since the water it floats on is a channel's.
func TestABoatStandOfAnotherHabitOnNoTerrainIsRefused(t *testing.T) {
	text := strings.Replace(canalStandsText("4.5"),
		"        habit: frontage\n        count: 3\n        seed: 5\n"+
			"        complexity: hull\n        frontage:\n          of: canal\n"+
			"          side: right\n          setback: {value: -2, unit: m}\n"+
			"          gap: {value: 0, unit: m}\n",
		"        habit: scatter\n        count: 3\n        seed: 5\n"+
			"        complexity: hull\n"+
			"        within: {of: x, from: 0, to: 360, near: 0, far: 10}\n", 1)
	// the stand is within sight of x, which the set must have an asset
	// of that name for the reader to take it
	text = strings.ReplaceAll(text, "  set:\n    stands:",
		"  set:\n    assets:\n      house:\n        name: x\n    stands:")
	// the refusal is held to words no other says: the one after it, of a
	// frontage along no channel, names the frontage too
	_, err := canalStood(t, text)
	if err == nil || !strings.Contains(err.Error(),
		"by the habit frontage, not") {
		t.Fatalf("a scattered boat stand on no terrain: %v", err)
	}
}

// A scene on groundless air stands its stands by the route a set is staged
// by: stand sends each stand to scatter, and scatter to the stand of a
// scene with no terrain, which stands the runs and the boats.
func TestGroundlessAirStandsItsStandsByTheStandRoute(t *testing.T) {
	read := groundlessAirRead(t, canalStandsText("4.5"))
	stage := testStaging()
	stage.recipes = read
	stage.groundlessAir = true
	if err := stage.stand(read); err != nil {
		t.Fatal(err)
	}
	if len(stage.stood) != 5 {
		t.Fatalf("%d places stood, want 2 runs and 3 boats", len(stage.stood))
	}
}

// Anti-test: placing a scene on groundless air no longer refuses a stand by
// the old words, which said stands need terrain. It may fail on what the
// scene lacks, a pinned shelf, and then it fails there.
func TestGroundlessAirDoesNotRefuseAStandForNeedingTerrain(t *testing.T) {
	read := groundlessAirRead(t, canalStandsText("4.5"))
	t.Setenv(sources.Placed, t.TempDir())
	stage := testStaging()
	stage.recipes = read
	err := stage.placeGroundlessAir(read, mustGroundlessAir(t, read))
	if err != nil && strings.Contains(err.Error(), "terrain-dependent") {
		t.Fatalf("a stand was refused for needing terrain: %v", err)
	}
}

// Anti-test: a setback that carries the line of fronts to the middle of the
// channel or past it is refused, naming the channel and how far its wall is.
func TestASetbackPastTheMiddleOfTheChannelIsRefused(t *testing.T) {
	text := strings.Replace(canalStandsText("4.5"),
		"setback: {value: -2, unit: m}", "setback: {value: -10, unit: m}", 1)
	_, err := canalStood(t, text)
	if err == nil || !strings.Contains(err.Error(), "channel canal") {
		t.Fatalf("a setback of -10 m on a 14 m canal: %v", err)
	}
}

// Anti-test: a channel the ocean's square does not reach has no water for a
// boat to float in, and the boat stand is refused.
func TestABoatStandBeyondTheOceanIsRefused(t *testing.T) {
	text := strings.Replace(canalStandsText("4.5"),
		"extent: {value: 256, unit: m}", "extent: {value: 64, unit: m}", 1)
	_, err := canalStood(t, text)
	if err == nil || !strings.Contains(err.Error(), "ocean's square") {
		t.Fatalf("a boat stand on a canal longer than the ocean: %v", err)
	}
}

// Anti-test: a kind that floats and is not a boat design has no draught,
// and a frontage of it is refused for that, not left to fail later.
func TestAFrontageOfAKindWithNoDraughtIsRefused(t *testing.T) {
	text := strings.Replace(canalStandsText("4.5"),
		"kind: sampan-at-anchor", "kind: alfa", 1)
	_, err := canalStood(t, text)
	if err == nil || !strings.Contains(err.Error(), "not a boat") {
		t.Fatalf("a frontage of the Alfa: %v", err)
	}
}

// Anti-test: a stand on a scene with no terrain that needs terrain heights
// is refused and says what it needs.
func TestAStandOnNoTerrainThatNeedsHeightsIsRefused(t *testing.T) {
	text := strings.Replace(canalStandsText("4.5"),
		"        count: 4\n", "        count: 4\n        above: 10\n", 1)
	_, err := canalStood(t, text)
	if err == nil || !strings.Contains(err.Error(), "says above") {
		t.Fatalf("a stand above 10 m on no terrain: %v", err)
	}
}
