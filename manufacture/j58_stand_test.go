package manufacture

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// standNear is how near two lengths of the stand must agree.
const standNear = 1e-9 // picked: sums of a few lengths, to rounding

// standParts is the stand's parts at a detail, none shared.
func standParts(detail string) []part {
	return j58StandRead.stood(j58Read, detail).object.parts
}

// standPartsOf is the stand's parts of one look, at quick detail.
func standPartsOf(of slot) []part {
	var found []part
	for _, each := range standParts("quick") {
		if each.of == of {
			found = append(found, each)
		}
	}
	return found
}

// The stand stands on its pad on the ground, inside its box, its top as
// high as its bellmouth's lip.
func TestTheJ58StandStandsOnItsPad(t *testing.T) {
	stand := j58StandRead
	box := mustMake(t, "j58-test-stand", 5).Bounds()
	every := partBounds(standParts("quick"))
	if math.Abs(every.Min.Y) > standNear ||
		math.Abs(every.Max.Y-box.Max.Y) > standNear ||
		every.Min.X < box.Min.X-standNear || every.Max.X > box.Max.X+standNear ||
		every.Min.Z < box.Min.Z-standNear || every.Max.Z > box.Max.Z+standNear {
		t.Errorf("the stand's parts reach %+v, its box is %+v", every, box)
	}
	pad := partBounds(standPartsOf(standPad))
	if math.Abs(pad.Max.Y-stand.Pad.Thick) > standNear {
		t.Errorf("the pad's top is %v up, want %v", pad.Max.Y, stand.Pad.Thick)
	}
}

// Each of the engine's four pins is held: a post beside it reaching over
// it, and a pin rod from the post ending inside the engine's lug, where
// the engine's own mount says the pin is.
func TestTheJ58StandHoldsTheEnginesPins(t *testing.T) {
	stand := j58StandRead
	pins := stand.pins(j58Read)
	rods := standPartsOf(standPin)
	if len(rods) != len(pins) || len(pins) != 4 {
		t.Fatalf("the stand has %d pin rods for %d pins", len(rods), len(pins))
	}
	for index, pin := range pins {
		inner := rods[index].space.Origin
		if math.Abs(inner.X-pin.X) > standNear || math.Abs(inner.Y-pin.Y) > standNear ||
			math.Abs(math.Abs(pin.Z)-math.Abs(inner.Z)-stand.Reach) > standNear {
			t.Errorf("pin rod %d starts at %+v, want %v inside pin %+v",
				index, inner, stand.Reach, pin)
		}
	}
	for index, pin := range pins {
		held := false
		for _, each := range standPartsOf(standCradle) {
			box := partBounds([]part{each})
			held = held || (box.Min.X < pin.X && box.Max.X > pin.X &&
				box.Max.Y > pin.Y && math.Abs(box.Min.Z) > math.Abs(pin.Z) &&
				box.Min.Z*pin.Z > 0)
		}
		if !held {
			t.Errorf("no post stands beside pin %d at %+v", index, pin)
		}
	}
}

// The stand holds a whole j58, built by the engine's own kind at the
// place the stand gives it: the same parts as the engine lying alone,
// each lifted by the same height and none turned, and each of the
// stand's pins ending inside one of its lugs.
func TestTheJ58StandHoldsAWholeEngine(t *testing.T) {
	stand, engine := j58StandRead, j58Read
	alone := j58Parts("quick")
	var held []part
	for _, each := range standParts("quick") {
		if each.of < j58Slots {
			held = append(held, each)
		}
	}
	if len(held) != len(alone) {
		t.Fatalf("the stand holds %d engine parts, the engine has %d",
			len(held), len(alone))
	}
	lift := stand.place(engine).Origin.Sub(engine.lying().Origin)
	for index, each := range held {
		moved := each.space.Origin.Sub(alone[index].space.Origin)
		if each.of != alone[index].of || moved.Sub(lift).Len() > standNear ||
			each.space.Orientation != alone[index].space.Orientation {
			t.Fatalf("engine part %d is moved %+v, want %+v", index, moved,
				lift)
		}
	}
	var lugs []part
	for _, each := range held {
		if each.of == j58Lug {
			lugs = append(lugs, each)
		}
	}
	for index, pin := range stand.pins(engine) {
		end := maths.V(pin.X, pin.Y, pin.Z-math.Copysign(stand.Reach, pin.Z))
		box := partBounds([]part{lugs[index]})
		if end.X < box.Min.X || end.X > box.Max.X || end.Y < box.Min.Y ||
			end.Y > box.Max.Y || end.Z < box.Min.Z || end.Z > box.Max.Z {
			t.Errorf("pin %d ends at %+v, outside its lug %+v", index, end, box)
		}
	}
}

