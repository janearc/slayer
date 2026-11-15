package set

import (
	"fmt"

	yaml "gopkg.in/yaml.v3"
)

// anchorOf is the one place a scene measures from: its latitude and
// longitude, its elevation in metres or lidar's own, and the datum they
// are on.
type anchorOf struct {
	lat, lon, elevation float64
	lidar               bool
	datum               string
}

// globeOf is the world a scene stands on: the planet, where its
// elevation comes from near the anchor, in which file, and beyond it,
// and how its detail is spent.
type globeOf struct {
	planet, near, file, beyond, detail string
}

// starSky is the sky drawn from a catalogue: its name, the direction
// and the radius in degrees round it that are drawn, and the faintest
// magnitude.
type starSky struct {
	from             string
	toward           skyPoint
	radius, faintest float64
}

// orbitOf is a thing in orbit: its height, the inclination of its orbit
// in degrees, and the thing it passes over.
type orbitOf struct {
	height      length
	inclination float64
	over        string
}

// attitudeOf is which way a thing points: straight down ("nadir"), at a
// place on the sky ("sky"), or at a thing ("object"); and the side that
// faces the sun, if it says.
type attitudeOf struct {
	point  string
	sky    *skyPoint
	object string
	shield string
}

// instrumentEye is what an instrument models: a pupil and a focal
// length, and the model.
type instrumentEye struct {
	pupil, focal length
	models       string
}

// anchorStep reads the scene's anchor, all of it.
func (reading reader) anchorStep(_, value *yaml.Node) error {
	got := &anchorOf{}
	err := reading.keys(value, map[string]step{
		"lat":       reading.number(&got.lat, -mostPole, mostPole),
		"lon":       reading.number(&got.lon, -mostHalfTurn, mostHalfTurn),
		"elevation": reading.elevationStep(got),
		"datum":     reading.word(&got.datum, "wgs84"),
	}, nil)
	if err == nil {
		err = reading.all(value, "an anchor", "lat", "lon", "elevation", "datum")
	}
	if err == nil {
		reading.set.anchor = got
	}
	return err
}

// mostHalfTurn is the most a longitude may be, either way, in degrees. It
// is defined as half a turn.
const mostHalfTurn = 180

// lidarWord is what an elevation says to take the terrain's own at the
// anchor.
const lidarWord = "lidar"

// elevationStep reads an anchor's elevation: metres, or lidar's own.
func (reading reader) elevationStep(into *anchorOf) step {
	return func(_, value *yaml.Node) error {
		err := reading.keys(value, map[string]step{
			"value": func(name, node *yaml.Node) error {
				if node.Value == lidarWord {
					into.lidar = true
					return nil
				}
				return reading.number(&into.elevation, -mostLength,
					mostLength)(name, node)
			},
			"unit": reading.only("m"),
		}, nil)
		if err == nil {
			err = reading.all(value, "an elevation", "value", "unit")
		}
		return err
	}
}

// globeStep reads the scene's globe, all of it.
func (reading reader) globeStep(_, value *yaml.Node) error {
	got := &globeOf{}
	err := reading.keys(value, map[string]step{
		"planet":  reading.word(&got.planet, "earth"),
		"terrain": reading.globeTerrainStep(got),
		"detail":  reading.word(&got.detail, "by-size"),
	}, nil)
	if err == nil {
		err = reading.all(value, "a globe", "planet", "terrain", "detail")
	}
	if err == nil {
		reading.set.globe = got
	}
	return err
}

// globeTerrainStep reads where a globe's elevation comes from, near the
// anchor and beyond it.
func (reading reader) globeTerrainStep(into *globeOf) step {
	return func(_, value *yaml.Node) error {
		err := reading.keys(value, map[string]step{
			"near":   reading.nearStep(into),
			"beyond": reading.beyondStep(into),
		}, nil)
		if err == nil {
			err = reading.all(value, "a globe's terrain", "near", "beyond")
		}
		return err
	}
}

// nearStep reads the elevation model near the anchor, by its source's
// key and, if it says, its file.
func (reading reader) nearStep(into *globeOf) step {
	return func(_, value *yaml.Node) error {
		err := reading.keys(value, map[string]step{
			"from": reading.text(&into.near),
			"file": reading.text(&into.file),
		}, nil)
		if err == nil {
			err = reading.all(value, "a near model", "from")
		}
		return err
	}
}

// beyondStep reads the coarser elevation model beyond the anchor's, by its
// source's key.
func (reading reader) beyondStep(into *globeOf) step {
	return func(_, value *yaml.Node) error {
		err := reading.keys(value, map[string]step{
			"from": reading.text(&into.beyond),
		}, nil)
		if err == nil {
			err = reading.all(value, "a far model", "from")
		}
		return err
	}
}

// starSkyStep reads the scene's sky: stars from a catalogue.
func (reading reader) starSkyStep(_, value *yaml.Node) error {
	got := &starSky{}
	err := reading.keys(value, map[string]step{
		"stars": reading.starsStep(got),
	}, nil)
	if err == nil {
		err = reading.all(value, "a sky", "stars")
	}
	if err == nil {
		reading.set.sky = got
	}
	return err
}

// starsStep reads where the stars are drawn from and toward, how far round
// it and how faint.
func (reading reader) starsStep(into *starSky) step {
	return func(_, value *yaml.Node) error {
		var toward *skyPoint
		err := reading.keys(value, map[string]step{
			"from":   reading.word(&into.from, "gaia"),
			"toward": reading.skyStep(&toward),
			"radius": reading.quantity(&into.radius, "deg", noAngle, mostTurn),
			"faintest": reading.number(&into.faintest, -mostMagnitude,
				mostMagnitude),
		}, nil)
		if err == nil {
			err = reading.all(value, "stars", "from", "toward", "radius",
				"faintest")
		}
		if err == nil {
			into.toward = *toward
		}
		return err
	}
}

