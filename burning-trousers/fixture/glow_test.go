package fixture_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/burning-trousers/fake"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// through is the light a ray sees through a slab of density k and depth
// d that glows g, before a sky of s, and beside it.
func through(t *testing.T, density, depth, glow, sky float64) (float64,
	float64) {
	t.Helper()
	slab := &thing.Volume{Medium: fake.NewSlab(maths.V(-1, -1, -depth/2),
		maths.V(1, 1, depth/2), density),
		Material: &thing.Material{Glow: maths.Grey(glow)}}
	world := &thing.World{Volumes: []*thing.Volume{slab},
		Sky: fake.NewSky(maths.Grey(sky))}
	eye := maths.V(0, 0, 5)
	ahead := maths.Ray{At: eye, Dir: maths.V(0, 0, -1)}
	beside := maths.Ray{At: maths.V(3, 0, 5), Dir: maths.V(0, 0, -1)}
	picture, err := trace.Render(world, maths.Subspace{},
		fake.NewAimed(ahead, beside), 0, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	return float64(picture.At(0, 0).R), float64(picture.At(1, 0).R)
}

// Kirchhoff: a slab that absorbs gives off what it absorbs. a uniform
// slab glowing g, of density k and depth d, before a sky of s, is
// g (1 - exp(-k d)) + s exp(-k d). thick, it is its glow and hides the
// sky; thin, it is its glow times k d over the sky; beside it, the sky.
//
// it holds to a frame's float32 while the march crosses the whole slab.
// a march may stop where less than a thousandth of what lies behind
// would get through, as trace.go says of its faint; past that the slab
// is held to within a thousandth of the brighter of glow and sky.
func TestAGlowingSlabIsKirchhoffs(t *testing.T) {
	const glow, sky, float32s, stops = 3.0, 0.25, 1e-6, 1e-3
	for _, slab := range [][2]float64{{0.5, 2}, {25, 2}, {5e-5, 2}} {
		density, depth := slab[0], slab[1]
		kept := math.Exp(-density * depth)
		want := glow*(1-kept) + sky*kept
		got, beside := through(t, density, depth, glow, sky)
		within := float32s * want
		if kept < stops {
			within = stops * math.Max(glow, sky)
		}
		if math.Abs(got-want) > within {
			t.Errorf("density %v: %.12f through the slab, kirchhoff says "+
				"%.12f", density, got, want)
		}
		if math.Abs(beside/sky-1) > 1e-12 {
			t.Errorf("density %v: beside the slab the sky is %v, not %v",
				density, beside, sky)
		}
	}
}
