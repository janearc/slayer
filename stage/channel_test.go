package stage_test

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

// aCanal is a small world with one channel along a line: 12 m between
// walls half a metre thick, the bed 3 m under ground at 0.5 m, the coping
// 0.2 m thick, the ground 20 m each side. The line is written where LINE
// stands.
const aCanal = `scene:
  name: a canal
  film:
    width: 64
    height: 36
    rays: 1
    wide: 60
    exposure: 1
    print:
      knee: 0.7
      top: 8
      source: eye
  world:
    unit: 1
    materials:
      brick:
        colour: {srgb_linear: [0.3, 0.1, 0.05]}
        source: eye
      stone:
        colour: {srgb_linear: [0.3, 0.3, 0.3]}
        source: eye
      paving:
        colour: {srgb_linear: [0.1, 0.1, 0.1]}
        source: eye
      silt:
        colour: {srgb_linear: [0.02, 0.02, 0.01]}
        source: eye
    channels:
      canal:
        line: LINE
        width: {value: 12, unit: m}
        depth: {value: 3, unit: m}
        top: {value: 0.5, unit: m}
        reach: {value: 20, unit: m}
        ground: paving
        bed: silt
        wall: {material: brick, thickness: {value: 0.5, unit: m}}
        coping:
          material: stone
          height: {value: 0.2, unit: m}
          overhang: {value: 0.1, unit: m}
        source: eye
`

// canalWorld is the canal along a line, built, or the refusal it meets.
func canalWorld(t *testing.T, line string) (*thing.World, error) {
	t.Helper()
	written := strings.Replace(aCanal, "LINE", line, 1)
	made, err := set.Read("canal.yaml", []byte(written))
	if err != nil {
		t.Fatal(err)
	}
	said, _ := made.World()
	return stage.World(said)
}

// shapeOf is the mesh a thing of a channel is.
func shapeOf(t *testing.T, world *thing.World, index int) *mesh.Mesh {
	t.Helper()
	shape, is := world.Things[index].Shape.(*mesh.Mesh)
	if !is {
		t.Fatalf("thing %d is %T, not a mesh", index, world.Things[index].Shape)
	}
	return shape
}

// a channel along a straight line is eleven solids, each closed, each in
// the material it was given: the bed, two walls, two copings and two
// stretches of ground, then a wall and a coping at each end.
func TestAStraightChannelIsElevenClosedSolids(t *testing.T) {
	world, err := canalWorld(t, "[[0, -100], [0, 100]]")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"silt", "brick", "brick", "stone", "stone", "paving",
		"paving", "brick", "stone", "brick", "stone"}
	if len(world.Things) != len(want) {
		t.Fatalf("the channel has %d solids, not %d", len(world.Things),
			len(want))
	}
	for index, name := range want {
		if got := world.Things[index].Material.Name; got != name {
			t.Errorf("solid %d wears %q, not %q", index, got, name)
		}
		if !shapeOf(t, world, index).Closed() {
			t.Errorf("solid %d (%s) is not closed", index, name)
		}
	}
}

// the bed lies where the channel says: between its walls, its upper face
// at the top less the depth, and a floor of one wall's thickness under it.
// East is x, up is y, and north is the world's minus z.
func TestTheBedLiesBetweenTheWallsAtItsDepth(t *testing.T) {
	world, err := canalWorld(t, "[[0, -100], [0, 100]]")
	if err != nil {
		t.Fatal(err)
	}
	bounds := shapeOf(t, world, 0).Bounds()
	got := []float64{bounds.Min.X, bounds.Max.X, bounds.Min.Y, bounds.Max.Y,
		bounds.Min.Z, bounds.Max.Z}
	want := []float64{-6, 6, -3, -2.5, -100, 100}
	for index := range want {
		if math.Abs(got[index]-want[index]) > 1e-9 {
			t.Errorf("the bed's bounds are %v, not %v", got, want)
			break
		}
	}
}

// the walls stand either side of the water, the left of the line's way
// to its west: one wall from 6 m to 6.5 m on each side, to the coping.
func TestTheWallsStandEitherSideOfTheWater(t *testing.T) {
	world, err := canalWorld(t, "[[0, -100], [0, 100]]")
	if err != nil {
		t.Fatal(err)
	}
	left, right := shapeOf(t, world, 1).Bounds(), shapeOf(t, world, 2).Bounds()
	if math.Abs(left.Min.X+6.5) > 1e-9 || math.Abs(left.Max.X+6) > 1e-9 ||
		math.Abs(right.Min.X-6) > 1e-9 || math.Abs(right.Max.X-6.5) > 1e-9 ||
		math.Abs(left.Max.Y-0.3) > 1e-9 {
		t.Errorf("the walls are %v and %v", left, right)
	}
}

// a channel along a line that bends is as solid as a straight one, each
// piece closed, the joint mitred.
func TestABentChannelIsStillClosed(t *testing.T) {
	world, err := canalWorld(t, "[[0, -100], [0, 0], [30, 100]]")
	if err != nil {
		t.Fatal(err)
	}
	for index := range world.Things {
		if !shapeOf(t, world, index).Closed() {
			t.Errorf("solid %d of the bent channel is not closed", index)
		}
	}
}

// a line that turns back on itself has no mitre, and is refused for it.
func TestALineThatTurnsBackIsRefused(t *testing.T) {
	_, err := canalWorld(t, "[[0, -50], [0, 50], [1, -50]]")
	if err == nil || !strings.Contains(err.Error(), "right angle") {
		t.Errorf("a hairpin built as %v", err)
	}
}

// a bend too sharp for the reach beside it is refused, since the ground
// would fold over itself on the inside of the bend.
func TestABendTooSharpForItsReachIsRefused(t *testing.T) {
	_, err := canalWorld(t, "[[0, 0], [5, 0], [8, 3]]")
	if err == nil || !strings.Contains(err.Error(), "too sharply") {
		t.Errorf("a short sharp bend built as %v", err)
	}
}
