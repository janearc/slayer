package set

import (
	"fmt"
	"maps"
	"slices"
	"time"

	yaml "gopkg.in/yaml.v3"
)

// Undrawn is a feature of a set that names schema version 2 which the
// reader keeps and no stage draws yet: the key's dotted name, as the
// schema's definitions call it, and the line and column it was written
// at.
type Undrawn struct {
	feature, says string
	line, column  int
}

// Feature is the feature's dotted name, such as scene.setting.
func (each Undrawn) Feature() string { return each.feature }

// Says is what the key says when it says one word or number, as written,
// and nothing when it says a block.
func (each Undrawn) Says() string { return each.says }

// Line is the line of the file the feature's key was written on.
func (each Undrawn) Line() int { return each.line }

// Column is the column of the file the feature's key was written at.
func (each Undrawn) Column() int { return each.column }

// Undrawn is every feature the set holds that nothing draws yet, in the
// order the file wrote them, each with where it was written.
func (made Set) Undrawn() []Undrawn { return slices.Clone(made.undrawn) }

// note keeps a feature of the key a node names as read and not yet
// drawn, with what the key says if it says a word or a number.
func (reading reader) note(feature string, key, value *yaml.Node) {
	said := ""
	if value.Kind == yaml.ScalarNode {
		said = value.Value
	}
	reading.set.undrawn = append(reading.set.undrawn, Undrawn{
		feature: feature, says: said, line: key.Line, column: key.Column})
}

// with is a block's steps with the steps of the keys only version 2 has
// added to them. A key in both is the later's.
func (reading reader) with(steps, more map[string]step) map[string]step {
	maps.Copy(steps, more)
	return steps
}

// onlyTwo is a step that reads a key only a file naming schema version 2
// may say: in any other file it refuses the key by name.
func (reading reader) onlyTwo(inner step) step {
	return func(name, value *yaml.Node) error {
		if !reading.second {
			return reading.refuse(name, "%s is a key of schema version 2, "+
				"and this file does not name that schema", name.Value)
		}
		return inner(name, value)
	}
}

// versionTwo is a step that reads a key of schema version 2 and keeps
// it as a feature that is read and not yet drawn.
func (reading reader) versionTwo(feature string, inner step) step {
	return reading.onlyTwo(func(name, value *yaml.Node) error {
		if err := inner(name, value); err != nil {
			return err
		}
		reading.note(feature, name, value)
		return nil
	})
}

// calendarDay is a day of the calendar, as a scene's date says it.
type calendarDay struct{ year, month, of int }

// dayLayout is how a date is written: year, month and day, with the
// zeros. It is the time package's reference date written that way.
const dayLayout = "2006-01-02"

// day is a step that reads a date, written year, month and day, that is
// a day of the calendar, into target.
func (reading reader) day(target *calendarDay) step {
	return func(name, value *yaml.Node) error {
		got, err := time.Parse(dayLayout, value.Value)
		if value.Kind != yaml.ScalarNode || err != nil {
			return reading.refuse(value, "%s takes a day of the calendar "+
				"as %s, not %q", name.Value, dayLayout, value.Value)
		}
		*target = calendarDay{got.Year(), int(got.Month()), got.Day()}
		return nil
	}
}

// settingGround and settingSpace are the two settings a scene has.
const (
	settingGround = "ground"
	settingSpace  = "space"
)

// setting is the scene's setting, the ground when it says none.
func (made Set) setting() string {
	if made.settingWord == "" {
		return settingGround
	}
	return made.settingWord
}

// InSpace is whether the scene says it is set in space, where a stage
// stands its things with no ground under them.
func (made Set) InSpace() bool { return made.setting() == settingSpace }

// date is the day the scene says it is, and whether it says one.
func (made Set) date() (calendarDay, bool) {
	if made.when == nil {
		return calendarDay{}, false
	}
	return *made.when, true
}

// spaceFrame is where a scene set in space is, and when: the place by its
// name, and a day, if it says one.
type spaceFrame struct {
	at   string
	date *calendarDay
}

// frame is the space frame the scene says, and whether it says one.
func (made Set) frame() (spaceFrame, bool) {
	if made.spaced == nil {
		return spaceFrame{}, false
	}
	return *made.spaced, true
}

// spaceScene is the keys of a scene that only schema version 2 has, and
// the steps that read them.
func (reading reader) spaceScene() map[string]step {
	return map[string]step{
		"setting": reading.onlyTwo(reading.settingStep),
		"date":    reading.versionTwo("scene.date", reading.dateStep),
		"frame":   reading.versionTwo("scene.frame", reading.frameStep),
		"anchor":  reading.versionTwo("scene.anchor", reading.anchorStep),
		"globe":   reading.versionTwo("scene.globe", reading.globeStep),
		"sky":     reading.versionTwo("scene.sky", reading.starSkyStep),
	}
}

// spaceFilm is the keys of a film that only schema version 2 has.
func (reading reader) spaceFilm() map[string]step {
	return map[string]step{
		"fps": reading.versionTwo("film.fps", reading.whole(
			&reading.set.film.fps, leastFrames, mostFrames)),
	}
}

