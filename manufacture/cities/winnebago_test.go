package cities

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// drawnWinnebago is the winnebago drawn at a level, or the test's end.
func drawnWinnebago(t *testing.T, level Detail) []Rendered {
	t.Helper()
	design, err := winnebago()
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Rendered(level)
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

// reach is the lowest and the highest corner of a drawn part along each
// axis, after it is placed on the vehicle.
func reach(one Rendered) (low, high maths.Vec) {
	low = maths.V(math.Inf(1), math.Inf(1), math.Inf(1))
	high = maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))
	for _, triangle := range one.Mesh.Triangles {
		for _, corner := range triangle {
			placed := one.Place.Out(corner)
			low = maths.V(math.Min(low.X, placed.X),
				math.Min(low.Y, placed.Y), math.Min(low.Z, placed.Z))
			high = maths.V(math.Max(high.X, placed.X),
				math.Max(high.Y, placed.Y), math.Max(high.Z, placed.Z))
		}
	}
	return low, high
}

// partsOf is the drawn parts made of a role.
func partsOf(t *testing.T, drawn []Rendered, role partRole) []Rendered {
	t.Helper()
	want, err := material(role)
	if err != nil {
		t.Fatal(err)
	}
	var found []Rendered
	for _, one := range drawn {
		if one.Material == want {
			found = append(found, one)
		}
	}
	return found
}

func TestAWinnebagoIsMadeOfPartsEachOfItsRole(t *testing.T) {
	drawn := drawnWinnebago(t, "quick")
	axles := 1 + int(mustChoose(t, "rear-axles"))
	windows := 2 * int(mustChoose(t, "window-count"))
	counts := map[partRole]int{roleBody: 1, roleCab: 1, roleTyre: 2 * axles,
		roleGlazing: windows + 1, roleEntry: 1}
	total := 0
	for role, want := range counts {
		if got := len(partsOf(t, drawn, role)); got != want {
			t.Errorf("%d parts of the %s role, want %d", got, role, want)
		}
		total += want
	}
	if len(drawn) != total {
		t.Errorf("%d parts, want %d", len(drawn), total)
	}
}

func TestAWinnebagoHasSixWheelsOnTheGround(t *testing.T) {
	tyres := partsOf(t, drawnWinnebago(t, "demo"), roleTyre)
	if len(tyres) != 6 {
		t.Fatalf("%d wheels, want six", len(tyres))
	}
	radius := mustChoose(t, "tyre-radius")
	for index, one := range tyres {
		low, high := reach(one)
		if math.Abs(low.Y) > tolerance ||
			math.Abs(high.Y-2*radius) > tolerance {
			t.Errorf("wheel %d spans %v to %v up, want 0 to %v", index,
				low.Y, high.Y, 2*radius)
		}
	}
}

func TestEachWheelStandsOutsideTheTrack(t *testing.T) {
	inner := mustChoose(t, "wheel-track") / 2
	outer := inner + mustChoose(t, "tyre-width")
	for index, one := range partsOf(t, drawnWinnebago(t, "demo"), roleTyre) {
		low, high := reach(one)
		kerb := math.Abs(high.X+inner) < tolerance &&
			math.Abs(low.X+outer) < tolerance
		off := math.Abs(low.X-inner) < tolerance &&
			math.Abs(high.X-outer) < tolerance
		if !kerb && !off {
			t.Errorf("wheel %d spans x %v to %v, want %v to %v out from "+
				"the middle", index, low.X, high.X, inner, outer)
		}
	}
}

func TestAWinnebagosSizesAreTheDatas(t *testing.T) {
	drawn := drawnWinnebago(t, "demo")
	body, cab := drawn[0], drawn[1]
	low, high := reach(body)
	length, width := mustChoose(t, "body-length"), mustChoose(t, "body-width")
	if math.Abs(high.Y-mustChoose(t, "roof-height")) > tolerance ||
		math.Abs(low.Y-mustChoose(t, "ground-clearance")) > tolerance ||
		math.Abs(high.X-width/2) > tolerance ||
		math.Abs(low.X+width/2) > tolerance ||
		math.Abs(high.Z-length/2) > tolerance ||
		math.Abs(low.Z+length/2) > tolerance {
		t.Errorf("the body spans %v to %v", low, high)
	}
	cabLow, cabHigh := reach(cab)
	if math.Abs(cabLow.Z-length/2) > tolerance ||
		math.Abs(cabHigh.Z-length/2-mustChoose(t, "cab-length")) > tolerance {
		t.Errorf("the cab spans %v to %v along, want %v long from the "+
			"body's front", cabLow.Z, cabHigh.Z, mustChoose(t, "cab-length"))
	}
	if cabHigh.Y >= high.Y {
		t.Errorf("the cab is %v high, and not lower than the body's %v",
			cabHigh.Y, high.Y)
	}
}

func TestTheEntryDoorIsOnTheKerbSideOfTheBody(t *testing.T) {
	doors := partsOf(t, drawnWinnebago(t, "quick"), roleEntry)
	if len(doors) != 1 {
		t.Fatalf("%d doors, want one", len(doors))
	}
	low, high := reach(doors[0])
	wall := mustChoose(t, "body-width") / 2
	half := mustChoose(t, "entry-thickness") / 2
	at := mustChoose(t, "entry-place")
	if math.Abs(low.X+wall+half) > tolerance ||
		math.Abs(high.X+wall-half) > tolerance {
		t.Errorf("the door spans x %v to %v, want it half out of the "+
			"kerb wall at %v", low.X, high.X, -wall)
	}
	across := mustChoose(t, "entry-width") / 2
	if math.Abs(low.Z-(at-across)) > tolerance ||
		math.Abs(high.Z-(at+across)) > tolerance {
		t.Errorf("the door spans %v to %v along, want %v to %v", low.Z,
			high.Z, at-across, at+across)
	}
}

