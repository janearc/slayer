package session

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/set"
)

// warmFixture is the things of the set these tests make ahead: two towers,
// which are drawn part by part, and three things of the stock, two of
// them one design, each standing so far north of the camera in metres.
var warmFixture = []struct {
	name, kind string
	seed       int
	north      float64
}{
	{"near tower", "tower", 1, 120},
	{"far tower", "tower", 2, 600},
	{"first spruce", "spruce", 3, 200},
	{"second spruce", "spruce", 3, 210},
	{"lone can", "can", 4, 150},
}

// warmSetText is the fixture as a set, each thing at an absolute place so
// that no ground is wanted to stand it.
func warmSetText() string {
	var text strings.Builder
	text.WriteString(`scene:
  name: made ahead
  film: {width: 1, height: 1, rays: 1}
  cameras:
    between:
      title: between
      location:
        loc_relative: {object: near tower}
      orientation:
        point_to: far tower
  recipes:
    warm-spruce: {plant: {family: conifer, file: ../manufacture/plants/conifers/conifer-profiles.yaml, species: spruce}}
  set:
    things:
`)
	for _, each := range warmFixture {
		selector := "kind: " + each.kind
		if each.kind == "spruce" {
			selector = "recipe: warm-spruce"
		}
		fmt.Fprintf(&text, `      %s:
        %s
        seed: %d
        complexity: quick
        position:
          east: {value: 0, unit: m}
          north: {value: %g, unit: m}
          up: {value: 0, unit: m}
`, each.name, selector, each.seed, each.north)
	}
	return text.String()
}

// warmStaging is a stage for the fixture whose camera is the one looking
// north from the origin, as the leveller sees it.
func warmStaging(t *testing.T, read set.Set) *staging {
	t.Helper()
	stage := testStaging()
	stage.given = mustParse(t)
	stage.recipes = read
	eye := lookingNorth()
	stage.sight = &eye
	return stage
}

// standEach stands each thing as stand's loop does, one after another,
// and says the bounds of what it stood.
func standEach(t *testing.T, stage *staging,
	things []set.Thing) []maths.Box {
	t.Helper()
	var bounds []maths.Box
	for index, each := range things {
		place, err := stage.thingPlace(each)
		if err != nil {
			t.Fatal(err)
		}
		stage.places[each.Name()] = place
		box, err := stage.putRecipe(each.Kind(), each.Recipe(),
			each.Complexity(), each.Seed(), uint64(index), place,
			each.Alight(), each.Name())
		if err != nil {
			t.Fatalf("%s: %v", each.Name(), err)
		}
		bounds = append(bounds, box)
	}
	return bounds
}

// sameStood checks two stages stood the same: the bounds of each thing,
// the tally, the number of parts in the world, the radii and the
// complexities counted.
func sameStood(t *testing.T, got, want *staging, gotBounds,
	wantBounds []maths.Box) {
	t.Helper()
	if !reflect.DeepEqual(gotBounds, wantBounds) {
		t.Errorf("bounds %v, want %v", gotBounds, wantBounds)
	}
	if got.made != want.made {
		t.Errorf("tally %+v, want %+v", got.made, want.made)
	}
	if len(got.world.Things) != len(want.world.Things) {
		t.Errorf("%d parts in the world, want %d", len(got.world.Things),
			len(want.world.Things))
	}
	if !reflect.DeepEqual(got.radii, want.radii) {
		t.Errorf("radii %v, want %v", got.radii, want.radii)
	}
	if !reflect.DeepEqual(got.rungs, want.rungs) {
		t.Errorf("rungs %+v, want %+v", got.rungs, want.rungs)
	}
}

// TestTheLoopTakesWhatWasMadeAhead checks that a stage whose things were
// made ahead stands what a stage that makes each as it comes stands, and
// that the loop took every one of them: no guess was wrong.
func TestTheLoopTakesWhatWasMadeAhead(t *testing.T) {
	t.Setenv(scrooge.Variable, "1")
	read := readPositionSet(t, warmSetText())
	things := read.Things()
	ahead := warmStaging(t, read)
	table := ahead.warmThings(things)
	if table == nil || len(table.composites) == 0 || len(table.wholes) == 0 {
		t.Fatalf("towers and spruces made ahead as %+v", table)
	}
	ahead.warm = table
	got := standEach(t, ahead, things)
	if table.taken != len(table.composites)+len(table.wholes) {
		t.Errorf("%d taken of %d made ahead", table.taken,
			len(table.composites)+len(table.wholes))
	}
	plain := warmStaging(t, read)
	sameStood(t, ahead, plain, got, standEach(t, plain, things))
}

// TestStandStandsWhatTheLoopAloneStands checks stand itself with things
// made ahead against the loop run by hand with none, and that the table
// is gone when the loop is.
func TestStandStandsWhatTheLoopAloneStands(t *testing.T) {
	t.Setenv(scrooge.Variable, "1")
	read := readPositionSet(t, warmSetText())
	ahead := warmStaging(t, read)
	if err := ahead.stand(read); err != nil {
		t.Fatal(err)
	}
	if ahead.warm != nil {
		t.Error("the table outlived the loop")
	}
	plain := warmStaging(t, read)
	standEach(t, plain, read.Things())
	sameStood(t, ahead, plain, nil, nil)
}