// settingStep reads the scene's setting. Only a setting in space is a
// feature the stages do not draw.
func (reading reader) settingStep(name, value *yaml.Node) error {
	err := reading.word(&reading.set.settingWord, settingGround,
		settingSpace)(name, value)
	if err == nil && reading.set.settingWord == settingSpace {
		reading.note("scene.setting", name, value)
	}
	return err
}

// dateStep reads the day the scene says it is.
func (reading reader) dateStep(name, value *yaml.Node) error {
	var got calendarDay
	if err := reading.day(&got)(name, value); err != nil {
		return err
	}
	reading.set.when = &got
	return nil
}

// frameStep reads where a scene in space is, and when.
func (reading reader) frameStep(_, value *yaml.Node) error {
	got := &spaceFrame{}
	var day calendarDay
	err := reading.keys(value, map[string]step{
		"at": reading.only("sun-earth-l2"),
		"date": func(name, node *yaml.Node) error {
			if err := reading.day(&day)(name, node); err != nil {
				return err
			}
			got.date = &day
			return nil
		},
	}, nil)
	if err == nil {
		err = reading.all(value, "a frame", "at")
	}
	if err == nil {
		got.at = valueOf(value, "at")
		reading.set.spaced = got
	}
	return err
}

// durationOf is how long a camera holds or moves: so many seconds, or for
// as long as the song does.
type durationOf struct {
	seconds float64
	song    bool
}

// songWord is what a duration says in place of a number of seconds.
const songWord = "song"

// durationStep reads a camera's duration, a number of seconds or the
// song's.
func (reading reader) durationStep(into **durationOf) step {
	return func(_, value *yaml.Node) error {
		got := &durationOf{}
		err := reading.keys(value, map[string]step{
			"value": func(name, node *yaml.Node) error {
				if node.Value == songWord {
					got.song = true
					return nil
				}
				return reading.number(&got.seconds, leastSeconds,
					mostSeconds)(name, node)
			},
			"unit": reading.only("s"),
		}, nil)
		if err == nil {
			err = reading.all(value, "a duration", "value", "unit")
		}
		if err == nil {
			*into = got
		}
		return err
	}
}

// leastSeconds is the shortest a duration may be. It is defined as none.
const leastSeconds = 0

// mostSeconds is the longest a duration may be, a year in seconds.
// Picked: no film runs past it.
const mostSeconds = 31536000

// leastFrames is the fewest frames a second a film may say. It is defined
// as one.
const leastFrames = 1

// mostFrames is the most frames a second a film may say, as the schema
// says it. Picked: the fastest a camera records.
const mostFrames = 240

// framesASecond is how many frames each second of the film holds, or
// zero when it says none.
func (made Film) framesASecond() int { return int(made.fps) }

// standing refuses a version 2 thing that does not say where it stands
// in exactly one way: by east and north together, or by a place, or by
// what it is around.
func (reading reader) standing(block *yaml.Node, made Thing) error {
	ways := noWays
	for _, way := range []bool{says(block, "east") || says(block, "north"),
		says(block, "place"), says(block, "orbit"), says(block, "around")} {
		if way {
			ways++
		}
	}
	switch {
	case says(block, "east") != says(block, "north"):
		return reading.refuse(block, "thing %s says east and north "+
			"together, or neither", made.name)
	case ways != oneWay:
		return reading.refuse(block, "thing %s says where it stands in "+
			"one way: east and north, or place, or orbit, or around; not "+
			"%d of them", made.name, ways)
	}
	return nil
}

// noWays is the count of ways a thing has said so far. It is defined as
// none.
const noWays = 0

// oneWay is the count of ways a thing says where it stands. It is
// defined as one.
const oneWay = 1

// foundSpace refuses what a version 2 scene says that its own things do
// not bear out: terrain under a scene in space, and a thing that goes
// around a thing the set does not have, or around itself, or around a
// thing that goes around it.
func (reading reader) foundSpace() error {
	made := reading.set
	if made.setting() == settingSpace && made.terrain != nil {
		return fmt.Errorf("%s: a scene in space has no terrain, and this "+
			"one says some", reading.file)
	}
	around := map[string]string{}
	for _, each := range made.things {
		if each.aroundName() != "" {
			around[each.name] = each.aroundName()
		}
	}
	for _, each := range made.things {
		if each.aroundName() == "" {
			continue
		}
		if !slices.ContainsFunc(made.things, func(other Thing) bool {
			return other.name == each.aroundName()
		}) {
			return fmt.Errorf("%s: thing %s goes around %q, and the set has "+
				"no thing by that name", reading.file, each.name,
				each.aroundName())
		}
		if loops(around, each.name) {
			return fmt.Errorf("%s: thing %s goes around itself, by way of "+
				"the things it goes around", reading.file, each.name)
		}
	}
	return nil
}

// foundVersionTwo refuses what a file that names schema version 2 says
// that its own set does not bear out, in the order the checks run.
func (reading reader) foundVersionTwo() error {
	for _, found := range []func() error{reading.foundSpace,
		reading.foundStar, reading.foundWorld, reading.foundCameras} {
		if err := found(); err != nil {
			return err
		}
	}
	return nil
}

// loops is whether following what each thing goes around, from a thing,
// comes back to it.
func loops(around map[string]string, from string) bool {
	at := from
	for range len(around) {
		next, goes := around[at]
		if !goes {
			return false
		}
		if next == from {
			return true
		}
		at = next
	}
	return false
}
