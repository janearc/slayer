package set

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// twoThings is a block of a star and a lantern, so a camera has things to
// stand by and look at, and lengths in radii have a star.
const twoThings = `  setting: space
  set:
    things:
      sun:
        kind: star
        seed: 1
        place: {at: origin}
      lamp:
        kind: lantern
        seed: 2
        around: sun
`

// withCamera reads a small version 2 set of two things and one camera of
// the lines given, which stand under the camera's name.
func withCamera(lines string) (Set, error) {
	text := strings.Replace(v2Text(defaultFilm, twoThings), defaultCamera,
		"  cameras:\n    cam:\n      title: c\n"+lines, 1)
	return Read("t.yaml", []byte(text))
}

// lampEye is the lines that give the lantern of twoThings an eye, as an
// instrument has.
const lampEye = "        eye:\n          pupil: {value: 1, unit: m}\n" +
	"          focal: {value: 2, unit: m}\n          models: diffraction-limited pupil\n"

// withCameraOf is withCamera for a set whose lantern says more.
func withCameraOf(more, lines string) (Set, error) {
	text := strings.Replace(v2Text(defaultFilm, twoThings+more), defaultCamera,
		"  cameras:\n    cam:\n      title: c\n"+lines, 1)
	return Read("t.yaml", []byte(text))
}

// A camera moves: from one place to another, looking at a thing, over
// seconds, easing as it says; each part is read into a typed field.
func TestCameraMovesAreRead(t *testing.T) {
	made, err := withCamera(`      moves:
        - from: {object: lamp, distance: {value: 12, unit: radii}, above_plane: {value: 25, unit: deg}}
          to: {object: lamp, behind: {value: 25, unit: m}, above: {value: 6, unit: m}}
          look: {object: sun, offset: {value: 0.8, unit: radii}}
          duration: {value: 10, unit: s}
          ease: in-out
        - look: {sky: {ra: 110.8, dec: -73.45}}
          field: {from: {value: 100, unit: km}, to: {value: 200, unit: m}}
          duration: {value: 4, unit: s}
`)
	if err != nil {
		t.Fatal(err)
	}
	moves := made.cameras[0].moves
	if len(moves) != 2 {
		t.Fatalf("%d moves read, want 2", len(moves))
	}
	first, second := moves[0], moves[1]
	if first.from.object != "lamp" || *first.from.distance != (length{12, "radii"}) ||
		*first.from.abovePlane != 25 || first.to.behind == nil ||
		*first.to.behind != (length{25, "m"}) || *first.to.above != (length{6, "m"}) {
		t.Errorf("the first move's places are %+v to %+v", first.from, first.to)
	}
	if first.look.object != "sun" || *first.look.offset != (length{0.8, "radii"}) ||
		first.seconds != 10 || first.ease != "in-out" {
		t.Errorf("the first move looks %+v for %v s, %q", first.look,
			first.seconds, first.ease)
	}
	if second.from != nil || second.to != nil || second.look.sky == nil ||
		*second.look.sky != (skyPoint{110.8, -73.45}) ||
		second.field == nil || second.field.from != (length{100, "km"}) ||
		second.field.to != (length{200, "m"}) || second.ease != "" {
		t.Errorf("the second move is %+v", second)
	}
}

// A camera stands at a place set by a thing, by a distance and an angle
// above the thing's plane, and points to a thing.
func TestCameraRelativeLocationIsRead(t *testing.T) {
	made, err := withCamera(`      orientation: {point_to: lamp}
      location:
        relative: {object: lamp, distance: {value: 3600000, unit: km}, above_plane: {value: 25, unit: deg}}
`)
	if err != nil {
		t.Fatal(err)
	}
	place := made.cameras[0].relative
	if place == nil || place.object != "lamp" ||
		*place.distance != (length{3600000, "km"}) || *place.abovePlane != 25 {
		t.Errorf("the camera's place is %+v", place)
	}
}

// A camera's eye is a person's, or an instrument of the set.
func TestCameraEyeIsAPersonsOrAnInstruments(t *testing.T) {
	made, err := withCameraOf(lampEye, "      eye: {instrument: lamp}\n      moves:\n"+
		"        - look: {object: sun}\n          duration: {value: 1, unit: s}\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := made.cameras[0].instrument; got != "lamp" {
		t.Errorf("the camera's instrument is %q, want lamp", got)
	}
	made, err = withCamera("      eye: human\n      orientation: {look_at: [0, 0, 2]}\n" +
		"      location: {at: [0, -5, 2]}\n")
	if err != nil || made.cameras[0].instrument != "" {
		t.Errorf("a human eye reads as %q, %v", made.cameras[0].instrument, err)
	}
}