// A higher axis lifts the engine with its pins: the engine goes where
// the stand puts it, and not where it was. The anti-test of the above.
func TestAHigherStandLiftsTheEngine(t *testing.T) {
	higher := strings.Replace(string(j58StandRaw), "axis: 1.5", "axis: 1.7", 1)
	stand, err := readJ58Stand([]byte(higher), j58Read)
	if err != nil {
		t.Fatal(err)
	}
	low := j58StandRead.pins(j58Read)[0]
	high := stand.pins(j58Read)[0]
	if math.Abs(high.Y-low.Y-0.2) > standNear {
		t.Errorf("a stand 0.2 higher lifts the pins from %v to %v", low.Y,
			high.Y)
	}
}

// The cradle clears the engine: its rails' top is below the engine's
// envelope, and every post stands further from the axis than the
// envelope reaches. The anti-test of a cradle through the engine.
func TestTheJ58StandCradleClearsTheEngine(t *testing.T) {
	stand, engine := j58StandRead, j58Read
	axis := stand.Pad.Thick + stand.Axis
	envelope := engine.Envelope / 2
	cradle := partBounds(standPartsOf(standCradle))
	if !(stand.railTop() < axis-envelope) || !(cradle.Max.Y > axis) {
		t.Errorf("the rails' top is %v and the posts reach %v; the "+
			"engine's underside is %v", stand.railTop(), cradle.Max.Y,
			axis-envelope)
	}
	for index, each := range standPartsOf(standCradle) {
		box := partBounds([]part{each})
		if box.Max.Y <= stand.railTop()+standNear {
			continue // a rail or a beam, under the engine
		}
		near := math.Min(math.Abs(box.Min.Z), math.Abs(box.Max.Z))
		if !(near > envelope) {
			t.Errorf("post %d stands %v from the axis, inside the "+
				"envelope %v", index, near, envelope)
		}
	}
}

// The thrust goes through the load cell: one cell along the axis from
// the cradle's front to the thrust post, which stands on the pad.
func TestTheJ58StandTakesThrustThroughItsCell(t *testing.T) {
	stand, engine := j58StandRead, j58Read
	cells, posts := standPartsOf(standCell), standPartsOf(standPost)
	if len(cells) != 1 || len(posts) != 1 {
		t.Fatalf("the stand has %d cells and %d thrust posts", len(cells),
			len(posts))
	}
	cell, post := partBounds(cells), partBounds(posts)
	front, _ := stand.railEnds(engine)
	if math.Abs(cell.Min.X-front) > standNear ||
		math.Abs(cell.Max.X-post.Min.X) > standNear ||
		math.Abs(post.Min.Y-stand.Pad.Thick) > standNear {
		t.Errorf("the cell runs %v to %v, the post from %v, %v up", cell.Min.X,
			cell.Max.X, post.Min.X, post.Min.Y)
	}
}

// The bellmouth's inside meets the engine's face at the face's radius
// and flares to its lip ahead of it, and its legs reach the pad.
func TestTheJ58BellmouthMeetsTheFace(t *testing.T) {
	stand, engine := j58StandRead, j58Read
	face, _ := stand.bellAt(engine, 1)
	lip, _ := stand.bellAt(engine, 0)
	if math.Abs(face[0]-engine.Radii.Face) > standNear || math.Abs(face[1]) > standNear ||
		math.Abs(lip[0]-stand.Bell.Lip) > standNear ||
		math.Abs(lip[1]+stand.Bell.Long) > standNear {
		t.Errorf("the bellmouth runs from %v to %v", lip, face)
	}
	legs := partBounds(standPartsOf(standBell))
	if math.Abs(legs.Min.Y-stand.Pad.Thick) > stand.Bell.Leg {
		t.Errorf("the bellmouth's legs reach %v, the pad's top is %v",
			legs.Min.Y, stand.Pad.Thick)
	}
}

