package main

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// windHorizon is defined as zero degrees of altitude above the horizon.
const windHorizon = 0

// atmosphere spans the terrain, including clearings between objects.
// The caller chooses its optics; tree count does not set an extinction rate.
func (stage *staging) atmosphere(read set.Set) error {
	for _, layer := range read.AtmosphericLayers() {
		if err := stage.atmosphereLayer(read, layer); err != nil {
			return err
		}
	}
	return nil
}

// atmosphereLayer retains only immutable terrain heights in its callback.
func (stage *staging) atmosphereLayer(read set.Set,
	layer set.AtmosphereLayer,
) error {
	name, albedo := layer.Weather(), layer.Albedo()
	mist, _ := read.Mist(name)
	from, speed := read.Wind()
	wind := skyward(from, windHorizon).Scale(-speed)
	weather := medium.NewWeather(mist.Extinction(), mist.Depth(),
		mist.Coverage(), mist.Seed(), wind)
	bounds := stage.groundExtent
	bounds.Max.Y += mist.Depth()
	floor := terrainFloor{window: stage.ground, lowest: stage.lowest}
	base, level := layer.Base()
	var field *medium.Layer
	var err error
	if level {
		bounds.Min.Y, bounds.Max.Y = base, base+mist.Depth()
		field, err = medium.NewDeck(weather, bounds, floor.height, base)
	} else {
		field, err = medium.NewLayer(weather, bounds, floor.height)
	}
	if err != nil {
		return err
	}
	material := thing.Diffuse(name, maths.Grey(albedo))
	stage.world.AddVolume(thing.NewVolume(field, material))
	return nil
}
