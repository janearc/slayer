package weathering_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/weathering"
)

// the numbers the vacancy tests lay out a street of panes by
const (
	// paneCount is how many panes the street holds, one a dwelling.
	paneCount = 40
	// paneSpacing is how far apart the panes are along the street, which
	// is also a dwelling's width, in metres.
	paneSpacing = 3.0
	// vacantReach is how far from the street's start vacancy reaches.
	vacantReach = 45.0
	// boardsName and brokenName are what the agent names what it adds.
	boardsName = "vacancy boards"
	brokenName = "vacancy broken"
)

// street is a world of panes in a row, one a dwelling, each a quad one
// metre square facing south, in a group of one material.
func street(t *testing.T) *thing.World {
	t.Helper()
	pane, err := mesh.New([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(1, 1, 0), maths.V(0, 1, 0)},
		[][3]int{{0, 1, 2}, {0, 2, 3}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	var members []shape.MeshMember
	for index := range paneCount {
		origin := maths.V(float64(index)*paneSpacing, 1, 0)
		members = append(members, shape.MeshMember{Mesh: pane,
			Space: maths.Subspace{Origin: origin}})
	}
	group, err := shape.NewMeshGroup(members)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: group, Material: &thing.Material{
		Name: "cities clear glazing", Through: 1}})
	return world
}

// emptyStreet is the agent that empties dwellings in the street, at an age.
func emptyStreet(age float64) weathering.Layer {
	empty := weathering.Agent{Kind: weathering.Vacancy, Name: "empty",
		On: []string{"glazing"}, Strength: 0.6, Settle: 20, Cell: paneSpacing,
		Boarded: 0.5, Broken: 0.5, Proud: 0.02, Plank: 0.25, Share: 0.8,
		Slant: 0.04, Reach: vacantReach, Facing: faceSouth,
		Colour: maths.Grey(0.1), Other: maths.Grey(0.01)}
	layer := layerOf(nil, empty)
	layer.Age = age
	return layer
}

// added is how many triangles Apply has put in a world under a name, and
// the farthest along the street any of them reaches.
func added(world *thing.World, name string) (faces int, farthest float64) {
	for _, each := range world.Things {
		made, ok := each.Shape.(*mesh.Mesh)
		if !ok || each.Material == nil || each.Material.Name != name {
			continue
		}
		faces += made.FaceCount()
		for _, point := range made.Points() {
			farthest = max(farthest, point.X)
		}
	}
	return faces, farthest
}

// TestVacancyBoardsAndBreaksWindows says that empty dwellings get boards
// and broken panes, and that none are added when there is no age.
func TestVacancyBoardsAndBreaksWindows(t *testing.T) {
	world := street(t)
	layer := emptyStreet(testAge)
	if err := weathering.Apply(world, &layer); err != nil {
		t.Fatal(err)
	}
	boards, _ := added(world, boardsName)
	shards, _ := added(world, brokenName)
	if boards == 0 || shards == 0 {
		t.Errorf("%d triangles of boards and %d of shards at forty years",
			boards, shards)
	}
	fresh := street(t)
	young := emptyStreet(0)
	if err := weathering.Apply(fresh, &young); err != nil {
		t.Fatal(err)
	}
	if boards, _ := added(fresh, boardsName); boards != 0 {
		t.Errorf("%d triangles of boards on a street with no age", boards)
	}
}

// TestVacancyOnlyRises says that more years leave no fewer dwellings
// empty, and the same layer leaves the same ones.
func TestVacancyOnlyRises(t *testing.T) {
	last := 0
	for _, age := range []float64{0, 2, 5, 10, 20, 40, 80, 160} {
		world := street(t)
		layer := emptyStreet(age)
		if err := weathering.Apply(world, &layer); err != nil {
			t.Fatal(err)
		}
		boards, _ := added(world, boardsName)
		shards, _ := added(world, brokenName)
		if got := boards + shards; got < last {
			t.Fatalf("%d triangles at %v years, and %d before", got, age, last)
		} else {
			last = got
		}
	}
	again := street(t)
	repeat := emptyStreet(160)
	if err := weathering.Apply(again, &repeat); err != nil {
		t.Fatal(err)
	}
	boards, _ := added(again, boardsName)
	shards, _ := added(again, brokenName)
	if boards+shards != last {
		t.Errorf("the same layer left %d, and then %d", last, boards+shards)
	}
}