func TestTheWindscreenStandsOnTheCabsFrontFace(t *testing.T) {
	front := math.Inf(-1)
	var screen Rendered
	for _, one := range partsOf(t, drawnWinnebago(t, "quick"), roleGlazing) {
		if _, high := reach(one); high.Z > front {
			front, screen = high.Z, one
		}
	}
	low, high := reach(screen)
	face := mustChoose(t, "body-length")/2 + mustChoose(t, "cab-length")
	half := mustChoose(t, "window-thickness") / 2
	if math.Abs(high.Z-face-half) > tolerance ||
		math.Abs(low.Z-face+half) > tolerance ||
		math.Abs(high.X+low.X) > tolerance {
		t.Errorf("the windscreen spans z %v to %v and x %v to %v, want "+
			"it centred on the cab's front face at %v", low.Z, high.Z,
			low.X, high.X, face)
	}
}

func TestWindowsAreTheSameOnEachSide(t *testing.T) {
	var kerb, off []float64
	for _, one := range partsOf(t, drawnWinnebago(t, "quick"), roleGlazing) {
		low, high := reach(one)
		if high.X-low.X > mustChoose(t, "window-thickness")+tolerance {
			continue
		}
		if high.X < 0 {
			kerb = append(kerb, low.Z)
		} else {
			off = append(off, low.Z)
		}
	}
	count := int(mustChoose(t, "window-count"))
	if len(kerb) != count || len(off) != count {
		t.Fatalf("%d windows on the kerb side and %d on the other, want "+
			"%d each", len(kerb), len(off), count)
	}
	for index := range kerb {
		if math.Abs(kerb[index]-off[index]) > tolerance {
			t.Errorf("window %d stands at %v and %v", index, kerb[index],
				off[index])
		}
	}
}

func TestEveryLevelDrawsClosedSolidsWoundOutward(t *testing.T) {
	for _, level := range []Detail{"hull", "clumps", "quick", "demo"} {
		for index, one := range drawnWinnebago(t, level) {
			if !closed(one.Mesh) || volume(one.Mesh) <= 0 {
				t.Errorf("%s: part %d is not a closed solid wound outward",
					level, index)
			}
		}
	}
}

func TestTheGlazingShinesMoreThanThePaintAndTheTyres(t *testing.T) {
	glass, _ := material(roleGlazing)
	for _, role := range []partRole{roleBody, roleCab, roleTyre, roleEntry} {
		other, err := material(role)
		if err != nil {
			t.Fatal(err)
		}
		if glass.Gloss <= other.Gloss {
			t.Errorf("the glazing shines %v and the %s %v", glass.Gloss,
				role, other.Gloss)
		}
	}
}

func TestAWinnebagoSleepsFourAndAPitchHoldsIt(t *testing.T) {
	facts, err := WinnebagoFacts()
	if err != nil {
		t.Fatal(err)
	}
	footprint := mustChoose(t, "body-width") * (mustChoose(t, "body-length") +
		mustChoose(t, "cab-length"))
	if facts.Sleeps != 4 {
		t.Errorf("it sleeps %d, want four", facts.Sleeps)
	}
	if facts.FloorArea <= 0 || facts.FloorArea > footprint {
		t.Errorf("the floor is %v square metres, under the footprint's %v",
			facts.FloorArea, footprint)
	}
	if facts.PitchArea <= footprint {
		t.Errorf("a pitch is %v square metres, and the vehicle's footprint "+
			"is %v", facts.PitchArea, footprint)
	}
}

func TestTwoDesignsDoNotNameTheSameChoice(t *testing.T) {
	const same = "numbers:\n- name: wall-radius\n  value: 3\n  says: a wall\n" +
		"  kind: picked\n  by: test\n"
	_, err := readBook(choiceFile{"one.yaml", []byte(same)},
		choiceFile{"two.yaml", []byte(same)})
	if err == nil {
		t.Error("a name two files give was read")
	}
}

func TestACountThatIsNotWholeIsRefused(t *testing.T) {
	reader := &choiceReader{}
	if reader.whole("body-width"); reader.trouble == nil {
		t.Error("a width of 2.5 was read as a count")
	}
	if got := reader.of("body-length"); got != 0 {
		t.Errorf("after a refusal the reader gave %v, not nought", got)
	}
}

func TestAChoiceNoDataNamesIsRefused(t *testing.T) {
	reader := &choiceReader{}
	if reader.of("no-such-choice"); reader.trouble == nil {
		t.Error("a name nothing gives was read")
	}
}

func TestAPaneOrAWheelWithNoSizeIsNotDrawn(t *testing.T) {
	if _, err := (pane{Role: roleGlazing}).Rendered("quick"); err == nil {
		t.Error("a pane with no size was drawn")
	}
	if _, err := (pane{Width: 1, Height: 1, Thickness: 1,
		Role: "thatch"}).Rendered("quick"); err == nil {
		t.Error("a pane of a role nothing is made of was drawn")
	}
	if _, err := (wheel{}).Rendered("quick"); err == nil {
		t.Error("a wheel with no size was drawn")
	}
}
