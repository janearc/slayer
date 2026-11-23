package set

import (
	"strings"
	"testing"
)

// defaultFilm is the film a small version 2 set says, as one line under
// scene.
const defaultFilm = "  film: {width: 64, height: 36, rays: 1, exposure: 1}\n"

// defaultCamera is the one camera a small version 2 set says, under
// scene.
const defaultCamera = `  cameras:
    only: {title: only, orientation: {look_at: [0, 0, 2]}, location: {at: [0, -5, 2]}}
`

// v2Text is a small set that names the version 2 schema, with the film and
// the lines added under scene.
func v2Text(film, add string) string {
	return `"$schema": ` + ecosystemV2Schema + `
scene:
  name: t
  seed: 1
  light: ambient
` + film + defaultCamera + add
}

// readV2 reads a small version 2 set with lines added under scene.
func readV2(add string) (Set, error) {
	source := v2Text(defaultFilm, add)
	return Read("t.yaml", []byte(source))
}

// stood is a block of things under set, each thing a name and its keys as
// the test writes them, indented to stand under scene.
func stood(each ...string) string {
	return "  set:\n    things:\n" + strings.Join(each, "")
}

// spruce is a thing of the usual kind, with more keys after its kind and
// seed, indented to stand under the thing's name.
func spruce(name, more string) string {
	return "      " + name + ":\n        kind: spruce\n        seed: 1\n" + more
}

// A scene says where it is, and when: its setting and its date, and a
// space scene's frame, each read into a typed field.
func TestSceneSettingDateAndFrameAreRead(t *testing.T) {
	made, err := readV2("  setting: space\n  date: \"2026-06-21\"\n" +
		"  frame:\n    at: sun-earth-l2\n    date: \"2026-06-22\"\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := made.setting(); got != "space" {
		t.Errorf("setting is %q, want space", got)
	}
	if got, said := made.date(); !said || got != (calendarDay{2026, 6, 21}) {
		t.Errorf("date is %v, %v, want 2026-06-21", got, said)
	}
	frame, said := made.frame()
	if !said || frame.at != "sun-earth-l2" ||
		frame.date == nil || *frame.date != (calendarDay{2026, 6, 22}) {
		t.Errorf("frame is %+v, %v, want sun-earth-l2 on 2026-06-22", frame, said)
	}
}

// A scene that says nothing of its setting is on the ground, says no date
// and has no frame; one that says ground says the same.
func TestSceneSaysGroundWhenItSaysNothing(t *testing.T) {
	for _, add := range []string{"", "  setting: ground\n"} {
		made, err := readV2(add)
		if err != nil {
			t.Fatal(err)
		}
		if _, said := made.date(); said || made.setting() != "ground" {
			t.Errorf("%q: setting %q, date said %v", add, made.setting(), said)
		}
		if _, said := made.frame(); said {
			t.Errorf("%q: a frame was said", add)
		}
	}
}

// A scene that says it is set in space is in space, as a stage asks it.
func TestASceneInSpaceSaysSo(t *testing.T) {
	made, err := readV2("  setting: space\n")
	if err != nil {
		t.Fatal(err)
	}
	if !made.InSpace() {
		t.Error("a scene set in space is not in space")
	}
}

// A scene that says it is on the ground, or says nothing, or names
// version 1, is not in space.
func TestASceneOnTheGroundIsNotInSpace(t *testing.T) {
	for _, add := range []string{"", "  setting: ground\n"} {
		made, err := readV2(add)
		if err != nil {
			t.Fatal(err)
		}
		if made.InSpace() {
			t.Errorf("%q: a scene on the ground is in space", add)
		}
	}
	first, err := Read("t.yaml", []byte("scene:\n  name: t\n"+defaultFilm+
		defaultCamera))
	if err != nil {
		t.Fatal(err)
	}
	if first.InSpace() {
		t.Error("a version 1 scene is in space")
	}
}

// A scene's setting, date and frame refuse what is not theirs; each
// refusal is of a set that is good but for that one line.
func TestSceneSettingDateAndFrameRefuseWhatIsNotTheirs(t *testing.T) {
	terrain := "  terrain:\n    from: usgs-kilauea-2019-dem\n" +
		"    file: unpacked/kil_2019dem/kil_2019dem.tif\n    west: 264500\n" +
		"    north: 2148100\n    across: 600\n    down: 600\n    step: 4\n"
	for why, add := range map[string]string{
		"a setting that is neither":        "  setting: orbit\n",
		"a date that is not a day":         "  date: \"the solstice\"\n",
		"a date that is not on a calendar": "  date: \"2026-02-30\"\n",
		"a frame at nowhere":               "  frame: {at: mars}\n",
		"a frame with no place":            "  frame: {date: \"2026-06-21\"}\n",
		"a frame's date off the calendar":  "  frame: {at: sun-earth-l2, date: \"2026-13-01\"}\n",
		"a space scene with terrain":       "  setting: space\n" + terrain,
	} {
		if _, err := readV2(add); err == nil {
			t.Errorf("%s was taken", why)
		}
	}
	if _, err := readV2("  setting: ground\n" + terrain); err != nil {
		t.Errorf("a ground scene with terrain is refused: %v", err)
	}
}

