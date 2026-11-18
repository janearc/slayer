package searching

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// setPlan is a plan with the values the world builders read.
func setPlan() Plan {
	return Plan{Azimuth: 25, Altitude: 55, Diameter: 38, Aperture: 8, ArrayCount: 6}
}

// newSet builds a site's set and fails the test if it cannot.
func newSet(t *testing.T, site string) *Set {
	t.Helper()
	set, err := NewSet(setPlan(), site)
	if err != nil {
		t.Fatalf("%s: %v", site, err)
	}
	return set
}

// TestGroundHeightFollowsTheSummit checks the authored ground at the
// centre, on the plateau and on the slope, against values worked by hand.
func TestGroundHeightFollowsTheSummit(t *testing.T) {
	for _, each := range []struct{ x, z, want float64 }{
		{0, 0, 0},
		{30, 0, 0},
		{30, 30, 0.407919},
		{100, 0, -6.05},
		{0, 100, -6.05},
		{-100, 0, -6.05},
	} {
		if got := groundHeight(each.x, each.z); !within(got, each.want, 1e-5) {
			t.Errorf("groundHeight(%g, %g) = %g, wanted %g", each.x, each.z, got, each.want)
		}
	}
}

// TestStringsRadioNamesTheTwoArraySites checks the site inventory.
func TestStringsRadioNamesTheTwoArraySites(t *testing.T) {
	for site, want := range map[string]bool{"radio-hawaii": true, "radio-atacama": true,
		"hawaii": false, "atacama": false, "blazar": false, "radio": false, "": false} {
		if got := stringsRadio(site); got != want {
			t.Errorf("stringsRadio(%q) = %v", site, got)
		}
	}
}

// TestNewSetBuildsAnObservatory checks the optical sites: one actor, four
// beams, the sun and moon lamps after them, two volumes, and a name.
func TestNewSetBuildsAnObservatory(t *testing.T) {
	hawaii, atacama := newSet(t, "hawaii"), newSet(t, "atacama")
	for _, set := range []*Set{hawaii, atacama} {
		if len(set.Actors) != 1 || len(set.Lasers) != 4 || len(set.World.Lamps) != 6 ||
			len(set.World.Volumes) != 2 || set.Mist == nil || set.Space {
			t.Errorf("%s: %d actors, %d lasers, %d lamps, %d volumes", set.Site,
				len(set.Actors), len(set.Lasers), len(set.World.Lamps), len(set.World.Volumes))
		}
		if len(set.Actors[0].things) == 0 {
			t.Errorf("%s: the observatory has no parts", set.Site)
		}
	}
	if hawaii.String() != "searching/hawaii" || atacama.String() != "searching/atacama" {
		t.Errorf("names %q, %q", hawaii, atacama)
	}
	if _, ok := hawaii.prototypes["coast-redwood/77"]; !ok {
		t.Error("hawaii has no trees")
	}
	if _, ok := atacama.prototypes["coast-redwood/77"]; ok {
		t.Error("atacama has trees")
	}
	if len(hawaii.World.Things) <= len(atacama.World.Things) {
		t.Errorf("hawaii has %d things, atacama %d", len(hawaii.World.Things), len(atacama.World.Things))
	}
}

// TestNewSetRefusesAnObservatoryWithNoDome checks that a zero diameter is
// refused by the factory and reported.
func TestNewSetRefusesAnObservatoryWithNoDome(t *testing.T) {
	plan := setPlan()
	plan.Diameter = 0
	set, err := NewSet(plan, "hawaii")
	if err == nil || set != nil {
		t.Fatalf("got %v, %v", set, err)
	}
}

