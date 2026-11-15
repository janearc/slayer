package set

import (
	"fmt"

	yaml "gopkg.in/yaml.v3"
)

// skyPoint is a direction on the sky, in degrees: right ascension and
// declination.
type skyPoint struct{ ra, dec float64 }

// relativePlace is a place set by a thing of the set: so far above it,
// behind it, up from it, off it, or at a distance from it, and at an
// angle above its plane. A part not said is nil.
type relativePlace struct {
	object                                string
	up, above, behind, standoff, distance *length
	abovePlane                            *float64 // degrees
}

// lookAt is what a move looks at: a thing or a place on the sky, and how
// far to one side of it.
type lookAt struct {
	object string
	sky    *skyPoint
	offset *length
}

// fieldOf is how wide a picture is across at what it looks at, at the start
// of a move and at its end.
type fieldOf struct{ from, to length }

// move is one move of a camera: from one place to another, or held where
// it is, looking, with the width the picture is across at what it looks
// at changing from one length to another, over some seconds, easing as
// it says.
type move struct {
	from, to *relativePlace
	look     lookAt
	field    *fieldOf
	seconds  float64
	ease     string
}

// spaceCamera is the keys of a camera that only schema version 2 has,
// read into the camera. Its eye is a person's or an instrument's.
func (reading reader) spaceCamera(shot *camera) map[string]step {
	return map[string]step{
		"duration": reading.versionTwo("camera.duration",
			reading.durationStep(&shot.duration)),
		"moves": reading.versionTwo("camera.moves",
			reading.movesStep(&shot.moves)),
		"eye": reading.eyeStep(shot),
	}
}

// spaceLocation is the keys of a camera's location that only schema
// version 2 has.
func (reading reader) spaceLocation(shot *camera) map[string]step {
	return map[string]step{
		"relative": reading.versionTwo("camera.location.relative",
			reading.relativeStep(&shot.relative)),
	}
}

// eyeStep reads a camera's eye: a person's, as every file may say, or in
// a file that names schema version 2 an instrument of the set, which is a
// feature read and not yet drawn.
func (reading reader) eyeStep(shot *camera) step {
	human := reading.only("human")
	return func(name, value *yaml.Node) error {
		if !reading.second || value.Kind == yaml.ScalarNode {
			return human(name, value)
		}
		err := reading.keys(value, map[string]step{
			"instrument": reading.text(&shot.instrument),
		}, nil)
		if err == nil {
			err = reading.all(value, "an eye", "instrument")
		}
		if err == nil {
			reading.note("camera.eye", name, value)
		}
		return err
	}
}

// movesStep reads a camera's moves, at least one, in order.
func (reading reader) movesStep(into *[]move) step {
	return func(name, value *yaml.Node) error {
		if value.Kind != yaml.SequenceNode || len(value.Content) == noMoves {
			return reading.refuse(value, "%s takes a list of at least one "+
				"move", name.Value)
		}
		for _, each := range value.Content {
			got := move{}
			if err := reading.moveStep(each, &got); err != nil {
				return err
			}
			*into = append(*into, got)
		}
		return nil
	}
}

// moveStep reads one move into got: it says what it looks at and how long
// it takes.
func (reading reader) moveStep(each *yaml.Node, got *move) error {
	err := reading.keys(each, map[string]step{
		"from":     reading.relativeStep(&got.from),
		"to":       reading.relativeStep(&got.to),
		"look":     reading.lookStep(&got.look),
		"field":    reading.fieldStep(&got.field),
		"duration": reading.quantity(&got.seconds, "s", leastSeconds, mostSeconds),
		"ease":     reading.word(&got.ease, "none", "in", "out", "in-out"),
	}, nil)
	if err == nil {
		err = reading.all(each, "a move", "look", "duration")
	}
	return err
}

// relativeStep reads a place set by a thing of the set.
func (reading reader) relativeStep(into **relativePlace) step {
	return func(_, value *yaml.Node) error {
		got := &relativePlace{}
		err := reading.keys(value, map[string]step{
			"object":      reading.text(&got.object),
			"up":          reading.lengthStep(&got.up),
			"above":       reading.lengthStep(&got.above),
			"behind":      reading.lengthStep(&got.behind),
			"standoff":    reading.lengthStep(&got.standoff),
			"distance":    reading.lengthStep(&got.distance),
			"above_plane": reading.degreesStep(&got.abovePlane),
		}, nil)
		if err == nil {
			err = reading.all(value, "a place", "object")
		}
		if err == nil {
			*into = got
		}
		return err
	}
}

