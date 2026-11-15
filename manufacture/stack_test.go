package manufacture

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The Apollo command and service module's sizes, as stacks.yaml gives
// them and where it says they were read.
const (
	apolloAcross = 3.91  // read: NASA, Apollo 204 spacecraft page, 12 ft 10 in
	apolloTall   = 10.11 // defined: the three sections' lengths, 2.36 + 4.52 + 3.23
	stackNear    = 1e-9  // picked: a sum of three lengths, to rounding
)

// The file of stacks reads whole, and every craft in it is a kind.
func TestEveryStackInTheFileIsAKind(t *testing.T) {
	if stacksError != nil {
		t.Fatal(stacksError)
	}
	if len(stacksRead) == 0 {
		t.Fatal("the file lists no stacks")
	}
	for _, row := range stacksRead {
		if _, err := KindNamed(row.Name); err != nil {
			t.Errorf("%s is in the file and is no kind: %v", row.Name, err)
		}
	}
}

// A stack stands on its own zero, as wide as its widest section and as
// tall as its sections are long, with a thing for every section in the
// section's own look. The anti-test: nothing of it is outside its box.
func TestAStackIsItsSectionsOneOnTheNext(t *testing.T) {
	made := mustMake(t, "apollo-csm", 5)
	box := made.Bounds()
	if math.Abs(box.Min.Y) > stackNear || math.Abs(box.Max.Y-apolloTall) > stackNear {
		t.Errorf("the stack stands from %v to %v, want nought to %v",
			box.Min.Y, box.Max.Y, apolloTall)
	}
	if math.Abs(box.Max.X-apolloAcross/2) > stackNear ||
		math.Abs(box.Min.Z+apolloAcross/2) > stackNear {
		t.Errorf("the stack's box is %+v, want %v across", box, apolloAcross)
	}
	things, err := made.Things(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"engine bell", "service module", "command module"}
	if len(things) != len(want) {
		t.Fatalf("the stack is %d things, want %d", len(things), len(want))
	}
	for at, each := range things {
		if each.Material.Name != want[at] {
			t.Errorf("section %d is %q, want %q", at, each.Material.Name, want[at])
		}
		inside := each.Shape.Bounds()
		if inside.Min.Y < box.Min.Y-stackNear || inside.Max.Y > box.Max.Y+stackNear {
			t.Errorf("the %s runs from %v to %v, outside the stack's box",
				want[at], inside.Min.Y, inside.Max.Y)
		}
	}
	// the sections meet: each begins where the one below it ends
	bell, module := things[0].Shape.Bounds(), things[1].Shape.Bounds()
	if math.Abs(bell.Max.Y-module.Min.Y) > stackNear {
		t.Errorf("the bell ends at %v and the module begins at %v",
			bell.Max.Y, module.Min.Y)
	}
}

