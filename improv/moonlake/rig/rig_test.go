package rig

import (
	"math"
	"testing"
)

// near is how close two lengths must be, in metres or as a share of a
// unit way. Picked by the test's author, fable-team, 2026-11-12: the
// set file gives places to a tenth of a metre.
const near = 0.1

// show is moonlake's layout as the film ran it: the eye 71.7 m over the
// lake and the decks 20.4 m up (the 4k recipe's overnight.sh).
var show = NewLayout(71.7, 20.4)

// apart is how far two points are.
func apart(one, other Point) float64 {
	gap := one.Add(other.Scale(-1))
	return math.Sqrt(gap.dot(gap))
}

// TestToward: north, east and up are where a compass says.
func TestToward(t *testing.T) {
	for _, each := range []struct {
		azimuth, altitude float64
		want              Point
	}{{0, 0, Point{0, 1, 0}}, {90, 0, Point{1, 0, 0}},
		{0, 90, Point{0, 0, 1}}, {180, 0, Point{0, -1, 0}}} {
		if got := Toward(each.azimuth, each.altitude); apart(got,
			each.want) > 1e-12 {
			t.Errorf("%v %v is %v", each.azimuth, each.altitude, got)
		}
	}
	if got := bearing(Point{}, Point{1, 1, 0}); math.Abs(got-45) > 1e-12 {
		t.Errorf("north east is %v", got)
	}
}

// TestHeadsStandOnTheTowers: each head stands over its tower's place in
// sets/moonlake.yaml, on the top deck; without towers, where it stood.
func TestHeadsStandOnTheTowers(t *testing.T) {
	towers := []Point{
		{264872.4 - eyeEast, 2147919.0 - eyeNorth},
		{264791.6 - eyeEast, 2147693.0 - eyeNorth},
		{263652.4 - eyeEast, 2148337.2 - eyeNorth},
		{263578.1 - eyeEast, 2148129.0 - eyeNorth},
	}
	heads := show.Heads()
	for i, head := range heads {
		flat := head.Origin
		flat[2] = 0
		if apart(flat, towers[i]) > near {
			t.Errorf("head %d stands at %v, its tower at %v", i, flat,
				towers[i])
		}
		deck := rigGround[i] + 20.4 - (lakeHeight + 71.7)
		if math.Abs(head.Origin[2]-deck) > 1e-9 {
			t.Errorf("head %d is %v up, its deck %v", i, head.Origin[2], deck)
		}
	}
	bare := NewLayout(71.7, 0).Heads()
	if bare[2].Origin[2] != show.Water()+shoreHeadUp {
		t.Errorf("without towers the shore head is %v up", bare[2].Origin[2])
	}
	if heads[2].Aim == heads[3].Aim || heads[0].Aim != azimuth+halfTurn {
		t.Errorf("the heads aim %v", heads)
	}
}

// TestFigureWays: a fan of ten beams spans the spread about its
// heading; one beam is the heading; a figure not known is the fan.
func TestFigureWays(t *testing.T) {
	ways, weight, split := FigureWays("fan", 90, 0, 20, 10, 0)
	if len(ways) != 10 || weight != 1 || !split {
		t.Fatalf("the fan is %d ways, %v, %v", len(ways), weight, split)
	}
	if apart(ways[0], Toward(80, 0)) > 1e-12 ||
		apart(ways[9], Toward(100, 0)) > 1e-12 {
		t.Errorf("the fan runs %v to %v", ways[0], ways[9])
	}
	unknown, _, _ := FigureWays("spiral", 90, 0, 20, 10, 0)
	if apart(unknown[3], ways[3]) != 0 {
		t.Error("an unknown figure is not the fan")
	}
	for figure, count := range map[string]int{"beam": 1, "pair": 2,
		"fan64": 64, "scan": 48, "cone": 40, "wave": 16} {
		ways, _, _ := FigureWays(figure, 90, -5, 20, 10, 0.25)
		if len(ways) != count {
			t.Errorf("%s has %d ways, not %d", figure, len(ways), count)
		}
		for _, way := range ways {
			if math.Abs(way.dot(way)-1) > 1e-9 {
				t.Errorf("%s has a way %v not a unit long", figure, way)
			}
		}
	}
}

