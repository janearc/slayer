package session

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// standCount is how many things the stand of the test's set holds.
const standCount = 20

// standReach is how far out the test's wedge reaches, in metres.
const standReach = 50

// standSetText is a set in space of one yurt and a stand of yurts in a
// disc about it, the stand saying one more key if it is given one.
func standSetText(extra string) string {
	return `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: stands
  setting: space
  light: none
  film: {width: 1, height: 1, rays: 1}
  cameras:
    between:
      title: between
      orientation: {look_at: [0, 0, 0]}
      location: {at: [0, -90, 2]}
  weather:
    wind:
      from: {value: 270, unit: deg}
      speed: {value: 2, unit: m/s}
    grove mist:
      kind: prescribed-mist
      extinction: {value: 0.006, unit: m^-1}
      depth: {value: 20, unit: m}
      coverage: 0.4
      seed: 23
  set:
    things:
      ger:
        kind: yurt
        seed: 1
        complexity: quick
        east: 0
        north: 0
        height: {value: 0, unit: m}
    stands:
      the district:
        kind: yurt
        count: 20
        seed: 2
        complexity: quick
        within: {of: ger, from: 0, to: 360, near: 8, far: 50}
` + extra
}

// A stand in a scene in space stands its things on the flat world
// ground, at the same height a thing with height 0 stands at, inside
// its wedge.
func TestAStandInSpaceStandsOnFlatGroundAtHeightZero(t *testing.T) {
	read := readPositionSet(t, standSetText(""))
	stage := testStaging()
	if err := stage.afloat(read); err != nil {
		t.Fatal(err)
	}
	if want := 1 + standCount; stage.made.things != want {
		t.Fatalf("%d things stood, want %d", stage.made.things, want)
	}
	stood := 0
	for _, each := range stage.stood {
		if each.of != "the district" {
			continue
		}
		stood++
		if each.height != 0 {
			t.Errorf("a stand's thing stands %v up", each.height)
		}
		out := math.Hypot(each.east, each.north)
		if out < 8 || out > standReach {
			t.Errorf("a stand's thing stands %v out of the wedge", out)
		}
	}
	if stood != standCount {
		t.Errorf("%d of the stand were noted, want %d", stood, standCount)
	}
	ger := stage.places["ger"]
	if ger != (maths.Vec{}) {
		t.Errorf("the ger stands at %v", ger)
	}
}

// The same stand and seed make the same places each time.
func TestAStandInSpaceIsTheSameEachTime(t *testing.T) {
	var runs [2][]standing
	for index := range runs {
		stage := testStaging()
		positions := readPositionSet(t, standSetText(""))
		if err := stage.afloat(positions); err != nil {
			t.Fatal(err)
		}
		runs[index] = stage.stood
	}
	if len(runs[0]) != len(runs[1]) {
		t.Fatalf("%d places, then %d", len(runs[0]), len(runs[1]))
	}
	for index := range runs[0] {
		if runs[0][index] != runs[1][index] {
			t.Fatalf("place %d was %v, then %v", index, runs[0][index],
				runs[1][index])
		}
	}
}

// Anti-test: a stand in space that needs terrain for anything else is
// refused, and the refusal names the stand and what it needs.
func TestAStandInSpaceThatNeedsTerrainIsRefused(t *testing.T) {
	for name, row := range map[string]struct{ extra, says string }{
		"above": {"        above: 10\n", "says above"},
		"below": {"        below: 10\n", "says below"},
		"mist": {"        atmosphere: {weather: grove mist, extent: " +
			"canopy-and-ground, fade-at-boundary: true}\n",
			"says atmosphere"},
		"fire": {"        alight: 1\n", "says alight"},
	} {
		text := strings.Replace(standSetText(""),
			"        within:", row.extra+"        within:", 1)
		stage := testStaging()
		err := stage.afloat(readPositionSet(t, text))
		if err == nil || !strings.Contains(err.Error(), "stand the district") ||
			!strings.Contains(err.Error(), row.says) {
			t.Errorf("%s: afloat says %v, want it to name the stand and "+
				"say %q", name, err, row.says)
		}
	}
}

// Anti-test: a stand in space with no wedge has no ground to grow over
// and is refused, saying to name one.
func TestAStandInSpaceWithNoWedgeIsRefused(t *testing.T) {
	text := strings.Replace(standSetText(""),
		"        within: {of: ger, from: 0, to: 360, near: 8, far: 50}\n", "", 1)
	stage := testStaging()
	err := stage.afloat(readPositionSet(t, text))
	if err == nil || !strings.Contains(err.Error(), "say within") {
		t.Errorf("a stand with no wedge: %v", err)
	}
}
