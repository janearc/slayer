package set_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// fizzSet is the fizz's set file as it ships, changed by each pair of
// was and now.
func fizzSet(t *testing.T, changes ...string) string {
	t.Helper()
	source, err := os.ReadFile("../sets/fizz.yaml")
	if err != nil {
		t.Fatal(err)
	}
	written := string(source)
	for i := 0; i+1 < len(changes); i += 2 {
		if !strings.Contains(written, changes[i]) {
			t.Fatalf("%q is not in the fizz", changes[i])
		}
		written = strings.Replace(written, changes[i], changes[i+1], 1)
	}
	return written
}

// fizzRead is the fizz's world, read as written or changed.
func fizzRead(t *testing.T, changes ...string) set.World {
	t.Helper()
	made, err := set.Read("fizz.yaml", []byte(fizzSet(t, changes...)))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := made.World()
	return world
}

// a fizz is its colour, its foot, its column, its rise and its churn.
func TestAFizzIsWhatItSays(t *testing.T) {
	fizz := fizzRead(t, "foot: [0, 0, 0]", "foot: [1, 2, 3]").Fizzes()[0]
	radius, height, thick := fizz.Column()
	seed, churn := fizz.Churned()
	named, _ := fizz.Colour().Swatch()
	if fizz.Name() != "column" || named != "paint" ||
		fizz.Foot() != maths.V(1, 3, -2) || radius != 0.4 || height != 2 ||
		thick != 4 || fizz.Rise() != 0.3 || seed != 43 || churn != 0.75 {
		t.Errorf("the fizz is %+v", fizz)
	}
}

// a fizz is not a wisp: a world that says one fizz and no wisp holds one
// fizz and no wisp, and its fizzes come in the order of its keys.
func TestAFizzIsNotAWisp(t *testing.T) {
	world := fizzRead(t)
	if len(world.Fizzes()) != 1 || len(world.Wisps()) != 0 {
		t.Errorf("%d fizzes and %d wisps", len(world.Fizzes()),
			len(world.Wisps()))
	}
	order := strings.Join(world.Order(), " ")
	if order != "materials boxes fizzes suns" {
		t.Errorf("the order is %q", order)
	}
}

// what a world hands back of its fizzes is its own copy.
func TestAWorldsFizzesAreACopy(t *testing.T) {
	world := fizzRead(t)
	world.Fizzes()[0] = set.Fizz{}
	if world.Fizzes()[0].Name() != "column" {
		t.Error("changing the list handed back changed the world's")
	}
}

// a fizz may stand still: a churn of nought and a rise of nought are
// numbers a fizz may say, and are read as said.
func TestAFizzMayStandStill(t *testing.T) {
	fizz := fizzRead(t, "rise: 0.3", "rise: 0", "churn: 0.75",
		"churn: 0").Fizzes()[0]
	if _, churn := fizz.Churned(); fizz.Rise() != 0 || churn != 0 {
		t.Errorf("the still fizz is %+v", fizz)
	}
}

// a fizz's mistakes are refused where they are written.
func TestAFizzsMistakesAreRefused(t *testing.T) {
	for _, mistake := range []struct{ was, now, said string }{
		{"        rise: 0.3\n", "",
			"a fizz says [colour foot radius height thick rise churn " +
				"seed source], all of them, and this has no rise"},
		{"        seed: 43\n        source: eye\n", "        seed: 43\n",
			"all of them, and this has no source"},
		{"radius: 0.4", "radius: 0", "radius takes a number from 5e-324"},
		{"height: 2", "height: 0", "height takes a number from 5e-324"},
		{"thick: 4", "thick: 0", "thick takes a number from 5e-324"},
		{"rise: 0.3", "rise: -0.3", "rise takes a number from 0"},
		{"churn: 0.75", "churn: 1.5", "churn takes a number from 0 to 1"},
		{"churn: 0.75", "churn: -0.1", "churn takes a number from 0 to 1"},
		{"seed: 43", "seed: 43.5", "seed takes a whole number"},
		{"seed: 43", "seed: 43\n        scale: 2",
			"scale is not a key here"},
		{"    fizzes:", "    fizz:", "fizz is not a key here"},
	} {
		_, err := set.Read("fizz.yaml", []byte(fizzSet(t, mistake.was,
			mistake.now)))
		if err == nil || !strings.Contains(err.Error(), mistake.said) {
			t.Errorf("%q: refused with %v, wanted %q", mistake.now, err,
				mistake.said)
		}
	}
}

// what a fizz may say at its edges is taken: the smallest column and
// the greatest churn are not mistakes.
func TestAFizzsEdgesAreTaken(t *testing.T) {
	for _, edge := range []struct{ was, now string }{
		{"churn: 0.75", "churn: 1"},
		{"radius: 0.4", "radius: 5e-324"},
		{"seed: 43", "seed: 0"},
	} {
		if _, err := set.Read("fizz.yaml", []byte(fizzSet(t, edge.was,
			edge.now))); err != nil {
			t.Errorf("%q: refused with %v", edge.now, err)
		}
	}
}
