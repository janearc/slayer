package hillside

import (
	"errors"
	"io"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/forest"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// mustParse is what the command is asked by these arguments, drawn
// small.
func mustParse(t *testing.T, arguments ...string) asked {
	t.Helper()
	given, err := parse(append([]string{"-w", "96", "-h", "60", "-n",
		"1"}, arguments...), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	return given
}

// only is the arguments that ask for none of any kind but these.
func only(counts map[string]string) []string {
	var arguments []string
	for _, name := range flagKinds() {
		count, asked := counts[name]
		if !asked {
			count = "0"
		}
		arguments = append(arguments, "-"+name, count)
	}
	return arguments
}

// Legacy flags retain their defaults; new recipes use the YAML front door.
func TestLegacyKindFlagsKeepTheirDefaults(t *testing.T) {
	given := mustParse(t)
	// Keep the user-visible legacy counts independent of the production
	// map; reading usual here lets a changed default change its own oracle.
	expectedCounts := map[string]int{
		"spruce": 5, "fern": 24, "boulder": 8, "cobble": 20,
		"log": 3, "can": 1, "campfire": 1, "spotlight": 1,
	}
	for _, name := range flagKinds() {
		count, has := given.counts[name]
		if !has || count != expectedCounts[name] {
			t.Errorf("%s: %d by default, has a flag: %v", name,
				count, has)
		}
	}
	asked := mustParse(t, "-spruce", "2", "-maser", "3", "-night",
		"-seed", "7", "-gpu", "-at", "2s", "-apart", "4",
		"-exposure", "9", "-faces", "5000", "-out", "there.png")
	want := mustParse(t)
	want.counts["spruce"], want.counts["maser"] = 2, 3
	want.night, want.onGPU, want.seed, want.at = true, true, 7,
		2*time.Second
	want.apart, want.exposure, want.faces, want.out = 4, 9, 5000,
		"there.png"
	if !reflect.DeepEqual(asked, want) {
		t.Errorf("every flag given, and got %+v, not %+v", asked, want)
	}
	if _, err := parse([]string{"-ottoman", "1"},
		io.Discard); err == nil {
		t.Error("an ottoman was allowed")
	}
}

// what cannot be drawn is refused when it is asked for, by the flag
// that asked it, before anything is made.
func TestWhatCannotBeDrawnIsRefusedAtTheFlags(t *testing.T) {
	refused := map[string][]string{
		"-spruce":   {"-spruce", "-1"},
		"-fern":     {"-fern", "99999999999"},
		"-n":        {"-n", "0"},
		"-w":        {"-w", "0"},
		"-h":        {"-h", "-3"},
		"-apart":    {"-apart", "0"},
		"-exposure": {"-exposure", "-1"},
		"-faces":    {"-faces", "0"},
	}
	for flag, arguments := range refused {
		_, err := parse(arguments, io.Discard)
		if !errors.Is(err, errAsked) {
			t.Errorf("%v: %v, not %v", arguments, err, errAsked)
			continue
		}
		name := strings.TrimPrefix(flag, "-")
		if !strings.Contains(err.Error(), name) {
			t.Errorf("%v is refused without naming %s: %v",
				arguments, flag, err)
		}
	}
	// more things than faces allowed is refused by adding the counts
	// up, not by making the things: this would be a list of a hundred
	// thousand million names. a review ran it past a minute.
	started := time.Now()
	_, err := parse([]string{"-faces", "100", "-boulder", "60",
		"-cobble", "60"}, io.Discard)
	if !errors.Is(err, errAsked) || time.Since(started) > time.Second {
		t.Errorf("120 things in 100 faces: %v after %v", err,
			time.Since(started))
	}
}

// what is asked for is what is stood on the hillside, each kind as
// many times as asked, in an order the seed shuffles.
// the shuffle reaches every place, the second as well as the last:
// two things take turns to be first as the seed changes.
func TestTheShuffleReachesEveryPlace(t *testing.T) {
	given := mustParse(t, only(map[string]string{"can": "1",
		"log": "1"})...)
	first := map[string]int{}
	for seed := range uint64(40) {
		given.seed = seed
		first[order(given)[0]]++
	}
	if first["can"] < 8 || first["log"] < 8 {
		t.Errorf("of 40 seeds the can is first %d times and the "+
			"log %d", first["can"], first["log"])
	}
}

func TestTheOrderIsWhatWasAskedShuffledByTheSeed(t *testing.T) {
	given := mustParse(t, only(map[string]string{"spruce": "3",
		"fern": "5", "cobble": "4"})...)
	names := order(given)
	again := order(given)
	if !reflect.DeepEqual(names, again) {
		t.Error("the same seed gave another order")
	}
	counted := map[string]int{}
	for _, name := range names {
		counted[name]++
	}
	want := map[string]int{"spruce": 3, "fern": 5, "cobble": 4}
	if !reflect.DeepEqual(counted, want) {
		t.Errorf("stood %v, asked for %v", counted, want)
	}
	if sort.StringsAreSorted(names) {
		t.Error("the kinds stand in the alphabet's order, unshuffled")
	}
	given.seed = 2
	if reflect.DeepEqual(order(given), names) {
		t.Error("another seed gave the same order")
	}
}

// a hillside counts what it holds: its things, the lamps that light
// it (the sun, the light come back from the far side, and each lamp a
// thing carries that an eye can see), and the lamps no eye can see.
func TestAHillsideCountsWhatItHolds(t *testing.T) {
	given := mustParse(t, only(map[string]string{"spruce": "1",
		"fern": "2", "boulder": "2", "campfire": "1", "spotlight": "1",
		"maser": "1"})...)
	world, made, err := hillside(given)
	if err != nil {
		t.Fatal(err)
	}
	if made.things != 8 || made.lamps != 4 || made.unseen != 1 {
		t.Errorf("%d things, %d lamps and %d unseen; asked for 8 "+
			"things, a fire and a spot under a sun, and a maser",
			made.things, made.lamps, made.unseen)
	}
	if len(world.Lamps) != made.lamps {
		t.Errorf("%d lamps in the world, %d counted",
			len(world.Lamps), made.lamps)
	}
	// the ground, and a thing or more for each made thing
	if len(world.Things) < 1+made.things || made.faces < 1000 {
		t.Errorf("%d things in the world of %d faces",
			len(world.Things), made.faces)
	}
	if said := made.String(); said == "" {
		t.Error("the tally says nothing")
	}
}

// boulders is a hillside of nine boulders and nothing else, so that
// the order is the same whatever the seed, and each thing is one
// thing.Thing: the world's things are the ground and then the nine, in the
// order of their places.
func boulders(t *testing.T, arguments ...string) (*thing.World, asked) {
	t.Helper()
	given := mustParse(t, append(only(map[string]string{
		"boulder": "9"}), arguments...)...)
	world, _, err := hillside(given)
	if err != nil {
		t.Fatal(err)
	}
	return world, given
}

// a thing stands where forest placed it, turned and sized as forest
// said: its place, its turn about the up axis and its size are all
// the placement's, and the places are the hillside's seed's.
func TestAThingStandsTurnsAndIsSizedAsForestPlacedIt(t *testing.T) {
	world, given := boulders(t)
	places, err := forest.Placements(given.seed, 3, given.apart, strays)
	if err != nil {
		t.Fatal(err)
	}
	middle := maths.V(given.apart, 0, given.apart)
	for index, placed := range places {
		got := world.Things[1+index].At(0)
		want := maths.Subspace{Origin: placed.At().Sub(middle),
			Orientation: maths.About(maths.YAxis, placed.Turn()),
			Scale:       placed.Size()}
		if !got.Origin.Near(want.Origin, 1e-9) || got.Scale !=
			want.Scale || got.Orientation != want.Orientation {
			t.Errorf("boulder %d stands %+v, forest placed it %+v",
				index, got, want)
		}
	}
	same, _ := boulders(t)
	other, _ := boulders(t, "-seed", "2")
	moved := false
	for index := range places {
		at := world.Things[1+index].At(0).Origin
		if at != same.Things[1+index].At(0).Origin {
			t.Fatalf("boulder %d moved for the same seed", index)
		}
		moved = moved || at != other.Things[1+index].At(0).Origin
	}
	if !moved {
		t.Error("another seed stood everything in the same places")
	}
}

// -apart is how far apart things stand: twice as far apart, and from
// the same seed every thing is twice as far from the middle.
func TestThingsStandAsFarApartAsAsked(t *testing.T) {
	near, _ := boulders(t, "-apart", "2")
	far, _ := boulders(t, "-apart", "4")
	for index := range 9 {
		close := near.Things[1+index].At(0).Origin
		wide := far.Things[1+index].At(0).Origin
		if !wide.Near(close.Scale(2), 1e-9) {
			t.Errorf("boulder %d is at %v four apart and %v two "+
				"apart", index, wide, close)
		}
	}
}

// sizes is the size of every thing on a hillside but its ground.
func sizes(world *thing.World) map[maths.Vec]bool {
	found := map[maths.Vec]bool{}
	for _, each := range world.Things[1:] {
		box := each.Shape.Bounds()
		found[box.Max.Sub(box.Min)] = true
	}
	return found
}

// each thing is made from a seed of its own, and no hillside's things
// are another's: nine boulders are nine sizes, and none of seed one's
// nine is among seed two's. the first version of stir made seed two
// seed one moved along a place, and eight of nine were shared;
// a review found it.
func TestNoTwoThingsAndNoTwoHillsidesShareASeed(t *testing.T) {
	one, _ := boulders(t)
	two, _ := boulders(t, "-seed", "2")
	first, second := sizes(one), sizes(two)
	if len(first) != 9 || len(second) != 9 {
		t.Fatalf("nine boulders of %d sizes, and of %d", len(first),
			len(second))
	}
	for size := range first {
		if second[size] {
			t.Errorf("a boulder %v is on both hillsides", size)
		}
	}
	seeds := map[uint64]bool{}
	for seed := range uint64(30) {
		for place := range 30 {
			seeds[stir(seed, place)] = true
		}
	}
	if len(seeds) != 900 {
		t.Errorf("30 hillsides of 30 places have %d seeds between "+
			"them", len(seeds))
	}
}

// carried is the lamps the hillside's things carry: every lamp after
// the sun and the light come back from the far side.
func carried(t *testing.T, arguments ...string) []emit.Lit {
	t.Helper()
	world, _, err := hillside(mustParse(t, append(only(
		map[string]string{"campfire": "3", "spotlight": "2"}),
		arguments...)...))
	if err != nil {
		t.Fatal(err)
	}
	var lamps []emit.Lit
	for _, lamp := range world.Lamps[2:] {
		lamps = append(lamps, lamp.(emit.Lit))
	}
	return lamps
}

// the lamps a hillside's things carry are as bright as -exposure
// says, flicker with -at, start clear of their housings, and never
// have less than no light in any colour, which a fire's deep red
// would have in blue.
func TestCarriedLampsFollowTheirFlags(t *testing.T) {
	plain := carried(t)
	bright := carried(t, "-exposure", "4000")
	later := carried(t, "-at", "300ms")
	if len(plain) != 5 {
		t.Fatalf("%d lamps carried by three fires and two spots",
			len(plain))
	}
	flickered := false
	for index, lamp := range plain {
		twice := bright[index].Strength.R / lamp.Strength.R
		if !(math.Abs(twice-2) <= 1e-9) {
			t.Errorf("lamp %d is %v times as bright at twice the "+
				"exposure", index, twice)
		}
		if lamp.Strength.R < 0 || lamp.Strength.G < 0 ||
			lamp.Strength.B < 0 {
			t.Errorf("lamp %d has less than no light: %v", index,
				lamp.Strength)
		}
		if lamp.Beyond <= 0 {
			t.Errorf("lamp %d starts in its own housing", index)
		}
		flickered = flickered || later[index].Strength != lamp.Strength
	}
	if !flickered {
		t.Error("no lamp is any different a third of a second on")
	}
}

// the eye stands back as far as the hillside is wide, and looks at
// its middle: a hillside twice as wide is seen from further off, and
// the middle is ahead of the eye either way.
func TestTheEyeStandsBackFromAWiderHillside(t *testing.T) {
	near, far := eyeAt(10), eyeAt(40)
	if far.Origin.Len() < 2*near.Origin.Len() {
		t.Errorf("the eye is %v from a hillside of 10 and %v from "+
			"one of 40", near.Origin.Len(), far.Origin.Len())
	}
	// off its front corner: to the right, above and behind, each by
	// a share of the hillside's width past where it stands for none
	none, small, large := eyeAt(0).Origin, near.Origin.Sub(
		eyeAt(0).Origin), far.Origin.Sub(eyeAt(0).Origin)
	if !none.Near(maths.V(0, 2.2, 4), 1e-12) || !small.Near(maths.V(3.5, 1.2, 8), 1e-12) ||
		small.Y <= 0 || small.Z <= 0 || !large.Near(small.Scale(4),
		1e-9) {
		t.Errorf("the eye is at %v for no hillside, %v more for 10 "+
			"and %v more for 40", none, small, large)
	}
	for _, eye := range []maths.Subspace{near, far} {
		toMiddle := maths.V(0, 1.4, 0).Sub(eye.Origin).Unit()
		if !eye.Forward().Near(toMiddle, 1e-9) {
			t.Errorf("the eye at %v looks %v, not at the middle",
				eye.Origin, eye.Forward())
		}
	}
}

// a hillside of more faces than allowed is refused, saying how many
// and which flag allows more.
func TestTooManyFacesIsRefused(t *testing.T) {
	given := mustParse(t, append(only(map[string]string{
		"spruce": "2"}), "-faces", "500")...)
	_, _, err := hillside(given)
	if !errors.Is(err, errTooMany) {
		t.Errorf("two spruces in 500 faces: %v, not %v", err,
			errTooMany)
	}
}

// light is all the light of a picture together.
func light(drawn *frame.Frame) float64 {
	sum := 0.0
	for _, pixel := range drawn.Pix {
		sum += float64(pixel.R + pixel.G + pixel.B)
	}
	return sum
}

// picture is a hillside drawn on the cpu.
func picture(t *testing.T, arguments ...string) *frame.Frame {
	t.Helper()
	given := mustParse(t, arguments...)
	world, made, err := hillside(given)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := draw(world, given, made.across)
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

func TestALampsLightIsNeverBelowNought(t *testing.T) {
	got := seen(maths.RGB(0.09, 0.01, -0.002))
	if got != maths.RGB(0.09, 0.01, 0) {
		t.Errorf("a glow a little under nought in blue came out %v",
			got)
	}
}

// the bounds at their edges: exactly as many things as faces allowed
// is asked for and one more is not; a picture one pixel square with
// one ray is a picture; no exposure at all is refused.
func TestTheFlagsBoundsAtTheirEdges(t *testing.T) {
	nine := only(map[string]string{"boulder": "9"})
	allowed := [][]string{
		append(nine[:len(nine):len(nine)], "-faces", "9"),
		{"-w", "1", "-h", "1", "-n", "1"},
	}
	for _, arguments := range allowed {
		if _, err := parse(arguments, io.Discard); err != nil {
			t.Errorf("%v was refused: %v", arguments, err)
		}
	}
	refused := [][]string{
		append(nine[:len(nine):len(nine)], "-faces", "8"),
		{"-exposure", "0"},
		{"-apart", "-1"},
	}
	for _, arguments := range refused {
		_, err := parse(arguments, io.Discard)
		if !errors.Is(err, errAsked) {
			t.Errorf("%v: %v, not %v", arguments, err, errAsked)
		}
	}
}

// the sun gives the ground the light the scene says, by day and by
// night, and a share of it comes back from the far side: each is a
// lamp so far off that its light falls alike across the hillside,
// made strong enough to arrive as that much.
func TestTheSunGivesTheGroundItsLight(t *testing.T) {
	levels := map[string]float64{"": day, "-night": moonlit}
	for flag, level := range levels {
		var arguments []string
		if flag != "" {
			arguments = append(arguments, flag)
		}
		world, err := bare(mustParse(t, arguments...), 20)
		if err != nil {
			t.Fatal(err)
		}
		wants := []float64{level, fill * level}
		for index, want := range wants {
			lamp := world.Lamps[index].(emit.Lit)
			_, _, here := lamp.Light(maths.Vec{}, maths.Up)
			_, _, there := lamp.Light(maths.V(15, 0, -15), maths.Up)
			if !(math.Abs(here.G-want) <= 1e-9*want) ||
				!(math.Abs(there.G-want) <= 0.02*want) {
				t.Errorf("%q lamp %d gives the ground %v at "+
					"the middle and %v at the edge, not %v",
					flag, index, here.G, there.G, want)
			}
		}
	}
}

// a hillside is as far across as its places a side times how far
// apart they are, and one thing alone is one place.
func TestAHillsideIsAsWideAsItsPlaces(t *testing.T) {
	widths := map[string]float64{"1": 3, "4": 6, "5": 9, "9": 9}
	for count, want := range widths {
		given := mustParse(t, append(only(map[string]string{
			"cobble": count}), "-apart", "3")...)
		_, made, err := hillside(given)
		if err != nil {
			t.Fatal(err)
		}
		if made.across != want {
			t.Errorf("%s cobbles three apart are %v across, not %v",
				count, made.across, want)
		}
	}
}

// the tally says what there is and no more: the lamps no eye sees
// are mentioned only when there are some.
func TestTheTallySaysWhatThereIs(t *testing.T) {
	plain := tally{things: 3, faces: 40, kept: 40, lamps: 2}.String()
	if plain != "3 things of 40 faces, lit by 2 lamps" {
		t.Errorf("it says %q", plain)
	}
	dark := tally{things: 3, faces: 40, kept: 40, lamps: 2,
		unseen: 1}.String()
	if !strings.HasPrefix(dark, plain) || !strings.Contains(dark,
		"1 more shine where no eye sees") {
		t.Errorf("with a maser it says %q", dark)
	}
}

// the ground lies under the hillside with its middle under the
// hillside's, level, and reaches well past it every way: at least so
// far across, and so many times the hillside.
func TestTheGroundLiesUnderTheHillside(t *testing.T) {
	sizes := map[float64]float64{10: 160, 100: 600}
	for across, want := range sizes {
		world, err := bare(mustParse(t), across)
		if err != nil {
			t.Fatal(err)
		}
		ground := world.Things[0]
		box, at := ground.Shape.Bounds(), ground.At(0).Origin
		low, high := box.Min.Add(at), box.Max.Add(at)
		half := want / 2
		if !low.Near(maths.V(-half, 0, -half), 1e-9) ||
			!high.Near(maths.V(half, 0, half), 1e-9) {
			t.Errorf("under a hillside %v across the ground runs "+
				"from %v to %v, not %v either way", across, low,
				high, half)
		}
	}
}

// a hillside of exactly as many faces as allowed is made, and one
// face fewer allowed refuses it.
func TestAHillsideOfExactlyItsFacesIsMade(t *testing.T) {
	stones := only(map[string]string{"boulder": "3"})
	_, made, err := hillside(mustParse(t, stones...))
	if err != nil {
		t.Fatal(err)
	}
	exact := append(stones[:len(stones):len(stones)], "-faces",
		strconv.Itoa(made.faces))
	if _, _, err := hillside(mustParse(t, exact...)); err != nil {
		t.Errorf("%d faces in %d allowed: %v", made.faces, made.faces,
			err)
	}
	under := append(stones[:len(stones):len(stones)], "-faces",
		strconv.Itoa(made.faces-1))
	if _, _, err := hillside(mustParse(t, under...)); !errors.Is(err,
		errTooMany) {
		t.Errorf("%d faces in one fewer: %v, not %v", made.faces, err,
			errTooMany)
	}
}
