package cities

import (
	"math"
	"math/rand/v2"
	"testing"
)

// The constants of splitmix64, written again here so that the sum below is
// worked apart from the code under test.
const (
	workedStep   = 0x9e3779b97f4a7c15 // defined: 2^64 over the golden ratio
	workedFirst  = 0xbf58476d1ce4e5b9 // defined: splitmix64's first multiplier
	workedSecond = 0x94d049bb133111eb // defined: splitmix64's second multiplier
	workedAbove  = 30                 // defined: splitmix64's first shift
	workedMiddle = 27                 // defined: splitmix64's second shift
	workedLast   = 31                 // defined: splitmix64's last shift
	// workedSpan is the largest number of members a test asks for.
	workedSpan = 1000
	// workedSeed is a seed the tests lay from.
	workedSeed = 9
	// workedMember is a member the tests follow.
	workedMember = 4
)

// workedMix is the finishing step of splitmix64, worked here again.
func workedMix(number uint64) uint64 {
	number = (number ^ number>>workedAbove) * workedFirst
	number = (number ^ number>>workedMiddle) * workedSecond
	return number ^ number>>workedLast
}

// workedStream is the stream of a member of a stand: the seed plus one more
// than the member's number times the golden step is mixed into the first
// half, and the first half plus the golden step, mixed, is the second.
func workedStream(seed uint64, member int) *rand.Rand {
	first := workedMix(seed + uint64(member+1)*workedStep)
	return rand.New(rand.NewPCG(first, workedMix(first+workedStep)))
}

// theWedge is a wedge about 10 east and 20 north, from bearing 30 to 150,
// between radii 5 and 50: it turns 150-30 = 120 degrees.
func theWedge() Ground {
	return Ground{East: 10, North: 20, From: 30, To: 150, Near: 5, Far: 50}
}

// theRectangle is the rectangle 100 to 160 east and 200 to 330 north: it is
// 160-100 = 60 wide and 330-200 = 130 deep.
func theRectangle() Ground {
	return Ground{Box: &Quarter{West: 100, South: 200, East: 160,
		North: 330}}
}

// workedWedge is the place of a member over theWedge, worked by hand: the
// bearing is 30 plus a draw of the 120 degrees, then the distance is the
// square root of 5 squared plus a draw of (50 squared less 5 squared), laid
// out from the centre. The stream is given back for what is drawn next.
func workedWedge(seed uint64, member int) (east, north float64,
	rest *rand.Rand) {
	rest = workedStream(seed, member)
	bearing := (30 + rest.Float64()*120) * math.Pi / 180
	out := math.Sqrt(5*5 + rest.Float64()*(50*50-5*5))
	return 10 + out*math.Sin(bearing), 20 + out*math.Cos(bearing), rest
}

// workedRectangle is the place of a member over theRectangle, worked by
// hand: east is 100 plus a draw of 60, then north is 200 plus a draw of 130.
func workedRectangle(seed uint64, member int) (east, north float64,
	rest *rand.Rand) {
	rest = workedStream(seed, member)
	return 100 + rest.Float64()*60, 200 + rest.Float64()*130, rest
}

// laidSite is the site at the index of a scatter of the many, or the
// test's end.
func laidSite(t *testing.T, ground Ground, item Item, seed Seed, many,
	index int) Site {
	t.Helper()
	laid, err := scatter{}.Sites(ground, item, seed, many)
	if err != nil {
		t.Fatal(err)
	}
	return laid.Sites[index]
}

// near is whether two numbers are within the tests' tolerance.
func near(first, second float64) bool {
	return math.Abs(first-second) <= tolerance
}

// A member of a scatter over a wedge is placed from its own stream: the
// bearing, then the distance, as the sum is worked by hand.
func TestAScatterPlacesEachMemberOverAWedgeFromItsOwnStream(t *testing.T) {
	t.Parallel()
	for member := range 8 {
		site := laidSite(t, theWedge(), Item{}, workedSeed, 8, member)
		east, north, _ := workedWedge(workedSeed, member)
		if !near(site.East, east) || !near(site.North, north) {
			t.Errorf("member %d is at %v, %v; worked by hand %v, %v",
				member, site.East, site.North, east, north)
		}
	}
}

// With the seed or the member number changed by one the worked place is
// another: the stream is keyed by both.
func TestAScatterOverAWedgeIsNotTheNeighbouringSeedOrMember(t *testing.T) {
	t.Parallel()
	site := laidSite(t, theWedge(), Item{}, workedSeed, 8, workedMember)
	for name, other := range map[string][2]float64{
		"seed":   pairOf(workedWedge(workedSeed+1, workedMember)),
		"member": pairOf(workedWedge(workedSeed, workedMember+1)),
	} {
		if near(site.East, other[0]) && near(site.North, other[1]) {
			t.Errorf("a changed %s gives the same place %v", name, other)
		}
	}
}

