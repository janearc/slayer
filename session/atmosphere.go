package session

import (
	"fmt"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/scene/medium"
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

// atmosphereLayer stages one layer as the manufactory's atmosphere, which
// retains only immutable terrain heights in its callback.
func (stage *staging) atmosphereLayer(read set.Set,
	layer set.AtmosphereLayer,
) error {
	name := layer.Weather()
	mist, _ := read.Mist(name)
	made := stage.atmosphereOf(read, layer, mist)
	if err := made.Staged(stage.world, stage.where); err != nil {
		return fmt.Errorf("%s: atmosphere layer of %q: %w",
			stage.given.set, name, err)
	}
	return nil
}

// atmosphereOf is the manufactory's atmosphere of a layer of the set and
// its mist, over the terrain the stage has laid.
func (stage *staging) atmosphereOf(read set.Set, layer set.AtmosphereLayer,
	mist set.Mist) manufacture.Atmosphere {
	made := manufacture.Atmosphere{Name: layer.Weather(),
		Weather: airWeather(read, mist), Albedo: layer.Albedo(),
		Over: stage.groundExtent, Beyond: layer.Beyond(),
		Floor: terrainFloor{window: stage.ground,
			lowest: stage.lowest}.height,
		Nitrogen: mist.NitrogenDioxide()}
	made.Toward, made.Reach, made.Least = layer.Toward()
	made.Angstrom, made.Angstromed = mist.Angstrom()
	made.CloudBase, made.CloudTop, made.Cloud = mist.Cloud()
	made.Base, made.Level = layer.Base()
	return made
}

// airWeather is the mist's weather for the medium, with the set's wind.
func airWeather(read set.Set, mist set.Mist) medium.GroveWeather {
	from, speed := read.Wind()
	wind := skyward(from, windHorizon).Scale(-speed)
	weather := medium.NewWeather(mist.Extinction(), mist.Depth(),
		mist.Coverage(), mist.Seed(), wind)
	weather.Asymmetry = mist.Asymmetry()
	weather.Edge = mist.Edge()
	weather.PatchScale = mist.Size()
	return weather
}
