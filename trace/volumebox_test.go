package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// puffAside is a world of one puff of radius two, moved off the origin
// and turned, placed as a picture would place it.
func puffAside(t *testing.T) placedVolume {
	t.Helper()
	world := thing.NewWorld()
	puffAt := thing.Still(maths.Subspace{Origin: maths.V(10, 0, 0),
		Orientation: maths.About(maths.YAxis, 0.7)})
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 2, Thick: 3},
		Motion: puffAt})
	volumes := placeVolumes(world, 0)
	if len(volumes) != 1 || !volumes[0].walled {
		t.Fatalf("%d volumes placed, want one with a world box",
			len(volumes))
	}
	return volumes[0]
}

// TestARayThatMissesAVolumesBoxMeetsNothing says that a ray that passes
// the volume's world box by is told so, takes nothing from the volume,
// and crosses none of it; and that a ray through the volume is not.
func TestARayThatMissesAVolumesBoxMeetsNothing(t *testing.T) {
	volume := puffAside(t)
	by := maths.Ray{At: maths.V(0, 5, 0), Dir: maths.XAxis}
	if volume.meets(by, 100) {
		t.Error("a ray 5 m over a puff of radius 2 meets its box")
	}
	if depth := volume.opticalDepth(by.At, by.Dir, 100); depth != 0 {
		t.Errorf("a ray past the puff takes %v from it", depth)
	}
	if across := crossings([]placedVolume{volume}, by, 100); len(across) != 0 {
		t.Errorf("a ray past the puff crosses it %d times", len(across))
	}
	through := maths.Ray{At: maths.V(0, 0, 0), Dir: maths.XAxis}
	if !volume.meets(through, 100) {
		t.Error("a ray through the puff misses its box")
	}
	if depth := volume.opticalDepth(through.At, through.Dir, 100); !(depth > 0) {
		t.Errorf("a ray through the puff takes %v from it", depth)
	}
}

// TestTheWorldBoxHoldsTheVolume is the anti-test of the box: every
// corner of the volume's own box, taken out to the world, is in the
// world box, so a ray that meets the volume never misses the box.
func TestTheWorldBoxHoldsTheVolume(t *testing.T) {
	volume := puffAside(t)
	for corner := range 8 {
		out := volume.at.Out(cornerOf(volume.bounds, corner))
		inside := out.X >= volume.world.Min.X && out.X <= volume.world.Max.X &&
			out.Y >= volume.world.Min.Y && out.Y <= volume.world.Max.Y &&
			out.Z >= volume.world.Min.Z && out.Z <= volume.world.Max.Z
		if !inside {
			t.Errorf("corner %d at %v is outside the world box %v", corner,
				out, volume.world)
		}
	}
	if math.IsInf(volume.world.Max.X, 0) {
		t.Error("a puff's world box has no edge")
	}
}