// The stand flattens at every detail and has no face of no area.
func TestTheJ58StandFlattensWithEveryFaceWhole(t *testing.T) {
	for _, detail := range j58Details {
		made, err := j58Stand{}.madeAt(Sow(5), detail)
		if err != nil {
			t.Fatalf("%s: %v", detail, err)
		}
		if _, err := made.Things(maths.Local()); err != nil {
			t.Errorf("%s: %v", detail, err)
		}
		for index, each := range standParts(detail) {
			for _, triangle := range each.skin() {
				if !(triangle[1].Sub(triangle[0]).Cross(triangle[2].Sub(
					triangle[0])).Len()/2 > j58Area) {
					t.Fatalf("%s: part %d has a sliver", detail, index)
				}
			}
		}
	}
}

// Every part of the stand wears its own look.
func TestTheJ58StandWearsItsLooks(t *testing.T) {
	dressed, err := j58StandRead.looks()
	if err != nil {
		t.Fatal(err)
	}
	if len(dressed) != len(standNames) {
		t.Errorf("the stand wears %d looks, want %d", len(dressed),
			len(standNames))
	}
	for where, look := range dressed {
		if look.Name != standNames[where] {
			t.Errorf("look %d is %q", where, look.Name)
		}
	}
	if dressed[standBell].Metal == nil || dressed[standPad].Metal != nil {
		t.Errorf("the bellmouth should be metal and the pad not")
	}
}

// Sizes that cannot hold the engine are refused when the file is read,
// by the name of what is wrong.
func TestAJ58StandThatCannotBeBuiltIsRefused(t *testing.T) {
	whole := string(j58StandRaw)
	refused := map[string]struct{ was, is, names string }{
		"no pad":         {"long: 7.5", "long: 0", "pad long"},
		"a low axis":     {"axis: 1.5", "axis: 1.0", "cradle"},
		"posts too near": {"out: 0.3", "out: 0.05", "cradle"},
		"long pins":      {"reach: 0.05", "reach: 0.2", "cradle"},
		"a narrow bell":  {"lip: 0.9", "lip: 0.5", "bellmouth"},
		"a short pad":    {"long: 7.5", "long: 5", "pad"},
		"a narrow pad":   {"wide: 3.6", "wide: 1.5", "pad"},
		"one flexure":    {"count: 3", "count: 1", "flexures"},
		"no concrete":    {"material: none", "material: concrete", "concrete"},
		"a bright pad":   {"[0.52, 0.51, 0.49]", "[1.52, 0.51, 0.49]", "colour"},
		"read nowhere":   {"read: >-", "unread: >-", "unread"},
	}
	for name, each := range refused {
		if !strings.Contains(whole, each.was) {
			t.Fatalf("%s: the file has no %q to change", name, each.was)
		}
		changed := strings.Replace(whole, each.was, each.is, 1)
		_, err := readJ58Stand([]byte(changed), j58Read)
		if err == nil {
			t.Errorf("%s: read without refusal", name)
			continue
		}
		if !strings.Contains(err.Error(), each.names) {
			t.Errorf("%s: refused as %q, which does not name %q", name,
				err, each.names)
		}
	}
}

// Sizes one step inside each limit are read: the anti-test of the
// refusals above.
func TestAJ58StandOneStepInsideItsLimitsIsRead(t *testing.T) {
	whole := string(j58StandRaw)
	accepted := map[string]struct{ was, is string }{
		"a lower axis":  {"axis: 1.5", "axis: 1.2"},
		"two flexures":  {"count: 3", "count: 2"},
		"a smaller pad": {"long: 7.5", "long: 6.5"},
		"a wide bell":   {"lip: 0.9", "lip: 1.1"},
	}
	for name, each := range accepted {
		if !strings.Contains(whole, each.was) {
			t.Fatalf("%s: the file has no %q to change", name, each.was)
		}
		changed := strings.Replace(whole, each.was, each.is, 1)
		if _, err := readJ58Stand([]byte(changed), j58Read); err != nil {
			t.Errorf("%s: refused as %v", name, err)
		}
	}
}