// TestNewSetBuildsAnArray checks the radio sites: one actor a dish, none of
// the beams, each dish on the golden-angle spiral, and a wider layout and
// larger dishes at the second site.
func TestNewSetBuildsAnArray(t *testing.T) {
	hawaii, atacama := newSet(t, "radio-hawaii"), newSet(t, "radio-atacama")
	for _, set := range []*Set{hawaii, atacama} {
		if len(set.Actors) != 6 || len(set.Lasers) != 0 || len(set.World.Lamps) != 2 {
			t.Fatalf("%s: %d actors, %d lasers, %d lamps", set.Site, len(set.Actors), len(set.Lasers), len(set.World.Lamps))
		}
	}
	if hawaii.Actors[0].at != (maths.V(0, 0, 0)) {
		t.Errorf("first dish at %v", hawaii.Actors[0].at)
	}
	for index := 1; index < 6; index++ {
		radius := 17 * math.Sqrt(float64(index))
		for _, each := range []struct {
			set    *Set
			factor float64
			scale  float64
		}{{hawaii, 1, 1}, {atacama, 1.55, 1.4}} {
			dish := each.set.Actors[index]
			if got := math.Hypot(dish.at.X, dish.at.Z); !within(got, radius*each.factor, 1e-9) || dish.scale != each.scale {
				t.Errorf("%s dish %d: radius %g scale %g, wanted %g and %g", each.set.Site, index, got, dish.scale, radius*each.factor, each.scale)
			}
		}
	}
	// The second dish sits a golden angle round: its angle from the first
	// axis is about 137.5 degrees, so x is negative and z positive.
	second := hawaii.Actors[1].at
	if second.X >= 0 || second.Z <= 0 || !within(second.X, -12.5353, 1e-3) {
		t.Errorf("second dish at %v", second)
	}
	if want := groundHeight(second.X, second.Z); second.Y != want {
		t.Errorf("second dish stands at height %g, ground is %g", second.Y, want)
	}
}

// TestNewSetBuildsTheBlazar checks the space set: no ground, four volumes
// and the three materials the views change.
func TestNewSetBuildsTheBlazar(t *testing.T) {
	set := newSet(t, "blazar")
	if !set.Space || len(set.Actors) != 0 || len(set.World.Volumes) != 4 || len(set.World.Things) != 1 ||
		set.Disc == nil || set.Jet == nil || set.Lobe == nil || set.JetField == nil || set.World.Sky == nil {
		t.Fatalf("blazar: %d volumes, %d things, space %v", len(set.World.Volumes), len(set.World.Things), set.Space)
	}
	if set.String() != "searching/blazar" {
		t.Errorf("name %q", set)
	}
}

// TestSceneryReusesItsPrototype checks that a second boulder of the same
// seed adds the same parts without growing the prototype cache, and that an
// unknown kind is refused.
func TestSceneryReusesItsPrototype(t *testing.T) {
	set := &Set{World: thing.NewWorld(), prototypes: map[string][]thing.Thing{}}
	if err := set.scenery("boulder", 5, maths.V(1, 2, 3), 1); err != nil {
		t.Fatal(err)
	}
	first := len(set.World.Things)
	if first == 0 || len(set.prototypes) != 1 {
		t.Fatalf("%d things, %d prototypes", first, len(set.prototypes))
	}
	if err := set.scenery("boulder", 5, maths.V(40, 2, 3), 2); err != nil {
		t.Fatal(err)
	}
	if len(set.World.Things) != 2*first || len(set.prototypes) != 1 {
		t.Fatalf("%d things, %d prototypes after a second", len(set.World.Things), len(set.prototypes))
	}
	last := set.World.Things[len(set.World.Things)-1]
	pose := last.Motion.At(0)
	if pose.Origin != (maths.V(40, 2, 3)) || pose.Scale != 2 {
		t.Errorf("second boulder pose %+v", pose)
	}
	if err := set.scenery("no-such-thing", 1, maths.Vec{}, 1); err == nil {
		t.Error("an unknown kind was made")
	}
}

