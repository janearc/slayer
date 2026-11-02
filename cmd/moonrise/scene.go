package main

import (
	"math"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// the scene's numbers: measured ones name their source, picked ones
// say they are.
const (
	// the moon's mean radius and distance, in metres, and its
	// geometric albedo: nasa's moon fact sheet. its surface is drawn
	// as matte, which the moon is not; its brightening at full is not
	// here.
	moonRadius, moonDistance, moonAlbedo = 1737.4e3, 384.4e6, 0.12
	// earthRadius is the earth's mean radius, the iugg's 6371 km
	earthRadius = 6371e3
	// eyeHeight is a person's eyes over the ground, picked
	eyeHeight = 1.7
	// sunlight is the light the sun lands, hillside's day, picked
	sunlight = 1.5
)

// soil is the earth's colour, picked by looking, as hillside's is.
var soil = maths.RGB(0.2, 0.17, 0.1)

// scene is the world, the eye, and where the moon's middle is.
type scene struct {
	world *thing.World
	eye   maths.Subspace
	moon  maths.Vec
}

// compose is the scene as asked: the earth, a tree on it at the
// origin, the eye back from it, and the sun and moon either side.
func compose(given asked) (scene, error) {
	world := thing.NewWorld()
	earth, err := globe.New(earthRadius)
	if err != nil {
		return scene{}, err
	}
	world.Add(thing.Thing{Shape: earth,
		Material: &thing.Material{Name: "earth", Colour: soil},
		Motion: thing.Still(maths.Subspace{
			Origin: maths.V(0, -earthRadius, 0)})})
	if err := planted(world, given); err != nil {
		return scene{}, err
	}
	eye := maths.V(0, eyeHeight, given.away)
	moon, err := risen(world, eye, given.moon)
	if err != nil {
		return scene{}, err
	}
	if err := lit(world, given.sun); err != nil {
		return scene{}, err
	}
	return scene{world: world, eye: maths.Subspace{Origin: eye,
		Orientation: maths.LookAt(eye, moon)}, moon: moon}, nil
}

// planted stands one tree of the kind and seed asked at the origin.
func planted(world *thing.World, given asked) error {
	row, err := manufacture.KindNamed(given.kind)
	if err != nil {
		return err
	}
	tree, err := manufacture.Make(row, manufacture.Sow(given.seed))
	if err != nil {
		return err
	}
	things, err := tree.Things(maths.Subspace{})
	if err != nil {
		return err
	}
	for _, each := range things {
		world.Add(each)
	}
	return nil
}

// risen puts the moon straight ahead of the eye at an altitude, at its
// own distance, and is where its middle is.
func risen(world *thing.World, eye maths.Vec, degrees float64) (maths.Vec,
	error) {
	up := degrees * math.Pi / 180
	centre := eye.Add(maths.V(0, math.Sin(up), -math.Cos(up)).
		Scale(moonDistance))
	moon, err := globe.New(moonRadius)
	if err != nil {
		return maths.Vec{}, err
	}
	world.Add(thing.Thing{Shape: moon,
		Material: &thing.Material{Name: "moon",
			Colour: maths.Grey(moonAlbedo)},
		Motion: thing.Still(maths.Subspace{Origin: centre})})
	return centre, nil
}

// lit puts the sun behind the eye at an altitude, at an endless
// distance.
func lit(world *thing.World, degrees float64) error {
	up := degrees * math.Pi / 180
	sun, err := emit.NewDistant(maths.V(0, math.Sin(up), math.Cos(up)),
		maths.Grey(sunlight))
	if err != nil {
		return err
	}
	world.Lamp(sun)
	return nil
}
