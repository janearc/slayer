package cities

import (
	"math"
	"slices"
	"strings"
	"testing"
)

// theGround is a whole circle of the radius, about the map's origin, with
// a hole in the middle the size of a vehicle's pitch.
func theGround(far float64) Ground {
	return Ground{Near: 12, Far: far, To: fullDegrees}
}

// anItem is the winnebago's footprint, from its data, or the test's end.
func anItem(t *testing.T, kind string) Item {
	t.Helper()
	item, err := ItemOf(kind)
	if err != nil {
		t.Fatal(err)
	}
	return item
}

// sitesOf is the sites a habit lays, or the test's end.
func sitesOf(t *testing.T, habit Habit, ground Ground, item Item, seed Seed,
	most int) Laid {
	t.Helper()
	laid, err := habit.Sites(ground, item, seed, most)
	if err != nil {
		t.Fatal(err)
	}
	return laid
}

func TestLatticePitchesNeverOverlap(t *testing.T) {
	item := anItem(t, "winnebago")
	sites := sitesOf(t, lattice{}, theGround(150), item, 1, 0).Sites
	laneEast, laneNorth := toward(item.Pitch.Heading)
	for first := range sites {
		for _, second := range sites[first+1:] {
			east, north := second.East-sites[first].East,
				second.North-sites[first].North
			along := math.Abs(east*laneEast + north*laneNorth)
			across := math.Abs(east*laneNorth - north*laneEast)
			if along < item.Pitch.Depth-tolerance &&
				across < item.Pitch.Width-tolerance {
				t.Fatalf("pitches at %v and %v overlap", sites[first], second)
			}
		}
	}
	if len(sites) < 100 {
		t.Errorf("only %d pitches in a circle 150 m across", len(sites))
	}
}

func TestEveryLatticeSiteLiesInsideTheGround(t *testing.T) {
	item := anItem(t, "winnebago")
	for name, ground := range map[string]Ground{
		"a circle": theGround(150),
		"a wedge":  {Near: 12, Far: 150, From: 45, To: 135},
		"a wedge past north": {East: 40, North: -30, Near: 12, Far: 150,
			From: 300, To: 60},
	} {
		sites := sitesOf(t, lattice{}, ground, item, 1, 0).Sites
		if len(sites) == 0 {
			t.Errorf("%s: no pitch fits", name)
		}
		for _, site := range sites {
			if !ground.contains(site.East, site.North) ||
				!holds(ground, site, item.Pitch) {
				t.Fatalf("%s: the site %v is not wholly inside", name, site)
			}
		}
	}
}

func TestEveryLatticeItemFacesAlongItsLaneTheSameWay(t *testing.T) {
	item := anItem(t, "winnebago")
	for _, site := range sitesOf(t, lattice{}, theGround(150), item, 1, 0).Sites {
		if !site.Faced || site.Heading != item.Pitch.Heading {
			t.Fatalf("the site %v does not face along the lanes", site)
		}
	}
}

// The park of sets/winnebago-field.yaml: a disc of radius 564 m with a
// hole of 12 m. Its count is an output, and is logged for the set's
// comment; it can be no more than the ground over the pitch and its lane.
func TestTheParkOfTheFieldSetFitsFewerThanTheGroundOverAPitch(t *testing.T) {
	item := anItem(t, "winnebago")
	ground := Ground{Near: 12, Far: 564, To: fullDegrees}
	count := len(sitesOf(t, lattice{}, ground, item, 2, 0).Sites)
	t.Logf("the park of the field set fits %d pitches", count)
	ceiling := math.Pi * ground.Far * ground.Far /
		(item.Pitch.Depth * (item.Pitch.Width + item.Pitch.Lane))
	if float64(count) > ceiling || count < 3000 {
		t.Errorf("%d pitches, under the ceiling of %v", count, ceiling)
	}
}

func TestAWiderLaneFitsFewerPitches(t *testing.T) {
	item := anItem(t, "winnebago")
	narrowGround := theGround(300)
	narrow := len(sitesOf(t, lattice{}, narrowGround, item, 1, 0).Sites)
	item.Pitch.Lane *= 2
	wideGround := theGround(300)
	wide := len(sitesOf(t, lattice{}, wideGround, item, 1, 0).Sites)
	if wide >= narrow {
		t.Errorf("%d pitches with the lane doubled, %d without", wide, narrow)
	}
}

func TestALatticeGivenACountTakesThatManyOfThosePitchesThatFit(t *testing.T) {
	item, ground := anItem(t, "winnebago"), theGround(150)
	all := sitesOf(t, lattice{}, ground, item, 1, 0).Sites
	some := sitesOf(t, lattice{}, ground, item, 1, 10)
	if len(some.Sites) != 10 {
		t.Fatalf("%d sites, want ten", len(some.Sites))
	}
	for _, site := range some.Sites {
		if !slices.Contains(all, site) {
			t.Errorf("the site %v is not one that fits", site)
		}
	}
	more := sitesOf(t, lattice{}, ground, item, 1, len(all)+5)
	if len(more.Sites) != len(all) || !strings.Contains(more.Said,
		"fewer than the") {
		t.Errorf("%d sites and %q when %d fit and more were asked for",
			len(more.Sites), more.Said, len(all))
	}
}

