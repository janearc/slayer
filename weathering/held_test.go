package weathering

import (
	"math"
	"sync"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// lookers is how many goroutines look in one shelter at once in the test
// of the first look: more than one, so the first is raced. picked.
const lookers = 16

// roofPlate is a flat square of smallSide at height, as a shape.Mesh of
// two triangles in its own space.
func roofPlate(height float64) *shape.Mesh {
	return &shape.Mesh{Vertices: []maths.Vec{maths.V(0, height, 0),
		maths.V(smallSide, height, 0), maths.V(smallSide, height, smallSide),
		maths.V(0, height, smallSide)},
		Faces: [][3]int{{0, 1, 2}, {0, 2, 3}}}
}

// inPlace is the way out of a shape's own space that leaves it where it
// is.
func inPlace(point maths.Vec) maths.Vec { return point }

// underRoof is a point under the middle of a roof plate, on the ground.
var underRoof = maths.V(smallSide/2, 0, smallSide/2)

// TestNothingIsLaidBeforeTheFirstLook says that a held surface costs no
// triangle until the shelter is looked in, and that the first look finds
// it as a shelter built at once would.
func TestNothingIsLaidBeforeTheFirstLook(t *testing.T) {
	shelter := NewShelter(shelterBlock)
	shelter.hold(roofPlate(plateHeight), inPlace, "roof")
	if len(shelter.all) != 0 {
		t.Fatalf("%d triangles laid before a look, want none",
			len(shelter.all))
	}
	rise, name, found := shelter.Above(underRoof, 2*plateHeight)
	if !found || name != "roof" || math.Abs(rise-plateHeight) > 1e-9 {
		t.Errorf("%v %q %v looking up, want %v to a roof", rise, name,
			found, plateHeight)
	}
	if len(shelter.all) != facesOfPlate || shelter.held != nil {
		t.Errorf("%d triangles and %d held after a look, want %d and none",
			len(shelter.all), len(shelter.held), facesOfPlate)
	}
}

// TestManyFirstLooksLayTheShelterOnce says that goroutines looking in a
// shelter at once all find the same roof, and that it is laid once, not
// once a goroutine.
func TestManyFirstLooksLayTheShelterOnce(t *testing.T) {
	shelter := NewShelter(shelterBlock)
	shelter.hold(roofPlate(plateHeight), inPlace, "roof")
	var looking sync.WaitGroup
	names := make([]string, lookers)
	for index := range lookers {
		looking.Go(func() {
			_, names[index], _ = shelter.Above(underRoof, 2*plateHeight)
		})
	}
	looking.Wait()
	for index, name := range names {
		if name != "roof" {
			t.Errorf("look %d found %q, want the roof", index, name)
		}
	}
	if len(shelter.all) != facesOfPlate {
		t.Errorf("%d triangles laid, want %d", len(shelter.all),
			facesOfPlate)
	}
}

// roofedWorld is a world of one roof plate, which holds still, and a
// layer over nothing but its shelter.
func roofedWorld() (*thing.World, *Layer) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: roofPlate(plateHeight),
		Material: &thing.Material{Name: "roof"}})
	return world, &Layer{Setting: Setting{Block: shelterBlock}}
}

// TestARoofInTheWorldAtApplyIsFound says that a roof standing when the
// layer is applied is in the shelter, though it is laid only at the look.
func TestARoofInTheWorldAtApplyIsFound(t *testing.T) {
	world, layer := roofedWorld()
	if err := Apply(world, layer); err != nil {
		t.Fatal(err)
	}
	if _, name, found := layer.Over.Above(underRoof,
		2*plateHeight); !found || name != "roof" {
		t.Errorf("found %q %v, want the roof", name, found)
	}
}

// TestARoofAddedAfterApplyIsNotFound is the anti-test of the one above:
// a roof put in the world after the layer was applied, as vacancy's
// boards are, is not in the shelter, though nothing was laid until the
// look. The shelter is the world as Apply found it.
func TestARoofAddedAfterApplyIsNotFound(t *testing.T) {
	world, layer := roofedWorld()
	world.Things = nil
	if err := Apply(world, layer); err != nil {
		t.Fatal(err)
	}
	world.Add(thing.Thing{Shape: roofPlate(plateHeight),
		Material: &thing.Material{Name: "late roof"}})
	if _, name, found := layer.Over.Above(underRoof,
		2*plateHeight); found {
		t.Errorf("found %q, a roof added after Apply", name)
	}
}
