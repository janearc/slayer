package session

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// townOfTheSteepCoast is the stand of the steep coast set, with its habit
// replaced by another when one is given.
func townOfTheSteepCoast(t *testing.T, habit string) set.Stand {
	t.Helper()
	data, err := os.ReadFile("../sets/terraces-on-a-steep-coast.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if habit != "" {
		text = strings.Replace(text, "        habit: contour\n",
			"        habit: "+habit+"\n", 1)
		text = strings.Replace(text, "        climb:\n"+
			"          rows: 4\n          runs: 2\n"+
			"          lane: {value: 4, unit: m}\n"+
			"          door: downhill\n", "", 1)
	}
	read, err := set.Read("coast.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return read.Stands()[0]
}

func TestEveryRunOfAContourIsMadeAfreshFromItsNumberAmongTheSites(t *testing.T) {
	town := townOfTheSteepCoast(t, "")
	const seed, number = 7, 3
	want := uint64(seed)<<gridPickShift + number
	if got := standPick(town, seed, number); got != want {
		t.Errorf("the run numbered %d is made from %d, not %d", number, got,
			want)
	}
	if standPick(town, seed, 4) == standPick(town, seed, 5) {
		t.Error("two runs of a contour are made from the same number")
	}
}

func TestAHabitThatIsNotAGridOrAContourStillMakesAFewAndStandsThemAgain(t *testing.T) {
	scatter := townOfTheSteepCoast(t, "lattice")
	const seed = 7
	same := standPick(scatter, seed, 0) == standPick(scatter, seed, standVariety)
	if !same {
		t.Error("a lattice made its thing afresh at each site")
	}
	want := uint64(seed)*standVariety + 2
	if got := standPick(scatter, seed, 2); got != want {
		t.Errorf("a lattice's third site is made from %d, not %d", got, want)
	}
}
