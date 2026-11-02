package trace

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// cloudAtTheMiddle is a world with one thick puff of cloud, three across
// in radius, at the world's middle, placed as the tracer places it
func cloudAtTheMiddle() []placedVolume {
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 3, Thick: 2},
		Material: &thing.Material{Name: "cloud",
			Colour: maths.RGB(0.5, 0.5, 0.5)}})
	return placeVolumes(world, 0)
}

// A lamp five short of the middle, lighting a point ten short, has
// nothing between them; the cloud is behind the lamp, and must not take
// any of the light. the march in the first trace ran on past the lamp and
// counted the cloud, which is how the orbital's lantern lit none of its
// nebula.
func TestADarkCloudBehindALampDoesNotDimWhatItLights(t *testing.T) {
	volumes := cloudAtTheMiddle()
	point, toward := maths.V(0, 0, -10), maths.V(0, 0, 1)
	got := paintedClear(volumes, point, toward, 5).R
	if !(got >= 0.999) {
		t.Fatalf("a cloud behind the lamp let through %v, not all",
			got)
	}
}

// The same cloud between the point and a lamp beyond it must still dim
// the light, so a march that counts nothing at all cannot pass the test
// above.
func TestACloudBetweenALampAndWhatItLightsStillDimsIt(t *testing.T) {
	volumes := cloudAtTheMiddle()
	point, toward := maths.V(0, 0, -10), maths.V(0, 0, 1)
	got := paintedClear(volumes, point, toward, 20).R
	if !(got <= 0.5) {
		t.Fatalf("a thick cloud between the lamp and the point let "+
			"through %v of the light", got)
	}
}
