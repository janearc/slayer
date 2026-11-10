package set

import (
	"fmt"
	"math"
	"slices"

	yaml "gopkg.in/yaml.v3"
)

// Atmosphere is regional prescribed weather and its authored scattering
// albedo. Empty means no regional field; stands can still carry canopy mist.
func (made Set) Atmosphere() (string, float64) {
	return made.atmosphere, made.atmosphereAlbedo
}

// AtmosphereLayer binds prescribed weather to a terrain footprint. An
// optional base is a level height in scene metres above the terrain minimum.
type AtmosphereLayer struct {
	weather string
	albedo  float64
	base    float64
	level   bool
}

// Weather names the checked weather recipe for this layer.
func (layer AtmosphereLayer) Weather() string { return layer.weather }

// Albedo is the authored diffuse scattering share.
func (layer AtmosphereLayer) Albedo() float64 { return layer.albedo }

// Base selects a level deck; without a base, the layer follows terrain.
func (layer AtmosphereLayer) Base() (float64, bool) {
	return layer.base, layer.level
}

// AtmosphericLayers returns explicit layers or the original single field.
// The returned slice belongs to the caller.
func (made Set) AtmosphericLayers() []AtmosphereLayer {
	if made.atmosphere == "" {
		return slices.Clone(made.atmosphereLayers)
	}
	return []AtmosphereLayer{{weather: made.atmosphere,
		albedo: made.atmosphereAlbedo}}
}

// atmosphere checks explicit optical appearance and the supported footprint.
func (reading reader) atmosphere(_, block *yaml.Node) error {
	if says(block, "layers") {
		return reading.keys(block, map[string]step{
			"layers": reading.atmosphereLayers,
		}, nil)
	}
	err := reading.keys(block, map[string]step{
		"weather": reading.text(&reading.set.atmosphere),
		"extent":  reading.only("terrain"),
		"albedo": reading.number(&reading.set.atmosphereAlbedo,
			clearAlbedo, fullAlbedo),
	}, nil)
	if err != nil {
		return err
	}
	return reading.all(block, "atmosphere", "weather", "extent", "albedo")
}

// atmosphereLayers reads named overlapping fields in declaration order.
func (reading reader) atmosphereLayers(_, block *yaml.Node) error {
	if len(block.Content) == noAtmosphereLayers {
		return reading.refuse(block,
			"atmosphere needs at least one layer")
	}
	return reading.each(block, reading.atmosphereLayer)
}

// atmosphereLayer reads one field without changing any earlier layer.
func (reading reader) atmosphereLayer(_, node *yaml.Node) error {
	layer := AtmosphereLayer{}
	err := reading.keys(node, map[string]step{
		"weather": reading.text(&layer.weather),
		"extent":  reading.only("terrain"),
		"albedo": reading.number(&layer.albedo,
			clearAlbedo, fullAlbedo),
		"base": reading.quantity(&layer.base, "m",
			-math.MaxFloat64, math.MaxFloat64),
	}, nil)
	if err != nil {
		return err
	}
	if err := reading.all(node, "atmosphere layer",
		"weather", "extent", "albedo"); err != nil {
		return err
	}
	layer.level = says(node, "base")
	reading.set.atmosphereLayers = append(
		reading.set.atmosphereLayers, layer)
	return nil
}

// noAtmosphereLayers is defined as the empty layer count.
const noAtmosphereLayers = 0

// clearAlbedo and fullAlbedo are the defined endpoints of a unit albedo.
const clearAlbedo, fullAlbedo = 0, 1

// foundAtmosphere checks references after weather and terrain are read.
func (reading reader) foundAtmosphere() error {
	for _, layer := range reading.set.AtmosphericLayers() {
		if _, found := reading.set.weather[layer.weather]; !found {
			return fmt.Errorf("%s: atmosphere names unknown "+
				"weather %q",
				reading.file, layer.weather)
		}
		if reading.set.terrain == nil {
			return fmt.Errorf("%s: a terrain atmosphere needs "+
				"terrain",
				reading.file)
		}
	}
	return nil
}
