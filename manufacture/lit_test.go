package manufacture

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// the kinds that carry a light.
var lamps = []string{"campfire", "spotlight", "maser"}

func TestEveryLitKindIsASoundThingCarryingOneLight(t *testing.T) {
	for _, name := range lamps {
		for number := range uint64(20) {
			made := mustMake(t, name, number)
			if _, err := made.things(maths.Subspace{}); err != nil {
				t.Fatalf("%s, seed %d: %v", name, number, err)
			}
			if len(made.lights) != 1 {
				t.Fatalf("%s, seed %d: %d lights", name, number,
					len(made.lights))
			}
		}
		one, same := mustMake(t, name, 3), mustMake(t, name, 3)
		other := mustMake(t, name, 4)
		if one.lights[0] != same.lights[0] {
			t.Errorf("%s: the same seed lit it another way", name)
		}
		if one.lights[0] == other.lights[0] {
			t.Errorf("%s: another seed lit it the same way", name)
		}
	}
}

// a campfire is stones in a ring, sticks leaned in over the middle,
// and a fire: a glow, every way, flickering, a hand above the ground
// in the middle of the ring. the stones and the sticks are two
// matters and so two things, of two looks.
func TestACampfireIsStonesSticksAndAFire(t *testing.T) {
	row := fireRow()
	for number := range uint64(20) {
		made := mustMake(t, "campfire", number)
		things, err := made.things(maths.Subspace{})
		if err != nil {
			t.Fatal(err)
		}
		if len(things) != 2 ||
			things[0].Material == things[1].Material {
			t.Fatalf("seed %d: %d things, not stones and sticks",
				number, len(things))
		}
		stones, sticks := 0, 0
		sizes := map[form]bool{}
		for _, one := range made.parts {
			foot := one.space.Origin
			out := math.Hypot(foot.X, foot.Z)
			if _, stone := one.shape.(lump); stone {
				stones++
				sizes[one.shape] = true
				if !row.ring.holds(out) {
					t.Errorf("a stone %v out", out)
				}
				continue
			}
			sticks++
			// a stick stands inside the stones and leans in: its
			// length runs back toward the middle
			way := one.space.Orientation.Y
			inward := -(way.X*foot.X + way.Z*foot.Z)
			if out >= row.ring.least || inward <= 0 || way.Y <= 0 {
				t.Errorf("seed %d: a stick at %v running %v",
					number, foot, way)
			}
		}
		// each stone is a stone of its own, from its own seed
		if len(sizes) != stones {
			t.Errorf("seed %d: %d stones of only %d shapes", number,
				stones, len(sizes))
		}
		if !row.stones.holds(float64(stones)) ||
			!row.sticks.holds(float64(sticks)) {
			t.Errorf("seed %d: %d stones and %d sticks", number,
				stones, sticks)
		}
		fire := made.lights[0]
		half, _ := fire.made.angles()
		if !fire.made.colour.glows() || half.InRadians() != math.Pi {
			t.Errorf("seed %d: no glow every way", number)
		}
		if !row.heat.holds(fire.made.colour.heat.InKelvin()) ||
			!row.power.holds(fire.made.power.InWatts()) {
			t.Errorf("seed %d: a fire of %v K and %v W", number,
				fire.made.colour.heat.InKelvin(),
				fire.made.power.InWatts())
		}
		if fire.at.X != 0 || fire.at.Z != 0 || fire.at.Y <= 0 ||
			fire.at.Y > row.ring.least {
			t.Errorf("seed %d: the fire is at %v", number, fire.at)
		}
		if fire.made.share(0) == fire.made.share(time.Second/3) {
			t.Errorf("seed %d: the fire does not flicker", number)
		}
	}
}

// a spotlight is a head on a stand, and its light is a warm cone out
// of the front of the head, the way the head points.
func TestASpotlightShinesTheWayItsHeadPoints(t *testing.T) {
	row := spotRow()
	for number := range uint64(20) {
		made := mustMake(t, "spotlight", number)
		beam := made.lights[0]
		head := made.parts[len(made.parts)-1]
		axis := head.space.Orientation.Y
		if !beam.toward.Near(axis, close) {
			t.Errorf("seed %d: the beam goes %v, the head %v",
				number, beam.toward, axis)
		}
		// the head sits on top of its stand, over the stand's foot
		post := made.parts[0].shape.(tube)
		top := maths.V(0, inScene(post.length), 0)
		if !head.space.Origin.Near(top, close) {
			t.Errorf("seed %d: the head is at %v, the stand's top "+
				"at %v", number, head.space.Origin, top)
		}
		long := inScene(head.shape.(tube).length)
		front := head.space.Origin.Add(axis.Scale(long))
		if !beam.at.Near(front, 1e-9) {
			t.Errorf("seed %d: the light is at %v, the head's "+
				"front at %v", number, beam.at, front)
		}
		// the light starts past the rim of the head it sits in
		rim, out := inScene(head.shape.(tube).tip), inScene(beam.beyond)
		if out < rim {
			t.Errorf("seed %d: the light starts %v out, inside a "+
				"head %v across", number, out, rim)
		}
		half, soft := beam.made.angles()
		if !row.half.holds(half.InDegrees()) || soft.InDegrees() <= 0 {
			t.Errorf("seed %d: a cone of %v with an edge of %v",
				number, half.InDegrees(), soft.InDegrees())
		}
		if !beam.made.colour.glows() || axis.Y >= 0 {
			t.Errorf("seed %d: no warm light tipped down", number)
		}
	}
}