// TestUpdatePlacesTheCameraByShot checks the eye, worked from the camera
// angle by hand, for the three kinds of ground site, and the lens widening
// on a flash.
func TestUpdatePlacesTheCameraByShot(t *testing.T) {
	for _, each := range []struct {
		site    string
		x, y, z float64
	}{
		{"hawaii", 42.261826, 27, -90.630779},
		{"atacama", 86.054177, 27, -50.937987},
		{"radio-hawaii", 80.29747, 75, -172.19848},
	} {
		set := newSet(t, each.site)
		eye, wide, err := set.Update(setPlan(), State{Aim: 25, Flash: 1})
		if err != nil {
			t.Fatal(err)
		}
		got := eye.Origin
		if !within(got.X, each.x, 1e-5) || !within(got.Y, each.y, 1e-9) || !within(got.Z, each.z, 1e-5) || wide != 52 {
			t.Errorf("%s: eye %v, lens %g", each.site, got, wide)
		}
	}
	set := newSet(t, "hawaii")
	if _, wide, err := set.Update(setPlan(), State{}); err != nil || wide != 55 {
		t.Errorf("an ordinary frame has lens %g, %v", wide, err)
	}
}

// TestUpdateTurnsEveryActor checks that each part of each actor is posed at
// its place, turned by the aim: a quarter turn moves local x to world z.
func TestUpdateTurnsEveryActor(t *testing.T) {
	set := newSet(t, "radio-hawaii")
	if _, _, err := set.Update(setPlan(), State{Aim: 90}); err != nil {
		t.Fatal(err)
	}
	for index, dish := range set.Actors {
		for _, part := range dish.things {
			pose := part.Motion.At(0)
			if pose.Origin != dish.at || pose.Scale != dish.scale {
				t.Fatalf("dish %d pose %+v", index, pose)
			}
		}
	}
	part := set.Actors[1].things[0].Motion.At(0)
	got := part.Out(maths.V(1, 0, 0))
	want := set.Actors[1].at.Add(maths.V(0, 0, 1))
	if !near(got, want, 1e-9) {
		t.Errorf("local x maps to %v, wanted %v", got, want)
	}
}

// near reports whether two points agree to a tolerance.
func near(one, other maths.Vec, tolerance float64) bool {
	return one.Sub(other).Len() <= tolerance
}

// TestUpdateLightsByTheSceneryClock checks the sun and moon at midnight,
// noon and a time between, with strengths worked by hand: the sun is up
// only by day and the moon by night, and the lamps after the beams follow.
func TestUpdateLightsByTheSceneryClock(t *testing.T) {
	set := newSet(t, "hawaii")
	update := func(seconds float64) {
		t.Helper()
		if _, _, err := set.Update(setPlan(), State{SceneSeconds: seconds}); err != nil {
			t.Fatal(err)
		}
	}
	// At the clock's start the sun is straight down; its direction is
	// (0, -1, 0.35) before scaling, which is 1.0595 long.
	update(0)
	norm := math.Hypot(1, 0.35)
	if !within(set.Sun.Toward.Y, -1/norm, 1e-9) || set.Sun.Strength.R != 0 {
		t.Errorf("midnight sun %+v", set.Sun)
	}
	if !within(set.Moon.Toward.Y, 1/norm, 1e-9) || !within(set.Moon.Strength.R, 0.24/norm, 1e-9) ||
		!within(set.Moon.Strength.G, 0.36/norm, 1e-9) || !within(set.Moon.Strength.B, 0.64/norm, 1e-9) {
		t.Errorf("midnight moon %+v", set.Moon)
	}
	update(43200)
	if !within(set.Sun.Toward.Y, 1/norm, 1e-9) || !within(set.Sun.Strength.R, 0.10, 1e-9) ||
		!within(set.Sun.Strength.G, 0.065, 1e-9) || !within(set.Sun.Strength.B, 0.035, 1e-9) {
		t.Errorf("noon sun %+v", set.Sun)
	}
	if set.Moon.Strength.R != 0 || set.Moon.Strength.G != 0 || set.Moon.Strength.B != 0 {
		t.Errorf("noon moon %+v", set.Moon)
	}
	// A quarter day in, the sun is on the horizon and gives no light.
	update(21600)
	if !within(set.Sun.Toward.Y, 0, 1e-9) || set.Sun.Strength.R != 0 {
		t.Errorf("sunrise sun %+v", set.Sun)
	}
	if set.World.Lamps[4] != thing.Lamp(set.Sun) || set.World.Lamps[5] != thing.Lamp(set.Moon) {
		t.Error("the sun and moon lamps were not refreshed")
	}
	for index, laser := range set.Lasers {
		if set.World.Lamps[index] != thing.Lamp(laser) {
			t.Errorf("beam %d lamp was not refreshed", index)
		}
	}
	if _, ok := set.World.Sky.(siteSky); !ok {
		t.Errorf("sky is %T", set.World.Sky)
	}
}

