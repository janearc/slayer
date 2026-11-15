package main

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// cloudPlan describes authored radial weather, independent of the local site.
type cloudPlan struct {
	Low        float64    `yaml:"low-height-metres" json:"low-height-metres"`
	High       float64    `yaml:"high-height-metres" json:"high-height-metres"`
	PatchScale float64    `yaml:"patch-scale-metres" json:"patch-scale-metres"`
	Extinction float64    `yaml:"extinction-per-metre" json:"extinction-per-metre"`
	Coverage   float64    `yaml:"coverage" json:"coverage"`
	Seed       uint64     `yaml:"seed" json:"seed"`
	Wind       [3]float64 `yaml:"wind-metres-per-second" json:"wind-metres-per-second"`
	Colour     [3]float64 `yaml:"colour-rgb" json:"colour-rgb"`
}

// cloudLayers shares the terrain/ocean's planetary centre and render origin.
// Weather is global: it has no arbitrary boundary around the observatory.
func (plan plan) cloudLayers(world *thing.World, sphere frame.Sphere, origin frame.Relative) error {
	centre, err := origin.Point(sphere.Centre())
	if err != nil {
		return err
	}
	for _, cloud := range plan.Clouds {
		weather := medium.NewWeather(cloud.Extinction, cloud.High-cloud.Low, cloud.Coverage,
			cloud.Seed, maths.V(cloud.Wind[0], cloud.Wind[1], cloud.Wind[2]))
		weather.PatchScale = cloud.PatchScale
		field, err := medium.NewShellWeather(plan.Radius, cloud.Low, cloud.High, weather)
		if err != nil {
			return err
		}
		world.AddVolume(thing.Volume{Medium: field,
			Material: &thing.Material{Name: "authored planetary cloud", Colour: rgb(cloud.Colour)},
			Motion:   thing.Still(maths.NewSubspace(centre, maths.Orientation{}, 1))})
	}
	return nil
}
