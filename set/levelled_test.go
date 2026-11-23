package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestAThingMayBeLevelledAndTheSceneMayNot(t *testing.T) {
	levelled := strings.Replace(towerSet(""), "complexity: hull",
		"complexity: levelled", 1)
	made, err := set.Read("levelled.yaml", []byte(levelled))
	if err != nil {
		t.Fatal(err)
	}
	if things := made.Things(); len(things) != 1 ||
		things[0].Complexity() != "levelled" {
		t.Errorf("the tower is %+v", things)
	}
	scene := strings.Replace(towerSet(""), "  light: none",
		"  light: none\n  complexity: levelled", 1)
	if _, err := set.Read("scene.yaml", []byte(scene)); err == nil {
		t.Error("a scene said to be levelled was read")
	}
}
