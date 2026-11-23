package set

import (
	"os"
	"strings"
	"testing"
)

// weatheredSet is the set file the weathering tests read, which says every
// kind of agent the portofino study uses.
const weatheredSet = "../sets/weathering-portofino-old.yaml"

// readWeathered is the weathering study's text, and a copy of it read.
func readWeathered(t *testing.T) (string, Weathering) {
	t.Helper()
	raw, err := os.ReadFile(weatheredSet)
	if err != nil {
		t.Fatal(err)
	}
	read, err := Read(weatheredSet, raw)
	if err != nil {
		t.Fatal(err)
	}
	world, has := read.World()
	if !has {
		t.Fatal("the weathering study has no world")
	}
	layer, has := world.Weathering()
	if !has {
		t.Fatal("the weathering study has no weathering")
	}
	return string(raw), layer
}

// TestAWeatheringLayerIsRead says that a layer's numbers, its wind, and
// each agent's kind, words, colours, place and numbers are read as said.
func TestAWeatheringLayerIsRead(t *testing.T) {
	_, layer := readWeathered(t)
	if layer.Age() != 40 || layer.Seed() != 11 || layer.Block() != 2 {
		t.Errorf("read age %v, seed %v, block %v", layer.Age(), layer.Seed(),
			layer.Block())
	}
	if wind := layer.Wind(); wind.X != 1 || wind.Y != 0 || wind.Z != 0 {
		t.Errorf("read a wind of %v, and it blows east", wind)
	}
	kinds := map[string]bool{}
	for _, agent := range layer.Agents() {
		kinds[agent.Kind()] = true
	}
	for _, kind := range []string{"rain", "soot", "dust", "moss", "ivy",
		"graffiti", "spall", "vacancy"} {
		if !kinds[kind] {
			t.Errorf("the study says no %s agent", kind)
		}
	}
	for _, agent := range layer.Agents() {
		if agent.Name() != "empty" {
			continue
		}
		if agent.On()[0] != "glazing" || agent.Number("cell") != 5.7 ||
			agent.Facing().Z != 1 {
			t.Errorf("read the empty homes wrongly: %v, cell %v, facing %v",
				agent.On(), agent.Number("cell"), agent.Facing())
		}
		if _, said := agent.Colour(); !said {
			t.Errorf("the empty homes' boards have no colour")
		}
	}
}

// TestAWeatheringLayerRefusesWhatItDoesNotKnow says that an unknown kind,
// an unknown number, a layer with no block and a block of nought are each
// refused.
func TestAWeatheringLayerRefusesWhatItDoesNotKnow(t *testing.T) {
	text, _ := readWeathered(t)
	cases := map[string]string{
		"an unknown kind":   strings.Replace(text, "kind: moss", "kind: lichen", 1),
		"an unknown number": strings.Replace(text, "strength: 0.9", "vigour: 0.9", 1),
		"no block":          strings.Replace(text, "      block: 2\n", "", 1),
		"a block of nought": strings.Replace(text, "block: 2", "block: 0", 1),
		"a negative share":  strings.Replace(text, "share: 0.65", "share: -0.65", 1),
	}
	for name, changed := range cases {
		if changed == text {
			t.Fatalf("%s: the case changed nothing", name)
		}
		if _, err := Read(weatheredSet, []byte(changed)); err == nil {
			t.Errorf("%s was read", name)
		}
	}
}