// TestAWrongGuessIsMadeAsItWouldHaveBeen is the anti-test of the table: a
// table whose every thing is kept under a seed one off is taken from by
// none, and the loop makes each itself and stands what it always stood.
func TestAWrongGuessIsMadeAsItWouldHaveBeen(t *testing.T) {
	t.Setenv(scrooge.Variable, "1")
	read := readPositionSet(t, warmSetText())
	things := read.Things()
	ahead := warmStaging(t, read)
	right := ahead.warmThings(things)
	wrong := &warmed{composites: map[warmKey]kept{},
		wholes: map[wholeKey]kept{}}
	for key, one := range right.composites {
		key.seed++
		wrong.composites[key] = one
	}
	for key, one := range right.wholes {
		key.seed++
		wrong.wholes[key] = one
	}
	ahead.warm = wrong
	got := standEach(t, ahead, things)
	if wrong.taken != 0 {
		t.Errorf("%d taken from a table of wrong guesses", wrong.taken)
	}
	plain := warmStaging(t, read)
	sameStood(t, ahead, plain, got, standEach(t, plain, things))
}

// TestAKeyThatDiffersInOneInputMisses is the anti-test of the key: the
// table gives a thing back for the arguments it was made for, and for no
// arguments that differ in any one of them.
func TestAKeyThatDiffersInOneInputMisses(t *testing.T) {
	base := warmKey{kind: "tower", recipe: "t000", detail: "quick", seed: 1,
		name: "near tower", at: turnedAt(maths.V(1, 2, 3), 0.5)}
	table := &warmed{composites: map[warmKey]kept{base: {stored: 1}}}
	changes := map[string]func(key *warmKey){
		"kind":   func(key *warmKey) { key.kind = "terrace" },
		"recipe": func(key *warmKey) { key.recipe = "t001" },
		"detail": func(key *warmKey) { key.detail = "demo" },
		"seed":   func(key *warmKey) { key.seed++ },
		"name":   func(key *warmKey) { key.name = "far tower" },
		"origin": func(key *warmKey) { key.at.Origin.X++ },
		"turn":   func(key *warmKey) { key.at = turnedAt(key.at.Origin, 0.6) },
		"scale":  func(key *warmKey) { key.at.Scale = 2 },
	}
	for name, change := range changes {
		key := base
		change(&key)
		if _, taken := table.composite(key); taken {
			t.Errorf("a key differing in its %s was taken", name)
		}
	}
	if _, taken := table.composite(base); !taken || table.taken != 1 {
		t.Errorf("the key it was made for: taken %v, %d taken", taken,
			table.taken)
	}
}

// TestAWholeKeyThatDiffersInOneInputMisses is the same anti-test for a
// whole thing, whose name enters the key as how it is grown.
func TestAWholeKeyThatDiffersInOneInputMisses(t *testing.T) {
	base := wholeKey{kind: "spruce", recipe: "", detail: "quick", seed: 3,
		taller: 1, branched: false}
	table := &warmed{wholes: map[wholeKey]kept{base: {stored: 1}}}
	changes := map[string]func(key *wholeKey){
		"kind":     func(key *wholeKey) { key.kind = "norway" },
		"recipe":   func(key *wholeKey) { key.recipe = "tall" },
		"detail":   func(key *wholeKey) { key.detail = "demo" },
		"seed":     func(key *wholeKey) { key.seed++ },
		"taller":   func(key *wholeKey) { key.taller = 2 },
		"branched": func(key *wholeKey) { key.branched = true },
	}
	for name, change := range changes {
		key := base
		change(&key)
		if _, taken := table.whole(key); taken {
			t.Errorf("a key differing in its %s was taken", name)
		}
	}
	if _, taken := table.whole(base); !taken || table.taken != 1 {
		t.Errorf("the key it was made for: taken %v, %d taken", taken,
			table.taken)
	}
}

// TestNothingIsMadeAheadWhereScroogeIsNotAsked is the anti-test of the
// switch: unset, no table is made, nothing is learned of the designs, and
// stand says nothing of things made ahead.
func TestNothingIsMadeAheadWhereScroogeIsNotAsked(t *testing.T) {
	t.Setenv(scrooge.Variable, "")
	read := readPositionSet(t, warmSetText())
	stage := warmStaging(t, read)
	if table := stage.warmThings(read.Things()); table != nil {
		t.Errorf("a table was made: %+v", table)
	}
	if len(stage.facts) != 0 || len(stage.levellers) != 0 {
		t.Error("an unasked stage learned something of its designs")
	}
	if err := stage.stand(read); err != nil {
		t.Fatal(err)
	}
	for _, each := range stage.given.cached {
		if strings.Contains(each.String(), "made ahead") {
			t.Errorf("a note of things made ahead: %q", each.String())
		}
	}
}

// TestANilTableHasNothing checks the table the stage holds outside the
// loop gives nothing back, whatever is asked of it.
func TestANilTableHasNothing(t *testing.T) {
	var none *warmed
	if _, taken := none.composite(warmKey{}); taken {
		t.Error("a nil table gave a thing part by part")
	}
	if _, taken := none.whole(wholeKey{}); taken {
		t.Error("a nil table gave a whole thing")
	}
}
