package hillside

import (
	"fmt"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// groundSet is a set with a terrain and nothing more, with what more the
// terrain block says.
const groundSet = "scene:\n  name: a ground\n  light: sun\n" +
	"  terrain: {from: a-map, west: 0, north: 10, across: 10, down: 10, " +
	"step: 1%s}\n"

// terrainOf is the terrain of the ground set, with what more it says.
func terrainOf(t *testing.T, more string) set.Terrain {
	t.Helper()
	read, err := set.Read("ground.yaml", []byte(fmt.Sprintf(groundSet, more)))
	if err != nil {
		t.Fatal(err)
	}
	ground, has := read.Ground()
	if !has {
		t.Fatal("the set has no terrain")
	}
	return ground
}

// A ground whose set says its reflectance is made of that.
func TestTheGroundIsWhatTheSetSays(t *testing.T) {
	look := groundLook(terrainOf(t, ", reflectance: [0.04, 0.035, 0.03]"))
	if look.Colour != maths.RGB(0.04, 0.035, 0.03) { // picked: as said
		t.Errorf("the ground is %v", look.Colour)
	}
}

// Anti: a ground whose set says nothing is hillside's basalt, as before.
func TestAnUnsaidGroundIsBasalt(t *testing.T) {
	look := groundLook(terrainOf(t, ""))
	if look.Colour != basalt || look.Name != "basalt" {
		t.Errorf("the ground is %s, %v", look.Name, look.Colour)
	}
}
