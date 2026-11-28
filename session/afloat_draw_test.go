//go:build draw

package session

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/moonlight"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/session/placement"
)

// afloatScene is a small set in space: a can at the origin and a boulder
// east, north and up of it, lit by a sun below level, seen from the side.
const afloatScene = `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: afloat
  seed: 1
  setting: space
  light: sun
  sun: {azimuth: 180, altitude: -45}
  set:
    things:
      the can:
        kind: can
        seed: 1
        east: 0
        north: 0
      the stone:
        kind: boulder
        seed: 2
        east: 3
        north: 4
        height: {value: 5, unit: m}
  film: {width: 24, height: 16, rays: 1, exposure: 2}
  cameras:
    aside:
      title: aside
      location: {at: [-10, 0, 0]}
      orientation: {look_at: [0, 0, 0]}
`

// afloatSet is the path of the small set in space, changed as asked, in a
// fresh directory.
func afloatSet(t *testing.T, was, now string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "afloat.yaml")
	if was != "" && !strings.Contains(afloatScene, was) {
		t.Fatalf("the set holds no %q", was)
	}
	text := strings.Replace(afloatScene, was, now, 1)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// nearPoint is whether two points are within a millimetre's millionth.
func nearPoint(first, second maths.Vec) bool {
	return first.Sub(second).Len() < 1e-9
}

// A set in space stands each craft where its east, north and height say,
// with no ground under it, against darkness, and its camera finds them.
func TestASetAfloatStandsItsCraft(t *testing.T) {
	given := smallAsked()
	given.set = afloatSet(t, "", "")
	world, made, given, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if made.things != 2 || given.eye == nil {
		t.Fatalf("%d things tallied, eye %v", made.things, given.eye)
	}
	stage := map[string]maths.Vec{"the can": {},
		"the stone": maths.V(3, 5, -4)}
	for _, each := range world.Things {
		at := each.At(0).Origin
		if !nearPoint(at, stage["the can"]) &&
			!nearPoint(at, stage["the stone"]) {
			t.Errorf("a part stands at %v, where no craft is", at)
		}
	}
	if _, dark := world.Sky.(darkness); !dark {
		t.Errorf("the sky is %T, want darkness", world.Sky)
	}
	if !nearPoint(given.eye.Origin, maths.V(-10, 0, 0)) {
		t.Errorf("the eye is at %v", given.eye.Origin)
	}
}

// A space recipe constructs the model named by its specification, rather
// than falling back to the generic spacecraft kind.
func TestASetAfloatBuildsItsSpacecraftRecipe(t *testing.T) {
	text := `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: roman recipe
  setting: space
  light: sun
  sun: {azimuth: 180, altitude: 45}
  recipes:
    roman observatory:
      kind: space-observatory
      spacecraft:
        design: roman
        aperture: {value: 2.4, unit: m}
  set:
    things:
      roman:
        recipe: roman observatory
        seed: 1
        east: 0
        north: 0
  film: {width: 24, height: 16, rays: 1, exposure: 1}
  cameras:
    aside:
      title: aside
      location: {at: [-20, 4, -20]}
      orientation: {look_at: [0, 0, 0]}
`
	path := filepath.Join(t.TempDir(), "roman-recipe.yaml")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	given := smallAsked()
	given.set = path
	world, made, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if made.things != 1 || len(world.Things) == 0 {
		t.Fatalf("made %d things, world has %d", made.things, len(world.Things))
	}
	for _, each := range world.Things {
		if each.Material.Name == "primary mirror" {
			return
		}
	}
	t.Fatal("the Roman recipe fell back to another spacecraft or was not drawn")
}

func TestASetAfloatCanUseModeledMoonlight(t *testing.T) {
	given := smallAsked()
	given.set = afloatSet(t, "light: sun", "light: moon\n  moon: {azimuth: 180, altitude: 45, disc: false}")
	world, made, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if made.lamps != 1 || len(world.Lamps) != 1 {
		t.Fatalf("moonlight lamps=%d/%d, want one physically sourced distant moon", made.lamps, len(world.Lamps))
	}
	if _, ok := world.Lamps[0].(emit.Distant); !ok {
		t.Fatalf("moonlight source is %T, want emit.Distant", world.Lamps[0])
	}
	if _, ok := world.Sky.(moonlight.Skies); !ok {
		t.Fatalf("moonlit sky is %T, want computed moonlight skies", world.Sky)
	}
	markers := placement.WorkingLights(world)
	if len(markers) != 1 || markers[0].Kind != "Moon" || markers[0].Author.Name != "moon" {
		t.Fatalf("producer lost its authored Moon identity: %+v", markers)
	}
}

