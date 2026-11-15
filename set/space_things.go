package set

import (
	"fmt"
	"maps"
	"slices"

	"github.com/janearc/slayer/manufacture"
	yaml "gopkg.in/yaml.v3"
)

// placeAt is where the thing says it stands: the origin, the anchor or
// the frame; or nothing, when it says none.
func (made Thing) placeAt() string { return made.space.place }

// aroundName is the name of the thing this one goes around, or nothing.
func (made Thing) aroundName() string { return made.space.around }

// spaceThing is what a thing says in schema version 2 that the ground's
// things do not. A key a thing does not say leaves its field empty.
type spaceThing struct {
	place, around string
	star, limb    string
	corona        *corona
	panels        string
	turning       *bool
	radius        *length
	width, walls  *length
	lean          *float64 // degrees
	sense         int
	kelvin        *kelvinRange
	thickness     *float64
	neon, argon   *float64
	glow          *float64
	orbit         *orbitOf
	attitude      *attitudeOf
	eye           *instrumentEye
}

// corona is a star's corona: how far it reaches, in the star's radii,
// how many streamers it has, and the seed they are drawn from.
type corona struct {
	reach, streamers, seed float64
}

// kelvinRange is the coolest and the hottest temperature a thing's lights
// have, as black bodies', in kelvin.
type kelvinRange struct{ from, to float64 }

// length is a length as a set says it: a number and its unit, which is
// metres, kilometres, or the radii of the star the set is about.
type length struct {
	value float64
	unit  string
}

// spaceKinds are the kinds of thing only schema version 2 has.
var spaceKinds = []string{"point", "star", "lantern", "culture-orbital",
	"nebula"}

// kindKeys are the keys that belong to one kind of thing, by kind: a
// thing of another kind that says one is refused.
var kindKeys = map[string][]string{
	"star":            {"star", "limb", "corona"},
	"lantern":         {"panels", "turning"},
	"culture-orbital": {"radius", "width", "walls", "lean", "sense", "lights"},
	"nebula":          {"thickness", "neon", "argon", "glow"},
}

// kind reads a thing's kind: any kind the manufactory makes, and in a
// file that names schema version 2, the kinds that only it has, which are
// features read and not yet drawn.
func (reading reader) kind(name, value *yaml.Node) error {
	words := manufacture.KindNames()
	if reading.second {
		words = append(slices.Clone(words), spaceKinds...)
	}
	if err := reading.only(words...)(name, value); err != nil {
		return err
	}
	if slices.Contains(spaceKinds, value.Value) {
		reading.note("thing.kind", name, value)
	}
	return nil
}

// owned refuses a thing that says a key of one kind and is another.
func (reading reader) owned(block *yaml.Node, made Thing) error {
	for _, key := range block.Content[firstKey:] {
		for kind, keys := range kindKeys {
			if kind != made.kind && slices.Contains(keys, key.Value) &&
				says(block, key.Value) && isKey(block, key) {
				return reading.refuse(key, "%s is a key of a %s, and thing "+
					"%s is a %s", key.Value, kind, made.name, made.kind)
			}
		}
	}
	return nil
}

// isKey is whether a node of a block is one of its keys, not a value.
func isKey(block *yaml.Node, node *yaml.Node) bool {
	for at := firstKey; at+valueAfter < len(block.Content); at += pairWidth {
		if block.Content[at] == node {
			return true
		}
	}
	return false
}

// spaceThing is the keys of a thing that only schema version 2 has, read
// into the thing.
func (reading reader) spaceThing(made *Thing) map[string]step {
	steps := reading.placing(&made.space)
	maps.Copy(steps, reading.starring(&made.space))
	maps.Copy(steps, reading.lanterning(&made.space))
	maps.Copy(steps, reading.orbiting(&made.space))
	maps.Copy(steps, reading.nebulous(&made.space))
	maps.Copy(steps, reading.spacecraft(&made.space))
	return steps
}

// placing is the keys that say where a thing stands in space.
func (reading reader) placing(into *spaceThing) map[string]step {
	return map[string]step{
		"place": reading.versionTwo("thing.place",
			reading.placeStep(&into.place)),
		"around": reading.versionTwo("thing.around",
			reading.text(&into.around)),
	}
}

