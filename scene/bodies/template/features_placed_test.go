package template

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/world"
)

// featureSplits is how finely the worlds the features are asked of are
// split: fine enough that a direction typed a little off lands on
// another tile.
const featureSplits = 12

// plain is a body's ground made into a plain world, the one its
// Features ask where the land is, on a sphere split n times. no body is
// placed: only the terrain is worked out.
func plain(t *testing.T, b bodies.Body, n int) *world.World {
	t.Helper()
	terrain := b.Ground.Terrain
	terrain.Tiling = hexes.Sphere{N: n}
	return world.New(terrain)
}

// drowned is a world with no land: the sphere split n times, every tile
// laid under deep water by a patch as wide as the sphere.
func drowned(t *testing.T, n int) *world.World {
	t.Helper()
	w := world.New(world.Terrain{Tiling: hexes.Sphere{N: n}, Seed: 1, Plates: 4,
		Features: []world.Feature{world.Patch{At: maths.XAxis, Reach: 10, Of: world.Deep}}})
	if _, found := w.Find(maths.XAxis, world.Land); found {
		t.Fatal("the drowned world has land")
	}
	return w
}

// landNear is the centre of the land tile nearest a direction, by the
// world's own Find.
func landNear(t *testing.T, w *world.World, toward maths.Vec) maths.Vec {
	t.Helper()
	tile, ok := w.Find(toward, world.Land)
	if !ok {
		t.Fatalf("no land toward %v", toward)
	}
	return w.Tiling.Centre(tile)
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// on venus, maxwell is a plateau on the land toward (0.2, 0.9, 0.4), 3
// tile spacings wide and 0.9 high, and maat a dead volcano on the land
// toward (0.9, 0.05, 0.4), 3 spacings wide and 0.5 high.
//
// the oracle is the world's Find, asked here with the directions typed
// again, on venus's own terrain at 12 splits.
func TestVenusPutsMaxwellAndMaatWhereItIsTold(t *testing.T) {
	b := Venus()
	w := plain(t, b, featureSplits)
	step := w.Tiling.Spacing()
	got, err := b.Ground.Features(w)
	if err != nil {
		t.Fatal(err)
	}
	holds(t, "venus's features", got, []world.Feature{
		world.Plateau{At: landNear(t, w, maths.V(0.2, 0.9, 0.4)), Reach: 3 * step, Top: 0.9},
		world.Volcano{At: landNear(t, w, maths.V(0.9, 0.05, 0.4)), Reach: 3 * step,
			Rise: 0.5, Dead: true}})
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// on mars, tharsis is a dead volcano on the land toward (0.9, 0.3,
// 0.3), 4 spacings wide and 0.6 high.
//
// the canyon runs from the unit vector of that centre plus (0.3, -0.1,
// -0.5) to the unit vector of that point plus (0.5, 0, 0.2), 1.2
// spacings wide and 0.5 deep.
func TestMarsPutsTharsisAndTheCanyonWhereItIsTold(t *testing.T) {
	b := Mars()
	w := plain(t, b, featureSplits)
	step := w.Tiling.Spacing()
	got, err := b.Ground.Features(w)
	if err != nil {
		t.Fatal(err)
	}
	at := landNear(t, w, maths.V(0.9, 0.3, 0.3))
	from := at.Add(maths.V(0.3, -0.1, -0.5)).Unit()
	to := from.Add(maths.V(0.5, 0, 0.2)).Unit()
	holds(t, "mars's features", got, []world.Feature{
		world.Volcano{At: at, Reach: 4 * step, Rise: 0.6, Dead: true},
		world.Canyon{From: from, To: to, Width: 1.2 * step, Depth: 0.5}})
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// on pluto, the heart is a patch of ice on the land toward (0.2, 0.1,
// 1), 5 spacings wide.
func TestPlutoLaysItsHeartOfIceWhereItIsTold(t *testing.T) {
	b := Pluto()
	w := plain(t, b, featureSplits)
	got, err := b.Ground.Features(w)
	if err != nil {
		t.Fatal(err)
	}
	holds(t, "pluto's features", got, []world.Feature{
		world.Patch{At: landNear(t, w, maths.V(0.2, 0.1, 1)),
			Reach: 5 * w.Tiling.Spacing(), Of: world.Ice}})
}

// the template's figure, held so nothing moves; from memory, not
// checked against a source.
//
// io has a dozen live volcanoes, the k-th on the land toward (cos 2.4k,
// 0.8 sin 1.7k, sin 2.4k) for k from 0 to 11, each 2.5 spacings wide
// and 0.3 high.
//
// worked by hand: for k = 1 that direction is (-0.737, 0.793, 0.675),
// which the last check holds the typed formula to.
func TestIoPutsADozenLiveVolcanoes(t *testing.T) {
	b := Jupiter().Moons[0].Body
	w := plain(t, b, featureSplits)
	step := w.Tiling.Spacing()
	got, err := b.Ground.Features(w)
	if err != nil {
		t.Fatal(err)
	}
	var want []world.Feature
	for k := 0; k < 12; k++ {
		x := float64(k)
		toward := maths.V(math.Cos(x*2.4), 0.8*math.Sin(x*1.7), math.Sin(x*2.4))
		want = append(want, world.Volcano{At: landNear(t, w, toward),
			Reach: 2.5 * step, Rise: 0.3})
	}
	holds(t, "io's features", got, want)
	one := maths.V(math.Cos(2.4), 0.8*math.Sin(1.7), math.Sin(2.4))
	if d := one.Sub(maths.V(-0.737, 0.793, 0.675)).Len(); !(d < 1e-3) {
		t.Errorf("the direction toward the second volcano is %v, worked by hand "+
			"as (-0.737, 0.793, 0.675)", one)
	}
}

// world.Find on a sphere fails only where there is no land at all: from
// any of three hundred directions it finds land in venus's world, which
// has some, and none in the drowned one.
//
// this is why venus's second refusal, for maat, can not be reached when
// the first, for maxwell, did not refuse (see .mutants).
func TestFindOnASphereFailsOnlyWhereThereIsNoLand(t *testing.T) {
	land, sea := plain(t, Venus(), 4), drowned(t, 4)
	for k := 0; k < 300; k++ {
		x := float64(k)
		toward := maths.V(math.Cos(x*2.4), math.Sin(x*1.3), math.Sin(x*2.4)+0.1)
		if _, ok := land.Find(toward, world.Land); !ok {
			t.Errorf("no land found toward %v in venus's world", toward)
		}
		if _, ok := sea.Find(toward, world.Land); ok {
			t.Errorf("land found toward %v in the drowned world", toward)
		}
	}
}

// with no land to put a feature on, venus, mars and pluto refuse, by
// name and with nothing placed, and io, which leaves a volcano out
// where it has no land near, places none and does not refuse.
func TestWithNoLandVenusMarsAndPlutoRefuseAndIoPlacesNone(t *testing.T) {
	w := drowned(t, 4)
	for _, c := range []struct {
		body bodies.Body
		want string
	}{
		{Venus(), "no land for maxwell"},
		{Mars(), "no land for the volcano"},
		{Pluto(), "no land for the heart"},
	} {
		got, err := c.body.Ground.Features(w)
		if err == nil || err.Error() != c.want {
			t.Errorf("%s: error is %v, want %q", c.body.Name, err, c.want)
		}
		if got != nil {
			t.Errorf("%s: placed %v along with the refusal", c.body.Name, got)
		}
	}
	io := Jupiter().Moons[0].Body
	got, err := io.Ground.Features(w)
	if err != nil || len(got) != 0 {
		t.Errorf("io on a drowned world gave %v and %v, want none and no error", got, err)
	}
}
