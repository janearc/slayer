package trace

import (
	"math"
	"testing"

	kerr "github.com/janearc/celestial/hole"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// compass is a sky whose light is the way a ray looks: what comes back
// from it says where a ray went.
type compass struct{}

// Light is the direction looked toward, as light.
func (compass) Light(toward maths.Vec) maths.Light {
	return maths.Light{R: toward.X, G: toward.Y, B: toward.Z}
}

// holeWorld is an empty world with a hole of unit mass at its middle,
// reaching forty masses, under a compass sky.
func holeWorld(motion thing.Motion) *thing.World {
	world := thing.NewWorld()
	world.Sky = compass{}
	world.AddField(thing.Field{Mass: 1, Reach: 40, Motion: motion})
	return world
}

// A ray enters the nearest field it will cross, and how far along: at zero when
// it starts inside, never when it misses or has passed, and a field on a motion
// of scale two reaches twice as far.
func TestEntering(t *testing.T) {
	fields := placeFields(holeWorld(nil), 0)
	at := func(from, dir maths.Vec) (float64, bool) {
		ray := maths.Ray{At: from, Dir: dir.Unit()}
		into, ok := entering(fields, ray)
		return into.dist, ok
	}
	ahead := maths.V(0, 0, 1)
	enters, ok := at(maths.V(0, 0, -100), ahead)
	if !ok || !within(enters, 60, 1e-9) {
		t.Errorf("from a hundred out it enters at %v, %v", enters, ok)
	}
	if enters, ok := at(maths.V(0, 0, -10), ahead); !ok || enters != 0 {
		t.Errorf("from inside it enters at %v, %v", enters, ok)
	}
	if _, ok := at(maths.V(0, 50, -100), ahead); ok {
		t.Error("a ray passing fifty out entered")
	}
	if _, ok := at(maths.V(0, 0, 100), ahead); ok {
		t.Error("a ray already past the field entered")
	}
	doubled := thing.Still(maths.Subspace{Scale: 2})
	twice := placeFields(holeWorld(doubled), 0)
	far := maths.Ray{At: maths.V(0, 0, -200), Dir: ahead}
	into, ok := entering(twice, far)
	enters = into.dist
	if !ok || !within(enters, 120, 1e-9) {
		t.Errorf("at scale two it enters at %v, %v, not 120", enters,
			ok)
	}
}

// A ray through the edge of the reach comes out within a degree of
// straight, bent toward the hole. That is the tracer cutting the
// bending off at the reach, and not the truth: such a ray really turns
// about six degrees. One aimed at the hole is black and covers its
// pixel.
func TestARayLeavesOrFalls(t *testing.T) {
	world := holeWorld(nil)
	edge := lone(world, maths.Ray{At: maths.V(0, 39.9, -100),
		Dir: maths.V(0, 0, 1)}, 0)
	out := maths.V(float64(edge.R), float64(edge.G), float64(edge.B))
	if !(out.Len() >= 0.5) {
		t.Fatalf("the ray past the edge came back %v", edge)
	}
	turned := math.Acos(math.Min(1, out.Unit().Z)) / maths.Degree
	if !(turned <= 1 && out.Y < 0) {
		t.Errorf("past the edge the ray turned %.3f degrees, toward %v",
			turned, out.Unit())
	}
	inward := maths.Ray{At: maths.V(0, 0, -100), Dir: maths.V(0, 0, 1)}
	in := lone(world, inward, 0)
	if in.Light() != (maths.Light{}) || in.Cover != 1 {
		t.Errorf("the ray into the hole came back %v", in)
	}
}

// The march's first step starts on the reach, and does not leave by it
// even when it starts a hair outside, as rounding may put it; a later
// step that far out does.
func TestTheFirstStepNeverLeaves(t *testing.T) {
	world := holeWorld(nil)
	scene := stage(world, 0)
	field := scene.fields[0]
	ray := maths.Ray{At: maths.V(0, 0, -100), Dir: maths.V(0, 0, 1)}
	fresh := func() *marcher {
		return newMarcher(scene, look{ray: ray},
			entry{field: field, dist: 60 - 1e-6})
	}
	marching := fresh()
	radius := field.hole.Radius(marching.ray.Space())
	if !(radius > field.source.Reach) {
		t.Fatalf("the march starts at %v, not outside the reach",
			radius)
	}
	if _, done := marching.march(0); done {
		t.Error("the first step left the field")
	}
	if _, done := fresh().march(1); !done {
		t.Error("a later step outside the reach stayed in")
	}
}

// A field's gain on a volume's light depends on the ray's momentum
// where the light is gathered, so the momentum is handed to gain and
// two momenta give two gains. Before it was an argument it was a field
// of the shifter that a caller had to remember to set.
func TestAGainDependsOnTheRaysMomentum(t *testing.T) {
	world := holeWorld(nil)
	world.AddVolume(thing.Volume{Medium: plate(0.5, -1, 1),
		Material: glows(maths.Grey(1))})
	scene := stage(world, 0)
	inward := maths.Ray{At: maths.V(0, 5, -100), Dir: maths.V(0, 0, 1)}
	into, enters := entering(scene.fields, inward)
	if !enters {
		t.Fatal("the ray misses the field")
	}
	walk := newMarcher(scene, look{ray: inward}, into)
	slab := crossings(scene.volumes, inward, math.Inf(1))[0]
	at := maths.V(0, 5, 0)
	arriving := walk.ray.Momentum()
	other := kerr.NewVec4(arriving.Time()*1.5, arriving.AlongX(),
		arriving.AlongY(), arriving.AlongZ())
	earlier, second := walk.shift.gain(slab, at, arriving),
		walk.shift.gain(slab, at, other)
	if !(earlier > 0) || within(second, earlier, 1e-6*earlier) {
		t.Errorf("two momenta give gains of %v and %v", earlier, second)
	}
	if got := walk.gainAt(other)(slab, at); got != second {
		t.Errorf("gainAt gives %v, gain itself %v", got, second)
	}
}

// A field is placed only if it has a mass and a reach greater than zero. One
// with either at zero or below is no field, and neither is one whose mass or
// reach is not a number, which no comparison with zero catches unless it is
// written the positive way.
func TestOnlyAFieldWithAMassAndAReachIsPlaced(t *testing.T) {
	world := thing.NewWorld()
	for _, each := range []thing.Field{
		{Mass: 1, Reach: 40},
		{Mass: 0, Reach: 40}, {Mass: -1, Reach: 40},
		{Mass: 1, Reach: 0}, {Mass: 1, Reach: -40},
		{Mass: math.NaN(), Reach: 40}, {Mass: 1, Reach: math.NaN()},
	} {
		world.AddField(each)
	}
	placed := placeFields(world, 0)
	if len(placed) != 1 || placed[0].source.Mass != 1 {
		t.Errorf("%d of seven fields are placed, and should be the "+
			"first alone", len(placed))
	}
}

// A ray whose light is all taken inside a field stops there, and the
// pixel is what was gathered on the way, wholly covered: the same
// light a thick lit smoke gives with no field round it.
func TestARaySpentInsideAFieldStopsThere(t *testing.T) {
	build := func(field bool) *thing.World {
		world := thing.NewWorld()
		world.AddVolume(thing.Volume{Medium: plate(40, -1, 1),
			Material: &thing.Material{Colour: maths.Grey(0.8)}})
		world.Lamp(fill)
		if field {
			world.AddField(thing.Field{Mass: 1e-6, Reach: 40})
		}
		return world
	}
	from := maths.Ray{At: maths.V(3, 0, -20), Dir: maths.V(0, 0, 1)}
	plain, spent := lone(build(false), from, 0), lone(build(true), from, 0)
	if !(plain.R > 0.05) || !within(float64(spent.R), float64(plain.R),
		1e-3) || spent.Cover != 1 {
		t.Errorf("spent in a field the pixel is %+v, and with none %+v",
			spent, plain)
	}
	if !(spent.Depth < 22) {
		t.Errorf("the spent ray went on to a depth of %v", spent.Depth)
	}
}

// A step that goes nowhere, as a ray with no momentum takes, gathers
// nothing, adds no depth and does not end the march.
func TestAStepThatGoesNowhereIsPassedOver(t *testing.T) {
	world := holeWorld(nil)
	world.AddVolume(thing.Volume{Medium: plate(0.5, -50, 50),
		Material: glows(maths.Grey(1))})
	scene := stage(world, 0)
	inward := maths.Ray{At: maths.V(0, 20, -100), Dir: maths.V(0, 0, 1)}
	into, enters := entering(scene.fields, inward)
	if !enters {
		t.Fatal("the ray misses the field")
	}
	walker := newMarcher(scene, look{ray: inward}, into)
	for i := 0; i < 3; i++ {
		if _, done := walker.march(i); done {
			t.Fatal("the march ended in its first steps")
		}
	}
	walker.ray = kerr.NewRay(walker.ray.Position(), kerr.NewVec4(0, 0, 0, 0))
	before := *walker
	drawn, done := walker.march(1)
	if done || drawn != (frame.Pixel{}) {
		t.Errorf("a step nowhere ended the march: %+v", drawn)
	}
	if walker.depth != before.depth || walker.keep != before.keep ||
		walker.inside != before.inside || walker.from != before.from {
		t.Errorf("a step nowhere changed the march: depth %v from %v, "+
			"keep %v, inside %v", walker.depth, before.depth,
			walker.keep, walker.inside)
	}
}

// turned is how many degrees a ray from far off, that would pass a
// hole of mass one and reach forty at a distance if it went straight,
// has turned by the time it shows the sky.
func turned(passing float64) float64 {
	from := maths.Ray{At: maths.V(passing, 0, -500),
		Dir: maths.V(0, 0, 1)}
	light := lone(holeWorld(nil), from, 0).Light()
	seen := maths.V(light.R, light.G, light.B).Unit()
	return math.Acos(seen.Z) / maths.Degree
}

// The tracer's own bending, as it is, so that a change to it is seen
// and the comments on loose and leave cannot go stale unnoticed. These
// are not the true angles: the hole's own march from far to far turns a
// ray more, and how near the tracer should come is a ruling to make.
//
// The numbers are what this code gave on 2026-10-15, after libholemath
// began stepping a ray by its own time, so that a photon's energy has
// no say in its path (libholemath 94300c5), which moved every angle.
// They hold to the hundredth on celestial's hole, 2026-11-01.
// That a ray passing at thirty turns more than one at twenty is the way
// out of a field entering it again, each time turning the ray a little.
func TestTheTracersOwnBendingIsAsMeasured(t *testing.T) {
	for _, each := range []struct{ passing, degrees float64 }{
		{10, 27.80}, {20, 9.73}, {30, 10.43},
	} {
		got := turned(each.passing)
		if !within(got, each.degrees, 0.01) {
			t.Errorf("a ray passing at %v turns %.2f degrees, and "+
				"turned %.2f when measured", each.passing, got,
				each.degrees)
		}
	}
}
