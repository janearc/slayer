package titan

import (
	"errors"
	"math"
	"sync"
	"testing"

	"github.com/janearc/celestial/grissom/mesh"
)

// earth is a radius to test on, in metres. It is a round number near the
// Earth's, not the Earth's.
const earth = 6.4e6

// chart is a map for testing, made of arithmetic: a grid of samples so
// many metres apart, each the sum of a few layers of noise whose size
// falls by a fixed share as their grain halves. A share of one half is
// ground with a Hurst exponent of one; a share near one is very rough.
type chart struct {
	finest float64 // metres between samples
	relief float64 // metres of height in the coarsest layer
	share  float64 // each finer layer's size against the one above
	hole   bool    // whether the map has nothing north of 60

	guard sync.Mutex
	asked int
}

func (c *chart) Source() string { return "a test chart, made of arithmetic" }

func (c *chart) Finest(mesh.Point) float64 { return c.finest }

// layers is how many layers of noise a chart has.
const layers = 6

func (c *chart) Height(at mesh.Point) (float64, bool, error) {
	c.guard.Lock()
	c.asked++
	c.guard.Unlock()
	if c.hole && at.Lat() > 60 {
		return 0, false, nil
	}
	// metres east and north on a plane, good enough near the equator
	east := at.Lon() * math.Pi / 180 * earth
	north := at.Lat() * math.Pi / 180 * earth
	var height float64
	size := c.relief
	for layer := layers - 1; layer >= 0; layer-- {
		grain := c.finest * math.Exp2(float64(layer))
		height += size * lattice(east/grain, north/grain, uint64(layer))
		size *= c.share
	}
	return height, true, nil
}

// lattice is smooth noise from -1 to 1 on a unit grid: each corner's
// hashed value, joined by straight lines.
func lattice(x, y float64, seed uint64) float64 {
	fx, fy := math.Floor(x), math.Floor(y)
	tx, ty := x-fx, y-fy
	at := func(dx, dy float64) float64 {
		bits := mix(mix(uint64(int64(fx+dx))+golden) ^
			mix(uint64(int64(fy+dy))+seed*golden))
		return 2*unit(bits) - 1
	}
	low := at(0, 0)*(1-tx) + at(1, 0)*tx
	high := at(0, 1)*(1-tx) + at(1, 1)*tx
	return low*(1-ty) + high*ty
}

func spot(t *testing.T, lon, lat float64) *mesh.Point {
	t.Helper()
	at, err := mesh.NewPoint(lon, lat)
	if err != nil {
		t.Fatal(err)
	}
	return at
}

