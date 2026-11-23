package cities

import (
	"math"
	"testing"
)

// drawnYurt is the yurt drawn for a level, or the test's end.
func drawnYurt(t *testing.T, level Detail) []Rendered {
	t.Helper()
	design, err := YurtAt(level)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Rendered(level)
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

// yurtRoles are the roles of the four parts a yurt has below the demo
// level, in order: the wall, the roof, the crown and the door.
var yurtRoles = []partRole{roleFrame, roleCover, roleCrown, roleDoor}

// outermost and innermost are how far out and in from the yurt's axis any
// corner of a drawn part stands, once placed.
func outermost(one Rendered) (out, in float64) {
	out, in = math.Inf(-1), math.Inf(1)
	for _, triangle := range one.Mesh.Triangles {
		for _, corner := range triangle {
			placed := one.Place.Out(corner)
			reach := math.Hypot(placed.X, placed.Z)
			out, in = math.Max(out, reach), math.Min(in, reach)
		}
	}
	return out, in
}

func TestTheDemoYurtIsDrawnInMorePartsThanTheQuickOne(t *testing.T) {
	quick, demo := drawnYurt(t, "quick"), drawnYurt(t, "demo")
	if len(demo) <= len(quick) {
		t.Errorf("%d parts at the demo level and %d quick", len(demo),
			len(quick))
	}
}

func TestTheYurtBelowTheDemoLevelIsTheYurtAsItWas(t *testing.T) {
	old, err := yurt()
	if err != nil {
		t.Fatal(err)
	}
	for _, level := range []Detail{"hull", "clumps", "quick"} {
		before, err := old.Rendered(level)
		if err != nil {
			t.Fatal(err)
		}
		if digest(before) != digest(drawnYurt(t, level)) {
			t.Errorf("the %s level changed", level)
		}
		if len(before) != len(yurtRoles) {
			t.Errorf("%d parts at the %s level, want %d", len(before),
				level, len(yurtRoles))
		}
		for index, role := range yurtRoles {
			want, err := material(role)
			if err != nil {
				t.Fatal(err)
			}
			if before[index].Material != want || want.Texture != nil {
				t.Errorf("%s: part %d is not plain of the %s role", level,
					index, role)
			}
		}
	}
}

func TestEveryPartOfTheDemoYurtIsInsideTheYurtsEnvelope(t *testing.T) {
	out := mustChoose(t, "wall-radius") + mustChoose(t, "yurt-reach")
	top := mustChoose(t, "roof-top") + mustChoose(t, "crown-rise") +
		mustChoose(t, "yurt-reach-top")
	for index, one := range drawnYurt(t, "demo") {
		low, high := reach(one)
		reached, _ := outermost(one)
		if low.Y < -tolerance || high.Y > top+tolerance ||
			reached > out+tolerance {
			t.Fatalf("part %d (%s) spans %v to %v up and reaches %v out, "+
				"want 0 to %v and %v", index, one.Mesh.Name, low.Y, high.Y,
				reached, top, out)
		}
	}
}

func TestTheSameYurtIsDrawnEachTime(t *testing.T) {
	if first, second := digest(drawnYurt(t, "demo")), digest(drawnYurt(t, "demo")); first != second {
		t.Error("two draws of the demo yurt differ")
	}
}

func TestTheDemoYurtHasEveryPartOfItsDecomposition(t *testing.T) {
	drawn := drawnYurt(t, "demo")
	sectors := int(mustChoose(t, "cover-sectors"))
	wall := sectors - (1 + 2*int(mustChoose(t, "door-gap-sectors")))
	for part, want := range map[string]int{
		"felt": wall + sectors, "canvas": wall + sectors, "rail": sectors,
		"rope":  int(mustChoose(t, "rope-count")) * sectors,
		"band":  int(mustChoose(t, "band-count"))*sectors + wall,
		"pole":  int(mustChoose(t, "pole-count")),
		"toono": int(mustChoose(t, "toono-segments")),
		"spoke": int(mustChoose(t, "toono-spokes")), "hub": 1,
		"jamb": 2, "lintel": 1, "door": 1,
		"panel": int(mustChoose(t, "door-panels")),
		"pipe":  1, "stove": 1, "carpet": 1, "bed": 2, "blanket": 2,
		"chest": 1, "floor": 1,
	} {
		if got := len(named(drawn, part)); got != want {
			t.Errorf("%d %s parts, want %d", got, part, want)
		}
	}
	radius := mustChoose(t, "wall-radius")
	pitch := mustChoose(t, "lattice-slat-pitch")
	stations := int(math.Round(2 * math.Pi * radius /
		pitch))
	if slats := len(named(drawn, "slat")); slats%2 != 0 ||
		slats >= 2*stations || slats < 2*(stations-12) {
		t.Errorf("%d slats for %d stations, two to each but the doorway's",
			slats, stations)
	}
}

func TestEveryPartOfTheDemoYurtIsAClosedSolidWoundOutward(t *testing.T) {
	for index, one := range drawnYurt(t, "demo") {
		if !closed(one.Mesh) || volume(one.Mesh) <= 0 {
			t.Fatalf("part %d (%s) is not a closed solid wound outward",
				index, one.Mesh.Name)
		}
	}
}

func TestTheDemoYurtHasNoSolidRoofOrWallAndSoItsCrownIsOpen(t *testing.T) {
	drawn := drawnYurt(t, "demo")
	for _, part := range []string{"wall", "roof", "lining", "crown"} {
		if got := len(named(drawn, part)); got != 0 {
			t.Errorf("%d solid %s parts at the demo level", got, part)
		}
	}
	opening := mustChoose(t, "crown-radius") - mustChoose(t, "toono-bar")
	for _, cover := range append(named(drawn, "felt"), named(drawn, "canvas")...) {
		low, high := reach(cover)
		_, in := outermost(cover)
		if high.Y > mustChoose(t, "roof-top")+tolerance ||
			(low.Y >= mustChoose(t, "wall-height")-tolerance &&
				in < opening-tolerance) {
			t.Fatalf("a %s part spans %v to %v up and comes in to %v, into "+
				"the crown's opening of radius %v", cover.Mesh.Name, low.Y,
				high.Y, in, opening)
		}
	}
}

func TestTheKhanaStandsInsideTheCover(t *testing.T) {
	drawn := drawnYurt(t, "demo")
	widest := math.Inf(-1)
	for _, slat := range named(drawn, "slat") {
		out, _ := outermost(slat)
		widest = math.Max(widest, out)
	}
	for _, cover := range named(drawn, "canvas") {
		_, high := reach(cover)
		_, in := outermost(cover)
		if high.Y <= mustChoose(t, "wall-height")+tolerance && in <= widest {
			t.Fatalf("the canvas begins %v out, inside the slats' %v", in,
				widest)
		}
	}
}

func TestTheRoofPolesRunFromTheWallsTopToTheCrownAndNoFurther(t *testing.T) {
	thickness := mustChoose(t, "pole-thickness")
	height := mustChoose(t, "wall-height")
	for _, pole := range named(drawnYurt(t, "demo"), "pole") {
		low, high := reach(pole)
		out, _ := outermost(pole)
		if math.Abs(high.Y-mustChoose(t, "roof-top")) > thickness ||
			low.Y > height || low.Y < height-thickness ||
			out > mustChoose(t, "wall-radius")+thickness {
			t.Fatalf("a pole spans %v to %v up and reaches %v out", low.Y,
				high.Y, out)
		}
	}
}

func TestTheStovepipeRisesThroughTheCrownRing(t *testing.T) {
	pipes := named(drawnYurt(t, "demo"), "pipe")
	if len(pipes) != 1 {
		t.Fatalf("%d stovepipes, want one", len(pipes))
	}
	low, high := reach(pipes[0])
	out, _ := outermost(pipes[0])
	want := mustChoose(t, "roof-top") + mustChoose(t, "crown-rise") +
		mustChoose(t, "pipe-rise")
	if math.Abs(high.Y-want) > tolerance ||
		low.Y > mustChoose(t, "stove-height")+mustChoose(t, "floor-height") ||
		out > mustChoose(t, "crown-radius")-mustChoose(t, "toono-bar") {
		t.Errorf("the pipe spans %v to %v up and reaches %v out, want it "+
			"to %v inside the ring", low.Y, high.Y, out, want)
	}
}

func TestTheDoorStandsOpenIntoTheYurtAndTheDoorwayIsClear(t *testing.T) {
	radius := mustChoose(t, "wall-radius")
	doors := named(drawnYurt(t, "demo"), "door")
	if len(doors) != 1 {
		t.Fatalf("%d doors, want one", len(doors))
	}
	low, high := reach(doors[0])
	if high.Z > radius+mustChoose(t, "door-thickness") ||
		low.Z > radius-mustChoose(t, "door-width")/2 {
		t.Errorf("the door spans z %v to %v, want it swung inside the wall "+
			"at %v", low.Z, high.Z, radius)
	}
	for _, slat := range named(drawnYurt(t, "demo"), "slat") {
		low, high := reach(slat)
		if (low.X+high.X)/2 > -mustChoose(t, "door-width")/2 &&
			(low.X+high.X)/2 < mustChoose(t, "door-width")/2 &&
			(low.Z+high.Z)/2 > 0 {
			t.Fatalf("a slat stands in the doorway at x %v", (low.X+high.X)/2)
		}
	}
}

func TestEveryMaterialAtTheDemoLevelThatSaysItIsTexturedIsAndOthersAreNot(
	t *testing.T) {
	drawn := drawnYurt(t, "demo")
	for _, part := range []string{"canvas", "felt", "rope", "band", "pole",
		"toono", "door", "panel", "carpet", "jamb", "bed", "blanket"} {
		for _, one := range named(drawn, part) {
			if one.Material.Texture == nil {
				t.Fatalf("a %s has no texture", part)
			}
		}
	}
	for _, role := range []partRole{roleCanvas, roleCover, roleRope, roleBand,
		rolePole, roleCrown, roleDoor, rolePaint, roleCarpet} {
		plain, err := material(role)
		if err != nil {
			t.Fatal(err)
		}
		if plain.Texture != nil {
			t.Errorf("the %s role's own material has a texture", role)
		}
	}
}