// starring is the keys of a star.
func (reading reader) starring(into *spaceThing) map[string]step {
	return map[string]step{
		"star": reading.versionTwo("thing.star",
			reading.word(&into.star, "sun")),
		"limb": reading.versionTwo("thing.limb",
			reading.word(&into.limb, "uniform", "darkened")),
		"corona": reading.versionTwo("thing.corona",
			reading.coronaStep(&into.corona)),
	}
}

// coronaStep reads a star's corona: how far it reaches, how many
// streamers and their seed, all three.
func (reading reader) coronaStep(into **corona) step {
	return func(_, value *yaml.Node) error {
		got := &corona{}
		err := reading.keys(value, map[string]step{
			"reach":     reading.number(&got.reach, leastReach, mostReach),
			"streamers": reading.whole(&got.streamers, leastCount, mostStreamers),
			"seed":      reading.whole(&got.seed, leastCount, mostSeed),
		}, nil)
		if err == nil {
			err = reading.all(value, "a corona", "reach", "streamers", "seed")
		}
		if err == nil && !(got.reach > noReach) {
			err = reading.refuse(value, "a corona reaches past the star")
		}
		if err == nil {
			*into = got
		}
		return err
	}
}

// noReach is a corona that reaches no further than its star. It is
// defined as none.
const noReach = 0

// leastReach is the least a corona's reach may be, in the star's radii.
// It is defined as none; the reader then refuses a reach of none.
const leastReach = 0

// mostReach is the most a corona's reach may be, in the star's radii.
// Picked: far beyond any star's corona.
const mostReach = 1e6

// leastCount is the least a count of streamers or a seed may be. It is
// defined as none.
const leastCount = 0

// mostStreamers is the most streamers a corona may have. Picked: more
// than a limb has room to draw.
const mostStreamers = 1e4

// lanterning is the keys of a lantern.
func (reading reader) lanterning(into *spaceThing) map[string]step {
	return map[string]step{
		"panels": reading.versionTwo("thing.panels",
			reading.word(&into.panels, "baseline")),
		"turning": reading.versionTwo("thing.turning",
			reading.maybeYes(&into.turning)),
	}
}

// maybeYes is a step that reads true or false into a pointer, which it
// leaves unset for a key not said.
func (reading reader) maybeYes(into **bool) step {
	return func(name, value *yaml.Node) error {
		var got bool
		if err := reading.yes(&got)(name, value); err != nil {
			return err
		}
		*into = &got
		return nil
	}
}

// maybe is a step that reads a number from least to most into a pointer.
func (reading reader) maybe(into **float64, least, most float64) step {
	return func(name, value *yaml.Node) error {
		var got float64
		if err := reading.number(&got, least, most)(name, value); err != nil {
			return err
		}
		*into = &got
		return nil
	}
}

// orbiting is the keys of a culture orbital.
func (reading reader) orbiting(into *spaceThing) map[string]step {
	return map[string]step{
		"radius": reading.versionTwo("thing.radius",
			reading.lengthStep(&into.radius)),
		"width": reading.versionTwo("thing.width",
			reading.lengthStep(&into.width)),
		"walls": reading.versionTwo("thing.walls",
			reading.lengthStep(&into.walls)),
		"lean":  reading.versionTwo("thing.lean", reading.leanStep(&into.lean)),
		"sense": reading.versionTwo("thing.sense", reading.senseStep(&into.sense)),
		"lights": reading.versionTwo("thing.lights",
			reading.lightsStep(&into.kelvin)),
	}
}

// leanStep reads a lean, a number of degrees from the axis.
func (reading reader) leanStep(into **float64) step {
	return func(_, value *yaml.Node) error {
		err := reading.keys(value, map[string]step{
			"value": reading.maybe(into, -mostLean, mostLean),
			"unit":  reading.only("deg"),
		}, nil)
		if err == nil {
			err = reading.all(value, "a lean", "value", "unit")
		}
		return err
	}
}

// mostLean is the most a thing leans from its axis, in degrees. It is
// defined as a right angle: past it the thing leans the other way.
const mostLean = 90

// senseStep reads which way a thing turns: one way, 1, or the other, -1.
func (reading reader) senseStep(into *int) step {
	return func(name, value *yaml.Node) error {
		var got float64
		if err := reading.number(&got, -turnsOne, turnsOne)(name, value); err != nil {
			return err
		}
		if got != turnsOne && got != -turnsOne {
			return reading.refuse(value, "%s takes 1 or -1, not %q",
				name.Value, value.Value)
		}
		*into = int(got)
		return nil
	}
}

