//go:build draw

package gpu

import (
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// tubeSeen is a world of the tubes, lit by one distant lamp, and an eye
// six metres off that looks at the middle of the first.
func tubeSeen(t *testing.T, tubes ...shape.Tube) (*thing.World, maths.Subspace) {
	t.Helper()
	world := tubeWorld(tubes...)
	for index := range world.Things {
		world.Things[index].Material = &thing.Material{
			Colour: maths.RGB(0.6, 0.5, 0.3)}
	}
	lamp, err := emit.NewDistant(maths.V(-1, 1, 3), maths.Grey(1))
	if err != nil {
		t.Fatal(err)
	}
	world.Lamp(lamp)
	from := maths.V(0, 1, 6)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 0, 0))}
}

// pixelsCovered is how many pixels of the gpu's picture of a world of the
// tubes the world covers.
func pixelsCovered(t *testing.T, tubes ...shape.Tube) int {
	t.Helper()
	world, eye := tubeSeen(t, tubes...)
	covered := 0
	for _, pixel := range drawn(t, world, eye).Pix {
		if pixel.Cover > 0 {
			covered++
		}
	}
	return covered
}

// tubeAgreement is the mean difference in light, as a share of the mean
// light, that the gpu's picture of a tube may have from the cpu's. A tube's
// wall is curved and is hit in float32 on the gpu and float64 on the cpu,
// so the two agree as other curved shapes' pictures do, to a part in five
// hundred, and not to the part in a hundred thousand of flat faces.
const tubeAgreement = 2e-3

// A tube is drawn on the gpu as on the cpu: it reaches the picture, covers
// the same pixels, and is as bright to within tubeAgreement.
func TestATubeIsDrawnLikeTheCPUDrawsIt(t *testing.T) {
	ring := shape.Tube{Radius: .5, Thick: .1, Height: 2}
	if covered := pixelsCovered(t, ring); covered < 100 {
		t.Fatalf("a tube covered %d pixels, want at least 100", covered)
	}
	world, eye := tubeSeen(t, ring)
	apart := drawnBoth(t, world, eye)
	if apart.cover != 0 || apart.mean > tubeAgreement {
		t.Fatalf("tube CPU/GPU difference: %+v", apart)
	}
}

// Anti-test: the count and the comparison above see a tube where there is
// one: a world with no tube covers no pixel, and its picture differs in
// cover from the picture of a world with one.
func TestAWorldWithNoTubeCoversNoPixel(t *testing.T) {
	if covered := pixelsCovered(t); covered != 0 {
		t.Errorf("a world with no tube covered %d pixels", covered)
	}
	empty, eye := tubeSeen(t)
	ring, _ := tubeSeen(t, shape.Tube{Radius: .5, Thick: .1, Height: 2})
	apart := compare(drawn(t, empty, eye), drawn(t, ring, eye), 96, 64)
	if apart.cover == 0 {
		t.Error("a world with a tube and one without covered the same")
	}
}