func TestTheSameSeedMakesTheSameSites(t *testing.T) {
	item, ground := anItem(t, "winnebago"), theGround(150)
	yurt := anItem(t, "yurt")
	for name, lay := range map[string]func(Seed) []Site{
		"lattice": func(seed Seed) []Site {
			return sitesOf(t, lattice{}, ground, item, seed, 20).Sites
		},
		"scatter": func(seed Seed) []Site {
			return sitesOf(t, scatter{}, ground, yurt, seed, 20).Sites
		},
	} {
		if !slices.Equal(lay(7), lay(7)) {
			t.Errorf("%s: seed 7 made two layouts", name)
		}
		if slices.Equal(lay(7), lay(8)) {
			t.Errorf("%s: two seeds made one layout", name)
		}
	}
}

func TestAScatterFacesWithinTheSpreadOfThePreferredHeading(t *testing.T) {
	yurt := anItem(t, "yurt")
	sites := sitesOf(t, scatter{}, theGround(500), yurt, 3, 500).Sites
	if len(sites) != 500 {
		t.Fatalf("%d sites, want 500", len(sites))
	}
	for _, site := range sites {
		off := math.Abs(site.Heading - yurt.Facing.Heading)
		if !site.Faced || off > yurt.Facing.Spread+tolerance {
			t.Fatalf("the site %v faces %v off the preferred way, spread "+
				"%v", site, off, yurt.Facing.Spread)
		}
	}
	if yurt.Facing.Heading != 180 {
		t.Errorf("gers face %v, want south", yurt.Facing.Heading)
	}
}

func TestAScatterOfItemsWithNoPreferenceLeavesTheFacingAlone(t *testing.T) {
	item := anItem(t, "winnebago")
	for _, site := range sitesOf(t, scatter{}, theGround(500), item, 3, 50).Sites {
		if site.Faced {
			t.Fatalf("the site %v is faced, and the data gives no facing", site)
		}
	}
}

func TestAScatterLiesInsideTheGround(t *testing.T) {
	ground := Ground{Near: 12, Far: 300, From: 300, To: 60}
	item := anItem(t, "yurt")
	for _, site := range sitesOf(t, scatter{}, ground, item, 4, 400).Sites {
		if !ground.contains(site.East, site.North) {
			t.Fatalf("the site %v is outside the wedge", site)
		}
	}
}

func TestAWedgeRunsRoundPastNorth(t *testing.T) {
	ground := Ground{Far: 100, From: 350, To: 10}
	for place, want := range map[[2]float64]bool{
		{0, 50}: true, {-5, 50}: true, {5, 50}: true, {50, 0}: false,
		{0, -50}: false, {0, 150}: false,
	} {
		if got := ground.contains(place[0], place[1]); got != want {
			t.Errorf("%v: contained %v, want %v", place, got, want)
		}
	}
}

func TestTheDataGivesTheWinnebagoAPitchAndTheYurtAFacing(t *testing.T) {
	winnebago, yurt := anItem(t, "winnebago"), anItem(t, "yurt")
	if !winnebago.Pitched || winnebago.Faced || winnebago.Pitch.Width != 8 ||
		winnebago.Pitch.Depth != 15 || winnebago.Pitch.Lane != 6 {
		t.Errorf("the winnebago is %+v", winnebago)
	}
	if !yurt.Faced || yurt.Pitched {
		t.Errorf("the yurt is %+v", yurt)
	}
	if bare := anItem(t, "nothing"); bare.Pitched || bare.Faced {
		t.Errorf("a kind with no data is %+v", bare)
	}
	facts, err := WinnebagoFacts()
	if err != nil {
		t.Fatal(err)
	}
	area := winnebago.Pitch.Width * winnebago.Pitch.Depth
	if area != facts.PitchArea {
		t.Errorf("the pitch is %v square metres by its sides and %v by "+
			"its area", area, facts.PitchArea)
	}
}

func TestALatticeOfAnItemWithNoPitchIsRefused(t *testing.T) {
	_, err := lattice{}.Sites(theGround(100), anItem(t, "yurt"), 1, 0)
	if err == nil || !strings.Contains(err.Error(), "yurt") {
		t.Errorf("a lattice of gers: %v", err)
	}
	pitch := anItem(t, "winnebago")
	pitch.Pitch.Depth = 0
	if _, err := (lattice{}).Sites(theGround(100), pitch, 1, 0); err == nil {
		t.Error("a lattice of pitches with no depth was laid")
	}
}

func TestAScatterWithNoCountIsRefused(t *testing.T) {
	_, err := scatter{}.Sites(theGround(100), anItem(t, "yurt"), 1, 0)
	if err == nil {
		t.Error("a scatter with no count was laid")
	}
}

func TestAHabitNoOneKnowsIsRefused(t *testing.T) {
	if _, err := HabitNamed("sprawl"); err == nil {
		t.Error("a habit nobody named was found")
	}
	for _, name := range HabitNames() {
		if _, err := HabitNamed(name); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestAPartOfAPitchIsRefused(t *testing.T) {
	const part = "numbers:\n- name: ledge-pitch-width\n  value: 8\n" +
		"  says: a pitch\n  kind: picked\n  by: test\n"
	book, err := readChoices("test.yaml", []byte(part))
	if err != nil {
		t.Fatal(err)
	}
	saved := choiceBook
	choiceBook = book
	defer func() { choiceBook = saved }()
	if _, err := ItemOf("ledge"); err == nil {
		t.Error("a pitch with a width and nothing else was read")
	}
}
