package manufacture

import (
	"errors"
	"math"
	"sort"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

func mustMake(t *testing.T, name string, number uint64) good {
	t.Helper()
	row, err := kindNamed(name)
	if err != nil {
		t.Fatal(err)
	}
	made, err := makeOf(row, sow(number))
	if err != nil {
		t.Fatalf("%s from seed %d: %v", name, number, err)
	}
	return made
}

// the solid kinds: everything asked for that is not a light.
var solids = []string{"spruce", "norway", "fern", "pebble", "cobble",
	"boulder", "log", "can", "bonfire", "home"}

// one number in, a whole thing out: every kind, from forty seeds,
// makes a thing slayer's mesh takes, already in its own looks.
func TestEveryKindMakesASoundThingFromAnySeed(t *testing.T) {
	for _, name := range solids {
		for number := range uint64(40) {
			made := mustMake(t, name, number)
			things, err := made.things(maths.Subspace{})
			if err != nil {
				t.Fatalf("%s, seed %d: %v", name, number, err)
			}
			if len(things) == 0 || things[0].Material == nil {
				t.Fatalf("%s, seed %d: %d things", name, number,
					len(things))
			}
		}
	}
}

// a surface gives back no more light than reaches it: every colour a kind is
// dressed in is a share of each colour of light, from zero to one, and so is
// how much of a mirror it is. a colour is picked by eye; that it is a colour at
// all is physics.
func TestEveryLookIsAShareOfTheLight(t *testing.T) {
	t.Parallel()
	for _, name := range kindNames() {
		for _, look := range mustMake(t, name, 1).dressed {
			colour := look.Colour
			for _, share := range []float64{colour.R, colour.G, colour.B,
				look.Gloss} {
				if share < 0 || share > 1 || math.IsNaN(share) {
					t.Errorf("%s's %s gives back %v of the light, "+
						"glossed %v", name, look.Name, colour, look.Gloss)
				}
			}
		}
	}
}

func TestTheSameSeedIsTheSameThingAndAnotherIsNot(t *testing.T) {
	for _, name := range solids {
		if name == "can" {
			continue // every can is the same can; see its own test
		}
		one, same := mustMake(t, name, 3), mustMake(t, name, 3)
		other := mustMake(t, name, 4)
		if len(one.parts) != len(same.parts) {
			t.Fatalf("%s: the same seed made %d parts and then %d",
				name, len(one.parts), len(same.parts))
		}
		for index := range one.parts {
			if one.parts[index] != same.parts[index] {
				t.Fatalf("%s: part %d changed for one seed",
					name, index)
			}
		}
		if len(other.parts) == len(one.parts) &&
			other.parts[0] == one.parts[0] {
			t.Errorf("%s: another seed made the same thing", name)
		}
	}
}

// reach is how far a made thing reaches along x, y and z.
func reach(t *testing.T, made good) (float64, float64, float64) {
	t.Helper()
	var all surface
	for _, skin := range made.skins() {
		all = append(all, skin...)
	}
	wide, tall, deep, _ := extent(all)
	return wide, tall, deep
}

// a seed picks a thing's size from the range its kind allows, and
// over forty seeds the picks spread across that range.
func TestASeedPicksInsideItsKindsRange(t *testing.T) {
	sizes := map[string]span{
		"spruce":  spruceRow().tall,
		"norway":  norwayRow().tall,
		"pebble":  stoneRows()["pebble"].long,
		"cobble":  stoneRows()["cobble"].long,
		"boulder": stoneRows()["boulder"].long,
	}
	for name, allowed := range sizes {
		low, high := math.Inf(1), math.Inf(-1)
		for number := range uint64(40) {
			made := mustMake(t, name, number)
			size, _, _ := reach(t, made)
			if name == "spruce" || name == "norway" {
				// its height is its trunk's; a top branch may
				// reach a little above the trunk's own tip
				trunk := made.parts[0].shape.(tube)
				size = inScene(trunk.length)
			}
			if !allowed.holds(size) {
				t.Fatalf("%s from seed %d is %v, outside %v",
					name, number, size, allowed)
			}
			low, high = math.Min(low, size), math.Max(high, size)
		}
		if high-low < (allowed.most-allowed.least)/2 {
			t.Errorf("%s: forty seeds only pick from %v to %v of "+
				"%v to %v", name, low, high, allowed.least,
				allowed.most)
		}
	}
}

// a pebble is smaller than a cobble and a cobble than a boulder, with
// no size that is two of them.
func TestStonesAreNamedBySize(t *testing.T) {
	rows := stoneRows()
	pebble, cobble, boulder := rows["pebble"].long, rows["cobble"].long,
		rows["boulder"].long
	if pebble.most > cobble.least || cobble.most > boulder.least {
		t.Errorf("the stones' sizes overlap: %v, %v, %v", pebble,
			cobble, boulder)
	}
}

// a can is the can makers' can whatever its seed, and the seed says
// only whether it stands or lies. the makers' 211 by 413 is 2 11/16
// inches across and 4 13/16 tall: 68.2625 and 122.2375 mm, written
// here in millimetres and not read back from the row, so a row that
// swapped the two would not agree with itself and pass.
func TestACanIsACanStandingOrLying(t *testing.T) {
	const tall, across = 0.1222375, 0.0682625
	standing, lying := 0, 0
	for number := range uint64(40) {
		wide, high, deep := reach(t, mustMake(t, "can", number))
		sizes := []float64{wide, high, deep}
		sort.Float64s(sizes)
		if math.Abs(sizes[2]-tall) > 1e-9 {
			t.Fatalf("seed %d: a can %v long, not %v", number,
				sizes, tall)
		}
		// across, a can of sixteen flat sides is its width from corner
		// to corner one way and a little less from flat to flat
		for _, width := range sizes[:2] {
			if width > across+1e-9 || width < 0.98*across {
				t.Fatalf("seed %d: a can %v across, not %v",
					number, width, across)
			}
		}
		if high == sizes[2] {
			standing++
		} else {
			lying++
		}
	}
	if standing == 0 || lying == 0 {
		t.Errorf("%d cans stand and %d lie", standing, lying)
	}
}

// a log lies on the ground: far longer than it is high, and its
// trunk resting on the ground, not sunk in it and not above it. a
// trunk of flat sides rests on a corner or on a flat as it is rolled,
// so its lowest point is within a twentieth of its height of the
// ground, and never under it.
func TestALogLiesOnTheGround(t *testing.T) {
	for number := range uint64(20) {
		made := mustMake(t, "log", number)
		trunk := made.parts[0].skin()
		wide, tall, _, lowest := extent(trunk)
		if wide < 4*tall {
			t.Errorf("seed %d: a log %v long and %v high", number,
				wide, tall)
		}
		if lowest < -1e-9 || lowest > tall/20 {
			t.Errorf("seed %d: its trunk's lowest point is at %v",
				number, lowest)
		}
		if len(made.parts) < 2 {
			t.Errorf("seed %d: a log with no stubs", number)
		}
	}
}

func TestAKindIsFoundByItsName(t *testing.T) {
	names := kindNames()
	for _, name := range solids {
		row, err := kindNamed(name)
		if err != nil || row.name != name {
			t.Errorf("%s: %v, %v", name, row.name, err)
		}
		if !strings.Contains(strings.Join(names, " "), name) {
			t.Errorf("%s is not among the kinds: %v", name, names)
		}
	}
	_, err := kindNamed("ottoman")
	if !errors.Is(err, errKind) {
		t.Fatalf("an ottoman: %v, not %v", err, errKind)
	}
	// the refusal says what there is, so nobody has to go and read
	if !strings.Contains(err.Error(), "boulder") {
		t.Errorf("the refusal does not list the kinds: %v", err)
	}
}

func TestMakingFromNoKindOrNoSeedIsRefused(t *testing.T) {
	row, err := kindNamed("fern")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := makeOf(kind{}, sow(1)); !errors.Is(err, errKind) {
		t.Errorf("a kind never made: %v, not %v", err, errKind)
	}
	if _, err := makeOf(row, seed{}); !errors.Is(err, errGood) {
		t.Errorf("a seed never sown: %v, not %v", err, errGood)
	}
}

// holds is whether a number is in a span, give or take rounding.
func (of span) holds(number float64) bool {
	return number >= of.least-close && number <= of.most+close
}

func TestASpanPicksFromItsLeastToItsMost(t *testing.T) {
	of := span{least: 2, most: 6}
	picks := map[float64]float64{0: 2, 0.5: 4, 1: 6}
	for share, want := range picks {
		if got := of.pick(share); got != want {
			t.Errorf("a share of %v picks %v, not %v", share, got,
				want)
		}
	}
	whole := map[float64]int{0: 2, 0.19: 2, 0.21: 3, 0.99: 6, 1: 6}
	for share, want := range whole {
		if got := of.count(share); got != want {
			t.Errorf("a share of %v counts %d, not %d", share, got,
				want)
		}
	}
}

// a thing is grown from its own seed, not only sized by it: beyond the
// few numbers a row picks, which way a spruce's first branch points,
// a fern's first frond and a log's first stub are the seed's too, and
// forty seeds point them forty ways. a stone's lump holds a sprout of
// the seed it was made from.
func TestAThingGrowsFromItsOwnSeed(t *testing.T) {
	for _, name := range []string{"spruce", "fern", "log"} {
		ways := map[maths.Vec]bool{}
		for number := range uint64(40) {
			first := mustMake(t, name, number).parts[1]
			ways[first.space.Orientation.Y] = true
		}
		if len(ways) < 30 {
			t.Errorf("%s: forty seeds point its first part "+
				"only %d ways", name, len(ways))
		}
	}
	for _, name := range []string{"pebble", "cobble", "boulder"} {
		stone := mustMake(t, name, 6).parts[0].shape.(lump)
		if want := sow(6).sprout(growing, 0); stone.from != want {
			t.Errorf("%s: its lump grows from %v, not a sprout of "+
				"its seed", name, stone.from)
		}
	}
}