// Nothing in a set in space stands on ground: no part of either craft is
// lifted to a ground's height, and none stands below the lowest said.
func TestASetAfloatHasNoGround(t *testing.T) {
	given := smallAsked()
	given.set = afloatSet(t, "", "")
	world, _, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	for _, each := range world.Things {
		if at := each.At(0).Origin; at.Y != 0 && at.Y != 5 {
			t.Errorf("a part stands %v up, not where the set says", at.Y)
		}
	}
	if given.floor != 0 {
		t.Errorf("the floor is %v, want none", given.floor)
	}
}

// The sun of a set in space is one distant lamp from its way, below
// level here, as bright as sunlight at the earth at the film's exposure.
func TestTheSunAfloatIsOneLampAnywhere(t *testing.T) {
	given := smallAsked()
	given.set = afloatSet(t, "", "")
	world, made, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Lamps) != 1 || made.lamps != 1 {
		t.Fatalf("%d lamps, tallied %d, want the sun's one",
			len(world.Lamps), made.lamps)
	}
	want, err := emit.NewDistant(skyward(180, -45),
		maths.Grey(sunlightAtEarth*2))
	if err != nil {
		t.Fatal(err)
	}
	gotWay, _, gotLight := world.Lamps[0].Light(maths.Vec{}, maths.Vec{})
	wantWay, _, wantLight := want.Light(maths.Vec{}, maths.Vec{})
	if !nearPoint(gotWay, wantWay) || gotLight != wantLight {
		t.Errorf("the sun is from %v with %v, want from %v with %v",
			gotWay, gotLight, wantWay, wantLight)
	}
}

// A set in space lit by none has no lamp, and its sun, if it says one,
// is not lit.
func TestASetAfloatLitByNoneHasNoSun(t *testing.T) {
	given := smallAsked()
	given.set = afloatSet(t, "light: sun", "light: none")
	world, made, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Lamps) != 0 || made.lamps != 0 {
		t.Errorf("%d lamps, tallied %d, want none", len(world.Lamps),
			made.lamps)
	}
}

// A craft that says a heading is turned by it, clockwise from north, as a
// thing on the ground is.
func TestACraftIsTurnedByItsHeading(t *testing.T) {
	given := smallAsked()
	given.set = afloatSet(t, "        north: 0\n",
		"        north: 0\n        heading: {value: 90, unit: deg}\n")
	world, _, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	turned := world.Things[0].At(0).Orientation.Out(maths.V(0, 0, -1))
	if !nearPoint(turned, maths.V(1, 0, 0)) {
		t.Errorf("a craft headed east faces %v, want east", turned)
	}
}

// A craft that says no heading is not turned: it has no seeded turn, as
// a thing on the ground has.
func TestACraftThatSaysNoHeadingIsNotTurned(t *testing.T) {
	given := smallAsked()
	given.set = afloatSet(t, "", "")
	world, _, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	for _, each := range world.Things {
		facing := each.At(0).Orientation.Out(maths.V(0, 0, -1))
		if !nearPoint(facing, maths.V(0, 0, -1)) {
			t.Errorf("a craft that says no heading faces %v", facing)
		}
	}
}

// A set in space whose world half says a sky has it hung, and not the
// darkness.
func TestASetAfloatHangsItsWorldsSky(t *testing.T) {
	given := smallAsked()
	given.set = afloatSet(t, "  film:", "  world:\n    sky:\n"+
		"      horizon: {srgb_linear: [0.1, 0.2, 0.3]}\n"+
		"      zenith: {srgb_linear: [0.1, 0.2, 0.3]}\n"+
		"      ground: {srgb_linear: [0.1, 0.2, 0.3]}\n"+
		"      source: eye\n  film:")
	world, _, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if _, hung := world.Sky.(sky.Gradient); !hung {
		t.Errorf("the sky is %T, want the world half's", world.Sky)
	}
}

// A set in space refuses, by name, what it says that the stage does not
// draw: a light that is not the sun, moon or none, a sun or moon not said, what stands
// on ground, a planet, a burning craft, and the lake.
func TestASetAfloatRefusesWhatItDoesNotDraw(t *testing.T) {
	stand := "  set:\n    stands:\n      trees: {kind: cobble, count: 1, " +
		"seed: 1}\n    things:\n"
	planet := "  planet:\n    name: p\n    made_up: true\n    radius: 1.0e7\n" +
		"    gravity: 10\n    from_star: 1\n    columns:\n      a:\n" +
		"        - {top: 0, bottom: 1, gases: {neon: 1}}\n    zones:\n" +
		"      - {south: -90, north: 90, column: a}\n  film:"
	for _, row := range []struct{ was, now, says string }{
		{"light: sun", "light: moon", "says no moon"},
		{"light: sun", "light: ambient", `not "ambient"`},
		{"  sun: {azimuth: 180, altitude: -45}\n", "", "says no sun"},
		{"  set:\n    things:\n", stand, "stand trees says no within"},
		{"  film:", planet, "says a planet"},
		{"        north: 0\n", "        north: 0\n        alight: true\n",
			"the can: a craft in space says alight"},
	} {
		given := smallAsked()
		given.set = afloatSet(t, row.was, row.now)
		_, _, _, err := staged(given)
		if err == nil || !strings.Contains(err.Error(), row.says) ||
			!strings.HasPrefix(err.Error(), given.set) {
			t.Errorf("%q: staged says %v, want it to name the set and "+
				"say %q", row.now, err, row.says)
		}
	}
	given := smallAsked()
	given.lake = 1
	given.said.fantasy = map[string]bool{"lake": true}
	given.set = afloatSet(t, "", "")
	if _, _, _, err := staged(given); err == nil ||
		!strings.Contains(err.Error(), "-lake") {
		t.Errorf("-lake in space: %v", err)
	}
}

