//go:build darwin

package gpu

import (
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
)

func TestSmokeLightingGridResolutionStudy(t *testing.T) {
	world, eye := fire(t)
	world.Volumes = world.Volumes[1:]
	world.Things[0].Material.Colour = maths.Light{}
	world.Lamps = nil
	world.Lamp(emit.Lit{At: maths.V(4, 8, 4), Strength: maths.RGB(40, 10, 2)})
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	direct, _, err := SampledAccelerated(world, eye, lens, 0, 96, 64, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, cells := range []int{32, 64, 65, 128} {
		cached, _, err := sampledLightingGrid(world, nil, eye, lens, 0, 96, 64, 2, cachedTraversal, cells)
		if err != nil {
			t.Fatal(err)
		}
		energy, apart, peak, worst := 0.0, 0.0, 0.0, 0.0
		for i, p := range direct.Pix {
			q := cached.Pix[i]
			for j, v := range []float32{p.R, p.G, p.B} {
				w := []float32{q.R, q.G, q.B}[j]
				if !(w >= 0) || math.IsInf(float64(w), 0) {
					t.Fatal("invalid cached radiance")
				}
				energy += math.Abs(float64(v))
				difference := math.Abs(float64(v - w))
				apart += difference
				peak = max(peak, float64(v))
				worst = max(worst, difference)
			}
		}
		if !(energy > 0 && peak > 0) {
			t.Fatal("unlit resolution fixture")
		}
		t.Logf("%d cells: summed-relative %.9g, worst/peak %.9g", cells, apart/energy, worst/peak)
		// The production 64-cell choice and finer controls retain the
		// existing two-percent fixture budget. 32 is a diagnostic comparison.
		if cells >= 64 && !(apart/energy <= .02) {
			t.Fatalf("%d-cell error %.9g", cells, apart/energy)
		}
	}
}