// TestUpdateAdvancesTheMist checks that the mist medium is replaced for each
// frame, and that a set with no mist and no beams still updates.
func TestUpdateAdvancesTheMist(t *testing.T) {
	set := newSet(t, "atacama")
	before := set.World.Volumes[0].Medium
	if _, _, err := set.Update(setPlan(), State{At: 3}); err != nil {
		t.Fatal(err)
	}
	if set.World.Volumes[0].Medium == before {
		t.Error("the mist did not advance")
	}
	bare := &Set{World: thing.NewWorld(), Site: "hawaii"}
	bare.World.Lamp(bare.Sun)
	bare.World.Lamp(bare.Moon)
	if _, _, err := bare.Update(setPlan(), State{}); err != nil {
		t.Fatalf("a set with no mist: %v", err)
	}
}

// TestUpdateOfTheBlazarChoosesTheView checks the space eye and the glow
// weights of each spectrum, and the fall back to the visible view.
func TestUpdateOfTheBlazarChoosesTheView(t *testing.T) {
	set := newSet(t, "blazar")
	eye, wide, err := set.Update(setPlan(), State{})
	if err != nil {
		t.Fatal(err)
	}
	if eye.Origin != (maths.V(8, 15, 68)) || wide != 52 {
		t.Errorf("eye %v, lens %g", eye.Origin, wide)
	}
	_, wide, err = set.Update(setPlan(), State{Flash: 1})
	if err != nil || wide != 49 {
		t.Errorf("flashing lens %g, %v", wide, err)
	}
	for _, each := range []struct {
		spectrum        string
		disc, jet, lobe float64
	}{
		{"radio", 0.07, 0.5, 4}, {"infrared", 1.5, 0.3, 0.3}, {"visible", 1, 1, 0.15},
		{"x-ray", 0.3, 2.5, 0.03}, {"gamma", 0.04, 5, 0.01}, {"unheard of", 1, 1, 0.15},
	} {
		if _, _, err := set.Update(setPlan(), State{Spectrum: each.spectrum, At: 2.5}); err != nil {
			t.Fatal(err)
		}
		if !within(set.Disc.Glow.R, 9*each.disc, 1e-9) || !within(set.Jet.Glow.B, 9*each.jet, 1e-9) ||
			!within(set.Lobe.Glow.B, 1.2*each.lobe, 1e-9) || set.JetField.Time != 2.5 {
			t.Errorf("%s: disc %v jet %v lobe %v time %g", each.spectrum, set.Disc.Glow, set.Jet.Glow, set.Lobe.Glow, set.JetField.Time)
		}
	}
	_, _, err = set.Update(setPlan(), State{Spectrum: "visible", Flash: 1})
	if err != nil || !within(set.Jet.Glow.B, 9*1.5, 1e-9) {
		t.Errorf("a flash brightens the jet: %v, %v", set.Jet.Glow, err)
	}
	moved := State{Camera: 25 * math.Pi}
	eye, _, err = set.Update(setPlan(), moved)
	if err != nil || !within(eye.Origin.X, 8+60*math.Sin(0.08), 1e-9) {
		t.Errorf("camera swing: eye %v, %v", eye.Origin, err)
	}
}