// What stands beside a stack stands round it, each against its foot's
// side: the Space Launch System is as wide as its core and a booster on
// either side, and as tall as the core's own sections.
func TestBoostersStandBesideTheirStack(t *testing.T) {
	const (
		core    = 8.4    // read: NASA, the core stage's width
		booster = 3.6576 // read: NASA, a booster's width, 12 feet
		tall    = 86.7   // defined: 64.6 + 8.4 + 13.7
	)
	made := mustMake(t, "sls-block-1", 5)
	box := made.Bounds()
	if math.Abs(box.Max.X-(core/2+booster)) > stackNear ||
		math.Abs(box.Max.Y-tall) > stackNear {
		t.Errorf("the stack's box is %+v, want %v out and %v tall", box,
			core/2+booster, tall)
	}
	parts, err := made.Things(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	// the booster cases reach out on both sides of the core and no
	// further: one against each side. However many things they are
	// handed back as, that is where their metal is.
	low, high := math.Inf(1), math.Inf(-1)
	for _, each := range parts {
		if each.Material.Name != "booster case" {
			continue
		}
		at, inside := each.At(0).Origin.X, each.Shape.Bounds()
		low = math.Min(low, at+inside.Min.X)
		high = math.Max(high, at+inside.Max.X)
	}
	want := core/2 + booster
	if math.Abs(high-want) > stackNear || math.Abs(low+want) > stackNear {
		t.Errorf("the booster cases reach from %v to %v, want %v either side",
			low, high, want)
	}
	// the anti-test: a stack with nothing beside it is no wider than itself
	plain := mustMake(t, "saturn-v", 5).Bounds()
	if math.Abs(plain.Max.X-10.0584/2) > stackNear {
		t.Errorf("a stack with no boosters reaches %v, want half of 10.0584",
			plain.Max.X)
	}
}

// The Apollo-Saturn V stands NASA's 363 feet: its sections are made to
// add up to it, and this holds them to it, so a section changed without
// the tower's remainder being worked again is caught. The anti-test is
// the launch vehicle alone, which is well short of it.
func TestTheApolloSaturnStandsNASAs363Feet(t *testing.T) {
	const (
		feet     = 0.3048 // defined: a foot, in metres
		onThePad = 363.0  // read: NASA, Apollo 4 press kit, the vehicle's height in feet with its spacecraft and escape system
	)
	whole := mustMake(t, "saturn-v-apollo", 5).Bounds()
	if math.Abs(whole.Max.Y-onThePad*feet) > 1e-9 {
		t.Errorf("the Apollo-Saturn V stands %v m, want %v", whole.Max.Y,
			onThePad*feet)
	}
	alone := mustMake(t, "saturn-v", 5).Bounds()
	if !(alone.Max.Y < onThePad*feet-20) {
		t.Errorf("the Saturn V alone stands %v m: not well short of the "+
			"whole vehicle", alone.Max.Y)
	}
}

// A painted section is the colour of its paint and no mirror: the Space
// Launch System's core is the rust swatch, redder than it is blue, and
// its boosters are chalk, near white. The anti-test is the Apollo
// service module, bare aluminium, which is a metal still.
func TestAPaintedSectionIsItsPaint(t *testing.T) {
	parts, err := mustMake(t, "sls-block-1", 5).Things(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, each := range parts {
		look := each.Material
		switch look.Name {
		case "core stage":
			seen++
			if look.Metal != nil || !(look.Colour.R > look.Colour.B) {
				t.Errorf("the core stage is %+v, metal %v: not rust paint",
					look.Colour, look.Metal != nil)
			}
		case "booster case":
			seen++
			if look.Metal != nil || !(look.Colour.B > 0.5) {
				t.Errorf("a booster case is %+v, metal %v: not chalk paint",
					look.Colour, look.Metal != nil)
			}
		}
	}
	if seen < 2 {
		t.Fatalf("saw %d of the core stage and a booster case", seen)
	}
	bare, err := mustMake(t, "apollo-csm", 5).Things(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	if bare[1].Material.Metal == nil {
		t.Error("the Apollo service module, bare aluminium, is no metal")
	}
}

// A stack that cannot be built is refused when the file is read, by the
// name of what is wrong with it.
func TestAStackThatCannotBeBuiltIsRefused(t *testing.T) {
	whole := string(stacksRaw)
	refused := map[string]struct{ was, is, names string }{
		"a section of no length": {"length: 4.52", "length: 0", "service module"},
		"a section of no width":  {"top: 0.6", "top: 0", "engine bell"},
		"a width below nought":   {"bottom: 2.501", "bottom: -1", "engine bell"},
		"a material not known":   {"material: titanium", "material: unobtainium", "unobtainium"},
		"a metal with no optics": {"material: titanium", "material: steel", "steel"},
		"a finish not known":     {"finish: weathered", "finish: sparkly", "sparkly"},
		"a craft with no name":   {"name: apollo-csm", "name: \"\"", "no name"},
		"a section with no part": {"part: service module", "part: \"\"", "apollo-csm"},
		"a key the file has not": {"length: 4.52", "lenght: 4.52", "lenght"},
		"a paint that is none":   {"paint: rust", "paint: mauve-ish", "mauve-ish"},
		"paint over no material": {"length: 64.6\n    material: aluminium", "length: 64.6\n    material: unobtainium", "unobtainium"},
		"paint on bare metal":    {"finish: weathered", "finish: weathered\n    paint: chalk", "not painted"},
		"no boosters beside":     {"count: 2", "count: 0", "sls-block-1"},
		"boosters underfoot":     {"raised: 0", "raised: -3", "sls-block-1"},
		"a booster of no width":  {"top: 0.6\n      length: 5.9496", "top: 0\n      length: 5.9496", "booster nose"},
	}
	for name, each := range refused {
		if !strings.Contains(whole, each.was) {
			t.Fatalf("%s: the file has no %q to change", name, each.was)
		}
		broken := strings.Replace(whole, each.was, each.is, 1)
		_, err := readStacks([]byte(broken))
		if err == nil {
			t.Errorf("%s: read without refusal", name)
			continue
		}
		if !strings.Contains(err.Error(), each.names) {
			t.Errorf("%s: refused as %q, which does not name %q", name, err,
				each.names)
		}
	}
	// whole files, for what the real one cannot be bent into
	bare := "stacks:\n- name: bare\n  says: nothing\n"
	tube := "  - part: tube\n    bottom: 1\n    top: 1\n    length: 1\n" +
		"    material: aluminium\n    finish: satin\n"
	files := map[string]string{
		"a craft with no sections": bare + "  sections: []\n",
		"a section read nowhere":   bare + "  sections:\n" + tube,
		"a name twice": bare + "  sections:\n" + tube + "    read: picked\n" +
			"- name: bare\n  says: again\n  sections:\n" + tube + "    read: picked\n",
	}
	for name, file := range files {
		_, err := readStacks([]byte(file))
		if err == nil || !strings.Contains(err.Error(), "bare") {
			t.Errorf("%s: read as %v, want a refusal that names the craft",
				name, err)
		}
	}
	// the anti-test: the same tube, read somewhere, is a craft
	sound := bare + "  sections:\n" + tube + "    read: picked\n"
	if read, err := readStacks([]byte(sound)); err != nil || len(read) != 1 {
		t.Errorf("a sound one-section craft read as %v, %v", read, err)
	}
}
