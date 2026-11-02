package shortcut

import (
	"strconv"
	"strings"
	"testing"
)

// each word parses to its own field and no other, and a set prints as
// the words in one fixed order: baked, skip, distance, once, cutoff.
func TestEachWordIsItsOwnField(t *testing.T) {
	for _, c := range []struct {
		text string
		want Set
	}{
		{"baked", Set{Baked: true}},
		{"baked(2)", Set{Baked: true, Resolution: 2}},
		{"baked(12)", Set{Baked: true, Resolution: 12}},
		{"baked(+8)", Set{Baked: true, Resolution: 8}},
		{"baked(256)", Set{Baked: true, Resolution: 256}},
		{"skip", Set{Skip: true}},
		{"distance", Set{Distance: true}},
		{"once", Set{Once: true}},
		{"cutoff", Set{Cutoff: true}},
		{"skip once", Set{Skip: true, Once: true}},
		{"cutoff baked(5)", Set{Baked: true, Resolution: 5, Cutoff: true}},
		{"baked(4) baked(9)", Set{Baked: true, Resolution: 9}},
		{"baked(9) baked(4)", Set{Baked: true, Resolution: 4}},
		{"baked(9) baked", Set{Baked: true, Resolution: 9}},
		{"baked skip  distance\tonce\ncutoff", All()},
		{"cutoff once distance skip baked", All()},
		{"  none ", Set{}},
		{"\tall\n", All()},
		{"", Set{}},
		{"   ", Set{}},
	} {
		got, err := Parse(c.text)
		if err != nil || got != c.want {
			t.Errorf("%q parses as %+v, %v; want %+v", c.text, got, err, c.want)
		}
	}
}

// a set prints as a sheet writes it. none is zero on every field, all is every
// field and no resolution; a resolution with the baked word off is not printed,
// since there is nothing to bake; a resolution that is not greater than zero is
// not printed.
func TestASetPrintsAsTheWords(t *testing.T) {
	for _, c := range []struct {
		set  Set
		want string
	}{
		{Set{}, "none"},
		{Set{Resolution: 40}, "none"},
		{All(), "all"},
		{Set{Baked: true}, "baked"},
		{Set{Baked: true, Resolution: -3}, "baked"},
		{Set{Baked: true, Resolution: 1}, "baked(1)"},
		{Set{Baked: true, Resolution: 64}, "baked(64)"},
		{Set{Skip: true}, "skip"},
		{Set{Distance: true}, "distance"},
		{Set{Once: true}, "once"},
		{Set{Cutoff: true}, "cutoff"},
		{Set{Skip: true, Resolution: 40}, "skip"},
		{Set{Cutoff: true, Once: true, Distance: true, Skip: true}, "skip distance once cutoff"},
		{Set{Cutoff: true, Baked: true, Skip: true}, "baked skip cutoff"},
		{Set{Baked: true, Skip: true, Distance: true, Once: true, Cutoff: true, Resolution: 64},
			"baked(64) skip distance once cutoff"},
		{Set{Baked: true, Skip: true, Distance: true, Once: true}, "baked skip distance once"},
		{Set{Skip: true, Distance: true, Once: true, Cutoff: true}, "skip distance once cutoff"},
	} {
		if got := c.set.String(); got != c.want {
			t.Errorf("%+v prints as %q, want %q", c.set, got, c.want)
		}
	}
}

// what a set prints, Parse reads back as the set, for every one of the
// thirty-two ways to turn five shortcuts on or off, at no resolution and at
// three; a resolution that was never printed is not read back.
func TestASetRoundTrips(t *testing.T) {
	for mask := 0; mask < 32; mask++ {
		for _, res := range []int{0, 2, 7, 256} {
			s := Set{Baked: mask&1 != 0, Skip: mask&2 != 0, Distance: mask&4 != 0,
				Once: mask&8 != 0, Cutoff: mask&16 != 0}
			if s.Baked {
				s.Resolution = res
			}
			text := s.String()
			back, err := Parse(text)
			if err != nil || back != s {
				t.Errorf("%+v prints as %q and reads back as %+v, %v", s, text, back, err)
			}
			if again := back.String(); again != text {
				t.Errorf("%q reads back and prints as %q", text, again)
			}
		}
	}
}

// Any is whether one of the five is on, and a resolution alone is not.
func TestAnyIsOneOfFive(t *testing.T) {
	if (Set{}).Any() || (Set{Resolution: 64}).Any() || None().Any() {
		t.Error("a set with nothing on has something on")
	}
	for _, s := range []Set{{Baked: true}, {Skip: true}, {Distance: true},
		{Once: true}, {Cutoff: true}, All()} {
		if !s.Any() {
			t.Errorf("%+v has one on and says it has none", s)
		}
	}
}

// Parse refuses what it cannot read and hands back no set: nothing from
// words before the bad one. a baked word wants a number from two to 256
// and says so; any other word is named with the words there are.
func TestParseRefusesWhatItCannotRead(t *testing.T) {
	for _, text := range []string{"baked(1)", "baked(0)", "baked(-2)", "baked(-8)",
		"baked(257)", "baked(x)", "baked()", "baked(3.5)",
		"skip baked(1)", "once baked(999)"} {
		got, err := Parse(text)
		if err == nil {
			t.Errorf("%q passed as %+v", text, got)
			continue
		}
		if got != (Set{}) {
			t.Errorf("%q was refused and still gave %+v", text, got)
		}
		if !strings.Contains(err.Error(), "from two to 256") {
			t.Errorf("%q: %v", text, err)
		}
	}
	for _, text := range []string{"fast", "NONE", "Baked", "none all", "all skip", "none skip",
		"baked(4", "baked4)", "baked( 4)", "baked(4)x", "xbaked(4)", "skip,once", "skip fast", "baked)"} {
		got, err := Parse(text)
		if err == nil {
			t.Errorf("%q passed as %+v", text, got)
			continue
		}
		if got != (Set{}) {
			t.Errorf("%q was refused and still gave %+v", text, got)
		}
		if !strings.Contains(err.Error(), "none of none, all, baked, baked(N), skip, distance, once, cutoff") {
			t.Errorf("%q: %v", text, err)
		}
	}
	_, err := Parse("skip fast")
	if err == nil || !strings.Contains(err.Error(), strconv.Quote("fast")) {
		t.Errorf("the bad word is not named: %v", err)
	}
	_, err = Parse("baked(300)")
	if err == nil || !strings.Contains(err.Error(), strconv.Quote("baked(300)")) {
		t.Errorf("the bad baked word is not named: %v", err)
	}
}