// TestSiteSkyHasDiscsAndStars checks the authored sky: the sun disc by day
// brightens its direction, the moon disc by night does, stars appear in
// some directions only at night, and the sun below the horizon draws none.
func TestSiteSkyHasDiscsAndStars(t *testing.T) {
	baseline := func(skybox siteSky, toward maths.Vec) maths.Light {
		horizon := maths.RGB(0.008+0.06*skybox.day, 0.011+0.022*skybox.day, 0.022+0.012*skybox.day)
		return sky.Gradient{Horizon: horizon, Zenith: maths.RGB(0.001, 0.002, 0.008).Scale(1 + skybox.day*3),
			Ground: maths.RGB(0.003, 0.004, 0.005)}.Light(toward.Unit())
	}
	day := siteSky{sun: maths.V(0, 1, 0), moon: maths.V(0, -1, 0), day: 1}
	if got, want := day.Light(maths.V(0, 1, 0)), baseline(day, maths.V(0, 1, 0)).Add(maths.RGB(0.8, 0.35, 0.08)); !lightNear(got, want) {
		t.Errorf("sun disc: %v, wanted %v", got, want)
	}
	if got, want := day.Light(maths.V(1, 1, 0)), baseline(day, maths.V(1, 1, 0)); !lightNear(got, want) {
		t.Errorf("sky off the disc: %v, wanted %v", got, want)
	}
	night := siteSky{sun: maths.V(0, -1, 0), moon: maths.V(0, 1, 0), day: 0}
	if got, want := night.Light(maths.V(0, 1, 0)), baseline(night, maths.V(0, 1, 0)).Add(maths.RGB(0.20, 0.27, 0.4)); !lightNear(got, want) {
		t.Errorf("moon disc: %v, wanted %v", got, want)
	}
	if got, want := night.Light(maths.V(0, -1, 0)), baseline(night, maths.V(0, -1, 0)); !lightNear(got, want) {
		t.Errorf("a sun below the horizon drew a disc: %v, wanted %v", got, want)
	}
	if got, want := (siteSky{sun: maths.V(0, 1, 0), moon: maths.V(0, 1, 0), day: 1}).Light(maths.V(0, 1, 0)),
		baseline(day, maths.V(0, 1, 0)).Add(maths.RGB(0.8, 0.35, 0.08)); !lightNear(got, want) {
		t.Errorf("sun and moon in one place: %v, wanted the sun", got)
	}
	// A scan of the noise found one star in four hundred thousand directions;
	// this is it. At night it lifts the sky by a tenth, as dusk and day do not.
	star := maths.V(-0.9545587021689343, -0.1841725, -0.23430359441824528)
	if got, want := night.Light(star), baseline(night, star).Add(maths.Grey(0.1)); !lightNear(got, want) {
		t.Errorf("star: %v, wanted %v", got, want)
	}
	if got, want := day.Light(star), baseline(day, star); !lightNear(got, want) {
		t.Errorf("the same direction by day: %v, wanted %v", got, want)
	}
	dusk := siteSky{sun: night.sun, moon: night.moon, day: 0.1}
	if got, want := dusk.Light(star), baseline(dusk, star); !lightNear(got, want) {
		t.Errorf("the same direction at dusk: %v, wanted %v", got, want)
	}
	neighbour := maths.V(-0.9545587021689343, -0.1941725, -0.23430359441824528)
	if got, want := night.Light(neighbour), baseline(night, neighbour); !lightNear(got, want) {
		t.Errorf("a direction beside the star: %v, wanted %v", got, want)
	}
}

// TestSetNamesItsSite checks the progress name of a hand-built set.
func TestSetNamesItsSite(t *testing.T) {
	if got := (&Set{Site: "somewhere"}).String(); got != "searching/somewhere" {
		t.Errorf("name %q", got)
	}
}

// lightNear reports whether two lights agree to a part in a trillion.
func lightNear(one, other maths.Light) bool {
	return within(one.R, other.R, 1e-12) && within(one.G, other.G, 1e-12) && within(one.B, other.B, 1e-12)
}

// TestUpdateRefusesALaserThatCannotBePlaced gives Update a plan whose
// diameter is not a number, so no beam can be aimed, and expects an error.
func TestUpdateRefusesALaserThatCannotBePlaced(t *testing.T) {
	set := newSet(t, "hawaii")
	plan := setPlan()
	plan.Diameter = math.NaN()
	if _, _, err := set.Update(plan, State{}); err == nil {
		t.Fatal("a beam with no position was accepted")
	}
}