// TestVacancyStaysWithinItsReach says that no pane past the agent's reach
// is boarded or broken.
func TestVacancyStaysWithinItsReach(t *testing.T) {
	world := street(t)
	layer := emptyStreet(testAge)
	layer.Agents[0].Reach = vacantReach / 3
	if err := weathering.Apply(world, &layer); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{boardsName, brokenName} {
		if _, farthest := added(world, name); farthest > vacantReach/3+1 {
			t.Errorf("%s as far as %v m, past a reach of %v", name, farthest,
				vacantReach/3)
		}
	}
}

// TestApplyNeedsABlock says that a layer with no block to keep what
// stands over a place by is refused.
func TestApplyNeedsABlock(t *testing.T) {
	layer := emptyStreet(testAge)
	layer.Block = 0
	if err := weathering.Apply(street(t), &layer); err == nil {
		t.Errorf("a layer with no block was taken")
	}
}

// TestDimmedGoesOutWhereEmpty says that the glow of a lamp is out in an
// empty dwelling and on in one that is lived in.
func TestDimmedGoesOutWhereEmpty(t *testing.T) {
	layer := emptyStreet(testAge)
	if err := weathering.Apply(street(t), &layer); err != nil {
		t.Fatal(err)
	}
	glow := maths.RGB(1, 0.5, 0.2)
	dimmed := weathering.Dimmed{Light: glow, Layer: &layer}
	out, lit := 0, 0
	for index := range paneCount {
		at := maths.V(float64(index)*paneSpacing+0.5, 1.5, 0)
		if dimmed.At(at) == (maths.Light{}) {
			out++
		} else {
			lit++
		}
	}
	if out == 0 || lit == 0 {
		t.Errorf("%d dwellings dark and %d lit, and want some of each", out,
			lit)
	}
}

// brickLike is a texture that says the left of a wall is mortar, for the
// test of what a weathered surface asks of its base.
type brickLike struct{}

// At is the brick's grey.
func (brickLike) At(maths.Vec) maths.Light { return maths.Grey(0.5) }

// InJoint is every point left of nought along x.
func (brickLike) InJoint(point maths.Vec) bool { return point.X < 0 }

// TestWeatheredPaintsItsBaseWithoutAFace says that a weathered surface
// asked with no way, as At asks, is what it was; asked with one, it is
// marked where an agent lands, and mortar is taken from the base.
func TestWeatheredPaintsItsBaseWithoutAFace(t *testing.T) {
	dust := weathering.Agent{Kind: weathering.Dust, Name: "dust",
		Strength: 0.5, Settle: 5, Tilt: 0.5, Share: 1, Catch: 1,
		Colour: maths.Grey(1)}
	layer := layerOf(weathering.NewShelter(testBlock), dust)
	weathered := weathering.Weathered{Base: brickLike{}, Name: "brick",
		Layer: &layer}
	point := maths.V(1, 0.05, 0)
	if got := weathered.At(point); got != maths.Grey(0.5) {
		t.Errorf("At painted %v, and the base is %v", got, maths.Grey(0.5))
	}
	brick := weathered.Facing(point, point, faceUp)
	mortar := weathered.Facing(maths.V(-1, 0.05, 0), maths.V(-1, 0.05, 0),
		faceUp)
	if !(brick.R > 0.5) || !(mortar.R > brick.R) {
		t.Errorf("dust lay %v on brick and %v on mortar", brick, mortar)
	}
	if wall := weathered.Facing(point, point, faceSouth); wall != maths.Grey(0.5) {
		t.Errorf("dust lay %v on a wall", wall)
	}
}