// A thing in space of a kind no stage builds is refused by its kind's
// name, before anything is staged.
func TestACraftOfAKindNotBuiltIsRefusedByName(t *testing.T) {
	for _, kind := range []string{"star", "lantern", "point", "nebula",
		"culture-orbital"} {
		given := smallAsked()
		given.set = afloatSet(t, "kind: can", "kind: "+kind)
		_, _, _, err := staged(given)
		if err == nil || !strings.Contains(err.Error(),
			"thing.kind: "+kind+" is read and not yet drawn") {
			t.Errorf("%s: staged says %v", kind, err)
		}
	}
}

// The sun's lamp is as bright as the measured constant at the film's
// exposure, and no other: a film twice as exposed has a sun twice as
// bright.
func TestTheSunAfloatFollowsTheExposure(t *testing.T) {
	given := smallAsked()
	given.set = afloatSet(t, "exposure: 2", "exposure: 4")
	world, _, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	_, _, light := world.Lamps[0].Light(maths.Vec{}, maths.Vec{})
	if got := light.G; math.Abs(got-sunlightAtEarth*4) > 1e-9 {
		t.Errorf("the sun at exposure 4 is %v, want %v", got,
			sunlightAtEarth*4)
	}
}

// sets/webb-at-l2.yaml stages what smoke 50 builds: Webb at the origin,
// the Larson at (0, 12, -12), the eye at (-10, 4, -30), the sun from due
// south and 45 degrees below level, and the Larson's own lamps lit.
func TestWebbAtL2StagesWhatSmokeFiftyBuilds(t *testing.T) {
	given := smallSetAsked(filepath.Join("..", "sets", "webb-at-l2.yaml"))
	world, made, given, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []maths.Vec{{}, maths.V(0, 12, -12)} {
		stood := false
		for _, each := range world.Things {
			stood = stood || nearPoint(each.At(0).Origin, want)
		}
		if !stood {
			t.Errorf("nothing stands at %v", want)
		}
	}
	if !nearPoint(given.eye.Origin, maths.V(-10, 4, -30)) {
		t.Errorf("the eye is at %v", given.eye.Origin)
	}
	way, _, _ := world.Lamps[0].Light(maths.Vec{}, maths.Vec{})
	if !nearPoint(way, maths.V(0, -math.Sin(math.Pi/4), math.Cos(math.Pi/4))) {
		t.Errorf("the sun is from %v", way)
	}
	if made.things != 2 || made.lamps < 2 {
		t.Errorf("%d things and %d lamps, want Webb, the Larson, the sun "+
			"and the Larson's", made.things, made.lamps)
	}
}

// Two craft of one kind, seed and complexity are made once and stood
// twice: the stock keeps one.
func TestCraftAlikeAreMadeOnce(t *testing.T) {
	given := smallAsked()
	given.set = afloatSet(t, "", "")
	_, once, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	given.set = afloatSet(t, "        kind: boulder\n        seed: 2\n",
		"        kind: can\n        seed: 1\n")
	_, twice, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	can := smallAsked()
	can.set = afloatSet(t, "      the stone:\n        kind: boulder\n"+
		"        seed: 2\n        east: 3\n        north: 4\n"+
		"        height: {value: 5, unit: m}\n", "")
	_, alone, _, err := staged(can)
	if err != nil {
		t.Fatal(err)
	}
	if twice.things != 2 || twice.kept != alone.kept ||
		once.kept == alone.kept {
		t.Errorf("two cans keep %d faces, one keeps %d, a can and a stone "+
			"%d", twice.kept, alone.kept, once.kept)
	}
}

// A craft of a complexity its kind is not made at is refused, by the
// set, the craft and the manufactory's reason.
func TestACraftAtAComplexityItsKindLacksIsRefused(t *testing.T) {
	given := smallAsked()
	given.set = afloatSet(t, "        north: 0\n",
		"        north: 0\n        complexity: demo\n")
	_, _, _, err := staged(given)
	if err == nil || !strings.HasPrefix(err.Error(), given.set+": the can: ") ||
		!strings.Contains(err.Error(), "demo") {
		t.Errorf("a can at demo: %v", err)
	}
}