// TestSweep: a cued sweep of nought is the middle of the swing; with
// none, the clock's.
func TestSweep(t *testing.T) {
	swing, tilt, spread := Sweep(3, 0)
	if swing != 0 || math.Abs(tilt-(tiltMiddle+tiltNod*math.Sin(
		tiltLead))) > 1e-12 || spread != spreadMiddle {
		t.Errorf("a cued sweep at nought is %v %v %v", swing, tilt, spread)
	}
	if clock, _, _ := Sweep(1, -1); clock != swingMost*math.Sin(swingRate) {
		t.Errorf("the clock's swing is %v", clock)
	}
}

// TestSearchlights: they stand behind the rim either side, the viewer's
// move turns them on the eye, a fan opens and closes, a chase runs.
func TestSearchlights(t *testing.T) {
	left, right := show.Searchlight(0), show.Searchlight(1)
	middle := left.Add(right).Scale(half)
	behind := show.farTree().Add(Toward(azimuth, 0).Scale(searchBehind)).
		Add(Point{0, 0, searchUp})
	if apart(middle, behind) > near ||
		math.Abs(apart(left, right)-2*searchApart) > 1e-9 {
		t.Errorf("the first two stand at %v and %v", left, right)
	}
	at := show.SearchWay("viewer", 0, 8, 0, 0)
	if home := show.Searchlight(0).Scale(-1).unit(); at.dot(home) < 0.999 {
		t.Errorf("a viewer beam points %v, not at the eye", at)
	}
	shut := show.SearchWay("fan", 2, 8, 0, 0)
	if math.Abs(math.Asin(shut[2])*halfTurn/math.Pi-fanTop) > 1e-9 {
		t.Errorf("a shut fan stands at %v", shut)
	}
	if sweep := show.SearchWay("chase", 0, 8, 0.1, 0.3); apart(sweep,
		show.SearchWay("sweep", 0, 8, 0.1, 0.3)) != 0 {
		t.Error("a chase does not aim as a sweep does")
	}
	if Running(8, 0) != 0 || Running(8, 0.5) != 0 || Running(8, 0.1) != 1 {
		t.Error("the chase does not run across the lights")
	}
}

// TestPlaces: the strobes stand behind the far tree, the backlights by
// the eye, the ship's apex over the water, and a beam down ends on it.
func TestPlaces(t *testing.T) {
	strobes := show.Strobes()
	if math.Abs(strobes[0][2]-show.farTree()[2]-whiteUp) > 1e-9 ||
		apart(strobes[0], strobes[1]) < magentaLeft {
		t.Errorf("the strobes stand at %v", strobes)
	}
	back := show.Backlights()
	if math.Abs(back[0][2]-(backlightGround-(lakeHeight+71.7))) > 1e-9 {
		t.Errorf("the backlights stand at %v", back)
	}
	top, foot := show.Ship(900, 450)
	if math.Abs(top[2]-foot[2]-450) > 1e-9 || foot[2] != show.Water() {
		t.Errorf("the ship's beam runs %v to %v", top, foot)
	}
	head := Point{0, 0, 10}
	if end := show.BeamEnd(head, Toward(0, -45), 4000); math.Abs(
		end[2]-show.Water()) > 1e-9 {
		t.Errorf("a beam down ends at %v, not on the water", end)
	}
	if end := show.BeamEnd(head, Toward(0, 10), 100); math.Abs(
		apart(end, head)-100) > 1e-9 {
		t.Errorf("a beam up ends at %v, not its reach", end)
	}
}