// A film says its frames a second, a whole number from 1 to 240.
func TestFilmFramesASecondAreRead(t *testing.T) {
	film := func(fps string) string {
		return "  film: {width: 64, height: 36, rays: 1, exposure: 1, fps: " +
			fps + "}\n"
	}
	source := v2Text(film("24"), "")
	made, err := Read("t.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if got := made.film.framesASecond(); got != 24 {
		t.Errorf("fps is %v, want 24", got)
	}
	for _, bad := range []string{"0", "241", "2.5", "-24", "fast"} {
		source := v2Text(film(bad), "")
		if _, err := Read("t.yaml", []byte(source)); err == nil {
			t.Errorf("fps %s was taken", bad)
		}
	}
	for _, good := range []string{"1", "240"} {
		source := v2Text(film(good), "")
		if _, err := Read("t.yaml", []byte(source)); err != nil {
			t.Errorf("fps %s is refused: %v", good, err)
		}
	}
}

// A thing is placed by its place, at the origin, the anchor or the frame,
// or by what it is around; each is read into a typed field.
func TestThingPlaceAndAroundAreRead(t *testing.T) {
	sun := spruce("sun", "        place: {at: origin}\n")
	moon := spruce("moon", "        around: sun\n")
	made, err := readV2("  setting: space\n" + stood(sun, moon))
	if err != nil {
		t.Fatal(err)
	}
	list := made.Things()
	if got := list[0].placeAt(); got != "origin" {
		t.Errorf("the sun's place is %q, want origin", got)
	}
	if got := list[1].aroundName(); got != "sun" {
		t.Errorf("the moon is around %q, want sun", got)
	}
	if got := list[0].aroundName(); got != "" {
		t.Errorf("the sun is around %q, want nothing", got)
	}
}

// A thing's place and its around refuse what is not theirs: a place at
// nowhere, an around that names no thing, a thing around itself, and two
// things around each other.
func TestThingPlaceAndAroundRefuseWhatIsNotTheirs(t *testing.T) {
	for why, add := range map[string]string{
		"a place at nowhere": stood(spruce("a",
			"        place: {at: elsewhere}\n")),
		"an around that names no thing": stood(spruce("a",
			"        around: nothing\n")),
		"a thing around itself": stood(spruce("a", "        around: a\n")),
		"two things around each other": stood(
			spruce("a", "        around: b\n"),
			spruce("b", "        around: a\n")),
	} {
		if _, err := readV2("  setting: space\n" + add); err == nil {
			t.Errorf("%s was taken", why)
		}
	}
}

// A version 2 thing says where it stands in one way: by east and north
// together, by a place, or by what it is around; never two, never none.
func TestThingStandsInOneWay(t *testing.T) {
	east := "        east: 1\n        north: 2\n"
	place := "        place: {at: origin}\n"
	around := "        around: base\n"
	base := spruce("base", east)
	for why, each := range map[string]struct {
		things string
		good   bool
	}{
		"east and north":          {stood(spruce("a", east)), true},
		"a place":                 {stood(spruce("a", place)), true},
		"an around":               {stood(base, spruce("a", around)), true},
		"none":                    {stood(spruce("a", "")), false},
		"east without north":      {stood(spruce("a", "        east: 1\n")), false},
		"east, north and a place": {stood(spruce("a", east+place)), false},
		"a place and an around":   {stood(base, spruce("a", place+around)), false},
	} {
		_, err := readV2("  setting: space\n" + each.things)
		if (err == nil) != each.good {
			t.Errorf("%s: read says %v, want good %v", why, err, each.good)
		}
	}
}

// A camera holds for a time, in seconds, or for as long as the song does.
func TestCameraDurationIsRead(t *testing.T) {
	camera := func(duration string) string {
		return "  cameras:\n    only:\n      title: only\n" +
			"      orientation: {look_at: [0, 0, 2]}\n" +
			"      location: {at: [0, -5, 2]}\n      duration: " + duration + "\n"
	}
	text := func(duration string) []byte {
		edited := strings.Replace(v2Text(defaultFilm, ""), defaultCamera,
			camera(duration), 1)
		return []byte(edited)
	}
	made, err := Read("t.yaml", text("{value: 6, unit: s}"))
	if err != nil {
		t.Fatal(err)
	}
	if got := made.cameras[0].duration; got == nil || got.seconds != 6 || got.song {
		t.Errorf("duration is %+v, want 6 seconds", got)
	}
	made, err = Read("t.yaml", text("{value: song, unit: s}"))
	if err != nil {
		t.Fatal(err)
	}
	if got := made.cameras[0].duration; got == nil || !got.song {
		t.Errorf("duration is %+v, want the song", got)
	}
	for _, bad := range []string{"{value: -1, unit: s}", "{value: forever, unit: s}",
		"{value: 6, unit: min}", "{value: 6}", "6"} {
		if _, err := Read("t.yaml", text(bad)); err == nil {
			t.Errorf("duration %s was taken", bad)
		}
	}
}

// A key only version 2 has is refused in a file that names no schema, by
// name, and says it is a key of version 2; and in one that names version
// 1, by that schema.
func TestKeysOfVersionTwoAreRefusedOutsideIt(t *testing.T) {
	legacy := strings.Replace(v2Text(defaultFilm, "  setting: space\n"),
		`"$schema": `+ecosystemV2Schema+"\n", "", 1)
	_, err := Read("t.yaml", []byte(legacy))
	if err == nil || !strings.Contains(err.Error(), "setting") ||
		!strings.Contains(err.Error(), "schema version 2") {
		t.Errorf("a legacy file with setting: read says %v", err)
	}
	named := strings.Replace(v2Text(defaultFilm, "  setting: space\n"),
		ecosystemV2Schema, EcosystemSchema, 1)
	if _, err := Read("t.yaml", []byte(named)); err == nil {
		t.Error("a version 1 file with setting: was taken")
	}
}
