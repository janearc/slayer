package set

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/scene/daylight"
)

// sunText is a small set under the version 2 schema, lit as the light
// says, with the lines added under scene.
func sunText(light, add string) []byte {
	return []byte(`"$schema": ` + ecosystemV2Schema + `
scene:
  name: t
  seed: 1
  light: ` + light + "\n" + defaultFilm + defaultCamera + add)
}

// workedPlace is a place, a date, a time and an offset, as a scene says
// them: forty north, on the prime meridian, at noon on the June solstice.
const workedPlace = "  location: {lat: 40, lon: 0}\n" +
	"  date: \"2026-06-21\"\n  timeofday: 1200\n" +
	"  utc-offset: {value: 0, unit: h}\n"

// A place, a date, a clock time and an offset work the sun: at noon on
// the solstice forty degrees north the sun stands over the south at an
// altitude of the complement of the latitude and the tilt of the earth.
func TestADateWorksTheSun(t *testing.T) {
	made, err := Read("t.yaml", sunText("sun", workedPlace))
	if err != nil {
		t.Fatal(err)
	}
	sun, said := made.Sunlit()
	if !said {
		t.Fatal("the scene says a date and has no sun")
	}
	azimuth, altitude := sun.Where()
	const tilt, within = 23.44, 2
	if want := 90 - 40 + tilt; math.Abs(altitude-want) > within {
		t.Errorf("altitude is %.2f, want %.2f within %d", altitude, want,
			within)
	}
	if math.Abs(azimuth-180) > within*2 {
		t.Errorf("azimuth is %.2f, want 180 within %d", azimuth, within*2)
	}
	if sun.Down() {
		t.Error("a sun at that altitude is down")
	}
}

// The sun is what scene/daylight says for the same instant, and the same
// instant said on two clocks is the same sun.
func TestTheWorkedSunIsDaylightsAtThatInstant(t *testing.T) {
	const place = "  location: {lat: 22.5238, lon: 114.0209}\n"
	for _, clock := range []struct{ time, offset string }{
		{"1725", "8"}, {"0925", "0"}, {"1255", "-4"}, {"2255", "5.5"},
	} {
		made, err := Read("t.yaml", sunText("sun", place+
			"  date: \"2026-11-20\"\n  timeofday: "+clock.time+
			"\n  utc-offset: {value: "+clock.offset+", unit: h}\n"))
		if err != nil {
			t.Fatalf("%v: %v", clock, err)
		}
		sun, _ := made.Sunlit()
		azimuth, altitude := sun.Where()
		instant := time.Date(2026, time.November, 20, 9, 25, 0, 0, time.UTC)
		if clock.offset == "5.5" {
			// 22:55 at +5:30 is 17:25 UTC the same day
			instant = time.Date(2026, time.November, 20, 17, 25, 0, 0, time.UTC)
		}
		if clock.time == "1255" {
			// 12:55 at -4 is 16:55 UTC
			instant = time.Date(2026, time.November, 20, 16, 55, 0, 0, time.UTC)
		}
		want, err := daylight.Sun(22.5238, 114.0209, instant)
		if err != nil {
			t.Fatal(err)
		}
		if azimuth != want.Azimuth || altitude != want.Altitude {
			t.Errorf("%v: sun at %.4f, %.4f; daylight says %.4f, %.4f", clock,
				azimuth, altitude, want.Azimuth, want.Altitude)
		}
	}
}

// A sun below the horizon is read, not refused: it is down, and a tint
// the scene said for it is kept.
func TestASunBelowTheHorizonIsDownNotRefused(t *testing.T) {
	night := strings.Replace(workedPlace, "timeofday: 1200", "timeofday: 0000", 1)
	made, err := Read("t.yaml", sunText("sun", night+
		"  sun: {tint: [1, 0.9, 0.8]}\n"))
	if err != nil {
		t.Fatal(err)
	}
	sun, said := made.Sunlit()
	if !said || !sun.Down() {
		t.Fatalf("midnight: sun said %v, down %v", said, sun.Down())
	}
	if red, green, blue := sun.Tint(); red != 1 || green != 0.9 || blue != 0.8 {
		t.Errorf("tint is %v %v %v, want 1 0.9 0.8", red, green, blue)
	}
}

// A sun said by azimuth and altitude is not down, whatever it says, and
// is read as it always was.
func TestATypedSunIsNeverWorkedDown(t *testing.T) {
	made, err := Read("t.yaml", sunText("sun",
		"  sun: {azimuth: 90, altitude: -5}\n"))
	if err != nil {
		t.Fatal(err)
	}
	sun, _ := made.Sunlit()
	if sun.Down() {
		t.Error("a typed sun is down")
	}
	if azimuth, altitude := sun.Where(); azimuth != 90 || altitude != -5 {
		t.Errorf("typed sun is %v, %v, want 90, -5", azimuth, altitude)
	}
}

// A scene that says its sun both ways is refused, and the refusal names
// both: the date and the azimuth and altitude.
func TestSayingTheSunBothWaysIsRefusedByName(t *testing.T) {
	_, err := Read("t.yaml", sunText("sun", workedPlace+
		"  sun: {azimuth: 180, altitude: 70}\n"))
	if err == nil {
		t.Fatal("a date and a typed sun were both taken")
	}
	for _, name := range []string{"date", "azimuth", "altitude"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("the refusal %q does not name %s", err, name)
		}
	}
}

