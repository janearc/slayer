package kernel_test

import (
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/burning-trousers/divergence"
	"github.com/janearc/slayer/burning-trousers/fake"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// scenes is how many scenes at random the kernel is drawn against the
// tracer in.
const scenes = 50

// between is a number at random from low to high.
func between(random *rand.Rand, low, high float64) float64 {
	return low + (high-low)*random.Float64()
}

// colour is a light at random, each part from zero to top.
func colour(random *rand.Rand, top float64) maths.Light {
	return maths.Light{R: between(random, 0, top),
		G: between(random, 0, top), B: between(random, 0, top)}
}

// surface is a material at random, glossy half the time.
func surface(random *rand.Rand) *thing.Material {
	made := &thing.Material{Colour: colour(random, 1)}
	if random.IntN(2) == 0 {
		made.Gloss, made.Index = random.Float64(), between(random, 1.1, 3)
	}
	return made
}

// arranged is the world a seed makes of scene/thing's parts: ground, one to
// six balls, one to three lamps and a sky.
func arranged(t *testing.T, seed uint64) *thing.World {
	t.Helper()
	random := rand.New(rand.NewPCG(seed, 2))
	world := &thing.World{Sky: fake.NewSky(colour(random, 0.5)),
		Things: []*thing.Thing{{Shape: ground(t, 50),
			Material: surface(random)}}}
	for range 1 + random.IntN(6) {
		centre := maths.V(between(random, -2, 2), between(random, 0, 2),
			between(random, -2, 2))
		world.Things = append(world.Things, &thing.Thing{
			Shape:    shape.Sphere{Radius: between(random, 0.1, 1)},
			Material: surface(random),
			Motion:   thing.Still(maths.Subspace{Origin: centre})})
	}
	for range 1 + random.IntN(3) {
		at := maths.V(between(random, -3, 3), between(random, 1, 5),
			between(random, -3, 3))
		world.Lamps = append(world.Lamps,
			emit.Lit{At: at, Strength: colour(random, 20)})
	}
	return world
}

// The kernel draws what the tracer draws: in every scene no more than
// the allowed share of pixels apart by more than a JND.
func TestTheKernelDrawsWhatTheTracerDraws(t *testing.T) {
	seen := looking(maths.V(0, 2, 6), maths.V(0, 0.5, 0), 48, 32)
	apart := 0
	var worst divergence.JNDs
	for seed := range uint64(scenes) {
		world := arranged(t, seed)
		cpu := seen.drawn(t, drawers[0], world)
		gpu := seen.drawn(t, drawers[1], world)
		measured, err := divergence.Compare(cpu, gpu)
		if err != nil {
			t.Fatal(err)
		}
		worst = max(worst, measured.Worst())
		if !measured.Within() {
			apart++
			t.Errorf("scene %d: %.1f%% of pixels past a JND, the worst %.1f",
				seed, 100*float64(measured.Past(1)), measured.Worst())
		}
	}
	t.Logf("%d of %d scenes apart; the worst pixel of all %.2f JND", apart,
		scenes, worst)
}
