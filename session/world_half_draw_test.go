//go:build draw

package session

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// the fixture's facts, each true by the fixture's definition (see
// terrainFixture): its first cell's north west corner, how wide its cells
// are, and how its heights climb: a cell's height is its column plus a
// thousand times its row, in eighths of a metre.
const (
	fixtureEast  = 1000.5  // defined: the east edge of the first cell
	fixtureNorth = 9000.25 // defined: the north edge of the first cell
	cellMetres   = 2.0     // defined: a cell is two metres across
	rowEighths   = 1000.0  // defined: each row down adds 1000 eighths
	eighth       = 8.0     // defined: heights are in eighths of a metre
)

// fixtureHeight is the fixture's ground at an easting and a northing, in
// metres: the height of the cell that holds the point.
func fixtureHeight(east, north float64) float64 {
	column := math.Floor((east - fixtureEast) / cellMetres)
	row := math.Floor((fixtureNorth - north) / cellMetres)
	return (column + rowEighths*row) / eighth
}

// a set whose things half stands a tree on that ground, and whose world
// half puts a box, one metre on a side, at the tree's foot and a bulb ten
// metres above it, both said as a map says a place: east, north, up.
const boxAndTree = `scene:
  name: a box at a tree's foot
  light: moon
  timeofday: 2200
  terrain:
    from: fixture
    file: fixture-plain.tif
    west: 1001.5
    north: 8995.25
    across: 150
    down: 100
    step: 2
  film:
    width: 24
    height: 16
    rays: 1
    exposure: 100000
  cameras:
    stone:
      orientation:
        point_to: the tree
        alpha: 0
      location:
        loc_relative:
          object: the stone
          standoff_x: 0
          standoff_y: 0
          standoff_z: 2
  set:
    things:
      the tree:
        kind: spruce
        seed: 3
        east: 1051.5
        north: 8955.25
      the stone:
        kind: cobble
        seed: 4
        east: 1050
        north: 8990
  world:
    materials:
      crate:
        colour: {srgb_linear: [0.5, 0.25, 0.125]}
        source: eye
    boxes:
      crate:
        material: crate
        middle: [1051.5, 8955.25, 2753.125]
        size: [1, 1, 1]
        source: eye
    lamps:
      bulb:
        at: [1051.5, 8955.25, 2763.125]
        temperature: 2700
        strength: 1
        radius: 0.05
        source: eye
`

// boxAndTreeAt is the box and tree set written onto the fixture's shelf,
// and its path.
func boxAndTreeAt(t *testing.T) string {
	t.Helper()
	path := onFixture(t, "", "")
	put(t, path, boxAndTree)
	return path
}

// boxAndTreeWith is the same set with a piece of its text changed.
func boxAndTreeWith(t *testing.T, was, now string) string {
	t.Helper()
	path := onFixture(t, "", "")
	put(t, path, replaced(t, boxAndTree, was, now))
	return path
}

// worldHalfless is the box and tree set with no world half: the set as
// far as its things.
func worldHalfless(t *testing.T) string {
	t.Helper()
	path := onFixture(t, "", "")
	whole, cut, _ := strings.Cut(boxAndTree, "  world:")
	if cut == "" {
		t.Fatal("the set has no world half to cut")
	}
	put(t, path, whole)
	return path
}

// treeAt is where the set stands a thing of a name, east and north.
func treeAt(t *testing.T, read set.Set, name string) (east, north float64) {
	t.Helper()
	for _, each := range read.Things() {
		if each.Name() == name {
			return each.At()
		}
	}
	t.Fatalf("the set has no %s", name)
	return east, north
}

// theFoot is where the tree's foot stands in the staged world, worked
// from the set's numbers: the window's west and north, the fixture's own
// height at the tree's place less its lowest, which is the window's first
// point, and so how far east, up and south of them it is.
func theFoot(t *testing.T, read set.Set) maths.Vec {
	t.Helper()
	ground, _ := read.Ground()
	west, northEdge, _, _, _ := ground.Window()
	lowest := fixtureHeight(west, northEdge)
	east, north := treeAt(t, read, "the tree")
	up := fixtureHeight(east, north) - lowest
	return maths.V(east-west, up, northEdge-north)
}

// the world half's box stands at the foot of the things half's tree: the
// box's middle is the tree's place, the window's east and north measured
// from its west and north and the height from its lowest point, as
// arithmetic from the set's numbers gives it; and the bulb is above the
// box, lighting from there, counted among the lamps.
func TestABoxFromTheWorldHalfStandsAtTheFootOfATree(t *testing.T) {
	given := mustParse(t)
	given.set = boxAndTreeAt(t)
	read := readSet(t, given.set)
	said, _ := read.World()
	world, made, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	foot := theFoot(t, read)
	crates, trees := nothing, nothing
	for _, each := range world.Things {
		switch {
		case each.Material != nil && each.Material.Name == "crate":
			crates++
			if each.At(nothing).Origin != foot {
				t.Errorf("the crate is at %v, not at %v",
					each.At(nothing).Origin, foot)
			}
		case each.At(nothing).Origin == foot:
			trees++
		}
	}
	if crates != len(said.Boxes()) || trees == nothing {
		t.Errorf("%d crates, %d things of the tree at its foot %v",
			crates, trees, foot)
	}
	for _, each := range said.Lamps() {
		bulb := lampAbove(t, read, foot, each.At())
		lit := nothing
		for _, lamp := range world.Lamps {
			if point, is := lamp.(emit.Lit); is && point.At == bulb {
				lit++
			}
		}
		if lit != len(said.Lamps()) || made.lamps != len(world.Lamps) {
			t.Errorf("%d bulbs at %v; %d lamps tallied of %d", lit, bulb,
				made.lamps, len(world.Lamps))
		}
	}
}

// lampAbove is where a bulb the world half puts at a place stands in the
// staged world: as far above the foot as the bulb is above the box's
// middle in the set, which stands on the tree's foot, and over the foot.
func lampAbove(t *testing.T, read set.Set, foot, bulb maths.Vec) maths.Vec {
	t.Helper()
	said, _ := read.World()
	for _, each := range said.Boxes() {
		return maths.V(foot.X, foot.Y+bulb.Y-each.Middle().Y, foot.Z)
	}
	t.Fatal("the set has no box")
	return foot
}

// a set with no world half stages as it did: no crate, and the lamps its
// light gives and nothing more.
func TestASetWithNoWorldHalfAddsNothing(t *testing.T) {
	given := mustParse(t)
	given.set = worldHalfless(t)
	world, _, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	for _, each := range world.Things {
		if each.Material != nil && each.Material.Name == "crate" {
			t.Error("a crate stands in a set that says none")
		}
	}
}

// a sky the world half says does not replace the sky the set's own light
// hung: the night's gradient stays.
func TestTheWorldHalfsSkyDoesNotReplaceTheSetsLight(t *testing.T) {
	given := mustParse(t)
	given.set = boxAndTreeWith(t, "    lamps:\n", "    sky:\n"+
		"      horizon: {srgb_linear: [0.9, 0.9, 0.9]}\n"+
		"      zenith: {srgb_linear: [0.9, 0.9, 0.9]}\n"+
		"      ground: {srgb_linear: [0.9, 0.9, 0.9]}\n"+
		"      source: eye\n    lamps:\n")
	world, _, given, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(world.Sky, skyFor(given)) {
		t.Errorf("the sky is %+v, not the night's", world.Sky)
	}
}