// A date with something missing is refused, and the refusal says which.
func TestADateWithWhatItNeedsMissingIsRefused(t *testing.T) {
	for why, test := range map[string]struct{ add, missing string }{
		"no place": {"  date: \"2026-06-21\"\n  timeofday: 1200\n" +
			"  utc-offset: {value: 0, unit: h}\n", "location"},
		"a place with no longitude": {"  location: {lat: 40}\n" +
			"  date: \"2026-06-21\"\n  timeofday: 1200\n" +
			"  utc-offset: {value: 0, unit: h}\n", "location"},
		"no time": {"  location: {lat: 40, lon: 0}\n" +
			"  date: \"2026-06-21\"\n" +
			"  utc-offset: {value: 0, unit: h}\n", "timeofday"},
		"no offset": {"  location: {lat: 40, lon: 0}\n" +
			"  date: \"2026-06-21\"\n  timeofday: 1200\n", "utc-offset"},
	} {
		_, err := Read("t.yaml", sunText("sun", test.add))
		if err == nil {
			t.Errorf("%s: the date was taken", why)
		} else if !strings.Contains(err.Error(), test.missing) {
			t.Errorf("%s: the refusal %q does not say %s is missing", why,
				err, test.missing)
		}
	}
}

// A time under the sun with no date to work it from is refused, a clock's
// offset with no date is refused, and a sun with no direction and no date
// is refused.
func TestATimeWithNoDateIsRefused(t *testing.T) {
	for why, test := range map[string]struct{ add, says string }{
		"a time under light: sun with a place": {
			"  location: {lat: 40, lon: 0}\n  timeofday: 1200\n", "no date"},
		"an offset with no date": {
			"  utc-offset: {value: 0, unit: h}\n", "utc-offset"},
		"a tint-only sun with no date": {
			"  sun: {tint: [1, 1, 1]}\n", "no date"},
	} {
		_, err := Read("t.yaml", sunText("sun", test.add))
		if err == nil {
			t.Errorf("%s was taken", why)
		} else if !strings.Contains(err.Error(), test.says) {
			t.Errorf("%s: the refusal %q does not say %q", why, err,
				test.says)
		}
	}
}

// Anti-tests: what is not a refusal. A time under a moon, ambient light
// or a typed sun is read as it always was, and a date under the moon is
// read when its sun is below the horizon.
func TestWhatIsNotRefused(t *testing.T) {
	for why, test := range map[string]struct{ light, add string }{
		"a time under ambient light": {"ambient",
			"  location: {lat: 40, lon: 0}\n  timeofday: 1200\n"},
		"a time under a typed sun": {"sun",
			"  location: {lat: 40, lon: 0}\n  timeofday: 1200\n" +
				"  sun: {azimuth: 180, altitude: 70}\n"},
		"a night date under the moon": {"moon", strings.Replace(workedPlace,
			"timeofday: 1200", "timeofday: 0000", 1) +
			"  moon: {azimuth: 90, altitude: 30}\n"},
	} {
		if _, err := Read("t.yaml", sunText(test.light, test.add)); err != nil {
			t.Errorf("%s was refused: %v", why, err)
		}
	}
}

// A date under the moon whose worked sun is above the horizon is refused,
// and the refusal names the date, the hour and what a moon scene needs.
func TestADayDateUnderTheMoonIsRefused(t *testing.T) {
	_, err := Read("t.yaml", sunText("moon", workedPlace+
		"  moon: {azimuth: 90, altitude: 30}\n"))
	if err == nil {
		t.Fatal("a moon scene at noon on the solstice was taken")
	}
	for _, word := range []string{"2026-06-21", "12:00",
		"sun below the horizon"} {
		if !strings.Contains(err.Error(), word) {
			t.Errorf("the refusal %q does not say %q", err, word)
		}
	}
}

// A sun that says one of its two directions is refused as one that says
// both or neither, whichever direction it leaves out.
func TestASunWithOneDirectionIsRefused(t *testing.T) {
	for why, sun := range map[string]string{
		"an azimuth and no altitude": "  sun: {azimuth: 90}\n",
		"an altitude and no azimuth": "  sun: {altitude: 30}\n",
	} {
		if _, err := Read("t.yaml", sunText("sun", sun)); err == nil {
			t.Errorf("a sun with %s was taken", why)
		}
	}
}

// An offset outside the zones there are, or one with no unit, is refused.
func TestAnOffsetNoClockHasIsRefused(t *testing.T) {
	for why, offset := range map[string]string{
		"fifteen hours ahead": "{value: 15, unit: h}",
		"thirteen behind":     "{value: -13, unit: h}",
		"minutes":             "{value: 60, unit: min}",
		"no unit":             "{value: 1}",
	} {
		add := strings.Replace(workedPlace, "{value: 0, unit: h}", offset, 1)
		if _, err := Read("t.yaml", sunText("sun", add)); err == nil {
			t.Errorf("%s was taken", why)
		}
	}
}

// In space a date is read and not worked: it is the stage's to draw, and
// the scene needs no place for it.
func TestADateInSpaceIsNotWorked(t *testing.T) {
	made, err := Read("t.yaml", sunText("ambient",
		"  setting: space\n  date: \"2026-06-21\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, said := made.Sunlit(); said {
		t.Error("a date in space worked a sun")
	}
}
