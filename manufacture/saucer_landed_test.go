package manufacture

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
	"pgregory.net/rapid"
)

// gearOfTheTests is the landed saucer's gear, read from the embedded
// file against the saucer's own size.
func gearOfTheTests(t *testing.T) (saucerGear, saucerSize) {
	t.Helper()
	size := saucerShape()
	gear, err := readGear(saucerLandedRaw, size)
	if err != nil {
		t.Fatal(err)
	}
	return gear, size
}

// geared counts a thing's parts that are gear: the legs' and the
// stairs', every slot from the legs' on.
func geared(made good) int {
	count := 0
	for _, each := range made.parts {
		if each.of >= saucerLeg {
			count++
		}
	}
	return count
}

// The landed Larson is the flying one with its gear added: three parts
// a leg, a strut, a brace and a pad, and every part the stairs
// constructor laid, and it reaches down to the ground the stairs set.
func TestTheLandedLarsonStandsOnItsGear(t *testing.T) {
	gear, size := gearOfTheTests(t)
	flying := mustMake(t, "mcrn-larson", 5)
	landed := mustMake(t, "mcrn-larson-landed", 5)
	const partsALeg = 3
	want := gear.Legs.Count*partsALeg + len(gear.pieces)
	if got := geared(landed); got != want {
		t.Errorf("the landed saucer has %d parts of gear, want %d", got, want)
	}
	if len(landed.parts) != len(flying.parts)+want {
		t.Errorf("landed has %d parts and flying %d; the gear is %d",
			len(landed.parts), len(flying.parts), want)
	}
	// landed, the saucer's origin is the ground under its feet, for a
	// scene stands a thing's origin on the ground: its box starts at
	// nought, and its top is the flying saucer's, raised by the height
	// the gear holds the hull at
	if landed.extent == nil || math.Abs(landed.extent.Min.Y) > 1e-12 {
		t.Errorf("the landed saucer's box starts at %v, not at the ground, nought",
			landed.extent)
	}
	held := -gear.ground(size)
	if math.Abs(landed.extent.Max.Y-(flying.extent.Max.Y+held)) > 1e-12 {
		t.Errorf("the landed saucer's top is at %v; the flying one's, %v, "+
			"raised by the %v m the gear holds it at, is %v",
			landed.extent.Max.Y, flying.extent.Max.Y, held,
			flying.extent.Max.Y+held)
	}
	// every light is above the ground, and the belly light under the hull
	lights, err := landed.Lights(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	const cabinBellyAndPortholes = 14
	if len(lights) != cabinBellyAndPortholes {
		t.Errorf("the landed saucer carries %d lights, want the cabin's, "+
			"the belly's and twelve portholes'", len(lights))
	}
	for _, each := range lights {
		if !(each.At().Y > 0) {
			t.Errorf("the light %q is at %v, not above the ground",
				each.Name(), each.At())
		}
		if each.Name() == bellyName && !(each.At().Y < held) {
			t.Errorf("the belly light is at %v, not under the hull at %v",
				each.At().Y, held)
		}
	}
	for of := range gear.worn {
		if landed.dressed[of] == nil {
			t.Errorf("the landed saucer has no look for slot %d", of)
		}
	}
}

// the anti-test of the one above: the saucer in flight has no gear at
// all, no look for any, and the box it always had
func TestTheFlyingLarsonCarriesNoGear(t *testing.T) {
	size := saucerShape()
	flying := mustMake(t, "mcrn-larson", 5)
	if got := geared(flying); got != 0 {
		t.Errorf("the saucer in flight carries %d parts of gear", got)
	}
	for of := range flying.dressed {
		if of >= saucerLeg {
			t.Errorf("the saucer in flight has a look for gear slot %d", of)
		}
	}
	if flying.extent == nil || *flying.extent != size.extent() {
		t.Errorf("the saucer in flight's box is %v, not %v",
			flying.extent, size.extent())
	}
	if flying.extent.Min.Y != -size.drop {
		t.Errorf("the saucer in flight reaches down to %v, not its belly "+
			"disc at %v", flying.extent.Min.Y, -size.drop)
	}
}

// The ground is where the stairs put it: the top tread's top is the
// underside of the belly disc, to rounding.
func TestTheTopTreadMeetsTheBellyDisc(t *testing.T) {
	gear, size := gearOfTheTests(t)
	ground := gear.ground(size)
	flight := gear.Stairs
	top := ground + flight.Treads.First.Y +
		float64(flight.Steps-1)*flight.Rise + flight.Treads.Thickness
	if math.Abs(top-(-size.drop)) > 1e-12 {
		t.Errorf("the top tread's top is at %v; the belly disc's underside "+
			"is at %v", top, -size.drop)
	}
}

// the anti-test of the one above: the ground is below the saucer, and a
// flight one step taller puts it one rise lower
func TestTheGroundIsBelowAndFollowsTheStairs(t *testing.T) {
	gear, size := gearOfTheTests(t)
	ground := gear.ground(size)
	if !(ground < -size.drop) {
		t.Errorf("the ground at %v is not below the belly disc at %v",
			ground, -size.drop)
	}
	taller := gear
	taller.Stairs.Steps++
	if got := taller.ground(size); math.Abs(got-(ground-gear.Stairs.Rise)) > 1e-12 {
		t.Errorf("one more step puts the ground at %v, not one rise below %v",
			got, ground)
	}
}

// hullAbove is the height of whatever of the hull hangs over a point of
// the saucer's own frame: the belly disc's underside inside its radius,
// the hull's underside outside it.
func hullAbove(size saucerSize, at maths.Vec) float64 {
	across := math.Hypot(at.X, at.Z)
	if across <= size.disc {
		return -size.drop
	}
	return size.under(across)
}

// No part of the stairs pokes up through the hull: every tread's top and
// both ends of every rod, radius and all, are at or below the hull over
// them.
func TestNoPartOfTheStairsPokesThroughTheHull(t *testing.T) {
	gear, size := gearOfTheTests(t)
	foot := maths.V(0, gear.ground(size), gear.Foot)
	const touching = 1e-9
	for index, piece := range gear.pieces {
		if piece.Box != nil {
			box := piece.Box
			for _, corner := range []maths.Vec{
				box.At, box.At.Add(maths.V(box.Size.X, 0, 0)),
				box.At.Add(maths.V(0, 0, box.Size.Z)),
				box.At.Add(maths.V(box.Size.X, 0, box.Size.Z))} {
				top := corner.Add(maths.V(0, box.Size.Y, 0)).Add(foot)
				if top.Y > hullAbove(size, top)+touching {
					t.Errorf("part %d, a %s box, reaches %v under a hull at %v",
						index, piece.Material, top.Y, hullAbove(size, top))
				}
			}
			continue
		}
		for _, end := range []maths.Vec{piece.Rod.From, piece.Rod.To} {
			at := end.Add(foot)
			if at.Y+piece.Rod.Radius > hullAbove(size, at)+touching {
				t.Errorf("part %d, a %s rod, reaches %v under a hull at %v",
					index, piece.Material, at.Y+piece.Rod.Radius,
					hullAbove(size, at))
			}
		}
	}
}

// the anti-test of the one above: the check can fail. A rail run on to
// the top of the flight, as the yard's stairs run theirs, would stand
// inside the hull.
func TestARailRunToTheTopWouldPokeThroughTheHull(t *testing.T) {
	gear, size := gearOfTheTests(t)
	rail := gear.Stairs.Rail
	slope := (rail.High.Y - rail.Low.Y) / (rail.Low.Z - rail.High.Z)
	top := -float64(gear.Stairs.Steps-1) * gear.Stairs.Going
	high := maths.V(rail.Offset, rail.Low.Y+slope*(rail.Low.Z-top), top)
	at := high.Add(maths.V(0, gear.ground(size), gear.Foot))
	if !(at.Y+rail.Radius > hullAbove(size, at)) {
		t.Errorf("a rail run to the top reaches %v, under a hull at %v: "+
			"the check above could not fail", at.Y+rail.Radius,
			hullAbove(size, at))
	}
}

// No leg stands in front of the stairs: a foot on the stairs' side of
// the hull stands clear of the flight's width, pad and all.
func TestNoLegStandsInFrontOfTheStairs(t *testing.T) {
	gear, _ := gearOfTheTests(t)
	legs := gear.Legs
	front := 0
	for each := range legs.Count {
		angle := legs.Turn*maths.Degree +
			fullTurn*float64(each)/float64(legs.Count)
		across, along := legs.Foot*math.Cos(angle), -legs.Foot*math.Sin(angle)
		if along <= 0 {
			continue
		}
		front++
		if !(math.Abs(across)-legs.Pad.Radius > gear.Stairs.Width/2) {
			t.Errorf("leg %d's foot, %v m to the side, stands in the stairs' "+
				"way, which is %v m wide", each, across, gear.Stairs.Width)
		}
	}
	if front == 0 {
		t.Error("no leg stands on the stairs' side of the hull at all")
	}
}

// the anti-test of the one above: turned a quarter turn, the first leg
// would stand right in front of the stairs
func TestALegTurnedAQuarterTurnWouldBlockTheStairs(t *testing.T) {
	gear, _ := gearOfTheTests(t)
	legs := gear.Legs
	const quarter = 270
	angle := quarter * maths.Degree
	across, along := legs.Foot*math.Cos(angle), -legs.Foot*math.Sin(angle)
	if !(along > 0 && math.Abs(across)-legs.Pad.Radius < gear.Stairs.Width/2) {
		t.Errorf("a leg at %d degrees stands at %v, %v: not in the stairs' way",
			quarter, across, along)
	}
}

// The gear's looks are made of what the yaml names, by thing: the legs
// titanium, the treads aluminium with aluminium's own published optics,
// and one slot for each material the stairs name.
func TestTheGearsLooksAreMadeOfWhatItNames(t *testing.T) {
	gear, _ := gearOfTheTests(t)
	legs := gear.worn[saucerLeg]
	if legs == nil || legs.Name != "titanium" || legs.Metal == nil {
		t.Fatalf("the legs' look is %+v, not titanium made a metal", legs)
	}
	if len(gear.slots) != 2 {
		t.Errorf("the stairs name %d materials, want aluminium and titanium",
			len(gear.slots))
	}
	treads := gear.worn[gear.slots["aluminium"]]
	if treads == nil || treads.Metal == nil {
		t.Fatalf("the treads' look is %+v, not aluminium made a metal", treads)
	}
	aluminium, err := materials.ByName("aluminium")
	if err != nil {
		t.Fatal(err)
	}
	const blue = 2
	if got, want := treads.Metal.N[blue], aluminium.Optics().At(materials.Blue).N(); got != want {
		t.Errorf("the treads' blue n is %v; aluminium's published row has %v",
			got, want)
	}
	for _, piece := range gear.pieces {
		if _, known := gear.slots[piece.Material]; !known {
			t.Errorf("a part of the stairs is made of %q, which has no slot",
				piece.Material)
		}
	}
}

// the refusals of readGear: a gear that cannot be built is not read.
// Each is the embedded file with one thing changed.
func TestAGearThatCannotBeBuiltIsRefused(t *testing.T) {
	size := saucerShape()
	whole := string(saucerLandedRaw)
	refused := map[string]struct{ was, is, names string }{
		"no legs":                 {"count: 3", "count: 0", "legs"},
		"a leg with no thickness": {"radius: 0.07", "radius: 0", "leg"},
		"a leg past the rim":      {"attach: 3.3", "attach: 9", "leg"},
		"a foot past the rim":     {"foot: 4.4", "foot: 5.9", "leg"},
		"a brace off the flat":    {"attach: 1.9", "attach: 2.5", "brace"},
		"a pad of no size":        {"thickness: 0.06", "thickness: 0", "pad"},
		"legs of nothing known":   {"  material: titanium\n  finish: weathered", "  material: unobtainium\n  finish: weathered", "unobtainium"},
		"legs finished unknown":   {"  material: titanium\n  finish: weathered", "  material: titanium\n  finish: sparkly", "sparkly"},
		"treads of no metal":      {"material: aluminium", "material: timber", "timber"},
		"stairs of no steps":      {"steps: 9", "steps: 0", "steps"},
		"one metal, two finishes": {"    material: titanium\n    finish: weathered", "    material: titanium\n    finish: satin", "titanium"},
		"not yaml at all":         {"legs:", "legs: [", ""},
	}
	for name, change := range refused {
		if !strings.Contains(whole, change.was) {
			t.Fatalf("%s: the file has no %q to change", name, change.was)
		}
		broken := strings.Replace(whole, change.was, change.is, 1)
		_, err := readGear([]byte(broken), size)
		if err == nil {
			t.Errorf("%s: the gear was read", name)
			continue
		}
		if !strings.Contains(err.Error(), change.names) {
			t.Errorf("%s: the refusal does not say %q: %v",
				name, change.names, err)
		}
	}
}

// the anti-test of the one above: the embedded file itself is read, with
// every part the flight lays: nine treads, and on each side a stringer,
// a rail and two posts.
func TestTheEmbeddedGearIsRead(t *testing.T) {
	gear, _ := gearOfTheTests(t)
	const sides, aSide = 2, 4
	if want := gear.Stairs.Steps + sides*aSide; len(gear.pieces) != want {
		t.Errorf("the stairs are %d parts, want %d", len(gear.pieces), want)
	}
	if saucerLandedError != nil {
		t.Errorf("the gear read at start refused: %v", saucerLandedError)
	}
	if got := landedGear(); len(got.pieces) != len(gear.pieces) {
		t.Errorf("the gear read at start has %d parts, a fresh reading %d",
			len(got.pieces), len(gear.pieces))
	}
}

// The hull's underside is flat across the belly and rises to the rim:
// nought at the flat's edge, the bowl's height at the rim.
func TestTheUndersideIsFlatThenRisesToTheRim(t *testing.T) {
	size := saucerShape()
	if got := size.under(0); got != 0 {
		t.Errorf("the underside at the axis is at %v", got)
	}
	if got := size.under(size.belly); got != 0 {
		t.Errorf("the underside at the flat's edge is at %v", got)
	}
	if got := size.under(size.radius); math.Abs(got-size.bowl) > 1e-12 {
		t.Errorf("the underside at the rim is at %v, not the bowl's %v",
			got, size.bowl)
	}
}

// the anti-test of the one above, as a property: the underside never
// dips below the flat, never stands above the rim, and never comes down
// going outward
func TestTheUndersideNeverDipsNorComesDownGoingOut(t *testing.T) {
	size := saucerShape()
	rapid.Check(t, func(draw *rapid.T) {
		near := rapid.Float64Range(0, size.radius).Draw(draw, "near")
		far := rapid.Float64Range(near, size.radius).Draw(draw, "far")
		low, high := size.under(near), size.under(far)
		if low < 0 || high > size.bowl+1e-12 {
			draw.Fatalf("the underside is at %v and %v, outside nought to %v",
				low, high, size.bowl)
		}
		if high < low {
			draw.Fatalf("the underside comes down going out: %v at %v, %v at %v",
				low, near, high, far)
		}
	})
}
