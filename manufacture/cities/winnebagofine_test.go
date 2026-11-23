package cities

import (
	"math"
	"testing"
)

// winnebagoAt is the winnebago drawn for a level, or the test's end.
func winnebagoAt(t *testing.T, level Detail) []Rendered {
	t.Helper()
	design, err := WinnebagoAt(level)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Rendered(level)
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

func TestTheDemoWinnebagoIsDrawnInMorePartsThanTheQuickOne(t *testing.T) {
	quick, demo := winnebagoAt(t, "quick"), winnebagoAt(t, "demo")
	if len(demo) <= len(quick) {
		t.Errorf("%d parts at the demo level and %d quick", len(demo),
			len(quick))
	}
}

func TestTheWinnebagoBelowTheDemoLevelIsTheWinnebagoAsItWas(t *testing.T) {
	old, err := winnebago()
	if err != nil {
		t.Fatal(err)
	}
	for _, level := range []Detail{"hull", "clumps", "quick"} {
		before, err := old.Rendered(level)
		if err != nil {
			t.Fatal(err)
		}
		if digest(before) != digest(winnebagoAt(t, level)) {
			t.Errorf("the %s level changed", level)
		}
		if got, want := len(before), winnebagoPartsBelowDemo(t); got != want {
			t.Errorf("%d parts at the %s level, want %d", got, level, want)
		}
	}
}

// winnebagoPartsBelowDemo is how many parts a winnebago has below the demo
// level: the body and the cab, a wheel at each end of each axle, a window
// each side at each place, the windscreen and the door.
func winnebagoPartsBelowDemo(t *testing.T) int {
	t.Helper()
	axles := 1 + int(mustChoose(t, "rear-axles"))
	windows := 2 * int(mustChoose(t, "window-count"))
	return 2 + 2*axles + windows + 1 + 1
}

func TestEveryPartOfTheDemoWinnebagoIsInsideItsEnvelope(t *testing.T) {
	side := mustChoose(t, "body-width")/2 + mustChoose(t, "reach-side")
	top := mustChoose(t, "roof-height") + mustChoose(t, "reach-top")
	end := mustChoose(t, "reach-end")
	rear := -mustChoose(t, "body-length")/2 - end
	front := mustChoose(t, "body-length")/2 + mustChoose(t, "cab-length") + end
	for index, one := range winnebagoAt(t, "demo") {
		low, high := reach(one)
		if low.X < -side-tolerance || high.X > side+tolerance ||
			low.Y < -tolerance || high.Y > top+tolerance ||
			low.Z < rear-tolerance || high.Z > front+tolerance {
			t.Errorf("part %d (%s) spans %v to %v, outside x +-%v, y 0 to "+
				"%v, z %v to %v", index, one.Mesh.Name, low, high, side, top,
				rear, front)
		}
	}
}

func TestTheSameWinnebagoIsDrawnEachTime(t *testing.T) {
	first, second := digest(winnebagoAt(t, "demo")), digest(winnebagoAt(t, "demo"))
	if first != second {
		t.Error("two draws of the demo winnebago differ")
	}
}

func TestEveryPartOfTheDemoWinnebagoIsAClosedSolidWoundOutward(t *testing.T) {
	for index, one := range winnebagoAt(t, "demo") {
		if !closed(one.Mesh) || volume(one.Mesh) <= 0 {
			t.Errorf("part %d (%s) is not a closed solid wound outward",
				index, one.Mesh.Name)
		}
	}
}

func TestTheDemoWinnebagoHasEveryPartOfItsDecomposition(t *testing.T) {
	drawn := winnebagoAt(t, "demo")
	windows := 2*int(mustChoose(t, "window-count")) + 2 + 1 + 1
	for part, want := range map[string]int{
		"frame": 4 * (windows + 1), "arch": 6, "mirror": 2, "arm": 4,
		"headlamp": 2, "taillamp": 2, "bumper": 1, "handle": 1, "step": 1, "riser": 1,
		"fabric": 1, "roller": 1, "plate": 1, "shroud": 1, "fan": 1,
		"grille": 3, "belt": 3,
	} {
		if got := len(named(drawn, part)); got != want {
			t.Errorf("%d %s parts, want %d", got, part, want)
		}
	}
	if got := len(named(drawn, "rung")); got < 3 {
		t.Errorf("%d rungs on the ladder", got)
	}
	if len(named(drawn, "seam")) < 2*4+1 {
		seams := len(named(drawn,
			"seam"))
		t.Errorf("%d seams, want several down each side", seams)
	}
}

func TestTheStepHasARiserFromTheGroundToTheDoorsSill(t *testing.T) {
	risers := named(winnebagoAt(t, "demo"), "riser")
	if len(risers) != 1 {
		t.Fatalf("%d risers, want one", len(risers))
	}
	low, high := reach(risers[0])
	if math.Abs(low.Y) > tolerance ||
		math.Abs(high.Y-mustChoose(t, "entry-sill")) > tolerance {
		t.Errorf("the riser spans %v to %v up, want 0 to the sill's %v",
			low.Y, high.Y, mustChoose(t, "entry-sill"))
	}
}

func TestTheCabsFrontCornersAreRounded(t *testing.T) {
	cab := drawnWinnebago(t, "quick")[1]
	rounded := winnebagoAt(t, "demo")[1]
	front := mustChoose(t, "body-length")/2 + mustChoose(t, "cab-length")
	flat := mustChoose(t, "cab-width")/2 - mustChoose(t, "nose-radius")
	if _, high := reach(cab); math.Abs(high.Z-front) > tolerance {
		t.Fatalf("the box cab reaches %v forward, want %v", high.Z, front)
	}
	_, high := reach(rounded)
	if math.Abs(high.Z-front) > tolerance {
		t.Errorf("the rounded cab reaches %v forward, want %v", high.Z, front)
	}
	for _, triangle := range rounded.Mesh.Triangles {
		for _, corner := range triangle {
			placed := rounded.Place.Out(corner)
			if placed.Z > front-tolerance && math.Abs(placed.X) > flat+tolerance {
				t.Fatalf("a corner of the front face stands at x %v, past "+
					"the flat's %v", placed.X, flat)
			}
		}
	}
}

func TestTheWindscreenFitsTheFlatOfTheNose(t *testing.T) {
	flat := mustChoose(t, "cab-width") - 2*mustChoose(t, "nose-radius")
	front := mustChoose(t, "body-length")/2 + mustChoose(t, "cab-length")
	for _, one := range partsOf(t, winnebagoAt(t, "demo"), roleGlazing) {
		low, high := reach(one)
		if high.Z > front-0.1 && high.X-low.X > flat+tolerance {
			t.Errorf("a windscreen %v wide, over a flat %v wide",
				high.X-low.X, flat)
		}
	}
}

func TestTheLampsAreLightsOfTheHeatAndStrengthDataGives(t *testing.T) {
	drawn := winnebagoAt(t, "demo")
	for _, name := range []string{"headlamp", "taillamp"} {
		for _, lamp := range named(drawn, name) {
			if lamp.Lamp == nil ||
				lamp.Lamp.Heat != mustChoose(t, name+"-heat") ||
				lamp.Lamp.Luminance != mustChoose(t, name+"-strength") {
				t.Fatalf("a %s is not the light data gives: %+v", name,
					lamp.Lamp)
			}
		}
	}
	for _, one := range drawn {
		if one.Lamp != nil && one.Mesh.Name != "headlamp" &&
			one.Mesh.Name != "taillamp" {
			t.Errorf("a %s is a light", one.Mesh.Name)
		}
	}
}

func TestEachWheelArchStandsBehindItsTyreOutsideTheWall(t *testing.T) {
	tyres := partsOf(t, winnebagoAt(t, "demo"), roleTyre)
	arches := named(winnebagoAt(t, "demo"), "arch")
	if len(arches) != len(tyres) {
		t.Fatalf("%d arches for %d tyres", len(arches), len(tyres))
	}
	for index, arch := range arches {
		low, high := reach(arch)
		tyreLow, tyreHigh := reach(tyres[index])
		if math.Abs((low.Z+high.Z)-(tyreLow.Z+tyreHigh.Z)) > tolerance {
			t.Errorf("arch %d is centred at z %v, its tyre at %v", index,
				(low.Z+high.Z)/2, (tyreLow.Z+tyreHigh.Z)/2)
		}
		if high.Y <= tyreHigh.Y {
			t.Errorf("arch %d tops out at %v, under its tyre's %v", index,
				high.Y, tyreHigh.Y)
		}
	}
}
