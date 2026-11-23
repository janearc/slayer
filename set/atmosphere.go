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
	world   bool // reaches the world, in schema version 2
	// beyond is how far past the terrain's edge the layer reaches, each
	// way across; the ground's edge height holds out there.
	beyond float64
	// toward is the thing the layer is thickest over, how far from it it
	// thins to its least, and that least as a share; "" for none.
	toward       string
	reach, least float64
}

// Beyond is how far past the terrain's edge the layer reaches, in metres,
// each way across: nought keeps it to the terrain.
func (layer AtmosphereLayer) Beyond() float64 { return layer.beyond }

// Toward is the named thing the layer is thickest over, the distance in
// metres from it at which it has thinned to its least, and the least as a
// share of its density; a name of "" is a layer as thick everywhere.
func (layer AtmosphereLayer) Toward() (string, float64, float64) {
	return layer.toward, layer.reach, layer.least
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
		"extent":  reading.extentStep(&layer.world),
		"albedo": reading.number(&layer.albedo,
			clearAlbedo, fullAlbedo),
		"base": reading.quantity(&layer.base, "m",
			-math.MaxFloat64, math.MaxFloat64),
		"beyond": reading.quantity(&layer.beyond, "m", 0, mostBeyond),
		"toward": reading.block(map[string]step{
			"object": reading.text(&layer.toward),
			"reach":  reading.quantity(&layer.reach, "m", 1, mostBeyond),
			"least":  reading.number(&layer.least, 0, 1)}),
	}, nil)
	if err != nil {
		return err
	}
	if err := reading.all(node, "atmosphere layer",
		"weather", "extent", "albedo"); err != nil {
		return err
	}
	layer.level = says(node, "base")
	if err := reading.towardSaid(node, layer); err != nil {
		return err
	}
	reading.set.atmosphereLayers = append(
		reading.set.atmosphereLayers, layer)
	return nil
}

// mostBeyond is the farthest a layer reaches past the terrain or thins
// toward a thing, in metres: a hundred kilometres. Picked, as a bound.
const mostBeyond = 1e5

// towardSaid checks a layer that thins away from a thing says the thing,
// its reach and its least, and that a layer over no terrain says neither
// that nor how far beyond the terrain it goes.
func (reading reader) towardSaid(node *yaml.Node,
	layer AtmosphereLayer) error {
	if layer.world && (says(node, "toward") || says(node, "beyond")) {
		return reading.refuse(node, "a world layer has no terrain to "+
			"reach beyond or thin across")
	}
	if !says(node, "toward") {
		return nil
	}
	return reading.all(valueNode(node, "toward"), "a layer's toward",
		"object", "reach", "least")
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
		if layer.world && reading.set.globe == nil &&
			reading.set.terrain != nil {
			return fmt.Errorf("%s: a world atmosphere needs a globe, "+
				"or no terrain, which makes it flat air", reading.file)
		}
		if layer.toward != "" && !reading.set.named(layer.toward) {
			return fmt.Errorf("%s: atmosphere thins toward unknown "+
				"object %q", reading.file, layer.toward)
		}
		if !layer.world && reading.set.terrain == nil {
			return fmt.Errorf("%s: a terrain atmosphere needs "+
				"terrain",
				reading.file)
		}
	}
	return nil
}