// a maser is a dish on a mount, and its light is hydrogen's line out
// of the dish, as narrow as the dish allows and no narrower. it
// cannot be seen.
func TestAMaserIsALineThroughItsDish(t *testing.T) {
	for number := range uint64(20) {
		made := mustMake(t, "maser", number)
		beam := made.lights[0]
		dish := made.parts[len(made.parts)-1]
		across := 2 * inScene(dish.shape.(tube).tip)
		half, _ := beam.made.angles()
		wave := inScene(beam.made.colour.wave)
		wide, want := half.InRadians(), 1.22*wave/across
		if math.Abs(wide-want) > 1e-9 {
			t.Errorf("seed %d: a beam of %v through a dish %v "+
				"across, not %v", number, wide, across, want)
		}
		// 21 cm: hydrogen's line, from memory
		if math.Abs(wave-0.211) > 0.001 {
			t.Errorf("seed %d: a line of %v m", number, wave)
		}
		if beam.made.bands().Power(bands.Of(550)) != 0 {
			t.Errorf("seed %d: the maser can be seen", number)
		}
		if !beam.toward.Near(dish.space.Orientation.Y, close) ||
			beam.toward.Y <= 0 {
			t.Errorf("seed %d: the beam goes %v, not up the dish",
				number, beam.toward)
		}
	}
}

// a thing placed somewhere takes its lights with it: where each is,
// and which way each shines.
func TestAThingPlacedTakesItsLightsWithIt(t *testing.T) {
	made := mustMake(t, "spotlight", 2)
	quarter := maths.About(maths.YAxis, math.Pi/2)
	there := maths.Subspace{Origin: maths.V(10, 0, -5),
		Orientation: quarter, Scale: 2}
	moved := made.placed(there)
	was, is := made.lights[0], moved.lights[0]
	if !is.at.Near(there.Out(was.at), 1e-9) {
		t.Errorf("the light is at %v, not %v", is.at, there.Out(was.at))
	}
	if !is.toward.Near(quarter.Out(was.toward), 1e-9) {
		t.Errorf("the light shines %v, not %v", is.toward,
			quarter.Out(was.toward))
	}
	if is.made != was.made {
		t.Error("moving a thing changed its light")
	}
	// a thing twice the size has a housing twice the size to clear
	far, near := inScene(is.beyond), inScene(was.beyond)
	if math.Abs(far-2*near) > close || near <= 0 {
		t.Errorf("the light starts %v out, and started %v", far, near)
	}
	if len(made.lights) != 1 || made.lights[0] != was {
		t.Error("placing a thing moved the one it was made from")
	}
}

// things gathered into one keep their own matters apart: three
// gathered are three things, each in its own look.
func TestThingsGatheredKeepTheirOwnLooks(t *testing.T) {
	all, err := gather(mustMake(t, "boulder", 1), mustMake(t, "log", 1),
		mustMake(t, "can", 1), mustMake(t, "spotlight", 1))
	if err != nil {
		t.Fatal(err)
	}
	things, err := all.things(maths.Subspace{})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, one := range things {
		seen[one.Material.Name] = true
	}
	for _, name := range []string{"granite", "bark", "aluminium"} {
		if !seen[name] {
			t.Errorf("no thing of %s among %d", name, len(things))
		}
	}
	if len(all.lights) != 1 {
		t.Errorf("%d lights gathered, of one", len(all.lights))
	}
}

// lighting a thing makes a thing with one more light, and leaves the
// thing it was made from as it was, even when two are lit from one.
func TestLightingAThingLeavesTheOneItWasMadeFromAlone(t *testing.T) {
	base := mustMake(t, "spotlight", 2)
	was := base.lights[0]
	red := mustLight(t, mustLine(t, 650e-9), 1, everyway{})
	green := mustLight(t, mustLine(t, 550e-9), 1, everyway{})
	none := measure.Metres(0)
	first := base.lit(red, maths.V(1, 0, 0), maths.V(0, 3, 0), none)
	second := base.lit(green, maths.V(2, 0, 0), maths.Up, none)
	if len(base.lights) != 1 || base.lights[0] != was {
		t.Error("lighting a thing changed the one it was made from")
	}
	if len(first.lights) != 2 || first.lights[0] != was ||
		first.lights[1].made != red {
		t.Error("the first thing lit does not carry its own light")
	}
	if len(second.lights) != 2 || second.lights[1].made != green {
		t.Error("the second thing lit does not carry its own light")
	}
	if got := first.lights[1].toward; got != maths.Up {
		t.Errorf("a way of %v was kept as %v, not a unit", 3, got)
	}
}

// a thing given no look is refused when a scene asks for it, and
// gathering it into another does not hide that: found by review, a
// good with no look gathered passed with a material of nothing.
func TestAThingWithNoLookIsRefusedGatheredToo(t *testing.T) {
	bare, err := newGood(mustTube(t, 1, 0.1, 0.1), mustCut(t, 6))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bare.things(maths.Subspace{}); !errors.Is(err,
		errLook) {
		t.Fatalf("alone: %v, not %v", err, errLook)
	}
	if _, err := gather(mustMake(t, "boulder", 1), bare); !errors.Is(
		err, errLook) {
		t.Errorf("gathered: %v, not %v", err, errLook)
	}
}
