package stage_test

import (
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

// what the tests count in, each true by definition
const (
	nothing = 0 // defined: no amount at all
	twice   = 2 // defined: two times
)

// unshifted is no shift at all.
var unshifted maths.Vec

// shiftOf is how the tests move the lot: by the middle of its first box,
// a place of whole quarters, so that moving a place of the lot by it is
// exact.
func shiftOf(t *testing.T, said set.World) maths.Vec {
	t.Helper()
	for _, each := range said.Boxes() {
		return each.Middle()
	}
	t.Fatal("the lot has no box")
	return unshifted
}

// timesOf are the times a moving thing in the lot is looked at: at the
// start of its run, halfway, at its end and a lap past it.
func timesOf(t *testing.T, said set.World) []time.Duration {
	t.Helper()
	for _, each := range said.Motions() {
		lap, _, _ := each.Run()
		return []time.Duration{nothing * time.Second, lap / twice, lap,
			lap * twice}
	}
	t.Fatal("the lot has no motion")
	return nil
}

// lotWorld is the lot's world as its set says it.
func lotWorld(t *testing.T) set.World {
	t.Helper()
	source, err := os.ReadFile("testdata/lot.yaml")
	if err != nil {
		t.Fatal(err)
	}
	made, err := set.Read("lot.yaml", source)
	if err != nil {
		t.Fatal(err)
	}
	said, _ := made.World()
	return said
}

// yardWorld is the yard's world as its set says it.
func yardWorld(t *testing.T) set.World {
	t.Helper()
	said, _ := read(t).World()
	return said
}

// addedTo is a world holding what its caller put in it, with a set's
// world added, shifted as given.
func addedTo(t *testing.T, world *thing.World, said set.World,
	shift maths.Vec) *thing.World {
	t.Helper()
	if err := stage.Add(world, said, shift); err != nil {
		t.Fatal(err)
	}
	return world
}

// built is the world a set's world builds, by World.
func builtFrom(t *testing.T, said set.World) *thing.World {
	t.Helper()
	world, err := stage.World(said)
	if err != nil {
		t.Fatal(err)
	}
	return world
}

// sameWorld says where two worlds differ, piece by piece, as exact as
// DeepEqual is: a float is the same float, and a NaN is never the same.
func sameWorld(t *testing.T, got, want *thing.World) {
	t.Helper()
	if len(got.Things) != len(want.Things) ||
		len(got.Volumes) != len(want.Volumes) ||
		len(got.Lamps) != len(want.Lamps) ||
		len(got.Fields) != len(want.Fields) {
		t.Fatalf("%d things, %d volumes, %d lamps, %d fields; wanted %d, "+
			"%d, %d, %d", len(got.Things), len(got.Volumes),
			len(got.Lamps), len(got.Fields), len(want.Things),
			len(want.Volumes), len(want.Lamps), len(want.Fields))
	}
	for i := range want.Things {
		if !reflect.DeepEqual(got.Things[i], want.Things[i]) {
			t.Errorf("thing %d is %+v, not %+v", i, got.Things[i],
				want.Things[i])
		}
	}
	for i := range want.Volumes {
		if !reflect.DeepEqual(got.Volumes[i], want.Volumes[i]) {
			t.Errorf("volume %d is %+v, not %+v", i, got.Volumes[i],
				want.Volumes[i])
		}
	}
	for i := range want.Lamps {
		if !reflect.DeepEqual(got.Lamps[i], want.Lamps[i]) {
			t.Errorf("lamp %d is %+v, not %+v", i, got.Lamps[i],
				want.Lamps[i])
		}
	}
	if !reflect.DeepEqual(got.Sky, want.Sky) {
		t.Errorf("the sky is %+v, not %+v", got.Sky, want.Sky)
	}
}

// a world added to a fresh world with no shift is the world World
// builds, piece by piece, for a set with every kind of piece and for one
// with a body.
func TestAddingToAFreshWorldIsWorld(t *testing.T) {
	for _, said := range []set.World{lotWorld(t), yardWorld(t)} {
		fresh := thing.NewWorld()
		added := addedTo(t, fresh, said, unshifted)
		sameWorld(t, added, builtFrom(t, said))
	}
}

// a world added to one that already holds things keeps what was there,
// in its order, and puts the set's pieces after it; its sky stays.
func TestAddingKeepsWhatWasThere(t *testing.T) {
	held, lot := builtFrom(t, yardWorld(t)), builtFrom(t, lotWorld(t))
	before := builtFrom(t, yardWorld(t))
	got := addedTo(t, held, lotWorld(t), unshifted)
	things, lamps := len(before.Things), len(before.Lamps)
	if len(got.Things) != things+len(lot.Things) ||
		len(got.Lamps) != lamps+len(lot.Lamps) {
		t.Fatalf("%d things and %d lamps; wanted %d and %d", len(got.Things),
			len(got.Lamps), things+len(lot.Things), lamps+len(lot.Lamps))
	}
	if !reflect.DeepEqual(got.Things[:things], before.Things) ||
		!reflect.DeepEqual(got.Things[things:], lot.Things) ||
		!reflect.DeepEqual(got.Lamps[:lamps], before.Lamps) ||
		!reflect.DeepEqual(got.Lamps[lamps:], lot.Lamps) {
		t.Error("what was there moved, or what was added is not the lot's")
	}
	if !reflect.DeepEqual(got.Sky, before.Sky) || got.Sky == nil {
		t.Errorf("the sky the world had was replaced by %+v", got.Sky)
	}
}

// a shifted add moves every thing, volume and lamp that has a place by
// the shift and moves nothing else: its shape, its material, its
// orientation, its light, a flood, a sun and the sky are as unshifted.
func TestAShiftMovesEveryPlaceByExactlyTheShift(t *testing.T) {
	said := lotWorld(t)
	shift, times := shiftOf(t, said), timesOf(t, said)
	plain := builtFrom(t, said)
	fresh := thing.NewWorld()
	got := addedTo(t, fresh, said, shift)
	if len(got.Things) != len(plain.Things) ||
		len(got.Volumes) != len(plain.Volumes) ||
		len(got.Lamps) != len(plain.Lamps) {
		t.Fatalf("%d things, %d volumes, %d lamps", len(got.Things),
			len(got.Volumes), len(got.Lamps))
	}
	for i, each := range plain.Things {
		if !reflect.DeepEqual(got.Things[i].Shape, each.Shape) ||
			!reflect.DeepEqual(got.Things[i].Material, each.Material) ||
			got.Things[i].Attrs != each.Attrs {
			t.Errorf("thing %d changed beyond its place: %+v", i,
				got.Things[i])
		}
		sameMotion(t, "thing", i, got.Things[i].Motion, each.Motion, shift,
			times)
	}
	for i, each := range plain.Volumes {
		if !reflect.DeepEqual(got.Volumes[i].Medium, each.Medium) ||
			!reflect.DeepEqual(got.Volumes[i].Material, each.Material) {
			t.Errorf("volume %d changed beyond its place: %+v", i,
				got.Volumes[i])
		}
		sameMotion(t, "volume", i, got.Volumes[i].Motion, each.Motion, shift,
			times)
	}
	for i, each := range plain.Lamps {
		if want := lampMoved(each, shift); !reflect.DeepEqual(got.Lamps[i],
			want) {
			t.Errorf("lamp %d is %+v, not %+v", i, got.Lamps[i], want)
		}
	}
	if !reflect.DeepEqual(got.Sky, plain.Sky) {
		t.Errorf("the sky moved: %+v", got.Sky)
	}
}

// lampMoved is a lamp as it should stand once shifted: a bulb and a spot
// by their place and not their aim, and a lamp with no place not at all.
func lampMoved(lamp thing.Lamp, shift maths.Vec) thing.Lamp {
	switch each := lamp.(type) {
	case emit.Lit:
		each.At = each.At.Add(shift)
		return each
	case film.Spot:
		each.At = each.At.Add(shift)
		return each
	}
	return lamp
}

// sameMotion says where a shifted motion is not the unshifted one moved:
// its place at each of the times is the shift off, and its way round and
// its size are the same.
func sameMotion(t *testing.T, of string, i int, got, was thing.Motion,
	shift maths.Vec, times []time.Duration) {
	t.Helper()
	for _, at := range times {
		have, want := got.At(at), was.At(at)
		if have.Origin != want.Origin.Add(shift) ||
			have.Orientation != want.Orientation ||
			have.Scale != want.Scale {
			t.Errorf("%s %d at %v is %+v, not %+v moved", of, i, at, have,
				want)
		}
	}
}

// lawn is the thing in a world that wears the grass: the yard's lawn.
func lawn(t *testing.T, world *thing.World) *thing.Thing {
	t.Helper()
	for _, each := range world.Things {
		if each.Material != nil && each.Material.Name == "grass" {
			return each
		}
	}
	t.Fatal("the world has no lawn")
	return nil
}

// no shift leaves a place as the set wrote it, to the bit: a box a set
// puts at a minus nought stands at a minus nought, which adding a plain
// nought would not leave it.
func TestNoShiftLeavesAMinusNoughtAlone(t *testing.T) {
	said, _ := read(t, "middle: [1, 2, 3]", "middle: [-0.0, 2, 3]").World()
	fresh := thing.NewWorld()
	east := lawn(t, addedTo(t, fresh, said, unshifted)).At(nothing).Origin.X
	if east != nothing || !math.Signbit(east) {
		t.Errorf("the lawn stands at %v east, not at a minus nought", east)
	}
}
