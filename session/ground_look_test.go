package session

import (
	"fmt"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
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
	source := fmt.Sprintf(groundSet, more)
	read, err := set.Read("ground.yaml", []byte(source))
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
	look := groundLook(terrainOf(t, ", reflectance: [0.04, 0.035, 0.03]"), 0)
	if look.Colour != maths.RGB(0.04, 0.035, 0.03) { // picked: as said
		t.Errorf("the ground is %v", look.Colour)
	}
}

// Anti: a ground whose set says nothing is hillside's basalt, as before.
func TestAnUnsaidGroundIsBasalt(t *testing.T) {
	look := groundLook(terrainOf(t, ""), 0)
	if look.Colour != basalt || look.Name != "basalt" {
		t.Errorf("the ground is %s, %v", look.Name, look.Colour)
	}
}

// coverSaid is a terrain's cover as a flow: rock everywhere, and sand from
// two metres under the sea to six over it.
const coverSaid = ", cover: {level: 0, layers: [" +
	"{name: rock, reflectance: [0.2, 0.2, 0.2], source: a test's}, " +
	"{name: sand, reflectance: [0.7, 0.6, 0.3], source: a test's, " +
	"height: {from: -2, until: 6, blend: 1}}]}"

// A ground whose set says a cover is painted by it, with the sea where the
// set says it is once the lowest ground is taken off: sand a metre over the
// sea and not fifty under it, whatever the height in the ground's space.
func TestTheGroundIsPaintedByItsCover(t *testing.T) {
	const lowest = -50.0 // picked: ground reaching fifty metres under the sea
	look := groundLook(terrainOf(t, coverSaid), lowest)
	painted, oriented := look.Texture.(texture.Oriented)
	if !oriented {
		t.Fatalf("the ground's texture is %T, not one read by its way",
			look.Texture)
	}
	level := maths.V(0, 1, 0)
	beach := painted.Facing(maths.V(0, -lowest+1, 0), maths.Vec{}, level)
	bed := painted.Facing(maths.V(0, 0, 0), maths.Vec{}, level)
	if beach != maths.RGB(0.7, 0.6, 0.3) || bed != maths.RGB(0.2, 0.2, 0.2) {
		t.Errorf("a metre over the sea is %v and fifty under it %v", beach,
			bed)
	}
}

// Anti: a ground whose set says a cover is not hillside's basalt and does
// not take a reflectance of its own: its colour is its first surface's.
func TestACoveredGroundStartsAsItsFirstSurface(t *testing.T) {
	look := groundLook(terrainOf(t, coverSaid), 0)
	if look.Name == "basalt" || look.Colour != maths.RGB(0.2, 0.2, 0.2) {
		t.Errorf("the ground is %s, %v", look.Name, look.Colour)
	}
}

// A covered ground carries its surfaces' names and where each one's
// reflectance came from, in the material's own words.
func TestACoveredGroundCarriesItsSurfacesAndSources(t *testing.T) {
	look := groundLook(terrainOf(t, coverSaid), 0)
	if look.Name != "ground cover: rock, sand" ||
		look.Source != "rock: a test's; sand: a test's" {
		t.Errorf("the ground is %q from %q", look.Name, look.Source)
	}
}
