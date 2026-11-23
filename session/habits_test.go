package session

import (
	"math"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// siteFacing is a site that faces a bearing.
func siteFacing(bearing float64) cities.Site {
	return cities.Site{Faced: true, Heading: bearing}
}

// readSetText is a set read from its text.
func readSetText(text string) (set.Set, error) {
	return set.Read("positions.yaml", []byte(text))
}

// standSetOf is the test's stand set with the stand's kind changed, its
// count taken out if none is asked for, and more keys said.
func standSetOf(kind string, count bool, keys string) string {
	text := strings.Replace(standSetText(""),
		"      the district:\n        kind: yurt\n",
		"      the district:\n        kind: "+kind+"\n", 1)
	if !count {
		text = strings.Replace(text, "        count: 20\n", "", 1)
	}
	return text + keys
}

// afloatStage is the stage of a set in space after its afloat, or the
// test's end.
func afloatStage(t *testing.T, text string) *staging {
	t.Helper()
	stage := testStaging()
	if err := stage.afloat(readPositionSet(t, text)); err != nil {
		t.Fatal(err)
	}
	return stage
}

// standStood is where the things of the stand were stood, in order.
func standStood(stage *staging) []standing {
	var stood []standing
	for _, each := range stage.stood {
		if each.of == "the district" {
			stood = append(stood, each)
		}
	}
	return stood
}

// The numbers of splitmix64, which a member's stream is keyed by, written
// out here so the working below is not the code's own.
const (
	splitStep   = 0x9e3779b97f4a7c15 // defined
	splitFirst  = 0xbf58476d1ce4e5b9 // defined
	splitSecond = 0x94d049bb133111eb // defined
)

// splitmix is the finishing step of splitmix64, worked here again.
func splitmix(number uint64) uint64 {
	number = (number ^ number>>30) * splitFirst
	number = (number ^ number>>27) * splitSecond
	return number ^ number>>31
}

// A stand with no habit places each of its members by itself: member n
// draws from a stream keyed by the stand's seed and n alone, its bearing
// evenly over the wedge and then its distance evenly over the area between
// the two radii. The places are an independent working of that sum.
func TestAStandWithNoHabitPlacesEachMemberByItself(t *testing.T) {
	staged := afloatStage(t, standSetOf("yurt", true, ""))
	stood := standStood(staged)
	if len(stood) != standCount {
		t.Fatalf("%d stood, want %d", len(stood), standCount)
	}
	for index, each := range stood {
		first := splitmix(2 + uint64(index+1)*splitStep)
		second := splitmix(first + splitStep)
		places := rand.New(rand.NewPCG(first, second))
		bearing := places.Float64() * 360 * math.Pi / 180
		out := math.Sqrt(8*8 + places.Float64()*(50*50-8*8))
		east, north := out*math.Sin(bearing), out*math.Cos(bearing)
		if math.Abs(each.east-east) > 1e-9 || math.Abs(each.north-north) > 1e-9 {
			t.Fatalf("place %d is %v, %v; the generator gives %v, %v", index,
				each.east, each.north, east, north)
		}
	}
}

// The scatter habit of a kind that prefers no facing is the stand as it
// was: the same things at the same places turned the same ways.
func TestAScatterHabitOfAKindWithNoFacingIsTheStandAsItWas(t *testing.T) {
	plain := afloatStage(t, standSetOf("winnebago", true, ""))
	scattered := afloatStage(t, standSetOf("winnebago", true,
		"        habit: scatter\n"))
	if len(plain.world.Things) != len(scattered.world.Things) {
		t.Fatalf("%d parts, then %d", len(plain.world.Things),
			len(scattered.world.Things))
	}
	for index, each := range plain.world.Things {
		if each.At(0) != scattered.world.Things[index].At(0) {
			t.Fatalf("part %d stands at %v, then %v", index, each.At(0),
				scattered.world.Things[index].At(0))
		}
	}
}

// A lattice stand with no count stands a thing on every pitch that fits,
// all at ground level, and says how many.
func TestALatticeStandStandsAThingOnEachPitchThatFits(t *testing.T) {
	stage := afloatStage(t, standSetOf("winnebago", false,
		"        habit: lattice\n"))
	stood := standStood(stage)
	if len(stood) < 10 {
		t.Fatalf("%d pitches fit, want many", len(stood))
	}
	for _, each := range stood {
		if each.height != 0 {
			t.Errorf("a thing stands %v up", each.height)
		}
	}
	if stage.made.things != 1+len(stood) {
		t.Errorf("%d things stood for %d pitches", stage.made.things,
			len(stood))
	}
	said := strings.Join(stage.given.departures, "\n")
	if !strings.Contains(said, "the district: a lattice laid") {
		t.Errorf("the stage said %q", said)
	}
}

// A lattice stand that gives a count takes at most that many, and says so
// when fewer fit.
func TestALatticeStandTakesAtMostItsCountAndSaysWhenFewerFit(t *testing.T) {
	few := afloatStage(t, standSetOf("winnebago", true,
		"        habit: lattice\n"))
	if got := len(standStood(few)); got != standCount {
		t.Errorf("%d stood, want the count of %d", got, standCount)
	}
	many := strings.Replace(standSetOf("winnebago", true,
		"        habit: lattice\n"), "count: 20", "count: 99999", 1)
	stage := afloatStage(t, many)
	if got := len(standStood(stage)); got >= 99999 || got == 0 {
		t.Errorf("%d stood of 99999 asked for", got)
	}
	if said := strings.Join(stage.given.departures, "\n"); !strings.Contains(
		said, "fewer than the 99999 asked for") {
		t.Errorf("the stage said %q", said)
	}
}

// A gers' site that faces south is turned not at all, as the ger's door is
// on its positive z side; east is a quarter turn; a site that leaves its
// facing alone has none.
func TestASiteFacesAsItsBearingSays(t *testing.T) {
	turned := func(bearing float64) float64 {
		turn := facingOf(siteFacing(bearing))
		if turn == nil {
			t.Fatalf("a faced site at %v has no turn", bearing)
		}
		return *turn
	}
	if got := turned(180); math.Abs(got) > 1e-12 {
		t.Errorf("south turns %v", got)
	}
	if got := turned(90); math.Abs(got-math.Pi/2) > 1e-12 {
		t.Errorf("east turns %v, want a quarter turn", got)
	}
	front := maths.Rotation(maths.YAxis, turned(90)).Out(maths.ZAxis)
	if front.X < 0.999 {
		t.Errorf("a front turned to face east looks along %v", front)
	}
}

// Anti-test: a lattice of a kind whose data gives no pitch is refused by
// the kind's name, and so is a habit with no wedge to lay over.
func TestAHabitThatCannotBeLaidIsRefused(t *testing.T) {
	stage := testStaging()
	source := standSetOf("yurt", false,
		"        habit: lattice\n")
	err := stage.afloat(readPositionSet(t, source))
	if err == nil || !strings.Contains(err.Error(), "yurt") {
		t.Errorf("a lattice of gers: %v", err)
	}
}

// Anti-test: the set reader refuses a stand with a habit and no wedge, a
// count-less stand that is not a lattice, and a habit nobody named.
func TestAStandThatSaysAHabitWrongIsRefusedByTheReader(t *testing.T) {
	within := "        within: {of: ger, from: 0, to: 360, near: 8, far: 50}\n"
	for name, text := range map[string]string{
		"a habit with no wedge": strings.Replace(standSetOf("winnebago",
			true, "        habit: lattice\n"), within, "", 1),
		"no count and no lattice": standSetOf("winnebago", false, ""),
		"a habit nobody named": standSetOf("winnebago", true,
			"        habit: sprawl\n"),
	} {
		if _, err := readSetText(text); err == nil {
			t.Errorf("%s was read", name)
		}
	}
}
