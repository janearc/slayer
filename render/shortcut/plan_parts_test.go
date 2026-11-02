package shortcut

import (
	"reflect"
	"testing"
)

// a part takes the set of the longest thing whose name it is, or begins
// with followed by a space; else the global. the things are told apart
// by the one field each has on, and the global by baked.
//
// "jupiter ring dust" begins both "jupiter" and "jupiter ring", and the
// longer is the one. "jupiter rings" does not begin "jupiter ring" and
// a space, so it is jupiter's, and "jupiterx" begins no name and a
// space, so it is nobody's.
//
// the loop runs each case many times because a map is walked in no
// order, and the longest must win in all of them.
func TestAPartTakesTheLongestThingItBeginsWith(t *testing.T) {
	global := Set{Baked: true}
	p := Plan{Global: global, Things: map[string]Set{
		"jupiter":      {Skip: true},
		"jupiter ring": {Once: true},
		"jup":          {Cutoff: true},
		"io":           {Distance: true},
	}}
	for _, c := range []struct {
		part string
		want Set
	}{
		{"jupiter", Set{Skip: true}},
		{"jupiter gas", Set{Skip: true}},
		{"jupiter air", Set{Skip: true}},
		{"jupiter ring", Set{Once: true}},
		{"jupiter ring dust", Set{Once: true}},
		{"jupiter rings", Set{Skip: true}},
		{"jupiter ringed moon", Set{Skip: true}},
		{"jupiterx", global},
		{"jup", Set{Cutoff: true}},
		{"jup moon", Set{Cutoff: true}},
		{"jupit", global},
		{"ju", global},
		{"io", Set{Distance: true}},
		{"io moon", Set{Distance: true}},
		{"iota", global},
		{"i", global},
		{"saturn", global},
		{"Jupiter", global},
		{" jupiter", global},
		{"", global},
	} {
		for rep := 0; rep < 40; rep++ {
			if got := p.ForPart(c.part); got != c.want {
				t.Fatalf("the part %q takes %+v, want %+v", c.part, got, c.want)
			}
		}
	}
	if got := (Plan{Global: global}).ForPart("jupiter gas"); got != global {
		t.Errorf("a plan with no things gave %+v", got)
	}
	if got := (Plan{}).ForPart("jupiter gas"); got != (Set{}) {
		t.Errorf("an empty plan gave %+v", got)
	}
}

// a thing with no name takes no part. a thing is found only when its
// name is longer than nothing, so a part with no name, as a volume
// with no material has, is the global's, and so is a part that begins
// with a space.
//
// this is how ForPart is written and no document says otherwise; it
// is held so that a change to it is seen.
func TestAThingWithNoNameTakesNoPart(t *testing.T) {
	global := Set{Baked: true}
	p := Plan{Global: global, Things: map[string]Set{"": {Skip: true}}}
	for _, part := range []string{"", " moon", "smoke"} {
		if got := p.ForPart(part); got != global {
			t.Errorf("the part %q takes %+v, want the global", part, got)
		}
	}
}

// For is the thing's own set by its whole name, or the global when the
// name is not there; it does not read a name by its beginning, and a
// thing whose own set is nothing is still its own, not the global.
func TestForTakesTheWholeNameOrTheGlobal(t *testing.T) {
	global := All()
	p := Plan{Global: global, Things: map[string]Set{
		"jupiter": {Skip: true},
		"quiet":   {},
	}}
	if got := p.For("jupiter"); got != (Set{Skip: true}) {
		t.Errorf("jupiter takes %+v", got)
	}
	if got := p.For("quiet"); got != (Set{}) {
		t.Errorf("a thing set to nothing takes %+v", got)
	}
	for _, name := range []string{"saturn", "jupiter gas", "jup", "Jupiter", ""} {
		if got := p.For(name); got != global {
			t.Errorf("%q is not a thing and takes %+v, want the global", name, got)
		}
	}
	if got := (Plan{Global: global}).For("jupiter"); got != global {
		t.Errorf("a plan with no things gave %+v", got)
	}
}

// With is a new plan: the plan it was made from keeps its things, a
// second thing keeps the first, and a thing named again takes the new
// set.
func TestWithMakesANewPlan(t *testing.T) {
	base := Plan{Global: Set{Cutoff: true}}
	one := base.With("a", Set{Skip: true})
	two := one.With("b", Set{Once: true})
	three := two.With("a", Set{Distance: true})
	if len(base.Things) != 0 || len(one.Things) != 1 || len(two.Things) != 2 {
		t.Fatalf("the things number %d, %d and %d; want 0, 1 and 2",
			len(base.Things), len(one.Things), len(two.Things))
	}
	if one.Things["a"] != (Set{Skip: true}) || two.Things["a"] != (Set{Skip: true}) {
		t.Errorf("a was changed under an earlier plan: %+v, %+v", one.Things["a"], two.Things["a"])
	}
	want := map[string]Set{"a": {Distance: true}, "b": {Once: true}}
	if !reflect.DeepEqual(three.Things, want) {
		t.Errorf("the third plan has %+v, want %+v", three.Things, want)
	}
	for _, p := range []Plan{one, two, three} {
		if p.Global != (Set{Cutoff: true}) {
			t.Errorf("the global became %+v", p.Global)
		}
	}
}

// the record is the global then each thing, in name order whatever
// order they went in.
func TestTheRecordIsInNameOrder(t *testing.T) {
	if got := (Plan{}).Record(); !reflect.DeepEqual(got, []string{"global: none"}) {
		t.Errorf("an empty plan records %q", got)
	}
	p := Plan{Global: Set{Baked: true, Cutoff: true}}.
		With("c", Set{Skip: true}).With("a", All()).With("b", Set{Baked: true, Resolution: 3})
	want := []string{"global: baked cutoff", "a: all", "b: baked(3)", "c: skip"}
	if got := p.Record(); !reflect.DeepEqual(got, want) {
		t.Errorf("the record is %q, want %q", got, want)
	}
}
