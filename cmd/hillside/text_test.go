package main

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/typography"
)

// withText is the fixture's set with a text on it, carrie in go's regular
// facing east a metre off the ground and turning once in eight seconds,
// and each pair of changes after it made to the file, the first of each
// pair replaced by the second.
func withText(t *testing.T, changes ...string) string {
	t.Helper()
	path := onFixture(t, "    stands:\n", "    texts:\n"+
		"      the word:\n        words: carrie\n        font: go\n"+
		"        size: 4\n        depth: 0.5\n        bevel: 0.04\n"+
		"        east: 1050\n        north: 8975\n        up: 1\n"+
		"        facing: 90\n        spin: 8\n    stands:\n")
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	scene := string(written)
	for i := 0; i+1 < len(changes); i += 2 {
		if !strings.Contains(scene, changes[i]) {
			t.Fatalf("the scene has no %q to change", changes[i])
		}
		scene = strings.Replace(scene, changes[i], changes[i+1], 1)
	}
	if err := os.WriteFile(path, []byte(scene), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// a text on the set stands its letters as things on the fixture's
// ground: six letters more than the set without it, each in the text's
// own material, its name a place a camera points at a metre above the
// ground, and its front turned east, then on round as its spin says.
func TestASetsTextStandsOnItsGround(t *testing.T) {
	given := mustParse(t)
	given.set = onFixture(t, "", "")
	_, plain, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	given = mustParse(t)
	given.set = withText(t, "point_to: the tree", "point_to: the word")
	world, made, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if made.things != plain.things+6 || made.faces <= plain.faces ||
		made.kept-plain.kept >= made.faces-plain.faces {
		t.Errorf("with the word %s; without %s", made, plain)
	}
	var letters int
	for _, each := range world.Things {
		if each.Material == nil || each.Material.Name != "the word" {
			continue
		}
		letters++
		east, north := maths.V(1, 0, 0), maths.V(0, 0, -1)
		if front := facing(each, 0); front.Sub(east).Len() > 1e-9 {
			t.Errorf("a letter faces %v at nought, not east", front)
		}
		later := facing(each, 2*time.Second)
		if later.Sub(north).Len() > 1e-9 {
			t.Errorf("two seconds on it faces %v, not north", later)
		}
	}
	if letters != 6 {
		t.Errorf("%d things in the word's material", letters)
	}
}

// facing is the way a thing's front, its own plus z, faces at a time.
func facing(one *thing.Thing, at time.Duration) maths.Vec {
	return one.Motion.At(at).Orientation.Out(maths.V(0, 0, 1))
}

// the letters stand their up above the ground: set at the stone's own
// place, the word's height in the places hillside writes is the
// stone's and a metre.
func TestATextStandsItsUpAboveTheGround(t *testing.T) {
	given := mustParse(t)
	given.set = withText(t, "north: 8975", "north: 8990")
	given.placed = filepath.Join(t.TempDir(), "places.csv")
	if _, _, _, err := staged(given); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(given.placed)
	if err != nil {
		t.Fatal(err)
	}
	heights := map[string]float64{}
	for _, line := range strings.Split(string(written), "\n")[1:] {
		fields := strings.Split(line, ",")
		if len(fields) == 5 {
			height, _ := strconv.ParseFloat(fields[4], 64)
			heights[fields[0]] = height
		}
	}
	word, stone := heights[`"the word"`], heights[`"the stone"`]
	if math.Abs(word-stone-1) > 0.05 || stone == 0 {
		t.Errorf("the word at %v, the stone at %v", word, stone)
	}
}

// a text whose font is not there, whose letters cannot be made, or that
// stands off the ground is refused by its name; and its faces count
// toward -faces, so a set under the limit without it is over with it.
func TestASetsTextThatCannotBeMadeIsRefused(t *testing.T) {
	given := mustParse(t)
	given.set = withText(t, "font: go", "font: no-such-font.ttf")
	if _, _, _, err := staged(given); err == nil ||
		!strings.Contains(err.Error(), "the word") {
		t.Errorf("a font not there: %v", err)
	}
	given = mustParse(t)
	given.set = withText(t, "bevel: 0.04", "bevel: 0.2")
	if _, _, _, err := staged(given); !errors.Is(err, typography.ErrType) {
		t.Errorf("a bevel past the stroke: %v", err)
	}
	given = mustParse(t)
	given.set = withText(t, "east: 1050\n        north: 8975",
		"east: 9999\n        north: 8975")
	if _, _, _, err := staged(given); err == nil ||
		!strings.Contains(err.Error(), "off the ground") {
		t.Errorf("a text off the ground: %v", err)
	}
	given = mustParse(t)
	given.set = onFixture(t, "", "")
	_, plain, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	given = mustParse(t, "-faces", strconv.Itoa(plain.kept+1))
	given.set = withText(t)
	if _, _, _, err := staged(given); !errors.Is(err, errTooMany) ||
		!strings.Contains(err.Error(), "the word") {
		t.Errorf("the word past -faces: %v", err)
	}
}

// a space in a text stands no letter: "car rie" is the six letters of
// "carrie", and the faces of its two r's, one mesh standing twice, are
// kept once.
func TestASpaceInATextStandsNothing(t *testing.T) {
	given := mustParse(t)
	given.set = withText(t)
	_, whole, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	given = mustParse(t)
	given.set = withText(t, "words: carrie", "words: car rie")
	_, spaced, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if spaced.things != whole.things || spaced.faces != whole.faces ||
		spaced.kept != whole.kept {
		t.Errorf("car rie %s; carrie %s", spaced, whole)
	}
}