// pairOf is an east and a north as a pair, the stream let go.
func pairOf(east, north float64, _ *rand.Rand) [2]float64 {
	return [2]float64{east, north}
}

// A member of a scatter over a rectangle is placed from its own stream:
// east, then north.
func TestAScatterPlacesEachMemberOverARectangleFromItsOwnStream(t *testing.T) {
	t.Parallel()
	for member := range 8 {
		site := laidSite(t, theRectangle(), Item{}, workedSeed, 8, member)
		east, north, _ := workedRectangle(workedSeed, member)
		if !near(site.East, east) || !near(site.North, north) {
			t.Errorf("member %d is at %v, %v; worked by hand %v, %v",
				member, site.East, site.North, east, north)
		}
	}
}

// With the seed or the member number changed by one the worked place over a
// rectangle is another.
func TestAScatterOverARectangleIsNotTheNeighbouringSeedOrMember(t *testing.T) {
	t.Parallel()
	site := laidSite(t, theRectangle(), Item{}, workedSeed, 8, workedMember)
	for name, other := range map[string][2]float64{
		"seed":   pairOf(workedRectangle(workedSeed+1, workedMember)),
		"member": pairOf(workedRectangle(workedSeed, workedMember+1)),
	} {
		if near(site.East, other[0]) && near(site.North, other[1]) {
			t.Errorf("a changed %s gives the same place %v", name, other)
		}
	}
}

// theFaced is an item that prefers to face 180 degrees, within 30 either
// way.
func theFaced() Item {
	return Item{Faced: true, Facing: habitFacing{Heading: 180, Spread: 30}}
}

// workedHeading is the heading of a member over theWedge, worked by hand:
// the draw after the two of its place, as a share of the way across the
// spread, which is 180 less 30 to 180 plus 30.
func workedHeading(seed uint64, member int) float64 {
	_, _, rest := workedWedge(seed, member)
	return 180 + (2*rest.Float64()-1)*30
}

// A faced member's heading is the third draw of its own stream, after its
// place, so it does not depend on the members before it.
func TestAScatterFacesEachMemberFromItsOwnStream(t *testing.T) {
	t.Parallel()
	for member := range 8 {
		site := laidSite(t, theWedge(), theFaced(), workedSeed, 8, member)
		if !site.Faced || !near(site.Heading, workedHeading(workedSeed,
			member)) {
			t.Errorf("member %d faces %v (faced %v); worked by hand %v",
				member, site.Heading, site.Faced,
				workedHeading(workedSeed, member))
		}
	}
}

// With the seed or the member number changed by one the worked heading is
// another.
func TestAScatterFacingIsNotTheNeighbouringSeedOrMember(t *testing.T) {
	t.Parallel()
	site := laidSite(t, theWedge(), theFaced(), workedSeed, 8, workedMember)
	for name, other := range map[string]float64{
		"seed":   workedHeading(workedSeed+1, workedMember),
		"member": workedHeading(workedSeed, workedMember+1),
	} {
		if near(site.Heading, other) {
			t.Errorf("a changed %s gives the same heading %v", name, other)
		}
	}
}

// A member's site is the same whether the stand asks for a few members or
// for a thousand, in its place and in its facing.
func TestAScatterMembersDoNotDependOnHowManyAreAsked(t *testing.T) {
	t.Parallel()
	for _, ground := range []Ground{theWedge(), theRectangle()} {
		few, err := scatter{}.Sites(ground, theFaced(), workedSeed, 5)
		if err != nil {
			t.Fatal(err)
		}
		many, err := scatter{}.Sites(ground, theFaced(), workedSeed,
			workedSpan)
		if err != nil {
			t.Fatal(err)
		}
		for member, site := range few.Sites {
			if site != many.Sites[member] {
				t.Errorf("member %d is %v of 5 and %v of %d", member, site,
					many.Sites[member], workedSpan)
			}
		}
	}
}

// Members of one stand are not one another's sites: the member asked for
// is the one of its own number and no other, a thousand being asked for.
func TestAScatterMemberIsNotItsNeighbourAmongAThousand(t *testing.T) {
	t.Parallel()
	laid, err := scatter{}.Sites(theWedge(), theFaced(), workedSeed,
		workedSpan)
	if err != nil {
		t.Fatal(err)
	}
	member := laid.Sites[workedMember]
	for _, other := range []int{workedMember - 1, workedMember + 1} {
		if laid.Sites[other] == member {
			t.Errorf("members %d and %d are both %v", workedMember, other,
				member)
		}
	}
}