func ground(t *testing.T, from Provider, seed uint64) *Ground {
	t.Helper()
	made, err := New(from, earth, seed)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// Asked for no more than the map has, the ground is the map's: measured,
// with nothing made, and the map's height at a place the map sampled.
func TestCoarseGroundIsMeasured(t *testing.T) {
	from := &chart{finest: 100, relief: 300, share: 0.6}
	world := ground(t, from, 1)
	for _, detail := range []float64{100e3, 5e3, 400} {
		got, err := world.At(spot(t, 12.3, 4.5), detail)
		if err != nil {
			t.Fatal(err)
		}
		if got.Kind() != Measured || got.Made() != 0 {
			t.Errorf("at %v m of detail the ground is %v, with %v m made",
				detail, got.Kind(), got.Made())
		}
		if got.Edge() > detail {
			t.Errorf("asked for %v m, got edges of %v m", detail, got.Edge())
		}
	}
	// a corner of the mesh is a place the map was asked about directly
	corner, _, _ := mesh.Faces()[3].Corners()
	want, _, _ := from.Height(corner)
	got, err := world.At(&corner, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.Height()-want) > 1e-6 {
		t.Errorf("at a corner the ground is %v, the map %v",
			got.Height(), want)
	}
}

// Asked for more than the map has, the ground says so: generated, some
// metres of it made, and never split finer than an inchworm.
func TestFineGroundIsGeneratedAndSaysSo(t *testing.T) {
	from := &chart{finest: 100, relief: 300, share: 0.6}
	world := ground(t, from, 1)
	at := spot(t, 12.3, 4.5)
	var movedAtAll bool
	for _, detail := range []float64{50, 1, 0.01, 1e-6} {
		got, err := world.At(at, detail)
		if err != nil {
			t.Fatal(err)
		}
		if got.Kind() != Generated {
			t.Errorf("at %v m of detail the ground is %v", detail, got.Kind())
		}
		if got.Made() != 0 {
			movedAtAll = true
		}
		if got.Edge() > math.Max(detail, Inchworm) {
			t.Errorf("asked for %v m, got edges of %v m", detail, got.Edge())
		}
		if got.Edge() < Inchworm/2 {
			t.Errorf("split to %v m, finer than an inchworm", got.Edge())
		}
		if got.Source() != from.Source() {
			t.Errorf("the sample forgot its map")
		}
	}
	if !movedAtAll {
		t.Errorf("rough ground was given no made relief at all")
	}
	finest, _ := world.At(at, 1e-6)
	if finest.Depth() != world.Depth(Inchworm) {
		t.Errorf("the finest depth is %d, an inchworm's is %d",
			finest.Depth(), world.Depth(Inchworm))
	}
}

// The same seed makes the same ground. Another seed makes other ground
// below the map, and the same ground where the map reaches.
func TestSeedPicksOnlyTheMadeGround(t *testing.T) {
	from := &chart{finest: 100, relief: 300, share: 0.6}
	one, again, other := ground(t, from, 7), ground(t, from, 7),
		ground(t, from, 8)
	at := spot(t, -70.2, -33.1)
	for _, detail := range []float64{2000, 0.05} {
		a, _ := one.At(at, detail)
		b, _ := again.At(at, detail)
		c, _ := other.At(at, detail)
		if a.Height() != b.Height() {
			t.Errorf("the same seed made different ground")
		}
		if (detail > 100) != (a.Height() == c.Height()) {
			t.Errorf("at %v m of detail, seeds 7 and 8 give %v and %v",
				detail, a.Height(), c.Height())
		}
	}
}

// The made ground takes after the map: flat stays flat, smooth slopes are
// carried straight on, and rougher maps get rougher ground.
func TestMadeGroundTakesAfterTheMap(t *testing.T) {
	made := func(from Provider) float64 {
		world := ground(t, from, 3)
		var squares float64
		const places = 200
		for i := range places {
			got, err := world.At(spot(t, 20+float64(i)*0.0137,
				10+float64(i)*0.0071), 0.01)
			if err != nil {
				t.Fatal(err)
			}
			squares += got.Made() * got.Made()
		}
		return math.Sqrt(squares / places)
	}
	flat := made(&chart{finest: 100, relief: 0, share: 0.6})
	gentle := made(&chart{finest: 100, relief: 300, share: 0.55})
	craggy := made(&chart{finest: 100, relief: 300, share: 0.9})
	if flat != 0 {
		t.Errorf("flat ground was given %v m of made relief", flat)
	}
	if !(gentle > 0 && craggy > 2*gentle) {
		t.Errorf("made relief: gentle %v m, craggy %v m", gentle, craggy)
	}
	// nothing made is bigger than what the map itself showed
	if craggy > 300 {
		t.Errorf("craggy ground was given %v m of made relief", craggy)
	}
}

// A corner, once it has a height, keeps it however much finer the ground
// is asked for: looking closer adds ground between the corners and moves
// none of them, but for the hint of noise laid over the made ground,
// which is allowed for here and is small.
func TestLookingCloserMovesNothing(t *testing.T) {
	from := &chart{finest: 100, relief: 300, share: 0.7}
	world := ground(t, from, 5)
	at := spot(t, 151.2, -33.9)
	toward := vectorOf(*at)
	cell, _ := mesh.Locate(at, 0)
	first, second, third := cell.Corners()
	var shape [corners]corner
	for i, each := range []mesh.Point{first, second, third} {
		shape[i], _ = world.read(vectorOf(each))
	}
	for depth := 1; depth <= 24; depth++ {
		var err error
		shape, err = world.step(shape, toward, depth)
		if err != nil {
			t.Fatal(err)
		}
		if depth < 14 {
			continue
		}
		for _, each := range shape {
			here := each.at.point()
			got, err := world.At(&here, Inchworm)
			if err != nil {
				t.Fatal(err)
			}
			// the noise at the corner: no more than a hint of each
			// made depth's spread, all its layers pulling one way
			var noise float64
			for made := 1; made <= world.Depth(Inchworm); made++ {
				if world.Edge(made) < each.spacing {
					noise += hint * world.spread(each, made)
				}
			}
			if each.height != 0 && noise > 0.05*300 {
				t.Fatalf("the noise at a corner may be %v m", noise)
			}
			// a corner's place goes through degrees and back on its
			// way in, so it lands a hair off the corner
			if math.Abs(got.Height()-each.height) > noise+1e-4 {
				t.Fatalf("a depth %d corner at %v m is %v m at an "+
					"inchworm", depth, each.height, got.Height())
			}
		}
	}
}

// The ground has no cracks: along a line of places a millimetre apart,
// crossing triangles of every size, no two neighbours differ by much.
func TestNoCracks(t *testing.T) {
	from := &chart{finest: 100, relief: 300, share: 0.6}
	world := ground(t, from, 9)
	const places, step = 4000, 0.001 // metres
	last := math.NaN()
	var worst float64
	for i := range places {
		lon := 30 + float64(i)*step/earth*180/math.Pi
		lat := 20 + float64(i)*step*0.37/earth*180/math.Pi
		got, err := world.At(spot(t, lon, lat), Inchworm)
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 {
			worst = math.Max(worst, math.Abs(got.Height()-last))
		}
		last = got.Height()
	}
	if worst > 0.02 {
		t.Errorf("neighbours a millimetre apart differ by %v m", worst)
	}
	if worst == 0 {
		t.Errorf("four metres of rough ground are perfectly level")
	}
}

// The map is asked about each of its corners once, and never about
// ground finer than it has.
func TestTheMapIsAskedOnce(t *testing.T) {
	from := &chart{finest: 100, relief: 300, share: 0.6}
	world := ground(t, from, 1)
	at := spot(t, 5, 5)
	if _, err := world.At(at, 0.01); err != nil {
		t.Fatal(err)
	}
	first := from.asked
	for range 50 {
		if _, err := world.At(at, 0.01); err != nil {
			t.Fatal(err)
		}
	}
	if from.asked != first {
		t.Errorf("the map was asked %d more times for ground it had "+
			"already given", from.asked-first)
	}
	if kept := world.Kept(); kept == 0 || kept > most {
		t.Errorf("the ground is keeping %d corners", kept)
	}
	// each kept corner cost one reading and two rings of six
	if want := world.Kept() * (1 + 2*around); first != want {
		t.Errorf("the map was asked %d times for %d corners", first,
			world.Kept())
	}
}

func TestRefusals(t *testing.T) {
	if _, err := New(nil, earth, 1); !errors.Is(err, Refused()) {
		t.Errorf("a ground with no map: %v", err)
	}
	from := &chart{finest: 100, relief: 300, share: 0.6, hole: true}
	if _, err := New(from, 0, 1); !errors.Is(err, Refused()) {
		t.Errorf("a ground of no size: %v", err)
	}
	world := ground(t, from, 1)
	if _, err := world.At(nil, 1); !errors.Is(err, Refused()) {
		t.Errorf("the ground at no place: %v", err)
	}
	if _, err := world.At(spot(t, 10, 80), 1); !errors.Is(err, Refused()) {
		t.Errorf("the ground where the map has none: %v", err)
	}
	if _, err := world.At(spot(t, 10, 0), 1); err != nil {
		t.Errorf("the ground where the map has some: %v", err)
	}
}

// The ladder of sizes: an edge halves with each depth, and an inchworm
// is the bottom of it.
func TestDepths(t *testing.T) {
	world := ground(t, &chart{finest: 100}, 1)
	for depth := range 30 {
		ratio := world.Edge(depth) / world.Edge(depth+1)
		if math.Abs(ratio-2) > 1e-12 {
			t.Fatalf("depth %d to the next is a ratio of %v", depth, ratio)
		}
	}
	bottom := world.Depth(Inchworm)
	if world.Edge(bottom) > Inchworm || world.Edge(bottom-1) <= Inchworm {
		t.Errorf("an inchworm is depth %d, with edges of %v m", bottom,
			world.Edge(bottom))
	}
	if world.Depth(0) != bottom || world.Depth(1e9) != 0 {
		t.Errorf("depths past the ends: %d and %d", world.Depth(0),
			world.Depth(1e9))
	}
	t.Logf("on a world of radius %.0f km: a face's edge is %.0f km; "+
		"an inchworm is depth %d; grissom's keys reach depth %d, "+
		"edges of %.3f m", earth/1000, world.Edge(0)/1000, bottom,
		mesh.Deepest, world.Edge(mesh.Deepest))
}

// Made ground is about as steep as the map showed: neither a hundred
// times steeper, as carrying the map's jaggedness down would make it, nor
// smoothed away.
func TestMadeGroundIsAboutAsSteepAsTheMap(t *testing.T) {
	from := &chart{finest: 100, relief: 300, share: 0.6}
	world := ground(t, from, 11)
	steep := func(lag float64) float64 {
		var squares float64
		const places = 400
		for i := range places {
			lon := 40 + float64(i)*0.011
			lat := 5 + float64(i)*0.007
			aside := lag / earth * 180 / math.Pi
			here, err := world.At(spot(t, lon, lat), lag)
			if err != nil {
				t.Fatal(err)
			}
			there, _ := world.At(spot(t, lon+aside, lat), lag)
			rise := (there.Height() - here.Height()) / lag
			squares += rise * rise
		}
		return math.Sqrt(squares / places)
	}
	map100 := steep(100)
	for _, lag := range []float64{10, 1, 0.1, 0.02} {
		if made := steep(lag); made > 2*map100 || made < map100/4 {
			t.Errorf("slopes over %v m are %v; the map's, over 100 m, "+
				"are %v", lag, made, map100)
		}
	}
}
