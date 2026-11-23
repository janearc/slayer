package set

import (
	"math"
	"strings"
	"time"

	"github.com/janearc/slayer/scene/daylight"
	yaml "gopkg.in/yaml.v3"
)

// The clock's offset from UTC, in hours, runs from the zone furthest behind
// to the zone furthest ahead: UTC-12 (Baker Island) to UTC+14 (the Line
// Islands). Source: the IANA time zone database.
const (
	leastOffsetHours = -12
	mostOffsetHours  = 14
)

// The size of the units a clock is read in. Defined: an hour is sixty
// minutes of sixty seconds.
const (
	minutesPerHour = 60
	secondsPerHour = 3600
)

// clockZoneName names the zone built from a scene's utc-offset. The name
// is only a label; the offset is what the zone holds.
const clockZoneName = "clock"

// dateStep reads the day the scene says it is. On the ground the day, with
// a place, a clock time and the clock's offset, works the sun; in space
// the day is read and no stage draws it yet.
func (reading reader) dateStep(name, value *yaml.Node) error {
	var got calendarDay
	if err := reading.day(&got)(name, value); err != nil {
		return err
	}
	reading.set.when = &got
	if reading.spaceCoordinates {
		reading.note("scene.date", name, value)
	}
	return nil
}

// sunned works the sun when the scene says a date: from its place, the
// date, its clock time and the clock's offset from UTC. A scene may say
// the sun that way or by azimuth and altitude, never both, and a scene
// with a time and nothing to work it from is refused.
func (reading reader) sunned(scene *yaml.Node) error {
	if reading.set.InSpace() {
		return nil
	}
	date := valueNode(scene, "date")
	if date == nil {
		return reading.unworked(scene)
	}
	if typedSun(scene) {
		return reading.refuse(valueNode(scene, "sun"), "the scene says a "+
			"date (line %d) and a sun with an azimuth and an altitude: the "+
			"date works the sun from a place, a clock time and the clock's "+
			"offset from UTC, so say the date or say the azimuth and "+
			"altitude, not both", date.Line)
	}
	if missing := missingForSun(scene); len(missing) > 0 {
		return reading.refuse(date, "date works the sun from a place, a "+
			"date, a clock time and the clock's offset from UTC, and this "+
			"scene has no %s", strings.Join(missing, " and no "))
	}
	return reading.work(scene, date)
}

// unworked refuses what a scene with no date says that only a date could
// use: a clock's offset, a sun with no direction, and a time that is asked
// to light the sun and has no date to do it from.
func (reading reader) unworked(scene *yaml.Node) error {
	if offset := valueNode(scene, "utc-offset"); offset != nil {
		return reading.refuse(offset, "utc-offset is the offset of the "+
			"clock that works the sun, and this scene says no date, so "+
			"nothing reads it")
	}
	if sun := valueNode(scene, "sun"); sun != nil && !typedSun(scene) {
		return reading.refuse(sun, "sun says no azimuth and no altitude, "+
			"and this scene says no date to work them from")
	}
	clock := valueNode(scene, "timeofday")
	if clock == nil || valueOf(scene, "light") != "sun" || typedSun(scene) {
		return nil
	}
	return reading.refuse(clock, "timeofday under light: sun works the "+
		"sun from a place, a date, a clock time and the clock's offset "+
		"from UTC, and this scene has no date: say a date, or say the "+
		"sun's azimuth and altitude")
}

// typedSun is whether the scene says the sun's direction outright.
func typedSun(scene *yaml.Node) bool {
	sun := valueNode(scene, "sun")
	return says(sun, "azimuth") || says(sun, "altitude")
}

// missingForSun is what the scene lacks of what a date needs to work the
// sun, by key, in the order the refusal names them.
func missingForSun(scene *yaml.Node) []string {
	var missing []string
	place := valueNode(scene, "location")
	if !says(place, "lat") || !says(place, "lon") {
		missing = append(missing, "location (lat and lon)")
	}
	if !says(scene, "timeofday") {
		missing = append(missing, "timeofday")
	}
	if !says(scene, "utc-offset") {
		missing = append(missing, "utc-offset")
	}
	return missing
}

// work puts the sun where scene/daylight finds it for the scene's place
// and the instant its date, clock time and offset make. A sun below the
// horizon is kept as it is: the sun is down. A tint the scene said for
// its sun is kept. A moon scene needs the sun below the horizon, so a
// sun above it there is refused.
func (reading reader) work(scene, date *yaml.Node) error {
	made := reading.set
	zone := time.FixedZone(clockZoneName,
		int(math.Round(made.offset*secondsPerHour)))
	day := *made.when
	instant := time.Date(day.year, time.Month(day.month), day.of,
		made.clock/minutesPerHour, made.clock%minutesPerHour, 0, 0, zone)
	where, err := daylight.Sun(made.latitude, made.longitude, instant)
	if err != nil {
		return reading.refuse(date, "%v", err)
	}
	if valueOf(scene, "light") == "moon" && where.Altitude > 0 {
		return reading.refuse(date, "a moon scene needs the sun below "+
			"the horizon, and on %s at %02d:%02d the sun stands "+
			"%.1f degrees above it: say a date and an hour of night",
			date.Value, made.clock/minutesPerHour,
			made.clock%minutesPerHour, where.Altitude)
	}
	worked := &Sun{azimuth: where.Azimuth, altitude: where.Altitude,
		worked: true}
	if made.sun != nil {
		worked.tint = made.sun.tint
	}
	made.sun = worked
	return nil
}

// Down is whether the sun is worked from a place and a time and stands at
// or below the horizon. A sun said by azimuth and altitude is never down:
// a stage refuses that one if it is not above the horizon.
func (made Sun) Down() bool { return made.worked && made.altitude <= 0 }