// turnsOne is the size of a sense of turning, either way. It is defined
// as one.
const turnsOne = 1

// lightsStep reads the lights a thing carries: the warmest and the
// coolest of their colours, as black bodies' temperatures in kelvin.
func (reading reader) lightsStep(into **kelvinRange) step {
	return func(_, value *yaml.Node) error {
		return reading.keys(value, map[string]step{
			"kelvin": reading.kelvinStep(into),
		}, nil)
	}
}

// kelvinStep reads a range of temperatures, from one to another, in kelvin.
func (reading reader) kelvinStep(into **kelvinRange) step {
	return func(_, value *yaml.Node) error {
		got := kelvinRange{}
		err := reading.keys(value, map[string]step{
			"from": reading.number(&got.from, leastKelvin, mostKelvin),
			"to":   reading.number(&got.to, leastKelvin, mostKelvin),
		}, nil)
		if err == nil {
			err = reading.all(value, "a range of kelvin", "from", "to")
		}
		if err == nil {
			*into = &got
		}
		return err
	}
}

// leastKelvin is the coolest a light's temperature may be, in kelvin. It
// is defined as one.
const leastKelvin = 1

// mostKelvin is the hottest a light's temperature may be, in kelvin.
// Picked: a million, which holds every light there is.
const mostKelvin = 1e6

// nebulous is the keys of a nebula.
func (reading reader) nebulous(into *spaceThing) map[string]step {
	return map[string]step{
		"thickness": reading.versionTwo("thing.thickness",
			reading.maybe(&into.thickness, leastThickness, mostThickness)),
		"neon": reading.versionTwo("thing.neon",
			reading.maybe(&into.neon, noShare, wholeShare)),
		"argon": reading.versionTwo("thing.argon",
			reading.maybe(&into.argon, noShare, wholeShare)),
		"glow": reading.versionTwo("thing.glow",
			reading.maybe(&into.glow, noShare, mostGlow)),
	}
}

// leastThickness is the least a nebula's thickness may be. Picked: the
// least the exposure's range takes.
const leastThickness = 1e-12

// mostThickness is the most a nebula's thickness may be. Picked: far
// thicker than any scene wants.
const mostThickness = 1e3

// noShare is a gas that is none of a nebula. It is defined as none.
const noShare = 0

// wholeShare is a gas that is all of a nebula. It is defined as the whole,
// one.
const wholeShare = 1

// mostGlow is the most a nebula's glow may be. Picked: a thousand times
// the base.
const mostGlow = 1e3

// lengthStep reads a length: a number and its unit, metres, kilometres or
// the radii of the star the set is about. A length in radii is kept to be
// checked for a star once the set is read.
func (reading reader) lengthStep(into **length) step {
	return func(_, value *yaml.Node) error {
		got := &length{}
		err := reading.keys(value, map[string]step{
			"value": reading.number(&got.value, -mostLength, mostLength),
			"unit":  reading.word(&got.unit, "m", "km", "radii"),
		}, nil)
		if err == nil {
			err = reading.all(value, "a length", "value", "unit")
		}
		if err == nil {
			*into = got
			if got.unit == "radii" {
				reading.set.radii = append(reading.set.radii,
					position{value.Line, value.Column})
			}
		}
		return err
	}
}

// mostLength is the most a length may be, in its own unit. Picked: well
// beyond any scene in metres.
const mostLength = 1e15

// position is a place in the file a set was read from.
type position struct{ line, column int }

// foundStar refuses a length in radii in a set that has no star for it to
// be the radii of.
func (reading reader) foundStar() error {
	made := reading.set
	if len(made.radii) == noRadii {
		return nil
	}
	if slices.ContainsFunc(made.things, func(each Thing) bool {
		return each.kind == "star"
	}) {
		return nil
	}
	return fmt.Errorf("%s:%d:%d: a length in radii is the radii of a "+
		"star, and the set has no star", reading.file, made.radii[firstKey].line,
		made.radii[firstKey].column)
}

// noRadii is the count of lengths in radii in a set that has none. It is
// defined as zero.
const noRadii = 0

// placeStep reads where a thing says it stands, by name.
func (reading reader) placeStep(into *string) step {
	return func(_, value *yaml.Node) error {
		err := reading.keys(value, map[string]step{
			"at": reading.word(into, "anchor", "frame", "origin"),
		}, nil)
		if err == nil {
			err = reading.all(value, "a place", "at")
		}
		return err
	}
}