// degreesStep reads an angle: a number of degrees, a turn either way at
// most, and its unit.
func (reading reader) degreesStep(into **float64) step {
	return func(_, value *yaml.Node) error {
		err := reading.keys(value, map[string]step{
			"value": reading.maybe(into, -mostTurn, mostTurn),
			"unit":  reading.only("deg"),
		}, nil)
		if err == nil {
			err = reading.all(value, "an angle", "value", "unit")
		}
		return err
	}
}

// mostTurn is the most an angle of the set's may be, either way, in
// degrees. It is defined as a whole turn.
const mostTurn = 360

// skyStep reads a direction on the sky: right ascension from 0 to 360
// degrees and declination from -90 to 90, both.
func (reading reader) skyStep(into **skyPoint) step {
	return func(_, value *yaml.Node) error {
		got := &skyPoint{}
		err := reading.keys(value, map[string]step{
			"ra":  reading.number(&got.ra, noAngle, mostTurn),
			"dec": reading.number(&got.dec, -mostPole, mostPole),
		}, nil)
		if err == nil {
			err = reading.all(value, "a direction on the sky", "ra", "dec")
		}
		if err == nil {
			*into = got
		}
		return err
	}
}

// noAngle is where an angle starts. It is defined as none.
const noAngle = 0

// mostPole is the most a latitude or a declination may be, either way.
// It is defined as a right angle.
const mostPole = 90

// lookStep reads what a move looks at: a thing or a place on the sky,
// one of them, and how far to one side of it.
func (reading reader) lookStep(into *lookAt) step {
	return func(_, value *yaml.Node) error {
		err := reading.keys(value, map[string]step{
			"object": reading.text(&into.object),
			"sky":    reading.skyStep(&into.sky),
			"offset": reading.lengthStep(&into.offset),
		}, nil)
		if err == nil && says(value, "object") == says(value, "sky") {
			err = reading.refuse(value, "a look says a thing or a place on "+
				"the sky, one of them")
		}
		return err
	}
}

// fieldStep reads how wide a picture is across at what it looks at, at the
// start of a move and at its end.
func (reading reader) fieldStep(into **fieldOf) step {
	return func(_, value *yaml.Node) error {
		var from, to *length
		err := reading.keys(value, map[string]step{
			"from": reading.lengthStep(&from),
			"to":   reading.lengthStep(&to),
		}, nil)
		if err == nil {
			err = reading.all(value, "a field", "from", "to")
		}
		if err == nil {
			*into = &fieldOf{from: *from, to: *to}
		}
		return err
	}
}

// foundCameras refuses a camera that stands by, looks at, or sees through
// a thing the set does not have.
func (reading reader) foundCameras() error {
	made := reading.set
	for _, each := range made.cameras {
		if err := reading.foundInstrument(each); err != nil {
			return err
		}
		for _, wanted := range each.wants() {
			if !made.named(wanted) {
				return fmt.Errorf("%s: camera %s wants %q, and the set has "+
					"no asset by that name", reading.file, each.name, wanted)
			}
		}
	}
	return nil
}

// wants is the names of the things a camera's version 2 keys use: its
// instrument, the thing its place is set by, and the things its moves
// stand by and look at.
func (shooting camera) wants() []string {
	var names []string
	if shooting.instrument != "" {
		names = append(names, shooting.instrument)
	}
	places := []*relativePlace{shooting.relative}
	for _, each := range shooting.moves {
		places = append(places, each.from, each.to)
		names = append(names, each.look.object)
	}
	for _, place := range places {
		if place != nil {
			names = append(names, place.object)
		}
	}
	return slicesWithout(names, "")
}

// slicesWithout is the names but for those that are a given one, in order.
func slicesWithout(names []string, not string) []string {
	var kept []string
	for _, each := range names {
		if each != not {
			kept = append(kept, each)
		}
	}
	return kept
}

// foundInstrument refuses a camera that sees through a thing that says no
// eye.
func (reading reader) foundInstrument(each camera) error {
	if each.instrument == "" {
		return nil
	}
	for _, held := range reading.set.things {
		if held.name == each.instrument && held.space.eye == nil {
			return fmt.Errorf("%s: camera %s sees through %q, which says "+
				"no eye", reading.file, each.name, each.instrument)
		}
	}
	return nil
}