// A camera's moves, place and eye refuse what is not theirs; each is a
// camera that is good but for those lines.
func TestCameraMovesPlaceAndEyeRefuseWhatIsNotTheirs(t *testing.T) {
	look := "        - look: {object: sun}\n          duration: {value: 1, unit: s}\n"
	for why, lines := range map[string]string{
		"a move with no look":                       "      moves:\n        - duration: {value: 1, unit: s}\n",
		"a move with no duration":                   "      moves:\n        - look: {object: sun}\n",
		"an ease nobody has":                        "      moves:\n        - look: {object: sun}\n          duration: {value: 1, unit: s}\n          ease: bounce\n",
		"a look at a thing and the sky":             "      moves:\n        - look: {object: sun, sky: {ra: 1, dec: 1}}\n          duration: {value: 1, unit: s}\n",
		"a look at a thing the set lacks":           "      moves:\n        - look: {object: moon}\n          duration: {value: 1, unit: s}\n",
		"a place by a thing the set lacks":          "      moves:\n        - from: {object: moon, up: {value: 1, unit: m}}\n          look: {object: sun}\n          duration: {value: 1, unit: s}\n",
		"a place with no thing":                     "      moves:\n        - from: {up: {value: 1, unit: m}}\n          look: {object: sun}\n          duration: {value: 1, unit: s}\n",
		"an angle past a turn":                      "      moves:\n        - from: {object: sun, above_plane: {value: 400, unit: deg}}\n          look: {object: sun}\n          duration: {value: 1, unit: s}\n",
		"a field with one end":                      "      moves:\n        - look: {object: sun}\n          field: {from: {value: 1, unit: km}}\n          duration: {value: 1, unit: s}\n",
		"a duration of the song":                    "      moves:\n        - look: {object: sun}\n          duration: {value: song, unit: s}\n",
		"no moves at all":                           "      moves: []\n",
		"an eye that is no instrument":              "      eye: {glass: lamp}\n      moves:\n" + look,
		"an instrument the set lacks":               "      eye: {instrument: moon}\n      moves:\n" + look,
		"a relative place of a thing the set lacks": "      orientation: {point_to: lamp}\n      location:\n        relative: {object: moon, distance: {value: 1, unit: km}}\n",
		"a relative place and an at":                "      orientation: {point_to: lamp}\n      location:\n        at: [0, -5, 2]\n        relative: {object: lamp, distance: {value: 1, unit: km}}\n",
		"a relative place and a standoff":           "      orientation: {point_to: lamp}\n      location:\n        loc_relative: {object: lamp, standoff_x: 1}\n        relative: {object: lamp, distance: {value: 1, unit: km}}\n",
	} {
		if _, err := withCamera(lines); err == nil {
			t.Errorf("%s was taken", why)
		}
	}
}

// A length in radii in a camera's move needs a star in the set, as any
// length does.
func TestCameraLengthInRadiiNeedsAStar(t *testing.T) {
	lines := "      moves:\n        - from: {object: lamp, distance: {value: 2, unit: radii}}\n" +
		"          look: {object: lamp}\n          duration: {value: 1, unit: s}\n"
	noStar := strings.Replace(v2Text(defaultFilm, "  setting: space\n  set:\n"+
		"    things:\n      lamp:\n        kind: lantern\n        seed: 2\n"+
		"        place: {at: origin}\n"), defaultCamera,
		"  cameras:\n    cam:\n      title: c\n"+lines, 1)
	if _, err := Read("t.yaml", []byte(noStar)); err == nil {
		t.Error("a move in radii was taken in a set with no star")
	}
	if _, err := withCamera(lines); err != nil {
		t.Errorf("a move in radii is refused beside a star: %v", err)
	}
}

// anywhere is the place every asset is found at, whatever its name.
func anywhere(string) (maths.Vec, bool) { return maths.Vec{}, true }

// A camera that moves, or stands by a relative place, is not one Shots
// makes yet: it refuses it by name and does not make a shot of it at a
// place it never said.
func TestShotsRefuseACameraThatMoves(t *testing.T) {
	made, err := withCamera("      moves:\n        - look: {object: sun}\n" +
		"          duration: {value: 1, unit: s}\n")
	if err != nil {
		t.Fatal(err)
	}
	_, err = made.Shots(anywhere)
	if err == nil || !strings.Contains(err.Error(), "camera cam") ||
		!strings.Contains(err.Error(), "does not make yet") {
		t.Errorf("Shots of a moving camera says %v", err)
	}
}

// What a version 2 camera says is refused in a file that names no schema,
// by name: its moves and its place by a thing.
func TestCameraKeysOfVersionTwoAreRefusedOutsideIt(t *testing.T) {
	legacy := strings.Replace(v2Text(defaultFilm, twoThings),
		`"$schema": `+ecosystemV2Schema+"\n", "", 1)
	moving := strings.Replace(legacy, defaultCamera, "  cameras:\n    cam:\n"+
		"      title: c\n      moves:\n        - look: {object: sun}\n"+
		"          duration: {value: 1, unit: s}\n", 1)
	if _, err := Read("t.yaml", []byte(moving)); err == nil ||
		!strings.Contains(err.Error(), "schema version 2") {
		t.Errorf("a legacy camera with moves: read says %v", err)
	}
	eyed := strings.Replace(legacy, defaultCamera, "  cameras:\n    cam:\n"+
		"      title: c\n      eye: {instrument: lamp}\n", 1)
	if _, err := Read("t.yaml", []byte(eyed)); err == nil ||
		!strings.Contains(err.Error(), "eye takes") {
		t.Errorf("a legacy camera with an instrument eye: read says %v", err)
	}
}
