package trace

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// everyFirst is the nearest thing a ray meets past near, by asking every
// thing in order: what the crowd is held to.
func everyFirst(things []placedThing, ray maths.Ray, near float64) (hit,
	bool) {
	nearest := hit{dist: nowhere}
	for i := range things {
		along, ok := things[i].meet(ray, near, nearest.dist)
		if ok && along < nearest.dist {
			nearest = hit{thing: things[i], dist: along}
		}
	}
	return nearest, nearest.dist < nowhere
}

// everyBlocked is whether anything that casts a shadow stands in a ray's
// way before far, by asking every thing: what the crowd is held to.
func everyBlocked(things []placedThing, ray maths.Ray, far float64) bool {
	far = math.Min(far, nowhere)
	for i := range things {
		placed := &things[i]
		if placed.thing.Clear() {
			continue
		}
		if _, ok := placed.meet(ray, 0, far); ok {
			return true
		}
	}
	return false
}

// between is a number at random from low to high.
func between(random *rand.Rand, low, high float64) float64 {
	return low + (high-low)*random.Float64()
}

// strewn is many balls and tubes at random, turned and scaled, some
// of them clear, with a plane under them that has no edge.
func strewn(seed uint64, many int) []placedThing {
	random := rand.New(rand.NewPCG(seed, 4))
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Motion: resting(maths.Vec{})})
	for range many {
		at := maths.Subspace{Origin: maths.V(between(random, -10, 10),
			between(random, 0, 3), between(random, -10, 10)),
			Orientation: maths.About(maths.V(random.Float64(), 1,
				random.Float64()).Unit(), between(random, 0, math.Pi)),
			Scale: between(random, 0.2, 2)}
		var solid shape.Shape = shape.Sphere{Radius: between(random, 0.1, 1)}
		if random.IntN(3) == 0 {
			solid = shape.Tube{Radius: 0.4, Thick: 0.1, Height: 0.8}
		}
		world.Add(thing.Thing{Shape: solid, Motion: thing.Still(at),
			Attrs: thing.Attrs{Clear: random.IntN(5) == 0}})
	}
	return place(world, 0)
}

// aimed is a ray at random from above and around the things, toward
// somewhere among them.
func aimed(random *rand.Rand) maths.Ray {
	from := maths.V(between(random, -15, 15), between(random, 1, 8),
		between(random, -15, 15))
	toward := maths.V(between(random, -10, 10), between(random, -1, 3),
		between(random, -10, 10))
	return maths.Ray{At: from, Dir: toward.Sub(from).Unit()}
}

// The crowd finds what asking every thing finds: the same thing at the same
// distance bit for bit, and the same shadow, for every ray.
func TestTheCrowdFindsWhatAskingEveryThingFinds(t *testing.T) {
	for _, many := range []int{0, 1, 3, 17, 300} {
		things := strewn(uint64(many), many)
		seen := gathered(things)
		random := rand.New(rand.NewPCG(uint64(many), 5))
		for range 2000 {
			ray, near := aimed(random), between(random, 0, 2)
			far := between(random, 0, 30)
			want, wanted := everyFirst(things, ray, near)
			got, found := seen.first(ray, near)
			if found != wanted || got.thing.thing != want.thing.thing ||
				math.Float64bits(got.dist) != math.Float64bits(want.dist) {
				t.Fatalf("%d things: the crowd met %v at %v; every thing, "+
					"%v at %v", many, found, got.dist, wanted, want.dist)
			}
			if seen.blocked(ray, far) != everyBlocked(things, ray, far) {
				t.Fatalf("%d things: the crowd and every thing disagree "+
					"on a shadow", many)
			}
		}
	}
}

// Two things met at the same distance go to the earlier, as asking
// every thing in order gives them, even where the tree holds the later
// one first: two balls either side of the ray, the later on the left,
// which the tree sorts first, and four more off the ray so it splits.
func TestATieGoesToTheEarlierThing(t *testing.T) {
	world := thing.NewWorld()
	for _, x := range []float64{1, -1, 10, 11, 12, 13} {
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 2},
			Motion: resting(maths.V(x, 0, 0))})
	}
	things := place(world, 0)
	want, _ := everyFirst(things, alongZ, 0)
	got, found := gathered(things).first(alongZ, 0)
	if !found || want.thing.thing != things[0].thing ||
		got.thing.thing != things[0].thing {
		t.Errorf("the tie went to another than the first thing")
	}
}

// starsOverAField is forty clear balls high up, stars drawn as balls,
// over forty that cast shadows near the ground.
func starsOverAField(random *rand.Rand) []placedThing {
	world := thing.NewWorld()
	for i := range 80 {
		star := i%2 == 0
		height := between(random, 0, 3)
		if star {
			height = between(random, 20, 25)
		}
		world.Add(thing.Thing{
			Shape: shape.Sphere{Radius: between(random, 0.1, 0.5)},
			Motion: resting(maths.V(between(random, -10, 10), height,
				between(random, -10, 10))),
			Attrs: thing.Attrs{Clear: star}})
	}
	return place(world, 0)
}

// A branch of only clear things casts no shadow, and a shadow ray
// passes over it without asking its box: the stars together make such
// branches, and the crowd still agrees with every thing on every
// shadow, rays through the stars and the field alike.
func TestAClearBranchIsPassedOver(t *testing.T) {
	random := rand.New(rand.NewPCG(9, 4))
	things := starsOverAField(random)
	seen := gathered(things)
	passed := 0
	for _, node := range seen.nodes {
		if !node.shadows {
			passed++
		}
	}
	if passed == 0 {
		t.Fatal("no branch holds only clear things")
	}
	for range 4000 {
		from := maths.V(between(random, -15, 15), between(random, 0, 30),
			between(random, -15, 15))
		toward := maths.V(between(random, -10, 10), between(random, 0, 25),
			between(random, -10, 10))
		ray := maths.Ray{At: from, Dir: toward.Sub(from).Unit()}
		far := between(random, 0, 40)
		if seen.blocked(ray, far) != everyBlocked(things, ray, far) {
			t.Fatalf("the crowd and every thing disagree on a shadow "+
				"from %v toward %v", from, toward)
		}
	}
}

// depth is how many branches deep the tree goes below a node.
func (seen crowd) depth(node int) int {
	if seen.nodes[node].left == 0 {
		return 1
	}
	return 1 + max(seen.depth(seen.nodes[node].left),
		seen.depth(seen.nodes[node].right))
}

// The tree over a hundred thousand things is shallow enough for a walk's
// stack many times over.
func TestTheTreeIsShallow(t *testing.T) {
	seen := gathered(strewn(6, 100_000))
	if depth := seen.depth(0); 2*depth > deepest {
		t.Errorf("the tree is %d deep; a walk holds %d", depth, deepest)
	}
}