// mostMagnitude is the most a magnitude may be, either way. Picked: far
// past the faintest star and the brightest.
const mostMagnitude = 100

// spacecraft is the keys of a thing that say where it is in orbit, which
// way it points and what its eye models, which only schema version 2 has.
func (reading reader) spacecraft(into *spaceThing) map[string]step {
	return map[string]step{
		"orbit": reading.versionTwo("thing.orbit", reading.orbitStep(into)),
		"attitude": reading.versionTwo("thing.attitude",
			reading.attitudeStep(into)),
		"eye": reading.versionTwo("thing.eye", reading.instrumentStep(into)),
	}
}

// orbitStep reads a thing's orbit, all of it.
func (reading reader) orbitStep(into *spaceThing) step {
	return func(_, value *yaml.Node) error {
		got := &orbitOf{}
		var height *length
		err := reading.keys(value, map[string]step{
			"height":      reading.lengthStep(&height),
			"inclination": reading.number(&got.inclination, noAngle, mostHalfTurn),
			"over":        reading.text(&got.over),
		}, nil)
		if err == nil {
			err = reading.all(value, "an orbit", "height", "inclination", "over")
		}
		if err == nil {
			got.height = *height
			into.orbit = got
		}
		return err
	}
}

// attitudeStep reads which way a thing points, and the side that faces the
// sun.
func (reading reader) attitudeStep(into *spaceThing) step {
	return func(_, value *yaml.Node) error {
		got := &attitudeOf{}
		err := reading.keys(value, map[string]step{
			"point":  reading.pointStep(got),
			"shield": reading.word(&got.shield, "sun"),
		}, nil)
		if err == nil {
			err = reading.all(value, "an attitude", "point")
		}
		if err == nil {
			into.attitude = got
		}
		return err
	}
}

// pointStep reads what a thing points at: straight down, a place on the
// sky, or a thing.
func (reading reader) pointStep(into *attitudeOf) step {
	return func(name, value *yaml.Node) error {
		if value.Kind == yaml.ScalarNode {
			err := reading.only("nadir")(name, value)
			into.point = value.Value
			return err
		}
		err := reading.keys(value, map[string]step{
			"sky":    reading.skyStep(&into.sky),
			"object": reading.text(&into.object),
		}, nil)
		if err == nil && says(value, "sky") == says(value, "object") {
			err = reading.refuse(value, "a point says a place on the sky "+
				"or a thing, one of them")
		}
		if says(value, "sky") {
			into.point = "sky"
		} else {
			into.point = "object"
		}
		return err
	}
}

// instrumentStep reads what a thing's eye models: a pupil and a focal
// length, and the model, all three.
func (reading reader) instrumentStep(into *spaceThing) step {
	return func(_, value *yaml.Node) error {
		got := &instrumentEye{}
		var pupil, focal *length
		err := reading.keys(value, map[string]step{
			"pupil":  reading.lengthStep(&pupil),
			"focal":  reading.lengthStep(&focal),
			"models": reading.only("diffraction-limited pupil"),
		}, nil)
		if err == nil {
			err = reading.all(value, "an eye", "pupil", "focal", "models")
		}
		if err == nil {
			got.pupil, got.focal = *pupil, *focal
			got.models = valueOf(value, "models")
			into.eye = got
		}
		return err
	}
}

// extentStep reads how far an atmosphere layer reaches: over the terrain,
// as every file may say, or in a file that names schema version 2 over the
// world, which is a feature read and not yet drawn.
func (reading reader) extentStep(world *bool) step {
	terrain := reading.only("terrain")
	return func(name, value *yaml.Node) error {
		if !reading.second || value.Value == "terrain" {
			return terrain(name, value)
		}
		if err := reading.only("terrain", "world")(name, value); err != nil {
			return err
		}
		*world = true
		reading.note("atmosphere.extent", name, value)
		return nil
	}
}

// foundWorld refuses what a version 2 scene says of its world that its own
// things and scene do not bear out: a place at an anchor or a frame the
// scene does not say, an orbit over, or a thing pointed at, that the set
// does not have.
func (reading reader) foundWorld() error {
	made := reading.set
	for _, each := range made.things {
		if err := reading.foundWhere(each); err != nil {
			return err
		}
	}
	return nil
}

// foundWhere refuses a thing that is placed at an anchor or frame the scene
// lacks, or is set over or toward a thing the set lacks.
func (reading reader) foundWhere(each Thing) error {
	made := reading.set
	_, framed := made.frame()
	switch {
	case each.placeAt() == "anchor" && made.anchor == nil:
		return fmt.Errorf("%s: thing %s is placed at the anchor, and the "+
			"scene says none", reading.file, each.name)
	case each.placeAt() == "frame" && !framed:
		return fmt.Errorf("%s: thing %s is placed at the frame, and the "+
			"scene says none", reading.file, each.name)
	}
	for _, wanted := range each.space.wants() {
		if wanted == each.name {
			return fmt.Errorf("%s: thing %s is set over or toward itself",
				reading.file, each.name)
		}
		if !made.named(wanted) {
			return fmt.Errorf("%s: thing %s wants %q, and the set has no "+
				"asset by that name", reading.file, each.name, wanted)
		}
	}
	return nil
}

// wants is the names of the things a thing's orbit passes over and its
// attitude points at.
func (made spaceThing) wants() []string {
	var names []string
	if made.orbit != nil {
		names = append(names, made.orbit.over)
	}
	if made.attitude != nil && made.attitude.object != "" {
		names = append(names, made.attitude.object)
	}
	return names
}
