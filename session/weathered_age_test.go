package session

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// ageSets are the three superblock sets and the age the weathering of each
// should come to, which follows the accretion the block is made at.
var ageSets = map[string]float64{
	"../sets/superblock-planned.yaml":  5,
	"../sets/superblock-half.yaml":     25,
	"../sets/superblock-accreted.yaml": 45,
}

// readAge is a set read from a file with one replacement made, and its
// weathering.
func readAge(t *testing.T, path, was, now string) (set.Set, set.Weathering) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(raw), was, now,
		1)
	read, err := set.Read(path, []byte(edited))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	layer, _ := world.Weathering()
	return read, layer
}

func TestTheWeatheringsAgeFollowsTheBlocksAccretion(t *testing.T) {
	making := &staging{given: asked{set: "age", seed: 3}}
	for path, want := range ageSets {
		read, layer := readAge(t, path, "from: block", "from: block")
		got, err := making.weatheredAge(read, layer)
		if err != nil || got != want {
			t.Errorf("%s: the age is %v, %v, want %v", path, got, err, want)
		}
	}
}

func TestAnAgeThatFollowsAThingTheSetDoesNotHaveIsRefused(t *testing.T) {
	making := &staging{given: asked{set: "age", seed: 3}}
	read, layer := readAge(t, "../sets/superblock-accreted.yaml",
		"from: block", "from: nothing")
	if got, err := making.weatheredAge(read, layer); err == nil {
		t.Errorf("an age of %v was found for a thing that is not there", got)
	}
}
