// Package scenes is scenes made at random from the harness's own parts,
// kept as numbers so each can be built at any scale and drawn any way:
// level ground, balls, lamps, a sky, perhaps a slab of gas, and an eye
// with the points it looks at. The same seed is the same scene.
package scenes

import (
	"math/rand/v2"

	"github.com/janearc/slayer/burning-trousers/fake"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Scene is an arrangement of the harness's parts.
type Scene struct {
	ground thing.Material
	balls  []ball
	lamps  []lamp
	sky    maths.Light
	slab   *slab
	eye    maths.Vec
	looks  []maths.Vec
}

// ball is a ball of one material.
type ball struct {
	centre   maths.Vec
	radius   float64
	material thing.Material
}

// lamp is a point lamp's place and its light at one unit away.
type lamp struct {
	at       maths.Vec
	strength maths.Light
}

// slab is a box of gas of one density.
type slab struct {
	low, high maths.Vec
	density   float64
}

// Arranged is the scene a seed makes: up to six balls, sixteen looks. A
// lit scene has lamps, and balls that may glow; an unlit one is lit by
// its sky alone, so nothing in it can be brighter than the sky.
func Arranged(seed uint64, lit bool) Scene {
	random := rand.New(rand.NewPCG(seed, 1))
	ground, sky := material(random, false), colour(random, 1)
	eye := maths.V(0, between(random, 1, 3), 6)
	made := Scene{ground: ground, sky: sky, eye: eye,
		balls: balls(random, lit), looks: looks(random, 16)}
	if lit {
		made.lamps = lamps(random)
	}
	if random.IntN(2) == 0 {
		low, high := maths.V(-2, 0, -1), maths.V(2, 2, 1)
		density := random.Float64()
		made.slab = &slab{low: low, high: high, density: density}
	}
	return made
}

// Crowd is a lit scene of many balls a seed makes, scattered over a
// field forty across, and looks in a grid of side by side across it.
func Crowd(seed uint64, many, side int) Scene {
	random := rand.New(rand.NewPCG(seed, 3))
	made := Scene{ground: material(random, false), sky: colour(random, 1),
		eye: maths.V(0, 12, 30), lamps: lamps(random)}
	for range many {
		x, y := between(random, -20, 20), between(random, 0, 4)
		centre := maths.V(x, y, between(random, -20, 20))
		radius := between(random, 0.1, 0.6)
		made.balls = append(made.balls, ball{centre: centre,
			radius: radius, material: material(random, true)})
	}
	for i := range side * side {
		across := float64(i%side) / float64(side)
		down := float64(i/side) / float64(side)
		at := maths.V(40*across-20, 0, 40*down-20)
		made.looks = append(made.looks, at)
	}
	return made
}

// Sky is the scene's sky, the same light every way.
func (made Scene) Sky() maths.Light { return made.sky }

// between is a number at random from low to high.
func between(random *rand.Rand, low, high float64) float64 {
	return low + (high-low)*random.Float64()
}

// colour is a light at random, each part from zero to top.
func colour(random *rand.Rand, top float64) maths.Light {
	return maths.Light{R: between(random, 0, top),
		G: between(random, 0, top), B: between(random, 0, top)}
}

// material is a surface at random that gives back no more than arrives,
// glossy half the time; when it may glow, it glows a third of the time.
func material(random *rand.Rand, glows bool) thing.Material {
	made := thing.Material{Colour: colour(random, 1)}
	if random.IntN(2) == 0 {
		made.Gloss = random.Float64()
		made.Index = between(random, 1.1, 3)
	}
	if glows && random.IntN(3) == 0 {
		made.Glow = colour(random, 2)
	}
	return made
}

// balls is one to six balls at random over the ground.
func balls(random *rand.Rand, glow bool) []ball {
	var made []ball
	for range 1 + random.IntN(6) {
		x, y := between(random, -2, 2), between(random, 0, 2)
		centre := maths.V(x, y, between(random, -2, 2))
		radius := between(random, 0.1, 1)
		made = append(made, ball{centre: centre, radius: radius,
			material: material(random, glow)})
	}
	return made
}

// lamps is one to three lamps at random above the ground.
func lamps(random *rand.Rand) []lamp {
	var made []lamp
	for range 1 + random.IntN(3) {
		at := maths.V(between(random, -3, 3), between(random, 1, 5),
			between(random, -3, 3))
		made = append(made, lamp{at: at, strength: colour(random, 20)})
	}
	return made
}

// looks is some points at random around the scene for the eye to look
// at, some of them past everything, at the sky.
func looks(random *rand.Rand, some int) []maths.Vec {
	var made []maths.Vec
	for range some {
		made = append(made, maths.V(between(random, -4, 4),
			between(random, -0.5, 3), between(random, -3, 3)))
	}
	return made
}

// World is the scene built at a scale: every length times by, each
// lamp's strength times by squared so the light it lands is the same,
// and the gas's density over by so the light it keeps is the same.
func (made Scene) World(by float64) *thing.World {
	ground := made.ground
	world := &thing.World{Sky: fake.NewSky(made.sky),
		Things: []*thing.Thing{{Shape: fake.NewPlane(50*by, nil),
			Material: &ground}}}
	for _, placed := range made.balls {
		centre, radius := placed.centre.Scale(by), placed.radius*by
		shape := fake.NewSphere(centre, radius, nil)
		material := placed.material
		world.Things = append(world.Things,
			&thing.Thing{Shape: shape, Material: &material})
	}
	for _, placed := range made.lamps {
		strength := placed.strength.Scale(by * by)
		lamp := fake.NewLamp(placed.at.Scale(by), strength, nil)
		world.Lamps = append(world.Lamps, lamp)
	}
	if made.slab != nil {
		gas := fake.NewSlab(made.slab.low.Scale(by),
			made.slab.high.Scale(by), made.slab.density/by)
		world.Volumes = []*thing.Volume{{Medium: gas}}
	}
	return world
}

// Rays is the rays from the eye to each look, at a scale.
func (made Scene) Rays(by float64) []maths.Ray {
	eye := made.eye.Scale(by)
	var rays []maths.Ray
	for _, at := range made.looks {
		toward := at.Scale(by).Sub(eye).Unit()
		rays = append(rays, maths.Ray{At: eye, Dir: toward})
	}
	return rays
}
