package fixture_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/burning-trousers/fake"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// The exponential law of attenuation: a slab of density k and depth d in front
// of a sky of 1 lets exp(-k d) of it through, and a ray beside the slab
// sees the sky whole.
func TestASlabPassesWhatTheLawOfAttenuationSays(t *testing.T) {
	const density, depth = 0.5, 2.0
	slab := &thing.Volume{Medium: fake.NewSlab(maths.V(-1, -1, -depth/2),
		maths.V(1, 1, depth/2), density)}
	world := &thing.World{Volumes: []*thing.Volume{slab},
		Sky: fake.NewSky(maths.Grey(1))}
	eye := maths.V(0, 0, 5)
	through := maths.Ray{At: eye, Dir: maths.V(0, 0, -1)}
	beside := maths.Ray{At: maths.V(3, 0, 5), Dir: maths.V(0, 0, -1)}
	picture, err := trace.Render(world, maths.Subspace{},
		fake.NewAimed(through, beside), 0, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := math.Exp(-density * depth)
	if got := float64(picture.At(0, 0).R); math.Abs(got/want-1) > 1e-3 {
		t.Errorf("through the slab %.5f is kept; the exponential law of attenuation says %.5f",
			got, want)
	}
	if got := float64(picture.At(1, 0).R); math.Abs(got-1) > 1e-6 {
		t.Errorf("beside the slab the sky is %.5f, not 1", got)
	}
}
