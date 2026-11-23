package cities

import (
	"fmt"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// shortSpan is the span, in metres, the bridge tests draw, short enough
// to draw quickly.
const shortSpan = 600

// bridgeOf is a bridge of a seed at an irregularity, of a span and depth,
// or the test's end.
func bridgeOf(t *testing.T, seed Seed, irregularity, span float64,
	depth int) *Bridge {
	t.Helper()
	bridge, err := NewBridge(seed, Irregularity{Value: irregularity}, span,
		depth)
	if err != nil {
		t.Fatalf("seed %d: %v", seed, err)
	}
	return bridge
}

func TestABridgesPylonsAreInhabitedTowersCountedFromItsSpan(t *testing.T) {
	spacing := drawerOf(t, bridgeCulture, 1).number("pylon-spacing", place{})
	for _, span := range []float64{600, 2000, 5000} {
		bridge := bridgeOf(t, 1, 0, span, 0)
		want := max(2, int(span/spacing))
		if len(bridge.Stations) != want || len(bridge.pylons) != want {
			t.Errorf("span %v: %d stations, %d pylons, want %d", span,
				len(bridge.Stations), len(bridge.pylons), want)
		}
		for index, at := range bridge.Stations {
			if !(at > -span/2 && at < span/2) {
				t.Errorf("station %d is at %v on a span of %v", index, at,
					span)
			}
			if len(bridge.pylons[index].Shafts) != 1 {
				t.Errorf("pylon %d is not one tower", index)
			}
		}
	}
	if len(bridgeOf(t, 1, 0, 5000, 0).Stations) <=
		len(bridgeOf(t, 1, 0, 600, 0).Stations) {
		t.Error("a longer span did not have more pylons")
	}
}

func TestABridgeAtNoIrregularityIsPlainAndFullIrregularityReachesItsFeatures(
	t *testing.T) {
	for seed := range Seed(10) {
		bridge := bridgeOf(t, seed, 0, shortSpan, 0)
		if bridge.Sag || bridge.Step || bridge.Second || bridge.Strands != 1 {
			t.Errorf("seed %d: a plain bridge sags %v, steps %v, has a "+
				"second deck %v and %d strands", seed, bridge.Sag,
				bridge.Step, bridge.Second, bridge.Strands)
		}
	}
	sag, step, second, strands := 0, 0, 0, 0
	for seed := range Seed(60) {
		bridge := bridgeOf(t, seed, 1, shortSpan, 0)
		sag += btoi(bridge.Sag)
		step += btoi(bridge.Step)
		second += btoi(bridge.Second)
		strands = max(strands, bridge.Strands)
	}
	if sag == 0 || step == 0 || second == 0 || strands < 3 {
		t.Errorf("sags %d, steps %d, second decks %d, most strands %d", sag,
			step, second, strands)
	}
}

func TestAnApproachStepsDownToTheGroundAtEachEnd(t *testing.T) {
	bridge := bridgeOf(t, 3, 0, 1000, 0)
	stretches := bridge.stretches()
	var before, after []stretch
	for _, each := range stretches {
		switch {
		case each.approach && each.to <= -bridge.Span/2+1e-9:
			before = append(before, each)
		case each.approach:
			after = append(after, each)
		}
	}
	if len(before) < 2 || len(before) != len(after) {
		t.Fatalf("%d steps before and %d after", len(before), len(after))
	}
	for index := 1; index < len(before); index++ {
		if !(before[index].height > before[index-1].height) ||
			!(after[index].height < after[index-1].height) {
			t.Errorf("the approaches do not step down toward the ground at "+
				"step %d", index)
		}
	}
	if before[0].height <= 0 || after[len(after)-1].height <= 0 ||
		before[0].height >= bridge.level(0) {
		t.Errorf("the last step is %v and %v high, the deck %v",
			before[0].height, after[len(after)-1].height, bridge.level(0))
	}
}

func TestACatenaryHangsEvenlyAndSagsByTheSagGiven(t *testing.T) {
	const length, sag = 700.0, 70.0
	if math.Abs(catenaryDrop(length, sag, 0)) > 1e-6 ||
		math.Abs(catenaryDrop(length, sag, length)) > 1e-6 {
		t.Error("a cable hangs below its supports")
	}
	if got := catenaryDrop(length, sag, length/2); math.Abs(got-sag) > 1e-6 {
		t.Errorf("the middle drops %v, want the sag %v", got, sag)
	}
	if math.Abs(catenaryDrop(length, sag, 100)-
		catenaryDrop(length, sag, length-100)) > 1e-6 {
		t.Error("a catenary is not even about its middle")
	}
	if !(catenaryDrop(length, sag, 100) < catenaryDrop(length, sag, 300)) {
		t.Error("a cable does not hang lower toward its middle")
	}
}

func TestABridgeHasCablesHangersAndLotsOnBothSidesOfTheStreet(t *testing.T) {
	bridge := bridgeOf(t, 2, 0, shortSpan, 0)
	design, err := bridge.Design("hull", 0)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Rendered("hull")
	if err != nil {
		t.Fatal(err)
	}
	if len(named(drawn, "cable")) == 0 || len(named(drawn, "hanger")) == 0 {
		cables := len(named(drawn, "cable"))
		hangers := len(named(drawn, "hanger"))
		t.Errorf("%d cable links, %d hangers", cables,
			hangers)
	}
	street := drawerOf(t, bridgeCulture, 2).fixed("street-width")
	north, south := 0, 0
	for _, wall := range named(drawn, "wall") {
		low, high := reach(wall)
		if low.Z < street/2 && high.Z > -street/2 {
			t.Fatalf("a wall at z %v to %v is in the street", low.Z, high.Z)
		}
		north += btoi(low.Z >= street/2)
		south += btoi(high.Z <= -street/2)
	}
	if north == 0 || south == 0 {
		t.Errorf("%d walls north of the street and %d south", north, south)
	}
	cable := bridge.pylonOffset(bridge.pylons[0])
	for _, hanger := range named(drawn, "hanger") {
		if low, high := reach(hanger); low.Z < -cable-1 || high.Z > cable+1 {
			t.Errorf("a hanger at z %v to %v", low.Z, high.Z)
		}
	}
}

func TestABridgeIsTheSameForTheSameSeedAndTheDepthBoundsTheNesting(t *testing.T) {
	firstBridge := bridgeOf(t, 4, 0.5, shortSpan, 0)
	first := digest(designOf(t, firstBridge))
	againBridge := bridgeOf(t, 4, 0.5, shortSpan, 0)
	again := digest(designOf(t, againBridge))
	if first != again {
		t.Error("one seed made two bridges")
	}
	shallow, deep := bridgeOf(t, 4, 0, 1000, 0), bridgeOf(t, 4, 0, 1000, 2)
	stretch := shallow.stretches()[len(shallow.stretches())/2]
	nested := 0
	for index := range 300 {
		if shallow.lotKind(stretch, index) == lotBridge {
			t.Fatalf("a bridge of depth nought has a bridge for lot %d", index)
		}
		nested += btoi(deep.lotKind(stretch, index) == lotBridge)
	}
	if nested == 0 {
		t.Error("a bridge of depth two never has a bridge for a lot")
	}
}

// designOf is a bridge drawn at the hull level, rendered, or the test's
// end.
func designOf(t *testing.T, bridge *Bridge) []Rendered {
	t.Helper()
	design, err := bridge.Design("hull", 0)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Rendered("hull")
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

func TestANestedBridgeIsOneDeeperAndIsRefusedBelowNought(t *testing.T) {
	deep := bridgeOf(t, 5, 0.4, 300, 2)
	design, err := deep.lot(lotBridge, 7, "hull")
	if err != nil || len(design.Elements) == 0 {
		t.Fatalf("a lot that is a bridge: %d parts, %v", len(design.Elements),
			err)
	}
	if _, err := NewBridge(1, Irregularity{Value: 2}, shortSpan, 0); err == nil {
		t.Error("an irregularity over one was accepted")
	}
	if _, err := deep.lot("cathedral", 1, "hull"); err == nil {
		t.Error("a lot of a kind that is not one was drawn")
	}
}

// kindsAtOrigin are every design the package makes, by a name, each drawn
// in its own space.
func kindsAtOrigin(t *testing.T) map[string]Design {
	t.Helper()
	made := map[string]Design{}
	for _, culture := range []string{"groningen", "portofino", "midwest"} {
		terrace, err := Terrace(culture, 3)
		if err != nil {
			t.Fatal(err)
		}
		made["terrace "+culture] = terrace
	}
	condominium, err := Condominium(4)
	if err != nil {
		t.Fatal(err)
	}
	made["condominium"] = condominium
	for _, seed := range []Seed{1, 2} {
		tower, err := NewTower(seed, Irregularity{Value: 0.8}, Pins{})
		if err != nil {
			t.Fatal(err)
		}
		if made[fmt.Sprint("tower ", seed)], err = tower.Design("quick",
			0); err != nil {
			t.Fatal(err)
		}
	}
	if made["bridge"], err = bridgeOf(t, 1, 0.3, 300, 1).Design("quick",
		0); err != nil {
		t.Fatal(err)
	}
	if made["yurt"], err = yurt(); err != nil {
		t.Fatal(err)
	}
	if made["winnebago"], err = winnebago(); err != nil {
		t.Fatal(err)
	}
	return made
}

func TestEveryKindCanBeMadeAwayFromTheOriginClosedAndWithNothingDoubled(
	t *testing.T) {
	offset := maths.V(1234.5, 56.25, -789.75)
	for name, design := range kindsAtOrigin(t) {
		at, err := design.Rendered("quick")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		moved, err := design.at(maths.NewSubspace(offset, maths.Identity,
			0)).Rendered("quick")
		if err != nil || len(moved) != len(at) || len(at) == 0 {
			t.Fatalf("%s: %d parts at the origin, %d away, %v", name,
				len(at), len(moved), err)
		}
		seen := map[string]bool{}
		for index := range at {
			lowAt, highAt := reach(at[index])
			lowAway, highAway := reach(moved[index])
			if lowAway.Sub(lowAt).Sub(offset).Len() > 1e-6 ||
				highAway.Sub(highAt).Sub(offset).Len() > 1e-6 {
				t.Fatalf("%s: part %d did not move by the offset", name,
					index)
			}
			if !closed(moved[index].Mesh) {
				t.Fatalf("%s: part %d (%s) is not closed", name, index,
					moved[index].Mesh.Name)
			}
			key := digest([]Rendered{moved[index]})
			if seen[key] {
				t.Fatalf("%s: part %d (%s) is drawn twice in one place",
					name, index, moved[index].Mesh.Name)
			}
			seen[key] = true
		}
	}
}
