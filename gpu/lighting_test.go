//go:build darwin

package gpu

import (
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"testing"
	"time"
)

// A cache must retain the moving source's response and account for its bake.
// The direct renderer is the reference; this is an approximation, with an
// authored two-percent summed-radiance error budget for this smoke fixture.
func TestCachedSmokeTracksDirectFireLighting(t *testing.T) {
	world, eye := fire(t)
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	world.Volumes = world.Volumes[1:]
	world.Things[0].Material.Colour = maths.Light{}
	var previous []float32
	for _, step := range []struct {
		at         time.Duration
		lamp       float64
		moveVolume bool
	}{{0, 0, false}, {0, .37, false}, {370 * time.Millisecond, .37, true}} {
		at := step.at
		world.Lamps = nil
		world.Lamp(emit.Lit{At: maths.V(4+step.lamp*3, 8, 4), Strength: maths.RGB(40, 10+step.lamp*80, 2)})
		if step.moveVolume {
			world.Volumes[0].Motion = thing.Still(maths.Subspace{Origin: maths.V(-1, 1, 0), Scale: 1.3, Orientation: maths.About(maths.ZAxis, .2)})
		}
		direct, _, err := SampledAccelerated(world, eye, lens, at, 96, 64, 2)
		if err != nil {
			t.Fatal(err)
		}
		cached, spent, err := SampledCached(world, eye, lens, at, 96, 64, 2)
		if err != nil {
			t.Fatal(err)
		}
		var errorSum, energy, changes float64
		now := make([]float32, len(cached.Pix))
		for i, p := range direct.Pix {
			q := cached.Pix[i]
			for j, v := range []float32{p.R, p.G, p.B} {
				w := []float32{q.R, q.G, q.B}[j]
				if math.IsNaN(float64(w)) || math.IsInf(float64(w), 0) || w < 0 {
					t.Fatal("invalid cached radiance")
				}
				errorSum += math.Abs(float64(v - w))
				energy += math.Abs(float64(v))
			}
			now[i] = q.G
			if previous != nil {
				changes += math.Abs(float64(q.G - previous[i]))
			}
		}
		relative := errorSum / energy
		t.Logf("at %v relative radiance error %.6f; cache %v trace %v", at, relative, spent.LightingGPU, spent.TraceGPU)
		if energy <= 0 || relative > .02 {
			t.Fatalf("cache changes radiance by %.6f", relative)
		}
		if previous != nil && changes == 0 {
			t.Fatal("changed lamp or volume did not change smoke")
		}
		previous = now
		if spent.LightingGPU <= 0 || spent.LightingWall < spent.LightingGPU {
			t.Fatalf("missing cache accounting: %+v", spent)
		}
	}
}

func TestLightingCacheLeavesCustomSmokeTexturesDirect(t *testing.T) {
	for _, shine := range []bool{false, true} {
		world, eye := fire(t)
		material := world.Volumes[1].Material
		if shine {
			material.Shine = banded{}
		} else {
			material.Texture = banded{}
		}
		encoded, err := encode(world, eye, film.Pinhole{}, 0, 4, 4)
		if err != nil {
			t.Fatal(err)
		}
		encoded.reserveLighting()
		if len(encoded.smokes) != 0 {
			t.Fatal("custom smoke material was cached")
		}
	}
}
